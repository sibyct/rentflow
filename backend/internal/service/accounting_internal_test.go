package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// fakeLedgerRepo implements only the LedgerRepository methods these
// tests exercise; embedding the interface means any other call panics
// with a nil dereference, which flags an unexpected dependency loudly.
type fakeLedgerRepo struct {
	domain.LedgerRepository

	settings   *domain.AccountingSettings
	leases     []*domain.LeaseBillingInfo
	candidates []*domain.LateFeeCandidate

	// existing keys (lease|period|type) already in the "database", so
	// InsertGenerated can honor the idempotency index.
	existing map[string]bool
	inserted []*domain.Transaction

	workOrderRow *domain.TransactionRow // what GetTransactionRowByWorkOrder returns; nil = not found
	created      []*domain.Transaction
	updated      []*domain.Transaction
	voided       []uuid.UUID
	templates    []*domain.Transaction
}

func (f *fakeLedgerRepo) GetSettings(_ context.Context, ownerID uuid.UUID) (*domain.AccountingSettings, error) {
	if f.settings != nil {
		return f.settings, nil
	}
	return domain.DefaultAccountingSettings(ownerID), nil
}

func (f *fakeLedgerRepo) ListLeasesToBill(context.Context, uuid.UUID, time.Time) ([]*domain.LeaseBillingInfo, error) {
	return f.leases, nil
}

func (f *fakeLedgerRepo) ListLateFeeCandidates(context.Context, uuid.UUID, time.Time, int) ([]*domain.LateFeeCandidate, error) {
	return f.candidates, nil
}

func txKey(t *domain.Transaction) string {
	if t.LeaseID != nil && t.Period != nil {
		return t.LeaseID.String() + "|" + t.Period.Format("2006-01-02") + "|" + string(t.Type)
	}
	if t.RecurrenceParentID != nil {
		return t.RecurrenceParentID.String() + "|" + t.IncurredOn.Format("2006-01-02")
	}
	return t.ID.String()
}

func (f *fakeLedgerRepo) InsertGenerated(_ context.Context, ts []*domain.Transaction, _ uuid.UUID) ([]*domain.Transaction, error) {
	if f.existing == nil {
		f.existing = map[string]bool{}
	}
	var out []*domain.Transaction
	for _, t := range ts {
		k := txKey(t)
		if f.existing[k] {
			continue
		}
		f.existing[k] = true
		out = append(out, t)
		f.inserted = append(f.inserted, t)
	}
	return out, nil
}

func (f *fakeLedgerRepo) GetTransactionRowByWorkOrder(context.Context, uuid.UUID) (*domain.TransactionRow, error) {
	if f.workOrderRow == nil {
		return nil, domain.ErrNotFound
	}
	return f.workOrderRow, nil
}

func (f *fakeLedgerRepo) CreateTransaction(_ context.Context, t *domain.Transaction, _ *domain.TransactionPayment, _ []domain.AuditEntry) error {
	f.created = append(f.created, t)
	return nil
}

func (f *fakeLedgerRepo) UpdateTransaction(_ context.Context, t *domain.Transaction, _ domain.AuditEntry) error {
	f.updated = append(f.updated, t)
	return nil
}

func (f *fakeLedgerRepo) VoidTransaction(_ context.Context, id uuid.UUID, _ time.Time, _ domain.AuditEntry) error {
	f.voided = append(f.voided, id)
	return nil
}

func (f *fakeLedgerRepo) ListRecurringTemplates(context.Context, uuid.UUID) ([]*domain.Transaction, error) {
	return f.templates, nil
}

func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func fixedClock(t time.Time) clock { return func() time.Time { return t } }

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestAllocatePayment(t *testing.T) {
	actor := uuid.New()
	row := func(amount, paid int64) *domain.TransactionRow {
		return &domain.TransactionRow{Transaction: domain.Transaction{ID: uuid.New(), AmountCents: amount}, PaidCents: paid}
	}
	rent, lateFee, charge := row(150000, 0), row(7500, 0), row(20000, 5000)
	open := []*domain.TransactionRow{rent, lateFee, charge}
	input := func(cents int64) domain.RecordPaymentInput {
		return domain.RecordPaymentInput{AmountCents: cents, PaidOn: day(2026, time.September, 10)}
	}

	t.Run("fills oldest first and stops when the money runs out", func(t *testing.T) {
		got, err := allocatePayment(open, actor, input(155000), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("payments = %d, want 2", len(got))
		}
		if got[0].TransactionID != rent.ID || got[0].AmountCents != 150000 {
			t.Errorf("first payment = %+v, want full rent", got[0])
		}
		if got[1].TransactionID != lateFee.ID || got[1].AmountCents != 5000 {
			t.Errorf("second payment = %+v, want 5000 toward the late fee", got[1])
		}
	})

	t.Run("pays exactly the whole balance", func(t *testing.T) {
		got, err := allocatePayment(open, actor, input(150000+7500+15000), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		var sum int64
		for _, p := range got {
			sum += p.AmountCents
		}
		if len(got) != 3 || sum != 172500 {
			t.Errorf("payments = %d totalling %d, want 3 totalling 172500", len(got), sum)
		}
	})

	t.Run("rejects an overpayment", func(t *testing.T) {
		_, err := allocatePayment(open, actor, input(172501), time.Now())
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})

	t.Run("nothing owed rejects any payment", func(t *testing.T) {
		_, err := allocatePayment(nil, actor, input(100), time.Now())
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})
}

func TestRentRoll_GenerateForPeriod(t *testing.T) {
	owner := uuid.New()
	now := day(2026, time.September, 20)
	dueDay31 := 31
	leaseA := &domain.LeaseBillingInfo{LeaseID: uuid.New(), OwnerID: owner, PropertyID: uuid.New(), UnitID: uuid.New(), MonthlyRentCents: 150000}
	leaseB := &domain.LeaseBillingInfo{LeaseID: uuid.New(), OwnerID: owner, PropertyID: uuid.New(), UnitID: uuid.New(), MonthlyRentCents: 90000, RentDueDay: &dueDay31}

	newSvc := func(repo *fakeLedgerRepo) *RentRollService {
		s := NewRentRollService(repo, discardLog())
		s.now = fixedClock(now)
		return s
	}

	t.Run("bills each lease with its due date and is idempotent", func(t *testing.T) {
		repo := &fakeLedgerRepo{leases: []*domain.LeaseBillingInfo{leaseA, leaseB}}
		svc := newSvc(repo)
		period := day(2026, time.September, 1)

		res, err := svc.GenerateForPeriod(context.Background(), owner, period)
		if err != nil {
			t.Fatal(err)
		}
		if res.RentCreated != 2 {
			t.Fatalf("RentCreated = %d, want 2", res.RentCreated)
		}
		byLease := map[uuid.UUID]*domain.Transaction{}
		for _, tx := range repo.inserted {
			byLease[*tx.LeaseID] = tx
		}
		if got := byLease[leaseA.LeaseID].DueOn; !got.Equal(day(2026, time.September, 1)) {
			t.Errorf("lease A due = %s, want the account default (the 1st)", got.Format("2006-01-02"))
		}
		if got := byLease[leaseB.LeaseID].DueOn; !got.Equal(day(2026, time.September, 30)) {
			t.Errorf("lease B due = %s, want the 31st clamped to Sept 30", got.Format("2006-01-02"))
		}
		if byLease[leaseA.LeaseID].AmountCents != 150000 || byLease[leaseA.LeaseID].Source != domain.TransactionSourceAutoRent {
			t.Errorf("lease A row = %+v", byLease[leaseA.LeaseID])
		}

		again, err := svc.GenerateForPeriod(context.Background(), owner, period)
		if err != nil {
			t.Fatal(err)
		}
		if again.RentCreated != 0 || len(repo.inserted) != 2 {
			t.Errorf("second run created %d (total rows %d); generation must be idempotent", again.RentCreated, len(repo.inserted))
		}
	})

	t.Run("refuses a future month", func(t *testing.T) {
		svc := newSvc(&fakeLedgerRepo{})
		_, err := svc.GenerateForPeriod(context.Background(), owner, day(2026, time.October, 1))
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})
}

func TestRentRoll_LateFees(t *testing.T) {
	owner := uuid.New()
	now := day(2026, time.September, 20)
	candidate := func(rent int64, leaseFee *int64) *domain.LateFeeCandidate {
		return &domain.LateFeeCandidate{
			RentTransactionID: uuid.New(), OwnerID: owner, LeaseID: uuid.New(), PropertyID: uuid.New(), UnitID: uuid.New(),
			Period: day(2026, time.September, 1), RentCents: rent, DueOn: day(2026, time.September, 1), GraceDays: 5,
			LeaseLateFeeCents: leaseFee,
		}
	}
	run := func(t *testing.T, settings *domain.AccountingSettings, cs ...*domain.LateFeeCandidate) []*domain.Transaction {
		t.Helper()
		repo := &fakeLedgerRepo{settings: settings, candidates: cs}
		svc := NewRentRollService(repo, discardLog())
		svc.now = fixedClock(now)
		if _, err := svc.GenerateForPeriod(context.Background(), owner, day(2026, time.September, 1)); err != nil {
			t.Fatal(err)
		}
		var fees []*domain.Transaction
		for _, tx := range repo.inserted {
			if tx.Type == domain.TransactionTypeLateFee {
				fees = append(fees, tx)
			}
		}
		return fees
	}
	settings := func(kind domain.LateFeeKind, value int64) *domain.AccountingSettings {
		s := domain.DefaultAccountingSettings(owner)
		s.LateFeeKind, s.LateFeeValue = kind, value
		return s
	}

	t.Run("flat account rule", func(t *testing.T) {
		fees := run(t, settings(domain.LateFeeKindFlat, 5000), candidate(150000, nil))
		if len(fees) != 1 || fees[0].AmountCents != 5000 || fees[0].Source != domain.TransactionSourceAutoLateFee {
			t.Errorf("fees = %+v", fees)
		}
	})
	t.Run("percent account rule", func(t *testing.T) {
		fees := run(t, settings(domain.LateFeeKindPercent, 500), candidate(150000, nil))
		if len(fees) != 1 || fees[0].AmountCents != 7500 {
			t.Errorf("fees = %+v, want one 7500 fee (5%% of 150000)", fees)
		}
	})
	t.Run("lease override beats the account rule", func(t *testing.T) {
		override := int64(2500)
		fees := run(t, settings(domain.LateFeeKindPercent, 500), candidate(150000, &override))
		if len(fees) != 1 || fees[0].AmountCents != 2500 {
			t.Errorf("fees = %+v, want the lease's own 2500", fees)
		}
	})
	t.Run("a zero rule creates no fee", func(t *testing.T) {
		if fees := run(t, settings(domain.LateFeeKindFlat, 0), candidate(150000, nil)); len(fees) != 0 {
			t.Errorf("fees = %+v, want none when the rule is off", fees)
		}
	})
}

func TestExpense_SyncFromWorkOrder(t *testing.T) {
	owner := uuid.New()
	now := day(2026, time.September, 20)
	cost := 250.50
	newWO := func(status domain.WorkOrderStatus, cost *float64) *domain.WorkOrder {
		completed := day(2026, time.September, 18)
		return &domain.WorkOrder{ID: uuid.New(), PropertyID: uuid.New(), Title: "Fix sink", Status: status, ActualCost: cost, CompletedAt: &completed, AssignedTo: "Handy Sam"}
	}
	newSvc := func(repo *fakeLedgerRepo) *ExpenseService {
		s := NewExpenseService(repo, nil, nil, nil, discardLog())
		s.now = fixedClock(now)
		return s
	}
	existing := func(amount, paid int64, voided bool) *domain.TransactionRow {
		row := &domain.TransactionRow{Transaction: domain.Transaction{ID: uuid.New(), AmountCents: amount}, PaidCents: paid}
		if voided {
			at := now
			row.VoidedAt = &at
		}
		return row
	}

	t.Run("completed with a cost creates one expense in cents", func(t *testing.T) {
		repo := &fakeLedgerRepo{}
		wo := newWO(domain.WorkOrderStatusCompleted, &cost)
		if err := newSvc(repo).SyncFromWorkOrder(context.Background(), owner, wo); err != nil {
			t.Fatal(err)
		}
		if len(repo.created) != 1 {
			t.Fatalf("created %d, want 1", len(repo.created))
		}
		got := repo.created[0]
		if got.AmountCents != 25050 || got.Source != domain.TransactionSourceWorkOrder || *got.WorkOrderID != wo.ID {
			t.Errorf("expense = %+v", got)
		}
		if !got.IncurredOn.Equal(day(2026, time.September, 18)) {
			t.Errorf("incurred_on = %s, want the completion date", got.IncurredOn.Format("2006-01-02"))
		}
		if got.VendorName != "Handy Sam" {
			t.Errorf("vendor_name = %q, want the freeform assignee as fallback", got.VendorName)
		}
	})

	t.Run("no cost, or not completed, creates nothing", func(t *testing.T) {
		repo := &fakeLedgerRepo{}
		svc := newSvc(repo)
		zero := 0.0
		for _, wo := range []*domain.WorkOrder{
			newWO(domain.WorkOrderStatusCompleted, nil),
			newWO(domain.WorkOrderStatusCompleted, &zero),
			newWO(domain.WorkOrderStatusInProgress, &cost),
		} {
			if err := svc.SyncFromWorkOrder(context.Background(), owner, wo); err != nil {
				t.Fatal(err)
			}
		}
		if len(repo.created) != 0 {
			t.Errorf("created %d expenses, want 0", len(repo.created))
		}
	})

	t.Run("a changed cost updates an unpaid expense", func(t *testing.T) {
		repo := &fakeLedgerRepo{workOrderRow: existing(10000, 0, false)}
		if err := newSvc(repo).SyncFromWorkOrder(context.Background(), owner, newWO(domain.WorkOrderStatusCompleted, &cost)); err != nil {
			t.Fatal(err)
		}
		if len(repo.updated) != 1 || repo.updated[0].AmountCents != 25050 {
			t.Errorf("updated = %+v, want one update to 25050", repo.updated)
		}
	})

	t.Run("a paid expense is never re-amounted", func(t *testing.T) {
		repo := &fakeLedgerRepo{workOrderRow: existing(10000, 10000, false)}
		if err := newSvc(repo).SyncFromWorkOrder(context.Background(), owner, newWO(domain.WorkOrderStatusCompleted, &cost)); err != nil {
			t.Fatal(err)
		}
		if len(repo.updated) != 0 {
			t.Errorf("updated = %+v, want none: money has already moved", repo.updated)
		}
	})

	t.Run("reopening voids an unpaid expense but not a paid one", func(t *testing.T) {
		unpaid := &fakeLedgerRepo{workOrderRow: existing(25050, 0, false)}
		if err := newSvc(unpaid).SyncFromWorkOrder(context.Background(), owner, newWO(domain.WorkOrderStatusInProgress, &cost)); err != nil {
			t.Fatal(err)
		}
		if len(unpaid.voided) != 1 {
			t.Errorf("voided = %d, want 1", len(unpaid.voided))
		}

		paid := &fakeLedgerRepo{workOrderRow: existing(25050, 25050, false)}
		if err := newSvc(paid).SyncFromWorkOrder(context.Background(), owner, newWO(domain.WorkOrderStatusInProgress, &cost)); err != nil {
			t.Fatal(err)
		}
		if len(paid.voided) != 0 {
			t.Errorf("voided = %d, want 0 for an expense with payments", len(paid.voided))
		}
	})

	t.Run("a voided expense is a tombstone", func(t *testing.T) {
		repo := &fakeLedgerRepo{workOrderRow: existing(25050, 0, true)}
		if err := newSvc(repo).SyncFromWorkOrder(context.Background(), owner, newWO(domain.WorkOrderStatusCompleted, &cost)); err != nil {
			t.Fatal(err)
		}
		if len(repo.created)+len(repo.updated)+len(repo.voided) != 0 {
			t.Errorf("a voided expense must be left alone: %+v", repo)
		}
	})
}

func TestExpense_GenerateRecurring(t *testing.T) {
	owner := uuid.New()
	monthly := domain.RecurrenceMonthly
	quarterly := domain.RecurrenceQuarterly
	due := day(2026, time.January, 15)
	monthlyTpl := &domain.Transaction{
		ID: uuid.New(), OwnerID: owner, PropertyID: uuid.New(), Kind: domain.TransactionKindExpense, Type: domain.TransactionTypeExpense,
		AmountCents: 10000, IncurredOn: day(2026, time.January, 31), DueOn: &due, IsRecurring: true, RecurrenceFrequency: &monthly,
	}
	quarterlyTpl := &domain.Transaction{
		ID: uuid.New(), OwnerID: owner, PropertyID: uuid.New(), Kind: domain.TransactionKindExpense, Type: domain.TransactionTypeExpense,
		AmountCents: 50000, IncurredOn: day(2026, time.January, 1), IsRecurring: true, RecurrenceFrequency: &quarterly,
	}

	repo := &fakeLedgerRepo{templates: []*domain.Transaction{monthlyTpl, quarterlyTpl}}
	svc := NewExpenseService(repo, nil, nil, nil, discardLog())
	svc.now = fixedClock(day(2026, time.April, 30))

	n, err := svc.GenerateRecurring(context.Background(), owner, day(2026, time.April, 30))
	if err != nil {
		t.Fatal(err)
	}
	// Monthly from Jan 31: Feb 28, Mar 31, Apr 30. Quarterly from Jan 1: Apr 1.
	if n != 4 {
		t.Fatalf("generated %d occurrences, want 4", n)
	}

	var monthlyDates []string
	for _, tx := range repo.inserted {
		if *tx.RecurrenceParentID == monthlyTpl.ID {
			monthlyDates = append(monthlyDates, tx.IncurredOn.Format("2006-01-02"))
			if tx.IsRecurring || tx.RecurrenceFrequency != nil {
				t.Errorf("occurrence %s must not itself be a recurring template", tx.IncurredOn.Format("2006-01-02"))
			}
			if tx.Source != domain.TransactionSourceRecurring {
				t.Errorf("source = %s, want recurring", tx.Source)
			}
			if tx.DueOn == nil || !tx.DueOn.Equal(tx.IncurredOn.AddDate(0, 0, -16)) {
				t.Errorf("due date should keep the template's offset from the incurred date, got %v", tx.DueOn)
			}
		}
	}
	want := []string{"2026-02-28", "2026-03-31", "2026-04-30"}
	if len(monthlyDates) != len(want) {
		t.Fatalf("monthly dates = %v, want %v", monthlyDates, want)
	}
	for i := range want {
		if monthlyDates[i] != want[i] {
			t.Errorf("monthly dates = %v, want %v (no month-end drift)", monthlyDates, want)
			break
		}
	}

	again, err := svc.GenerateRecurring(context.Background(), owner, day(2026, time.April, 30))
	if err != nil {
		t.Fatal(err)
	}
	if again != 0 {
		t.Errorf("second run generated %d, want 0 (idempotent)", again)
	}
}

func TestFillSeries(t *testing.T) {
	from := day(2026, time.April, 1)
	to := day(2026, time.September, 30)
	points := []domain.SeriesPoint{
		{Bucket: day(2026, time.June, 1), IncomeCents: 100, ExpenseCents: 40},
		{Bucket: day(2026, time.September, 1), IncomeCents: 300},
	}
	got := fillSeries(points, from, to, "month")
	if len(got) != 6 {
		t.Fatalf("buckets = %d, want 6 (Apr..Sep)", len(got))
	}
	if got[0].IncomeCents != 0 || got[2].IncomeCents != 100 || got[2].ExpenseCents != 40 || got[5].IncomeCents != 300 {
		t.Errorf("series = %+v", got)
	}

	// Weekly buckets start on Monday, matching Postgres date_trunc('week').
	weekly := fillSeries(nil, day(2026, time.September, 1), day(2026, time.September, 30), "week")
	if !weekly[0].Bucket.Equal(day(2026, time.August, 31)) {
		t.Errorf("first week bucket = %s, want Monday Aug 31", weekly[0].Bucket.Format("2006-01-02"))
	}
	if len(weekly) != 5 {
		t.Errorf("weekly buckets = %d, want 5", len(weekly))
	}
}
