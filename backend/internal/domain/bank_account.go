package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BankAccountType string

const (
	BankAccountTypeOperating    BankAccountType = "operating"
	BankAccountTypeDepositTrust BankAccountType = "security_deposit_trust"
)

func (t BankAccountType) Valid() bool {
	return t == BankAccountTypeOperating || t == BankAccountTypeDepositTrust
}

// BankAccountProviderManual is the only provider today: the balance is
// typed in and statement lines are entered or imported by hand. Provider
// and ExternalAccountID exist so a bank-feed integration (Plaid) can
// slot in later by writing the same statement-line rows the
// reconciliation screen already reads — nothing here is Plaid-specific.
const BankAccountProviderManual = "manual"

type BankAccount struct {
	ID                uuid.UUID
	OwnerID           uuid.UUID
	Nickname          string
	BankName          string
	Type              BankAccountType
	PropertyID        *uuid.UUID // nil = portfolio-wide
	BalanceCents      int64
	BalanceAsOf       *time.Time
	Last4             string
	Provider          string
	ExternalAccountID string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// BankAccountRow is a BankAccount plus what the list screen shows.
type BankAccountRow struct {
	BankAccount
	PropertyName          string
	UnmatchedLineCount    int
	UnmatchedPaymentCount int
}

type StatementLineSource string

const (
	StatementLineSourceManual StatementLineSource = "manual"
	StatementLineSourceCSV    StatementLineSource = "csv"
	StatementLineSourcePlaid  StatementLineSource = "plaid"
)

// StatementLine is one line of a bank statement. AmountCents is signed
// from the account's point of view: deposits positive, withdrawals
// negative.
type StatementLine struct {
	ID               uuid.UUID
	OwnerID          uuid.UUID
	BankAccountID    uuid.UUID
	PostedOn         time.Time
	Description      string
	AmountCents      int64
	Source           StatementLineSource
	ExternalID       string
	MatchedPaymentID *uuid.UUID
	MatchedAt        *time.Time
	CreatedAt        time.Time
}

// ReconcilablePayment is a ledger payment that could be matched to a
// statement line: signed positive for money received (income), negative
// for money paid out (expense), so it compares directly to a line.
type ReconcilablePayment struct {
	PaymentID     uuid.UUID
	TransactionID uuid.UUID
	AmountCents   int64
	PaidOn        time.Time
	Method        string
	Reference     string
	Description   string
}

type StatementLineFilter string

const (
	StatementLineAll       StatementLineFilter = ""
	StatementLineMatched   StatementLineFilter = "matched"
	StatementLineUnmatched StatementLineFilter = "unmatched"
)

type StatementLineListOptions struct {
	OwnerID       uuid.UUID
	BankAccountID uuid.UUID
	Filter        StatementLineFilter
	Limit         int
	Offset        int
}

type Reconciliation struct {
	Account                *BankAccount
	MatchedLineCount       int
	UnmatchedLineCount     int
	UnmatchedLinesCents    int64
	UnmatchedPayments      []*ReconcilablePayment
	UnmatchedPaymentsCents int64
}

type BankAccountRepository interface {
	Create(ctx context.Context, a *BankAccount, audit AuditEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*BankAccount, error)
	Update(ctx context.Context, a *BankAccount, audit AuditEntry) error
	Delete(ctx context.Context, id uuid.UUID, audit AuditEntry) error
	ListForOwner(ctx context.Context, ownerID uuid.UUID) ([]*BankAccountRow, error)

	// InsertLines inserts statement lines, skipping any whose
	// (account, external_id) already exists so a re-imported file or a
	// re-polled feed never duplicates. It returns how many were new.
	InsertLines(ctx context.Context, lines []*StatementLine) (int, error)
	GetLine(ctx context.Context, id uuid.UUID) (*StatementLine, error)
	DeleteLine(ctx context.Context, id uuid.UUID) error
	ListLines(ctx context.Context, opts StatementLineListOptions) ([]*StatementLine, int, error)
	// ListUnmatchedPayments returns live payments that are not matched to
	// any statement line and are either recorded against this account or
	// against none yet.
	ListUnmatchedPayments(ctx context.Context, ownerID, accountID uuid.UUID) ([]*ReconcilablePayment, error)
	// Match links a line to a payment atomically and, if the payment had
	// no bank account, assigns it this one.
	Match(ctx context.Context, lineID, paymentID uuid.UUID, at time.Time, audit AuditEntry) error
	Unmatch(ctx context.Context, lineID uuid.UUID, audit AuditEntry) error
	// PaymentSignedAmount returns a payment's signed cents (income +,
	// expense −) and whether it is live and owned by ownerID.
	PaymentSignedAmount(ctx context.Context, paymentID, ownerID uuid.UUID) (cents int64, ok bool, err error)
	LineCounts(ctx context.Context, accountID uuid.UUID) (matched, unmatched int, unmatchedCents int64, err error)
}

type CreateBankAccountInput struct {
	Nickname     string
	BankName     string
	Type         BankAccountType
	PropertyID   *uuid.UUID
	BalanceCents int64
	BalanceAsOf  *time.Time
	Last4        string
}

type UpdateBankAccountInput = CreateBankAccountInput

type StatementLineInput struct {
	PostedOn    time.Time
	Description string
	AmountCents int64
	ExternalID  string
}

type BankAccountService interface {
	Create(ctx context.Context, ownerID uuid.UUID, input CreateBankAccountInput) (*BankAccount, error)
	Get(ctx context.Context, ownerID, id uuid.UUID) (*BankAccount, error)
	Update(ctx context.Context, ownerID, id uuid.UUID, input UpdateBankAccountInput) (*BankAccount, error)
	Delete(ctx context.Context, ownerID, id uuid.UUID) error
	List(ctx context.Context, ownerID uuid.UUID) ([]*BankAccountRow, error)

	AddStatementLines(ctx context.Context, ownerID, accountID uuid.UUID, source StatementLineSource, lines []StatementLineInput) (added int, err error)
	ListStatementLines(ctx context.Context, ownerID uuid.UUID, opts StatementLineListOptions) ([]*StatementLine, int, error)
	DeleteStatementLine(ctx context.Context, ownerID, accountID, lineID uuid.UUID) error
	Match(ctx context.Context, ownerID, accountID, lineID, paymentID uuid.UUID) error
	Unmatch(ctx context.Context, ownerID, accountID, lineID uuid.UUID) error
	Reconciliation(ctx context.Context, ownerID, accountID uuid.UUID) (*Reconciliation, error)
}
