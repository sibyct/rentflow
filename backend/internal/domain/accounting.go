package domain

import (
	"context"
	"math"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// The central ledger. Every receivable (rent, late fee, one-off charge)
// and every expense is one Transaction; every settlement of one is a
// TransactionPayment. Rent Roll, Expenses and Charges are filtered views
// over these two tables, not tables of their own.
//
// All money is integer cents (int64) — never float64 — so sums and
// balances cannot drift. The legacy float64-dollar fields elsewhere
// (Lease.MonthlyRent, WorkOrder.ActualCost) are converted once, at the
// boundary, with DollarsToCents.

type TransactionKind string

const (
	TransactionKindIncome  TransactionKind = "income"
	TransactionKindExpense TransactionKind = "expense"
)

type TransactionType string

const (
	TransactionTypeRent    TransactionType = "rent"
	TransactionTypeLateFee TransactionType = "late_fee"
	TransactionTypeCharge  TransactionType = "charge"
	TransactionTypeExpense TransactionType = "expense"
)

type ChargeType string

const (
	ChargeTypeUtilityRebill ChargeType = "utility_rebill"
	ChargeTypeDamage        ChargeType = "damage"
	ChargeTypeAmenity       ChargeType = "amenity"
	ChargeTypeOther         ChargeType = "other"
)

// Valid deliberately excludes late fees: those are produced by the
// late-fee sweep (TransactionTypeLateFee), never entered by hand.
func (t ChargeType) Valid() bool {
	switch t {
	case ChargeTypeUtilityRebill, ChargeTypeDamage, ChargeTypeAmenity, ChargeTypeOther:
		return true
	default:
		return false
	}
}

type ExpenseCategory string

const (
	ExpenseCategoryRepairs       ExpenseCategory = "repairs"
	ExpenseCategoryUtilities     ExpenseCategory = "utilities"
	ExpenseCategoryInsurance     ExpenseCategory = "insurance"
	ExpenseCategoryPropertyTax   ExpenseCategory = "property_tax"
	ExpenseCategoryManagementFee ExpenseCategory = "management_fee"
	ExpenseCategoryLandscaping   ExpenseCategory = "landscaping"
	ExpenseCategorySupplies      ExpenseCategory = "supplies"
	ExpenseCategoryLegal         ExpenseCategory = "legal"
	ExpenseCategoryMortgage      ExpenseCategory = "mortgage"
	ExpenseCategoryOther         ExpenseCategory = "other"
)

func (c ExpenseCategory) Valid() bool {
	switch c {
	case ExpenseCategoryRepairs, ExpenseCategoryUtilities, ExpenseCategoryInsurance, ExpenseCategoryPropertyTax,
		ExpenseCategoryManagementFee, ExpenseCategoryLandscaping, ExpenseCategorySupplies, ExpenseCategoryLegal,
		ExpenseCategoryMortgage, ExpenseCategoryOther:
		return true
	default:
		return false
	}
}

type RecurrenceFrequency string

const (
	RecurrenceMonthly   RecurrenceFrequency = "monthly"
	RecurrenceQuarterly RecurrenceFrequency = "quarterly"
	RecurrenceYearly    RecurrenceFrequency = "yearly"
)

func (f RecurrenceFrequency) Valid() bool {
	switch f {
	case RecurrenceMonthly, RecurrenceQuarterly, RecurrenceYearly:
		return true
	default:
		return false
	}
}

// Next returns the date one period after from.
func (f RecurrenceFrequency) Next(from time.Time) time.Time {
	switch f {
	case RecurrenceQuarterly:
		return AddMonthsClamped(from, 3)
	case RecurrenceYearly:
		return AddMonthsClamped(from, 12)
	default:
		return AddMonthsClamped(from, 1)
	}
}

type TransactionSource string

const (
	TransactionSourceManual      TransactionSource = "manual"
	TransactionSourceAutoRent    TransactionSource = "auto_rent"
	TransactionSourceAutoLateFee TransactionSource = "auto_late_fee"
	TransactionSourceWorkOrder   TransactionSource = "work_order"
	TransactionSourceRecurring   TransactionSource = "recurring"
)

// Transaction is one ledger row. PropertyID is always set (that is what
// ties it to an owner's portfolio); the other links say what it is
// about: LeaseID for rent/late fees/charges, WorkOrderID + VendorID for
// an expense that came from maintenance work.
type Transaction struct {
	ID                  uuid.UUID
	OwnerID             uuid.UUID
	PropertyID          uuid.UUID
	UnitID              *uuid.UUID
	Kind                TransactionKind
	Type                TransactionType
	ChargeType          *ChargeType
	Category            *ExpenseCategory
	AmountCents         int64
	IncurredOn          time.Time
	DueOn               *time.Time
	Period              *time.Time // first day of the billing month; rent/late fee/charge only
	LeaseID             *uuid.UUID
	WorkOrderID         *uuid.UUID
	VendorID            *uuid.UUID
	VendorName          string // freeform fallback when there is no Vendor record
	Description         string
	TaxDeductible       bool
	IsRecurring         bool
	RecurrenceFrequency *RecurrenceFrequency
	RecurrenceParentID  *uuid.UUID
	AttachmentID        *uuid.UUID
	Source              TransactionSource
	VoidedAt            *time.Time
	CreatedBy           uuid.UUID
	CreatedAt           time.Time
	UpdatedBy           *uuid.UUID
	UpdatedAt           time.Time
}

func (t *Transaction) Voided() bool { return t.VoidedAt != nil }

type TransactionPayment struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	AmountCents   int64
	PaidOn        time.Time
	Method        string
	Reference     string
	BankAccountID *uuid.UUID
	VoidedAt      *time.Time
	RecordedBy    uuid.UUID
	CreatedAt     time.Time
}

// TransactionRow is the read model every list/detail screen shows: a
// Transaction plus its live payment total and the display context
// (property/unit/tenant/vendor names) joined in.
type TransactionRow struct {
	Transaction
	PaidCents      int64
	LastPaidOn     *time.Time
	PropertyName   string
	UnitName       string
	TenantName     string
	VendorCompany  string
	WorkOrderTitle string
}

func (r *TransactionRow) OutstandingCents() int64 {
	if out := r.AmountCents - r.PaidCents; out > 0 {
		return out
	}
	return 0
}

// PaymentStatus is the computed status shown on Rent Roll rows and
// Charges. It is derived, never stored (see the migration header).
type PaymentStatus string

const (
	PaymentStatusPaid    PaymentStatus = "paid"
	PaymentStatusPartial PaymentStatus = "partial"
	PaymentStatusLate    PaymentStatus = "late"
	PaymentStatusUnpaid  PaymentStatus = "unpaid"
)

func (s PaymentStatus) Valid() bool {
	switch s {
	case PaymentStatusPaid, PaymentStatusPartial, PaymentStatusLate, PaymentStatusUnpaid:
		return true
	default:
		return false
	}
}

// ReceivableStatus derives the status of an amount owed to us: paid once
// nothing is outstanding; late once today is past dueOn + graceDays;
// otherwise partial if anything has been paid, else unpaid. Late wins
// over partial — a half-paid rent that is past its grace period is late.
func ReceivableStatus(billedCents, paidCents int64, dueOn time.Time, graceDays int, now time.Time) PaymentStatus {
	if billedCents-paidCents <= 0 {
		return PaymentStatusPaid
	}
	lateAfter := truncateToDate(dueOn).AddDate(0, 0, graceDays)
	if truncateToDate(now).After(lateAfter) {
		return PaymentStatusLate
	}
	if paidCents > 0 {
		return PaymentStatusPartial
	}
	return PaymentStatusUnpaid
}

// IncomeStatus is ReceivableStatus for a single income row (charges use
// no grace period — a charge is late the day after it is due).
func (r *TransactionRow) IncomeStatus(graceDays int, now time.Time) PaymentStatus {
	due := r.IncurredOn
	if r.DueOn != nil {
		due = *r.DueOn
	}
	return ReceivableStatus(r.AmountCents, r.PaidCents, due, graceDays, now)
}

type ExpenseStatus string

const (
	ExpenseStatusUnpaid  ExpenseStatus = "unpaid"
	ExpenseStatusPaid    ExpenseStatus = "paid"
	ExpenseStatusOverdue ExpenseStatus = "overdue"
)

func (s ExpenseStatus) Valid() bool {
	switch s {
	case ExpenseStatusUnpaid, ExpenseStatusPaid, ExpenseStatusOverdue:
		return true
	default:
		return false
	}
}

// ExpenseStatus derives Unpaid/Paid/Overdue. An expense with no due date
// can never be overdue.
func (r *TransactionRow) ExpenseStatus(now time.Time) ExpenseStatus {
	if r.PaidCents >= r.AmountCents {
		return ExpenseStatusPaid
	}
	if r.DueOn != nil && truncateToDate(now).After(truncateToDate(*r.DueOn)) {
		return ExpenseStatusOverdue
	}
	return ExpenseStatusUnpaid
}

// AccountingSettings is the per-account late-fee rule. LateFeeValue is
// cents when LateFeeKind is flat and basis points of the month's rent
// when percent; zero disables late fees.
type LateFeeKind string

const (
	LateFeeKindFlat    LateFeeKind = "flat"
	LateFeeKindPercent LateFeeKind = "percent"
)

func (k LateFeeKind) Valid() bool { return k == LateFeeKindFlat || k == LateFeeKindPercent }

type AccountingSettings struct {
	OwnerID           uuid.UUID
	LateFeeKind       LateFeeKind
	LateFeeValue      int64
	GraceDays         int
	DefaultRentDueDay int
	UpdatedAt         time.Time
}

// DefaultAccountingSettings is what an account gets before it saves any:
// late fees off, 5 days' grace, rent due on the 1st.
func DefaultAccountingSettings(ownerID uuid.UUID) *AccountingSettings {
	return &AccountingSettings{
		OwnerID:           ownerID,
		LateFeeKind:       LateFeeKindFlat,
		LateFeeValue:      0,
		GraceDays:         5,
		DefaultRentDueDay: 1,
	}
}

// LateFeeCents computes the fee for one month's rent under a rule.
func LateFeeCents(kind LateFeeKind, value, rentCents int64) int64 {
	if value <= 0 {
		return 0
	}
	if kind == LateFeeKindPercent {
		return (rentCents*value + 5000) / 10000
	}
	return value
}

// LeaseBillingInfo is what the rent generator needs about one billable
// lease, with the lease-level overrides already resolved to cents.
type LeaseBillingInfo struct {
	LeaseID          uuid.UUID
	OwnerID          uuid.UUID
	PropertyID       uuid.UUID
	UnitID           uuid.UUID
	MonthlyRentCents int64
	RentDueDay       *int
}

// LateFeeCandidate is one rent row that has gone past its grace period
// unpaid and has no late-fee row yet.
type LateFeeCandidate struct {
	RentTransactionID uuid.UUID
	OwnerID           uuid.UUID
	LeaseID           uuid.UUID
	PropertyID        uuid.UUID
	UnitID            uuid.UUID
	Period            time.Time
	RentCents         int64
	DueOn             time.Time
	GraceDays         int
	// LeaseLateFeeCents is the lease's own flat late_fee_amount override,
	// nil when the lease has none (fall back to AccountingSettings).
	LeaseLateFeeCents *int64
}

// RentRollRow is one lease × billing month, aggregated over its rent,
// late-fee and charge rows so charges show up in the balance.
type RentRollRow struct {
	LeaseID      uuid.UUID
	PropertyID   uuid.UUID
	UnitID       uuid.UUID
	PropertyName string
	UnitName     string
	TenantName   string
	Period       time.Time
	RentCents    int64
	DueOn        time.Time
	BilledCents  int64
	PaidCents    int64
	LastPaidOn   *time.Time
	GraceDays    int
}

func (r *RentRollRow) OutstandingCents() int64 {
	if out := r.BilledCents - r.PaidCents; out > 0 {
		return out
	}
	return 0
}

func (r *RentRollRow) Status(now time.Time) PaymentStatus {
	return ReceivableStatus(r.BilledCents, r.PaidCents, r.DueOn, r.GraceDays, now)
}

type RentRollSortKey string

const (
	RentRollSortProperty    RentRollSortKey = "property"
	RentRollSortUnit        RentRollSortKey = "unit"
	RentRollSortDueOn       RentRollSortKey = "due_on"
	RentRollSortOutstanding RentRollSortKey = "outstanding"
)

type RentRollOptions struct {
	OwnerID    uuid.UUID
	PropertyID *uuid.UUID
	Status     *PaymentStatus
	PeriodFrom time.Time // first of month, inclusive
	PeriodTo   time.Time // first of month, inclusive
	Sort       RentRollSortKey
	SortDesc   bool
	Limit      int
	Offset     int
}

type ExpenseSortKey string

const (
	ExpenseSortIncurredOn ExpenseSortKey = "incurred_on"
	ExpenseSortAmount     ExpenseSortKey = "amount"
	ExpenseSortProperty   ExpenseSortKey = "property"
	ExpenseSortCategory   ExpenseSortKey = "category"
)

type ExpenseListOptions struct {
	OwnerID    uuid.UUID
	PropertyID *uuid.UUID
	Category   *ExpenseCategory
	VendorID   *uuid.UUID
	Status     *ExpenseStatus
	From       *time.Time
	To         *time.Time
	Search     string
	Sort       ExpenseSortKey
	SortDesc   bool
	Limit      int
	Offset     int
}

type ChargeListOptions struct {
	OwnerID    uuid.UUID
	PropertyID *uuid.UUID
	Status     *PaymentStatus
	Limit      int
	Offset     int
}

// AuditEntry is one line of the accounting change log.
type FieldChange struct {
	Old any `json:"old"`
	New any `json:"new"`
}

type AuditEntry struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	EntityType string
	EntityID   uuid.UUID
	ActorID    uuid.UUID
	Action     string
	Changes    map[string]FieldChange
	CreatedAt  time.Time
}

const (
	AuditEntityTransaction = "transaction"
	AuditEntityPayment     = "payment"
	AuditEntitySettings    = "settings"
	AuditEntityBankAccount = "bank_account"
	AuditEntityDeposit     = "deposit"
	AuditEntityStatement   = "owner_statement"
)

func NewAuditEntry(ownerID uuid.UUID, entityType string, entityID, actorID uuid.UUID, action string, changes map[string]FieldChange, now time.Time) AuditEntry {
	if changes == nil {
		changes = map[string]FieldChange{}
	}
	return AuditEntry{
		ID:         uuid.New(),
		OwnerID:    ownerID,
		EntityType: entityType,
		EntityID:   entityID,
		ActorID:    actorID,
		Action:     action,
		Changes:    changes,
		CreatedAt:  now,
	}
}

// Diff records every field that differs between before and after, keyed
// by column name. Pointer fields are compared by pointee so re-saving an
// unchanged value produces no noise.
func (t *Transaction) Diff(after *Transaction) map[string]FieldChange {
	changes := map[string]FieldChange{}
	add := func(field string, a, b any) {
		a, b = derefValue(a), derefValue(b)
		if !reflect.DeepEqual(a, b) {
			changes[field] = FieldChange{Old: a, New: b}
		}
	}
	add("property_id", t.PropertyID, after.PropertyID)
	add("unit_id", t.UnitID, after.UnitID)
	add("charge_type", t.ChargeType, after.ChargeType)
	add("category", t.Category, after.Category)
	add("amount_cents", t.AmountCents, after.AmountCents)
	add("incurred_on", dateString(&t.IncurredOn), dateString(&after.IncurredOn))
	add("due_on", dateString(t.DueOn), dateString(after.DueOn))
	add("vendor_id", t.VendorID, after.VendorID)
	add("vendor_name", t.VendorName, after.VendorName)
	add("description", t.Description, after.Description)
	add("tax_deductible", t.TaxDeductible, after.TaxDeductible)
	add("is_recurring", t.IsRecurring, after.IsRecurring)
	add("recurrence_frequency", t.RecurrenceFrequency, after.RecurrenceFrequency)
	add("attachment_id", t.AttachmentID, after.AttachmentID)
	return changes
}

func derefValue(v any) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		return rv.Elem().Interface()
	}
	return v
}

func dateString(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

// DollarsToCents converts a legacy float64 dollar amount to cents.
func DollarsToCents(dollars float64) int64 {
	return int64(math.Round(dollars * 100))
}

// FirstOfMonth returns the first day of t's month (UTC, midnight).
func FirstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// LastOfMonth returns the last day of t's month (UTC, midnight).
func LastOfMonth(t time.Time) time.Time {
	return FirstOfMonth(t).AddDate(0, 1, -1)
}

// AddMonthsClamped adds n months keeping the day of month where it
// exists and clamping to the last day otherwise (Jan 31 + 1 month =
// Feb 28/29, not Mar 3).
func AddMonthsClamped(t time.Time, n int) time.Time {
	first := time.Date(t.Year(), t.Month()+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	last := LastOfMonth(first).Day()
	day := t.Day()
	if day > last {
		day = last
	}
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, time.UTC)
}

// RentDueDate is the date rent for period falls due: dueDay clamped to
// the month's length (a 31st due day in February is the 28th/29th).
func RentDueDate(period time.Time, dueDay int) time.Time {
	first := FirstOfMonth(period)
	last := LastOfMonth(first).Day()
	if dueDay < 1 {
		dueDay = 1
	}
	if dueDay > last {
		dueDay = last
	}
	return time.Date(first.Year(), first.Month(), dueDay, 0, 0, 0, 0, time.UTC)
}

// RecordPaymentInput is one payment against one ledger row, or (for
// RecordLeasePayment) a lump sum allocated across a lease's open rows.
type RecordPaymentInput struct {
	AmountCents   int64
	PaidOn        time.Time
	Method        string
	Reference     string
	BankAccountID *uuid.UUID
}

type DashboardPeriod string

const (
	DashboardPeriodMonth DashboardPeriod = "month"
	DashboardPeriodSixMo DashboardPeriod = "6months"
	DashboardPeriodYTD   DashboardPeriod = "ytd"
)

func (p DashboardPeriod) Valid() bool {
	switch p {
	case DashboardPeriodMonth, DashboardPeriodSixMo, DashboardPeriodYTD:
		return true
	default:
		return false
	}
}

// DashboardTotals are the KPI numbers for the current calendar month,
// on a cash basis: income is money actually received, expenses are money
// actually paid out.
type DashboardTotals struct {
	CollectedCents       int64 // payments received this month
	ExpectedCents        int64 // income billed with a due date this month
	OutstandingCents     int64 // all open receivables due on/before today
	ExpensesPaidCents    int64 // expense payments made this month
	UpcomingExpenseCents int64 // unpaid expenses due within 30 days (incl. overdue)
	UpcomingExpenseCount int
	OverdueExpenseCount  int
	LateRentCount        int
}

type SeriesPoint struct {
	Bucket       time.Time
	IncomeCents  int64
	ExpenseCents int64
}

type AccountingDashboard struct {
	Totals         DashboardTotals
	NetIncomeCents int64
	Series         []SeriesPoint
	Period         DashboardPeriod
}

type LedgerRepository interface {
	// Writes. Each is atomic with the audit rows passed alongside it.
	CreateTransaction(ctx context.Context, t *Transaction, payment *TransactionPayment, audits []AuditEntry) error
	// InsertGenerated inserts rows produced by the rent / late-fee /
	// recurring generators, skipping any that collide with the
	// idempotency unique indexes, and returns only those actually
	// inserted. Audit entries are written for the inserted rows.
	InsertGenerated(ctx context.Context, ts []*Transaction, actorID uuid.UUID) ([]*Transaction, error)
	UpdateTransaction(ctx context.Context, t *Transaction, audit AuditEntry) error
	VoidTransaction(ctx context.Context, id uuid.UUID, at time.Time, audit AuditEntry) error
	AddPayments(ctx context.Context, payments []*TransactionPayment, audits []AuditEntry) error
	VoidPayment(ctx context.Context, id uuid.UUID, at time.Time, audit AuditEntry) error

	GetTransaction(ctx context.Context, id uuid.UUID) (*Transaction, error)
	GetTransactionRow(ctx context.Context, id uuid.UUID) (*TransactionRow, error)
	// GetTransactionRowByWorkOrder returns the (possibly voided) expense
	// auto-created for a work order, or ErrNotFound when there is none.
	GetTransactionRowByWorkOrder(ctx context.Context, workOrderID uuid.UUID) (*TransactionRow, error)
	// GetActiveLeaseIDForUnit returns the unit's active lease, or
	// ErrNotFound — what a one-off charge attaches to.
	GetActiveLeaseIDForUnit(ctx context.Context, unitID uuid.UUID) (uuid.UUID, error)
	// BankAccountOwnedBy / AttachmentOwnedBy back the ownership check on
	// an id a client supplies inside a ledger write (IDOR safety).
	BankAccountOwnedBy(ctx context.Context, id, ownerID uuid.UUID) (bool, error)
	AttachmentOwnedBy(ctx context.Context, id, ownerID uuid.UUID) (bool, error)
	GetPayment(ctx context.Context, id uuid.UUID) (*TransactionPayment, error)
	ListPayments(ctx context.Context, transactionID uuid.UUID) ([]*TransactionPayment, error)
	// ListOpenIncomeForLease returns a lease's income rows that still
	// have an outstanding balance, oldest due first (rent, then late
	// fee, then charges within a due date) — the order a lump-sum
	// payment is allocated in.
	ListOpenIncomeForLease(ctx context.Context, leaseID uuid.UUID) ([]*TransactionRow, error)

	ListRentRoll(ctx context.Context, opts RentRollOptions) ([]*RentRollRow, int, error)
	ListExpenses(ctx context.Context, opts ExpenseListOptions) ([]*TransactionRow, int, error)
	ListCharges(ctx context.Context, opts ChargeListOptions) ([]*TransactionRow, int, error)

	// ListLeasesToBill returns the owner's active leases that overlap
	// the month starting at period and have a positive monthly rent.
	ListLeasesToBill(ctx context.Context, ownerID uuid.UUID, period time.Time) ([]*LeaseBillingInfo, error)
	ListLateFeeCandidates(ctx context.Context, ownerID uuid.UUID, today time.Time, defaultGraceDays int) ([]*LateFeeCandidate, error)
	// ListDueRecurring returns recurring-expense templates whose next
	// occurrence is due on or before asOf and has not been generated.
	ListRecurringTemplates(ctx context.Context, ownerID uuid.UUID) ([]*Transaction, error)
	ListOwnerIDsWithActiveLeases(ctx context.Context) ([]uuid.UUID, error)
	// ListAccountIDs returns every account that owns at least one
	// property — who the scheduled jobs iterate over.
	ListAccountIDs(ctx context.Context) ([]uuid.UUID, error)

	GetSettings(ctx context.Context, ownerID uuid.UUID) (*AccountingSettings, error)
	UpsertSettings(ctx context.Context, s *AccountingSettings, audit AuditEntry) error
	ListAudit(ctx context.Context, ownerID uuid.UUID, entityType string, entityID uuid.UUID) ([]*AuditEntry, error)

	DashboardTotals(ctx context.Context, ownerID uuid.UUID, today time.Time) (*DashboardTotals, error)
	DashboardSeries(ctx context.Context, ownerID uuid.UUID, from, to time.Time, bucket string) ([]SeriesPoint, error)
}

type LedgerService interface {
	RecordPayment(ctx context.Context, actorID, transactionID uuid.UUID, input RecordPaymentInput) (*TransactionRow, error)
	RecordLeasePayment(ctx context.Context, actorID, leaseID uuid.UUID, input RecordPaymentInput) ([]*TransactionPayment, error)
	VoidTransaction(ctx context.Context, actorID, id uuid.UUID) error
	VoidPayment(ctx context.Context, actorID, paymentID uuid.UUID) error
	ListTransactionAudit(ctx context.Context, ownerID, transactionID uuid.UUID) ([]*AuditEntry, error)
	ListPayments(ctx context.Context, ownerID, transactionID uuid.UUID) ([]*TransactionPayment, error)
	GetSettings(ctx context.Context, ownerID uuid.UUID) (*AccountingSettings, error)
	UpdateSettings(ctx context.Context, ownerID uuid.UUID, input UpdateAccountingSettingsInput) (*AccountingSettings, error)
	Dashboard(ctx context.Context, ownerID uuid.UUID, period DashboardPeriod) (*AccountingDashboard, error)
}

type UpdateAccountingSettingsInput struct {
	LateFeeKind       LateFeeKind
	LateFeeValue      int64
	GraceDays         int
	DefaultRentDueDay int
}

type RentRollService interface {
	List(ctx context.Context, ownerID uuid.UUID, opts RentRollOptions) ([]*RentRollRow, int, error)
	// GenerateForPeriod creates the rent rows for the month starting at
	// period (and applies any late fees now due). Idempotent.
	GenerateForPeriod(ctx context.Context, ownerID uuid.UUID, period time.Time) (*GenerateResult, error)
}

type GenerateResult struct {
	RentCreated    int
	LateFeeCreated int
}

type CreateExpenseInput struct {
	PropertyID          uuid.UUID
	UnitID              *uuid.UUID
	Category            ExpenseCategory
	VendorID            *uuid.UUID
	VendorName          string
	AmountCents         int64
	IncurredOn          time.Time
	DueOn               *time.Time
	Description         string
	TaxDeductible       bool
	IsRecurring         bool
	RecurrenceFrequency *RecurrenceFrequency
	AttachmentID        *uuid.UUID
	// PaidOn, when set, records a full payment on creation ("Date paid").
	PaidOn        *time.Time
	PaymentMethod string
	BankAccountID *uuid.UUID
}

// UpdateExpenseInput is a full replace of the editable fields (payments
// are managed separately through RecordPayment / VoidPayment).
type UpdateExpenseInput struct {
	PropertyID          uuid.UUID
	UnitID              *uuid.UUID
	Category            ExpenseCategory
	VendorID            *uuid.UUID
	VendorName          string
	AmountCents         int64
	IncurredOn          time.Time
	DueOn               *time.Time
	Description         string
	TaxDeductible       bool
	IsRecurring         bool
	RecurrenceFrequency *RecurrenceFrequency
	AttachmentID        *uuid.UUID
}

// WorkOrderExpenseSyncer is the narrow port WorkOrderService depends on:
// it creates/updates/voids the expense linked to a work order as its
// status/actual cost change. Idempotent and safe to call on every
// work-order write.
type WorkOrderExpenseSyncer interface {
	SyncFromWorkOrder(ctx context.Context, ownerID uuid.UUID, w *WorkOrder) error
}

type ExpenseService interface {
	Create(ctx context.Context, ownerID uuid.UUID, input CreateExpenseInput) (*TransactionRow, error)
	Get(ctx context.Context, ownerID, id uuid.UUID) (*TransactionRow, error)
	List(ctx context.Context, ownerID uuid.UUID, opts ExpenseListOptions) ([]*TransactionRow, int, error)
	Update(ctx context.Context, ownerID, id uuid.UUID, input UpdateExpenseInput) (*TransactionRow, error)
	WorkOrderExpenseSyncer
	// GenerateRecurring materializes due occurrences of recurring
	// expense templates. Idempotent.
	GenerateRecurring(ctx context.Context, ownerID uuid.UUID, asOf time.Time) (int, error)
}

type CreateChargeInput struct {
	UnitID      uuid.UUID
	ChargeType  ChargeType
	AmountCents int64
	Description string
	DueOn       time.Time
}

type ChargeService interface {
	Create(ctx context.Context, ownerID uuid.UUID, input CreateChargeInput) (*TransactionRow, error)
	List(ctx context.Context, ownerID uuid.UUID, opts ChargeListOptions) ([]*TransactionRow, int, error)
}
