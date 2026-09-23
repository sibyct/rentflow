package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestReceivableStatus(t *testing.T) {
	due := date(2026, time.September, 1)
	tests := []struct {
		name         string
		billed, paid int64
		grace        int
		now          time.Time
		want         PaymentStatus
	}{
		{"fully paid", 100000, 100000, 5, date(2026, time.September, 20), PaymentStatusPaid},
		{"overpaid counts as paid", 100000, 120000, 5, date(2026, time.September, 20), PaymentStatusPaid},
		{"unpaid before due", 100000, 0, 5, date(2026, time.August, 30), PaymentStatusUnpaid},
		{"unpaid inside grace", 100000, 0, 5, date(2026, time.September, 6), PaymentStatusUnpaid},
		{"unpaid on last grace day is not yet late", 100000, 0, 5, date(2026, time.September, 6), PaymentStatusUnpaid},
		{"late the day after grace ends", 100000, 0, 5, date(2026, time.September, 7), PaymentStatusLate},
		{"partial inside grace", 100000, 40000, 5, date(2026, time.September, 3), PaymentStatusPartial},
		{"partial past grace is late", 100000, 40000, 5, date(2026, time.September, 20), PaymentStatusLate},
		{"zero grace is late the day after due", 100000, 0, 0, date(2026, time.September, 2), PaymentStatusLate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReceivableStatus(tt.billed, tt.paid, due, tt.grace, tt.now); got != tt.want {
				t.Errorf("ReceivableStatus = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTransactionRow_ExpenseStatus(t *testing.T) {
	now := date(2026, time.September, 15)
	past := date(2026, time.September, 1)
	future := date(2026, time.October, 1)

	tests := []struct {
		name string
		row  TransactionRow
		want ExpenseStatus
	}{
		{"paid in full", TransactionRow{Transaction: Transaction{AmountCents: 500, DueOn: &past}, PaidCents: 500}, ExpenseStatusPaid},
		{"unpaid, due in future", TransactionRow{Transaction: Transaction{AmountCents: 500, DueOn: &future}}, ExpenseStatusUnpaid},
		{"unpaid, no due date is never overdue", TransactionRow{Transaction: Transaction{AmountCents: 500}}, ExpenseStatusUnpaid},
		{"unpaid, past due is overdue", TransactionRow{Transaction: Transaction{AmountCents: 500, DueOn: &past}}, ExpenseStatusOverdue},
		{"partly paid, past due is still overdue", TransactionRow{Transaction: Transaction{AmountCents: 500, DueOn: &past}, PaidCents: 100}, ExpenseStatusOverdue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.row.ExpenseStatus(now); got != tt.want {
				t.Errorf("ExpenseStatus = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestLateFeeCents(t *testing.T) {
	tests := []struct {
		name  string
		kind  LateFeeKind
		value int64
		rent  int64
		want  int64
	}{
		{"flat", LateFeeKindFlat, 7500, 150000, 7500},
		{"percent 5%", LateFeeKindPercent, 500, 150000, 7500},
		{"percent rounds half up", LateFeeKindPercent, 333, 100001, 3330},
		{"zero disables", LateFeeKindFlat, 0, 150000, 0},
		{"negative disables", LateFeeKindPercent, -1, 150000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LateFeeCents(tt.kind, tt.value, tt.rent); got != tt.want {
				t.Errorf("LateFeeCents = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRentDueDate(t *testing.T) {
	tests := []struct {
		period time.Time
		day    int
		want   time.Time
	}{
		{date(2026, time.September, 1), 5, date(2026, time.September, 5)},
		{date(2026, time.February, 1), 31, date(2026, time.February, 28)},
		{date(2028, time.February, 1), 31, date(2028, time.February, 29)},
		{date(2026, time.April, 1), 31, date(2026, time.April, 30)},
		{date(2026, time.September, 17), 1, date(2026, time.September, 1)},
		{date(2026, time.September, 1), 0, date(2026, time.September, 1)},
	}
	for _, tt := range tests {
		if got := RentDueDate(tt.period, tt.day); !got.Equal(tt.want) {
			t.Errorf("RentDueDate(%s, %d) = %s, want %s", tt.period.Format("2006-01-02"), tt.day, got.Format("2006-01-02"), tt.want.Format("2006-01-02"))
		}
	}
}

func TestAddMonthsClamped(t *testing.T) {
	tests := []struct {
		from time.Time
		n    int
		want time.Time
	}{
		{date(2026, time.January, 31), 1, date(2026, time.February, 28)},
		{date(2026, time.January, 31), 2, date(2026, time.March, 31)},
		{date(2026, time.November, 15), 3, date(2027, time.February, 15)},
		{date(2028, time.February, 29), 12, date(2029, time.February, 28)},
	}
	for _, tt := range tests {
		if got := AddMonthsClamped(tt.from, tt.n); !got.Equal(tt.want) {
			t.Errorf("AddMonthsClamped(%s, %d) = %s, want %s", tt.from.Format("2006-01-02"), tt.n, got.Format("2006-01-02"), tt.want.Format("2006-01-02"))
		}
	}
}

func TestDollarsToCents(t *testing.T) {
	tests := []struct {
		in   float64
		want int64
	}{
		{0, 0}, {1500, 150000}, {19.99, 1999}, {0.1 + 0.2, 30}, {1234.565, 123457},
	}
	for _, tt := range tests {
		if got := DollarsToCents(tt.in); got != tt.want {
			t.Errorf("DollarsToCents(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestTransaction_Diff(t *testing.T) {
	cat := ExpenseCategoryRepairs
	vendor := uuid.New()
	before := &Transaction{AmountCents: 1000, Category: &cat, VendorID: &vendor, Description: "a", IncurredOn: date(2026, time.September, 1)}

	t.Run("identical produces no changes", func(t *testing.T) {
		sameCat, sameVendor := cat, vendor
		after := *before
		after.Category, after.VendorID = &sameCat, &sameVendor // different pointers, same values
		if changes := before.Diff(&after); len(changes) != 0 {
			t.Errorf("expected no changes, got %v", changes)
		}
	})

	t.Run("records only changed fields", func(t *testing.T) {
		after := *before
		after.AmountCents = 2500
		after.VendorID = nil
		changes := before.Diff(&after)
		if len(changes) != 2 {
			t.Fatalf("expected 2 changes, got %v", changes)
		}
		if changes["amount_cents"].Old != int64(1000) || changes["amount_cents"].New != int64(2500) {
			t.Errorf("amount_cents change = %+v", changes["amount_cents"])
		}
		if changes["vendor_id"].New != nil {
			t.Errorf("vendor_id new = %v, want nil", changes["vendor_id"].New)
		}
	})
}
