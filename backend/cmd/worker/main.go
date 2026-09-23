// Command worker runs RentFlow's scheduled and background jobs: the
// daily rent / late-fee generation, recurring expenses, monthly owner
// statements, and the email outbox drain. It is the same image as the
// API (a second binary, like cmd/migrate), deployed as its own Fly
// process group so a slow job can never stall an API request.
//
// Every job is idempotent: scheduled jobs claim a (job, run key) row in
// worker_runs before running — exactly one machine wins per key — and
// release it if the job fails so the next tick retries. Running two
// workers, or restarting one mid-day, never duplicates work.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"propertymanagement/internal/config"
	"propertymanagement/internal/domain"
	"propertymanagement/internal/infra/mail"
	"propertymanagement/internal/infra/pdf"
	"propertymanagement/internal/repository/postgres"
	"propertymanagement/internal/service"
)

const (
	tickInterval = time.Minute
	outboxBatch  = 20
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pool, err := postgres.NewPool(dbCtx, cfg.DatabaseURL, 5)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	propertyRepo := postgres.NewPropertyRepository(pool)
	unitRepo := postgres.NewUnitRepository(pool)
	vendorRepo := postgres.NewVendorRepository(pool)
	ledgerRepo := postgres.NewLedgerRepository(pool)
	ownerRepo := postgres.NewPropertyOwnerRepository(pool)
	statementRepo := postgres.NewOwnerStatementRepository(pool)
	outboxRepo := postgres.NewEmailOutboxRepository(pool)
	runs := postgres.NewWorkerRunRepository(pool)

	renderer := pdf.NewStatementRenderer()
	rentRoll := service.NewRentRollService(ledgerRepo, log)
	expenses := service.NewExpenseService(ledgerRepo, propertyRepo, unitRepo, vendorRepo, log)
	statements := service.NewOwnerStatementService(statementRepo, ownerRepo, propertyRepo, outboxRepo, renderer, cfg.MailEnabled(), cfg.PublicAppURL, log)

	var dispatcher *service.EmailDispatcher
	if cfg.MailEnabled() {
		mailer := mail.NewSMTP(mail.SMTPConfig{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom, TLS: cfg.SMTPTLS,
		})
		dispatcher = service.NewEmailDispatcher(outboxRepo, statementRepo, renderer, mailer, log)
	} else {
		log.Warn("SMTP_HOST not set; queued emails will wait in the outbox until email is configured")
	}

	w := &worker{log: log, runs: runs, ledger: ledgerRepo, rentRoll: rentRoll, expenses: expenses, statements: statements, dispatcher: dispatcher}
	log.Info("worker started", "tick", tickInterval.String())

	w.tick(ctx) // don't wait a full interval after a (re)start
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("worker stopping")
			return nil
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

type worker struct {
	log        *slog.Logger
	runs       domain.WorkerRunRepository
	ledger     domain.LedgerRepository
	rentRoll   domain.RentRollService
	expenses   domain.ExpenseService
	statements domain.OwnerStatementService
	dispatcher *service.EmailDispatcher
}

func (w *worker) tick(ctx context.Context) {
	now := time.Now().UTC()
	today := now.Format("2006-01-02")

	if w.dispatcher != nil {
		if sent, err := w.dispatcher.DrainOnce(ctx, outboxBatch); err != nil {
			w.log.ErrorContext(ctx, "outbox drain failed", slog.Any("error", err))
		} else if sent > 0 {
			w.log.InfoContext(ctx, "emails sent", slog.Int("count", sent))
		}
	}

	w.once(ctx, "rent-and-late-fees", today, func(ctx context.Context) error {
		owners, err := w.ledger.ListOwnerIDsWithActiveLeases(ctx)
		if err != nil {
			return err
		}
		month := domain.FirstOfMonth(now)
		for _, id := range owners {
			res, err := w.rentRoll.GenerateForPeriod(ctx, id, month)
			if err != nil {
				return fmt.Errorf("owner %s: %w", id, err)
			}
			if res.RentCreated+res.LateFeeCreated > 0 {
				w.log.InfoContext(ctx, "billing generated", slog.String("owner_id", id.String()), slog.Int("rent", res.RentCreated), slog.Int("late_fees", res.LateFeeCreated))
			}
		}
		return nil
	})

	w.once(ctx, "recurring-expenses", today, func(ctx context.Context) error {
		owners, err := w.ledger.ListAccountIDs(ctx)
		if err != nil {
			return err
		}
		for _, id := range owners {
			n, err := w.expenses.GenerateRecurring(ctx, id, now)
			if err != nil {
				return fmt.Errorf("owner %s: %w", id, err)
			}
			if n > 0 {
				w.log.InfoContext(ctx, "recurring expenses generated", slog.String("owner_id", id.String()), slog.Int("count", n))
			}
		}
		return nil
	})

	// Last month's statements, from the 2nd on so the previous month's
	// final payments have had a day to be recorded.
	if now.Day() >= 2 {
		w.once(ctx, "owner-statements", now.Format("2006-01"), func(ctx context.Context) error {
			owners, err := w.ledger.ListAccountIDs(ctx)
			if err != nil {
				return err
			}
			for _, id := range owners {
				n, err := w.statements.GenerateScheduled(ctx, id, now)
				if err != nil {
					return fmt.Errorf("owner %s: %w", id, err)
				}
				if n > 0 {
					w.log.InfoContext(ctx, "scheduled statements generated", slog.String("owner_id", id.String()), slog.Int("count", n))
				}
			}
			return nil
		})
	}
}

// once runs fn at most once per (job, key) across every worker machine.
func (w *worker) once(ctx context.Context, job, key string, fn func(context.Context) error) {
	claimed, err := w.runs.Claim(ctx, job, key)
	if err != nil {
		w.log.ErrorContext(ctx, "claiming job failed", slog.String("job", job), slog.Any("error", err))
		return
	}
	if !claimed {
		return
	}
	w.log.InfoContext(ctx, "job started", slog.String("job", job), slog.String("key", key))
	if err := fn(ctx); err != nil {
		w.log.ErrorContext(ctx, "job failed; will retry next tick", slog.String("job", job), slog.String("key", key), slog.Any("error", err))
		// Release with a fresh context: ctx may be the very thing that was cancelled.
		relCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if relErr := w.runs.Release(relCtx, job, key); relErr != nil {
			w.log.ErrorContext(ctx, "releasing job claim failed", slog.String("job", job), slog.Any("error", relErr))
		}
		return
	}
	w.log.InfoContext(ctx, "job finished", slog.String("job", job), slog.String("key", key))
}
