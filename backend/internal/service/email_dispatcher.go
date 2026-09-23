package service

import (
	"context"
	"fmt"
	"log/slog"

	"propertymanagement/internal/domain"
)

// EmailDispatcher drains the email outbox: claim → build (rendering a
// statement's PDF from its frozen snapshot) → send → mark. Delivery is
// at-least-once: a claim that dies mid-send is retried on the next tick,
// up to MaxEmailAttempts.
type EmailDispatcher struct {
	outbox     domain.EmailOutboxRepository
	statements domain.OwnerStatementRepository
	renderer   domain.StatementRenderer
	mailer     domain.Mailer
	log        *slog.Logger
	now        clock
}

func NewEmailDispatcher(outbox domain.EmailOutboxRepository, statements domain.OwnerStatementRepository, renderer domain.StatementRenderer, mailer domain.Mailer, log *slog.Logger) *EmailDispatcher {
	return &EmailDispatcher{outbox: outbox, statements: statements, renderer: renderer, mailer: mailer, log: log, now: systemClock}
}

// DrainOnce delivers up to limit pending emails and returns how many were sent.
func (d *EmailDispatcher) DrainOnce(ctx context.Context, limit int) (int, error) {
	pending, err := d.outbox.ClaimPending(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("drain outbox: %w", err)
	}
	sent := 0
	for _, e := range pending {
		if err := d.deliver(ctx, e); err != nil {
			final := e.Attempts >= domain.MaxEmailAttempts
			d.log.WarnContext(ctx, "email delivery failed", slog.String("email_id", e.ID.String()), slog.Int("attempt", e.Attempts), slog.Bool("final", final), slog.Any("error", err))
			if markErr := d.outbox.MarkFailed(ctx, e.ID, err.Error(), final); markErr != nil {
				d.log.ErrorContext(ctx, "recording email failure", slog.Any("error", markErr))
			}
			continue
		}
		sent++
	}
	return sent, nil
}

func (d *EmailDispatcher) deliver(ctx context.Context, e *domain.OutboxEmail) error {
	msg := domain.Email{To: e.To, Subject: e.Subject, HTML: e.HTML}

	var statement *domain.OwnerStatementRow
	if e.StatementID != nil {
		st, err := d.statements.GetByID(ctx, *e.StatementID)
		if err != nil {
			return fmt.Errorf("load statement: %w", err)
		}
		pdf, err := d.renderer.RenderOwnerStatement(&st.Snapshot)
		if err != nil {
			return fmt.Errorf("render statement pdf: %w", err)
		}
		msg.Attachments = append(msg.Attachments, domain.EmailAttachment{Filename: statementFilename(st), ContentType: "application/pdf", Data: pdf})
		statement = st
	}

	if err := d.mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	now := d.now()
	if err := d.outbox.MarkSent(ctx, e.ID, now); err != nil {
		return err
	}
	if statement != nil {
		audit := domain.NewAuditEntry(statement.OwnerID, domain.AuditEntityStatement, statement.ID, statement.OwnerID, "emailed", map[string]domain.FieldChange{
			"to": {New: e.To},
		}, now)
		if err := d.statements.MarkSent(ctx, statement.ID, now, audit); err != nil {
			return err
		}
	}
	return nil
}
