package service

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

const maxStatementDays = 366

// OwnerStatementService generates owner statements: for one property and
// period it pulls the cash-basis income (payments received) and expenses
// (payments made) from the ledger, deducts the owner's management fee,
// and freezes the result into an immutable snapshot. Everything shown
// afterwards — on screen, in the PDF, in the email — comes from that
// snapshot only.
type OwnerStatementService struct {
	repo         domain.OwnerStatementRepository
	ownerRepo    domain.PropertyOwnerRepository
	propertyRepo domain.PropertyRepository
	outbox       domain.EmailOutboxRepository
	renderer     domain.StatementRenderer
	mailEnabled  bool
	appURL       string
	log          *slog.Logger
	now          clock
}

func NewOwnerStatementService(
	repo domain.OwnerStatementRepository,
	ownerRepo domain.PropertyOwnerRepository,
	propertyRepo domain.PropertyRepository,
	outbox domain.EmailOutboxRepository,
	renderer domain.StatementRenderer,
	mailEnabled bool,
	appURL string,
	log *slog.Logger,
) *OwnerStatementService {
	return &OwnerStatementService{
		repo: repo, ownerRepo: ownerRepo, propertyRepo: propertyRepo, outbox: outbox, renderer: renderer,
		mailEnabled: mailEnabled, appURL: appURL, log: log, now: systemClock,
	}
}

var _ domain.OwnerStatementService = (*OwnerStatementService)(nil)

func sumItems(items []domain.StatementLineItem) int64 {
	var total int64
	for _, i := range items {
		total += i.AmountCents
	}
	return total
}

func (s *OwnerStatementService) Generate(ctx context.Context, ownerID, propertyID uuid.UUID, start, end time.Time) (*domain.OwnerStatementRow, error) {
	now := s.now()
	start, end = dateOf(start), dateOf(end)
	switch {
	case start.IsZero() || end.IsZero() || end.Before(start):
		return nil, fmt.Errorf("generate statement: %w", validationError("period_end", "must be on or after the start date"))
	case end.After(dateOf(now)):
		return nil, fmt.Errorf("generate statement: %w", validationError("period_end", "cannot be in the future"))
	case end.Sub(start) > maxStatementDays*24*time.Hour:
		return nil, fmt.Errorf("generate statement: %w", validationError("period_end", "a statement covers at most a year"))
	}

	prop, err := ownedProperty(ctx, s.propertyRepo, propertyID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("generate statement: %w", err)
	}
	owner, err := s.ownerRepo.GetForProperty(ctx, propertyID)
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("generate statement: %w", validationError("property_id", "assign an owner to this property first"))
		}
		return nil, fmt.Errorf("generate statement: %w", err)
	}

	income, expenses, err := s.repo.CollectPeriod(ctx, ownerID, propertyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("generate statement: %w", err)
	}
	incomeTotal, expenseTotal := sumItems(income), sumItems(expenses)
	fee := (incomeTotal*int64(owner.ManagementFeeBps) + 5000) / 10000

	snap := domain.StatementSnapshot{
		OwnerName: owner.Name, OwnerEmail: owner.Email, PropertyName: prop.Name, PropertyAddress: prop.FormattedAddress(),
		PeriodStart: start.Format("2006-01-02"), PeriodEnd: end.Format("2006-01-02"), GeneratedAt: now, FeeBps: owner.ManagementFeeBps,
		Income: income, Expenses: expenses, IncomeCents: incomeTotal, ExpensesCents: expenseTotal, FeeCents: fee,
		NetPayoutCents: incomeTotal - expenseTotal - fee,
	}
	ownerRef := owner.ID
	st := &domain.OwnerStatement{
		ID: uuid.New(), OwnerID: ownerID, PropertyOwnerID: &ownerRef, PropertyID: propertyID, PeriodStart: start, PeriodEnd: end,
		Status: domain.StatementStatusDraft, Snapshot: snap, GeneratedAt: now,
	}
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityStatement, st.ID, ownerID, "generated", map[string]domain.FieldChange{
		"period":         {New: snap.PeriodStart + " to " + snap.PeriodEnd},
		"income_cents":   {New: incomeTotal},
		"expenses_cents": {New: expenseTotal},
		"fee_cents":      {New: fee},
		"net_cents":      {New: snap.NetPayoutCents},
	}, now)
	if err := s.repo.CreateOrReplaceDraft(ctx, st, audit); err != nil {
		return nil, fmt.Errorf("generate statement: %w", err)
	}
	return s.repo.GetByID(ctx, st.ID)
}

func (s *OwnerStatementService) owned(ctx context.Context, ownerID, id uuid.UUID) (*domain.OwnerStatementRow, error) {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if st.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return st, nil
}

func (s *OwnerStatementService) Get(ctx context.Context, ownerID, id uuid.UUID) (*domain.OwnerStatementRow, error) {
	st, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("get statement %s: %w", id, err)
	}
	return st, nil
}

func (s *OwnerStatementService) List(ctx context.Context, ownerID uuid.UUID, opts domain.OwnerStatementListOptions) ([]*domain.OwnerStatementRow, int, error) {
	opts.OwnerID = ownerID
	opts.Limit, opts.Offset = clampPage(opts.Limit, opts.Offset)
	rows, total, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list statements: %w", err)
	}
	return rows, total, nil
}

func statementFilename(st *domain.OwnerStatementRow) string {
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, st.Snapshot.PropertyName)
	return fmt.Sprintf("owner-statement-%s-%s-to-%s.pdf", strings.Trim(name, "-"), st.Snapshot.PeriodStart, st.Snapshot.PeriodEnd)
}

func (s *OwnerStatementService) PDF(ctx context.Context, ownerID, id uuid.UUID) (string, []byte, error) {
	st, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return "", nil, fmt.Errorf("statement pdf %s: %w", id, err)
	}
	data, err := s.renderer.RenderOwnerStatement(&st.Snapshot)
	if err != nil {
		return "", nil, fmt.Errorf("statement pdf %s: %w", id, err)
	}
	return statementFilename(st), data, nil
}

// Send queues the statement's delivery email. Delivery itself happens in
// the worker (so a slow or down SMTP server never fails this request),
// and the statement only becomes Sent once the email has really gone out.
func (s *OwnerStatementService) Send(ctx context.Context, ownerID, id uuid.UUID) (*domain.OwnerStatementRow, error) {
	st, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("send statement %s: %w", id, err)
	}
	if !s.mailEnabled {
		return nil, fmt.Errorf("send statement %s: email is not configured: %w", id, domain.ErrUnavailable)
	}
	if st.Status == domain.StatementStatusPaid {
		return nil, fmt.Errorf("send statement %s: %w", id, domain.ErrConflict)
	}
	if st.Snapshot.OwnerEmail == "" {
		return nil, fmt.Errorf("send statement %s: %w", id, validationError("owner_email", "this owner has no email address on file"))
	}
	if st.EmailQueued {
		return st, nil
	}
	if err := s.enqueue(ctx, st); err != nil {
		return nil, fmt.Errorf("send statement %s: %w", id, err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *OwnerStatementService) enqueue(ctx context.Context, st *domain.OwnerStatementRow) error {
	snap := st.Snapshot
	period := snap.PeriodStart + " to " + snap.PeriodEnd
	body := fmt.Sprintf(`<p>Hello %s,</p>
<p>Your owner statement for <strong>%s</strong> (%s) is attached.</p>
<table cellpadding="4" style="border-collapse:collapse">
<tr><td>Income collected</td><td align="right">%s</td></tr>
<tr><td>Expenses paid</td><td align="right">%s</td></tr>
<tr><td>Management fee (%s)</td><td align="right">%s</td></tr>
<tr><td><strong>Net payout</strong></td><td align="right"><strong>%s</strong></td></tr>
</table>
<p>Reply to this email with any questions.</p>`,
		html.EscapeString(snap.OwnerName), html.EscapeString(snap.PropertyName), period,
		money(snap.IncomeCents), money(snap.ExpensesCents), feePercent(snap.FeeBps), money(snap.FeeCents), money(snap.NetPayoutCents))

	id := st.ID
	return s.outbox.Enqueue(ctx, &domain.OutboxEmail{
		ID: uuid.New(), OwnerID: st.OwnerID, To: snap.OwnerEmail,
		Subject:     fmt.Sprintf("Owner statement — %s — %s", snap.PropertyName, period),
		HTML:        body,
		StatementID: &id,
		CreatedAt:   s.now(),
	})
}

func money(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := fmt.Sprintf("%d", cents/100)
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return fmt.Sprintf("%s$%s.%02d", sign, b.String(), cents%100)
}

func feePercent(bps int) string { return fmt.Sprintf("%.2f%%", float64(bps)/100) }

func (s *OwnerStatementService) MarkSent(ctx context.Context, ownerID, id uuid.UUID) (*domain.OwnerStatementRow, error) {
	if _, err := s.owned(ctx, ownerID, id); err != nil {
		return nil, fmt.Errorf("mark statement %s sent: %w", id, err)
	}
	now := s.now()
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityStatement, id, ownerID, "marked_sent", nil, now)
	if err := s.repo.MarkSent(ctx, id, now, audit); err != nil {
		return nil, fmt.Errorf("mark statement %s sent: %w", id, err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *OwnerStatementService) MarkPaid(ctx context.Context, ownerID, id uuid.UUID, in domain.MarkStatementPaidInput) (*domain.OwnerStatementRow, error) {
	if _, err := s.owned(ctx, ownerID, id); err != nil {
		return nil, fmt.Errorf("mark statement %s paid: %w", id, err)
	}
	in.Method = strings.TrimSpace(in.Method)
	in.Reference = strings.TrimSpace(in.Reference)
	var verrs domain.ValidationErrors
	if in.AmountCents <= 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "amount_cents", Message: "must be greater than zero"})
	}
	if in.PaidOn.IsZero() {
		verrs = append(verrs, &domain.ValidationError{Field: "paid_on", Message: "is required"})
	}
	if len(in.Method) > 40 || len(in.Reference) > 120 {
		verrs = append(verrs, &domain.ValidationError{Field: "reference", Message: "is too long"})
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("mark statement %s paid: %w", id, verrs)
	}
	in.PaidOn = dateOf(in.PaidOn)
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityStatement, id, ownerID, "marked_paid", map[string]domain.FieldChange{
		"paid_cents": {New: in.AmountCents}, "paid_on": {New: in.PaidOn.Format("2006-01-02")}, "method": {New: in.Method}, "reference": {New: in.Reference},
	}, s.now())
	if err := s.repo.MarkPaid(ctx, id, in, audit); err != nil {
		return nil, fmt.Errorf("mark statement %s paid: %w", id, err)
	}
	return s.repo.GetByID(ctx, id)
}

// GenerateScheduled creates last month's statement for every property
// whose owner has scheduled statements turned on — skipping any that
// already exist, so it is safe to run repeatedly — and queues the email
// when SMTP is configured.
func (s *OwnerStatementService) GenerateScheduled(ctx context.Context, accountID uuid.UUID, asOf time.Time) (int, error) {
	targets, err := s.ownerRepo.ListAutoStatementTargets(ctx, accountID)
	if err != nil {
		return 0, fmt.Errorf("scheduled statements: %w", err)
	}
	firstThis := domain.FirstOfMonth(asOf)
	start := firstThis.AddDate(0, -1, 0)
	end := firstThis.AddDate(0, 0, -1)

	created := 0
	for _, t := range targets {
		exists, err := s.repo.Exists(ctx, t.PropertyID, start, end)
		if err != nil {
			return created, fmt.Errorf("scheduled statements: %w", err)
		}
		if exists {
			continue
		}
		st, err := s.Generate(ctx, accountID, t.PropertyID, start, end)
		if err != nil {
			s.log.WarnContext(ctx, "scheduled statement failed", slog.String("property_id", t.PropertyID.String()), slog.Any("error", err))
			continue
		}
		created++
		if s.mailEnabled && st.Snapshot.OwnerEmail != "" {
			if err := s.enqueue(ctx, st); err != nil {
				s.log.WarnContext(ctx, "queueing scheduled statement email failed", slog.String("statement_id", st.ID.String()), slog.Any("error", err))
			}
		}
	}
	return created, nil
}
