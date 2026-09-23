package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// LedgerService owns everything that changes an existing ledger row —
// payments, voids, the late-fee rule — plus the dashboard rollup. There
// is one account per user today, so actorID and ownerID are the same
// uuid; the audit trail still records both roles separately so it
// stays correct if staff accounts are ever added.
type LedgerService struct {
	repo         domain.LedgerRepository
	leaseRepo    domain.LeaseRepository
	unitRepo     domain.UnitRepository
	propertyRepo domain.PropertyRepository
	log          *slog.Logger
	now          clock
}

func NewLedgerService(repo domain.LedgerRepository, leaseRepo domain.LeaseRepository, unitRepo domain.UnitRepository, propertyRepo domain.PropertyRepository, log *slog.Logger) *LedgerService {
	return &LedgerService{repo: repo, leaseRepo: leaseRepo, unitRepo: unitRepo, propertyRepo: propertyRepo, log: log, now: systemClock}
}

var _ domain.LedgerService = (*LedgerService)(nil)

func validatePayment(input domain.RecordPaymentInput) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if input.AmountCents <= 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "amount_cents", Message: "must be greater than zero"})
	}
	if input.PaidOn.IsZero() {
		verrs = append(verrs, &domain.ValidationError{Field: "paid_on", Message: "is required"})
	}
	if len(input.Method) > 40 {
		verrs = append(verrs, &domain.ValidationError{Field: "method", Message: "is too long"})
	}
	if len(input.Reference) > 120 {
		verrs = append(verrs, &domain.ValidationError{Field: "reference", Message: "is too long"})
	}
	return verrs
}

func newPayment(transactionID, actorID uuid.UUID, input domain.RecordPaymentInput, now time.Time) *domain.TransactionPayment {
	return &domain.TransactionPayment{
		ID:            uuid.New(),
		TransactionID: transactionID,
		AmountCents:   input.AmountCents,
		PaidOn:        input.PaidOn,
		Method:        input.Method,
		Reference:     input.Reference,
		BankAccountID: input.BankAccountID,
		RecordedBy:    actorID,
		CreatedAt:     now,
	}
}

// paymentAudit logs a payment on its parent transaction's history, so
// one transaction's change log shows the whole story of that row.
func paymentAudit(ownerID, actorID uuid.UUID, p *domain.TransactionPayment, action string, now time.Time) domain.AuditEntry {
	return domain.NewAuditEntry(ownerID, domain.AuditEntityTransaction, p.TransactionID, actorID, action, map[string]domain.FieldChange{
		"payment_id":   {New: p.ID.String()},
		"amount_cents": {New: p.AmountCents},
		"paid_on":      {New: p.PaidOn.Format("2006-01-02")},
		"method":       {New: p.Method},
	}, now)
}

func (s *LedgerService) RecordPayment(ctx context.Context, actorID, transactionID uuid.UUID, input domain.RecordPaymentInput) (*domain.TransactionRow, error) {
	if verrs := validatePayment(input); len(verrs) > 0 {
		return nil, fmt.Errorf("record payment: %w", verrs)
	}
	t, err := ownedTransaction(ctx, s.repo, transactionID, actorID)
	if err != nil {
		return nil, fmt.Errorf("record payment on %s: %w", transactionID, err)
	}
	if t.Voided() {
		return nil, fmt.Errorf("record payment on %s: %w", transactionID, domain.ErrNotFound)
	}
	if err := checkOwnedRefs(ctx, s.repo, actorID, input.BankAccountID, nil); err != nil {
		return nil, fmt.Errorf("record payment on %s: %w", transactionID, err)
	}

	row, err := s.repo.GetTransactionRow(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("record payment on %s: %w", transactionID, err)
	}
	if input.AmountCents > row.OutstandingCents() {
		return nil, fmt.Errorf("record payment on %s: %w", transactionID,
			validationError("amount_cents", "exceeds the outstanding balance"))
	}

	now := s.now()
	p := newPayment(transactionID, actorID, input, now)
	if err := s.repo.AddPayments(ctx, []*domain.TransactionPayment{p}, []domain.AuditEntry{paymentAudit(actorID, actorID, p, "payment_recorded", now)}); err != nil {
		return nil, fmt.Errorf("record payment on %s: %w", transactionID, err)
	}
	return s.repo.GetTransactionRow(ctx, transactionID)
}

// RecordLeasePayment is the Rent Roll "Record Payment" action: one
// lump sum from a tenant, allocated oldest-due-first across the lease's
// open rent / late-fee / charge rows and written in a single database
// transaction (all rows or none).
func (s *LedgerService) RecordLeasePayment(ctx context.Context, actorID, leaseID uuid.UUID, input domain.RecordPaymentInput) ([]*domain.TransactionPayment, error) {
	if verrs := validatePayment(input); len(verrs) > 0 {
		return nil, fmt.Errorf("record lease payment: %w", verrs)
	}
	if _, _, _, err := ownedLease(ctx, s.leaseRepo, s.unitRepo, s.propertyRepo, leaseID, actorID); err != nil {
		return nil, fmt.Errorf("record lease payment for %s: %w", leaseID, err)
	}
	if err := checkOwnedRefs(ctx, s.repo, actorID, input.BankAccountID, nil); err != nil {
		return nil, fmt.Errorf("record lease payment for %s: %w", leaseID, err)
	}

	open, err := s.repo.ListOpenIncomeForLease(ctx, leaseID)
	if err != nil {
		return nil, fmt.Errorf("record lease payment for %s: %w", leaseID, err)
	}
	payments, err := allocatePayment(open, actorID, input, s.now())
	if err != nil {
		return nil, fmt.Errorf("record lease payment for %s: %w", leaseID, err)
	}

	now := s.now()
	audits := make([]domain.AuditEntry, len(payments))
	for i, p := range payments {
		audits[i] = paymentAudit(actorID, actorID, p, "payment_recorded", now)
	}
	if err := s.repo.AddPayments(ctx, payments, audits); err != nil {
		return nil, fmt.Errorf("record lease payment for %s: %w", leaseID, err)
	}
	return payments, nil
}

// allocatePayment splits input.AmountCents across rows in the order
// given (already oldest-due-first). It rejects a payment larger than the
// total outstanding rather than silently keeping a credit no screen
// could show.
func allocatePayment(open []*domain.TransactionRow, actorID uuid.UUID, input domain.RecordPaymentInput, now time.Time) ([]*domain.TransactionPayment, error) {
	remaining := input.AmountCents
	var payments []*domain.TransactionPayment
	for _, row := range open {
		if remaining == 0 {
			break
		}
		portion := row.OutstandingCents()
		if portion > remaining {
			portion = remaining
		}
		if portion <= 0 {
			continue
		}
		part := input
		part.AmountCents = portion
		payments = append(payments, newPayment(row.ID, actorID, part, now))
		remaining -= portion
	}
	if remaining > 0 {
		return nil, validationError("amount_cents", "exceeds the outstanding balance")
	}
	return payments, nil
}

func (s *LedgerService) VoidTransaction(ctx context.Context, actorID, id uuid.UUID) error {
	t, err := ownedTransaction(ctx, s.repo, id, actorID)
	if err != nil {
		return fmt.Errorf("void transaction %s: %w", id, err)
	}
	if t.Voided() {
		return fmt.Errorf("void transaction %s: %w", id, domain.ErrNotFound)
	}
	row, err := s.repo.GetTransactionRow(ctx, id)
	if err != nil {
		return fmt.Errorf("void transaction %s: %w", id, err)
	}
	if row.PaidCents > 0 {
		return fmt.Errorf("void transaction %s: %w", id,
			validationError("payments", "void the recorded payments first"))
	}
	now := s.now()
	audit := domain.NewAuditEntry(actorID, domain.AuditEntityTransaction, id, actorID, "voided", nil, now)
	if err := s.repo.VoidTransaction(ctx, id, now, audit); err != nil {
		return fmt.Errorf("void transaction %s: %w", id, err)
	}
	return nil
}

func (s *LedgerService) VoidPayment(ctx context.Context, actorID, paymentID uuid.UUID) error {
	p, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("void payment %s: %w", paymentID, err)
	}
	if _, err := ownedTransaction(ctx, s.repo, p.TransactionID, actorID); err != nil {
		return fmt.Errorf("void payment %s: %w", paymentID, err)
	}
	if p.VoidedAt != nil {
		return fmt.Errorf("void payment %s: %w", paymentID, domain.ErrNotFound)
	}
	now := s.now()
	if err := s.repo.VoidPayment(ctx, paymentID, now, paymentAudit(actorID, actorID, p, "payment_voided", now)); err != nil {
		return fmt.Errorf("void payment %s: %w", paymentID, err)
	}
	return nil
}

func (s *LedgerService) ListTransactionAudit(ctx context.Context, ownerID, transactionID uuid.UUID) ([]*domain.AuditEntry, error) {
	if _, err := ownedTransaction(ctx, s.repo, transactionID, ownerID); err != nil {
		return nil, fmt.Errorf("list audit for transaction %s: %w", transactionID, err)
	}
	entries, err := s.repo.ListAudit(ctx, ownerID, domain.AuditEntityTransaction, transactionID)
	if err != nil {
		return nil, fmt.Errorf("list audit for transaction %s: %w", transactionID, err)
	}
	return entries, nil
}

func (s *LedgerService) ListPayments(ctx context.Context, ownerID, transactionID uuid.UUID) ([]*domain.TransactionPayment, error) {
	if _, err := ownedTransaction(ctx, s.repo, transactionID, ownerID); err != nil {
		return nil, fmt.Errorf("list payments for transaction %s: %w", transactionID, err)
	}
	payments, err := s.repo.ListPayments(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("list payments for transaction %s: %w", transactionID, err)
	}
	return payments, nil
}

func (s *LedgerService) GetSettings(ctx context.Context, ownerID uuid.UUID) (*domain.AccountingSettings, error) {
	settings, err := s.repo.GetSettings(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("get accounting settings: %w", err)
	}
	return settings, nil
}

func (s *LedgerService) UpdateSettings(ctx context.Context, ownerID uuid.UUID, input domain.UpdateAccountingSettingsInput) (*domain.AccountingSettings, error) {
	var verrs domain.ValidationErrors
	if !input.LateFeeKind.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "late_fee_kind", Message: "must be flat or percent"})
	}
	if input.LateFeeValue < 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "late_fee_value", Message: "cannot be negative"})
	}
	if input.LateFeeKind == domain.LateFeeKindPercent && input.LateFeeValue > 10000 {
		verrs = append(verrs, &domain.ValidationError{Field: "late_fee_value", Message: "a percentage cannot exceed 100%"})
	}
	if input.GraceDays < 0 || input.GraceDays > 60 {
		verrs = append(verrs, &domain.ValidationError{Field: "grace_days", Message: "must be between 0 and 60"})
	}
	if input.DefaultRentDueDay < 1 || input.DefaultRentDueDay > 31 {
		verrs = append(verrs, &domain.ValidationError{Field: "default_rent_due_day", Message: "must be between 1 and 31"})
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("update accounting settings: %w", verrs)
	}

	before, err := s.repo.GetSettings(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("update accounting settings: %w", err)
	}
	now := s.now()
	after := &domain.AccountingSettings{
		OwnerID:           ownerID,
		LateFeeKind:       input.LateFeeKind,
		LateFeeValue:      input.LateFeeValue,
		GraceDays:         input.GraceDays,
		DefaultRentDueDay: input.DefaultRentDueDay,
		UpdatedAt:         now,
	}
	changes := map[string]domain.FieldChange{}
	if before.LateFeeKind != after.LateFeeKind {
		changes["late_fee_kind"] = domain.FieldChange{Old: string(before.LateFeeKind), New: string(after.LateFeeKind)}
	}
	if before.LateFeeValue != after.LateFeeValue {
		changes["late_fee_value"] = domain.FieldChange{Old: before.LateFeeValue, New: after.LateFeeValue}
	}
	if before.GraceDays != after.GraceDays {
		changes["grace_days"] = domain.FieldChange{Old: before.GraceDays, New: after.GraceDays}
	}
	if before.DefaultRentDueDay != after.DefaultRentDueDay {
		changes["default_rent_due_day"] = domain.FieldChange{Old: before.DefaultRentDueDay, New: after.DefaultRentDueDay}
	}
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntitySettings, ownerID, ownerID, "updated", changes, now)
	if err := s.repo.UpsertSettings(ctx, after, audit); err != nil {
		return nil, fmt.Errorf("update accounting settings: %w", err)
	}
	return after, nil
}

// Dashboard rolls the ledger up for the accounting home: cash-basis KPIs
// for the current month plus an income-vs-expense series for period.
func (s *LedgerService) Dashboard(ctx context.Context, ownerID uuid.UUID, period domain.DashboardPeriod) (*domain.AccountingDashboard, error) {
	if !period.Valid() {
		return nil, fmt.Errorf("accounting dashboard: %w", validationError("period", "must be month, 6months or ytd"))
	}
	today := dateOf(s.now())

	totals, err := s.repo.DashboardTotals(ctx, ownerID, today)
	if err != nil {
		return nil, fmt.Errorf("accounting dashboard: %w", err)
	}

	from, to, bucket := dashboardRange(today, period)
	points, err := s.repo.DashboardSeries(ctx, ownerID, from, to, bucket)
	if err != nil {
		return nil, fmt.Errorf("accounting dashboard: %w", err)
	}

	return &domain.AccountingDashboard{
		Totals:         *totals,
		NetIncomeCents: totals.CollectedCents - totals.ExpensesPaidCents,
		Series:         fillSeries(points, from, to, bucket),
		Period:         period,
	}, nil
}

func dashboardRange(today time.Time, period domain.DashboardPeriod) (from, to time.Time, bucket string) {
	switch period {
	case domain.DashboardPeriodSixMo:
		return domain.FirstOfMonth(today).AddDate(0, -5, 0), domain.LastOfMonth(today), "month"
	case domain.DashboardPeriodYTD:
		return time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, time.UTC), domain.LastOfMonth(today), "month"
	default:
		return domain.FirstOfMonth(today), domain.LastOfMonth(today), "week"
	}
}

// bucketStart normalizes a date to the start of its bucket — the Monday
// of its week, or the first of its month — matching Postgres date_trunc.
func bucketStart(t time.Time, bucket string) time.Time {
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if bucket == "month" {
		return domain.FirstOfMonth(t)
	}
	offset := (int(t.Weekday()) + 6) % 7 // Monday = 0
	return t.AddDate(0, 0, -offset)
}

// fillSeries returns one point per bucket in [from, to], zero-filling
// buckets with no activity so the chart's x-axis is continuous.
func fillSeries(points []domain.SeriesPoint, from, to time.Time, bucket string) []domain.SeriesPoint {
	byBucket := make(map[time.Time]domain.SeriesPoint, len(points))
	for _, p := range points {
		byBucket[bucketStart(p.Bucket, bucket)] = p
	}
	var out []domain.SeriesPoint
	for cur := bucketStart(from, bucket); !cur.After(to); {
		p := byBucket[cur]
		p.Bucket = cur
		out = append(out, p)
		if bucket == "month" {
			cur = cur.AddDate(0, 1, 0)
		} else {
			cur = cur.AddDate(0, 0, 7)
		}
	}
	return out
}
