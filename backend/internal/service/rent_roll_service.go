package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// maxRentRollMonths bounds how wide a date range one Rent Roll request
// can span.
const maxRentRollMonths = 24

// RentRollService turns active leases into monthly rent rows and applies
// late fees. Generation is idempotent (unique indexes on the ledger), so
// the worker, the manual "Generate" button and the lazy trigger on
// listing can all run for the same month safely.
type RentRollService struct {
	repo domain.LedgerRepository
	log  *slog.Logger
	now  clock
}

func NewRentRollService(repo domain.LedgerRepository, log *slog.Logger) *RentRollService {
	return &RentRollService{repo: repo, log: log, now: systemClock}
}

var _ domain.RentRollService = (*RentRollService)(nil)

// GenerateForPeriod bills every active lease for the month starting at
// period, then applies any late fees now due. There is no proration: a
// lease active at any point in the month is billed its full monthly
// rent. Months after the current one cannot be generated.
func (s *RentRollService) GenerateForPeriod(ctx context.Context, ownerID uuid.UUID, period time.Time) (*domain.GenerateResult, error) {
	now := s.now()
	period = domain.FirstOfMonth(period)
	if period.After(domain.FirstOfMonth(now)) {
		return nil, fmt.Errorf("generate rent for %s: %w", monthLabel(period),
			validationError("period", "cannot generate rent for a future month"))
	}

	settings, err := s.repo.GetSettings(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("generate rent: %w", err)
	}
	leases, err := s.repo.ListLeasesToBill(ctx, ownerID, period)
	if err != nil {
		return nil, fmt.Errorf("generate rent: %w", err)
	}

	rows := make([]*domain.Transaction, 0, len(leases))
	for _, l := range leases {
		dueDay := settings.DefaultRentDueDay
		if l.RentDueDay != nil {
			dueDay = *l.RentDueDay
		}
		due := domain.RentDueDate(period, dueDay)
		unitID := l.UnitID
		leaseID := l.LeaseID
		p := period
		rows = append(rows, &domain.Transaction{
			ID:          uuid.New(),
			OwnerID:     ownerID,
			PropertyID:  l.PropertyID,
			UnitID:      &unitID,
			Kind:        domain.TransactionKindIncome,
			Type:        domain.TransactionTypeRent,
			AmountCents: l.MonthlyRentCents,
			IncurredOn:  due,
			DueOn:       &due,
			Period:      &p,
			LeaseID:     &leaseID,
			Description: fmt.Sprintf("Rent for %s", monthLabel(period)),
			Source:      domain.TransactionSourceAutoRent,
			CreatedBy:   ownerID,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	created, err := s.repo.InsertGenerated(ctx, rows, ownerID)
	if err != nil {
		return nil, fmt.Errorf("generate rent: %w", err)
	}

	lateFees, err := s.applyLateFees(ctx, ownerID, settings, now)
	if err != nil {
		return nil, err
	}
	return &domain.GenerateResult{RentCreated: len(created), LateFeeCreated: lateFees}, nil
}

// applyLateFees adds a late-fee row for every rent row past its grace
// period. A lease's own flat late_fee_amount wins over the account rule;
// a zero fee (rule off) creates nothing.
func (s *RentRollService) applyLateFees(ctx context.Context, ownerID uuid.UUID, settings *domain.AccountingSettings, now time.Time) (int, error) {
	today := dateOf(now)
	candidates, err := s.repo.ListLateFeeCandidates(ctx, ownerID, today, settings.GraceDays)
	if err != nil {
		return 0, fmt.Errorf("apply late fees: %w", err)
	}

	var rows []*domain.Transaction
	for _, c := range candidates {
		var fee int64
		if c.LeaseLateFeeCents != nil {
			fee = *c.LeaseLateFeeCents
		} else {
			fee = domain.LateFeeCents(settings.LateFeeKind, settings.LateFeeValue, c.RentCents)
		}
		if fee <= 0 {
			continue
		}
		unitID := c.UnitID
		leaseID := c.LeaseID
		period := c.Period
		due := today
		rows = append(rows, &domain.Transaction{
			ID:          uuid.New(),
			OwnerID:     ownerID,
			PropertyID:  c.PropertyID,
			UnitID:      &unitID,
			Kind:        domain.TransactionKindIncome,
			Type:        domain.TransactionTypeLateFee,
			AmountCents: fee,
			IncurredOn:  today,
			DueOn:       &due,
			Period:      &period,
			LeaseID:     &leaseID,
			Description: fmt.Sprintf("Late fee for %s rent", monthLabel(c.Period)),
			Source:      domain.TransactionSourceAutoLateFee,
			CreatedBy:   ownerID,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	created, err := s.repo.InsertGenerated(ctx, rows, ownerID)
	if err != nil {
		return 0, fmt.Errorf("apply late fees: %w", err)
	}
	return len(created), nil
}

func (s *RentRollService) List(ctx context.Context, ownerID uuid.UUID, opts domain.RentRollOptions) ([]*domain.RentRollRow, int, error) {
	now := s.now()
	currentMonth := domain.FirstOfMonth(now)

	if opts.PeriodFrom.IsZero() {
		opts.PeriodFrom = currentMonth
	}
	if opts.PeriodTo.IsZero() {
		opts.PeriodTo = opts.PeriodFrom
	}
	opts.PeriodFrom = domain.FirstOfMonth(opts.PeriodFrom)
	opts.PeriodTo = domain.FirstOfMonth(opts.PeriodTo)
	if opts.PeriodTo.Before(opts.PeriodFrom) {
		return nil, 0, fmt.Errorf("list rent roll: %w", validationError("period_to", "cannot be before period_from"))
	}
	if opts.PeriodFrom.AddDate(0, maxRentRollMonths, 0).Before(opts.PeriodTo) {
		return nil, 0, fmt.Errorf("list rent roll: %w", validationError("period_to", fmt.Sprintf("range cannot exceed %d months", maxRentRollMonths)))
	}
	opts.OwnerID = ownerID
	opts.Limit, opts.Offset = clampPage(opts.Limit, opts.Offset)

	// Lazy trigger: opening a range that includes the current month
	// makes sure its rent rows exist (and late fees are up to date).
	// Past months are never generated implicitly — that would invent
	// arrears for periods this system never billed; use the explicit
	// Generate action for those. A failure here is logged, not fatal:
	// the roll can still render whatever rows exist.
	if !opts.PeriodFrom.After(currentMonth) && !opts.PeriodTo.Before(currentMonth) {
		if _, err := s.GenerateForPeriod(ctx, ownerID, currentMonth); err != nil {
			s.log.WarnContext(ctx, "rent roll: lazy generation failed", slog.String("owner_id", ownerID.String()), slog.Any("error", err))
		}
	}

	rows, total, err := s.repo.ListRentRoll(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list rent roll: %w", err)
	}
	return rows, total, nil
}
