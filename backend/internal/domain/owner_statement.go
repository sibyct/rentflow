package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PropertyOwner is the real-world owner a managed property is managed
// FOR — distinct from the user account (Property.OwnerID) that manages
// it. Owner statements are addressed to one. Property.OwnerName, the
// freeform text, stays as the fallback for properties not linked yet.
type PropertyOwner struct {
	ID               uuid.UUID
	OwnerID          uuid.UUID // the managing account
	Name             string
	Email            string
	Phone            string
	ManagementFeeBps int // basis points: 800 = 8.00% of collected income
	AutoStatements   bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type PropertyOwnerRow struct {
	PropertyOwner
	PropertyIDs   []uuid.UUID
	PropertyNames []string
}

type PropertyOwnerInput struct {
	Name             string
	Email            string
	Phone            string
	ManagementFeeBps int
	AutoStatements   bool
	PropertyIDs      []uuid.UUID
}

type PropertyOwnerRepository interface {
	Create(ctx context.Context, o *PropertyOwner, propertyIDs []uuid.UUID, audit AuditEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*PropertyOwnerRow, error)
	// Update replaces the owner's fields and the set of properties linked
	// to it, atomically.
	Update(ctx context.Context, o *PropertyOwner, propertyIDs []uuid.UUID, audit AuditEntry) error
	Delete(ctx context.Context, id uuid.UUID, audit AuditEntry) error
	ListForOwner(ctx context.Context, ownerID uuid.UUID) ([]*PropertyOwnerRow, error)
	// GetForProperty returns the owner a property is linked to, or
	// ErrNotFound when it has none.
	GetForProperty(ctx context.Context, propertyID uuid.UUID) (*PropertyOwner, error)
	// ListAutoStatementTargets returns (owner, property) pairs whose owner
	// has scheduled statements turned on.
	ListAutoStatementTargets(ctx context.Context, accountID uuid.UUID) ([]*AutoStatementTarget, error)
}

type AutoStatementTarget struct {
	Owner      *PropertyOwner
	PropertyID uuid.UUID
}

type PropertyOwnerService interface {
	Create(ctx context.Context, ownerID uuid.UUID, input PropertyOwnerInput) (*PropertyOwnerRow, error)
	Get(ctx context.Context, ownerID, id uuid.UUID) (*PropertyOwnerRow, error)
	Update(ctx context.Context, ownerID, id uuid.UUID, input PropertyOwnerInput) (*PropertyOwnerRow, error)
	Delete(ctx context.Context, ownerID, id uuid.UUID) error
	List(ctx context.Context, ownerID uuid.UUID) ([]*PropertyOwnerRow, error)
}

type StatementStatus string

const (
	StatementStatusDraft StatementStatus = "draft"
	StatementStatusSent  StatementStatus = "sent"
	StatementStatusPaid  StatementStatus = "paid"
)

func (s StatementStatus) Valid() bool {
	return s == StatementStatusDraft || s == StatementStatusSent || s == StatementStatusPaid
}

type StatementLineItem struct {
	Date        string `json:"date"`
	Description string `json:"description"`
	AmountCents int64  `json:"amount_cents"`
}

// StatementSnapshot is everything an owner statement shows, frozen at
// generation. A statement is rendered (on screen and to PDF) only from
// this, never re-derived from live data — so editing a payment, an
// expense or the owner's fee later can never change a statement that
// has already gone out.
type StatementSnapshot struct {
	OwnerName       string              `json:"owner_name"`
	OwnerEmail      string              `json:"owner_email"`
	PropertyName    string              `json:"property_name"`
	PropertyAddress string              `json:"property_address"`
	PeriodStart     string              `json:"period_start"`
	PeriodEnd       string              `json:"period_end"`
	GeneratedAt     time.Time           `json:"generated_at"`
	FeeBps          int                 `json:"fee_bps"`
	Income          []StatementLineItem `json:"income"`
	Expenses        []StatementLineItem `json:"expenses"`
	IncomeCents     int64               `json:"income_cents"`
	ExpensesCents   int64               `json:"expenses_cents"`
	FeeCents        int64               `json:"fee_cents"`
	NetPayoutCents  int64               `json:"net_payout_cents"`
}

type OwnerStatement struct {
	ID              uuid.UUID
	OwnerID         uuid.UUID
	PropertyOwnerID *uuid.UUID
	PropertyID      uuid.UUID
	PeriodStart     time.Time
	PeriodEnd       time.Time
	Status          StatementStatus
	Snapshot        StatementSnapshot
	GeneratedAt     time.Time
	SentAt          *time.Time
	PaidCents       *int64
	PaidOn          *time.Time
	PayMethod       string
	PayReference    string
}

type OwnerStatementRow struct {
	OwnerStatement
	PropertyName string
	OwnerName    string
	// EmailQueued is true while a delivery email for it is waiting in the outbox.
	EmailQueued bool
}

type OwnerStatementListOptions struct {
	OwnerID    uuid.UUID
	PropertyID *uuid.UUID
	Status     *StatementStatus
	Limit      int
	Offset     int
}

type MarkStatementPaidInput struct {
	AmountCents int64
	PaidOn      time.Time
	Method      string
	Reference   string
}

type OwnerStatementRepository interface {
	// CreateOrReplaceDraft inserts a statement, replacing an existing
	// draft for the same property and period; an existing sent or paid
	// one is immutable and yields ErrConflict.
	CreateOrReplaceDraft(ctx context.Context, s *OwnerStatement, audit AuditEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*OwnerStatementRow, error)
	Exists(ctx context.Context, propertyID uuid.UUID, start, end time.Time) (bool, error)
	List(ctx context.Context, opts OwnerStatementListOptions) ([]*OwnerStatementRow, int, error)
	MarkSent(ctx context.Context, id uuid.UUID, at time.Time, audit AuditEntry) error
	MarkPaid(ctx context.Context, id uuid.UUID, in MarkStatementPaidInput, audit AuditEntry) error
	// CollectPeriod gathers the cash-basis line items for a property and
	// period: payments received (income) and paid out (expenses) whose
	// payment date falls inside it.
	CollectPeriod(ctx context.Context, ownerID, propertyID uuid.UUID, start, end time.Time) (income, expenses []StatementLineItem, err error)
}

type OwnerStatementService interface {
	Generate(ctx context.Context, ownerID, propertyID uuid.UUID, start, end time.Time) (*OwnerStatementRow, error)
	Get(ctx context.Context, ownerID, id uuid.UUID) (*OwnerStatementRow, error)
	List(ctx context.Context, ownerID uuid.UUID, opts OwnerStatementListOptions) ([]*OwnerStatementRow, int, error)
	// Send queues the statement (with its PDF) for email delivery to the
	// property owner; it becomes Sent when the email actually goes out.
	Send(ctx context.Context, ownerID, id uuid.UUID) (*OwnerStatementRow, error)
	// MarkSent records that the statement was delivered by other means.
	MarkSent(ctx context.Context, ownerID, id uuid.UUID) (*OwnerStatementRow, error)
	MarkPaid(ctx context.Context, ownerID, id uuid.UUID, in MarkStatementPaidInput) (*OwnerStatementRow, error)
	PDF(ctx context.Context, ownerID, id uuid.UUID) (filename string, data []byte, err error)
	// GenerateScheduled creates last month's statement for every property
	// whose owner has scheduled statements on, and queues the email.
	GenerateScheduled(ctx context.Context, accountID uuid.UUID, asOf time.Time) (int, error)
}

// ---- Outbound email ----

type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

type Email struct {
	To          string
	Subject     string
	HTML        string
	Attachments []EmailAttachment
}

// Mailer is the outbound-email port (SMTP adapter in internal/infra/mail).
type Mailer interface {
	Send(ctx context.Context, e Email) error
}

// StatementRenderer renders an owner statement snapshot to a PDF
// (adapter in internal/infra/pdf).
type StatementRenderer interface {
	RenderOwnerStatement(s *StatementSnapshot) ([]byte, error)
}

type OutboxEmail struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	To          string
	Subject     string
	HTML        string
	StatementID *uuid.UUID
	Attempts    int
	CreatedAt   time.Time
}

// MaxEmailAttempts is how many times the outbox tries to deliver an
// email before marking it failed.
const MaxEmailAttempts = 5

type EmailOutboxRepository interface {
	Enqueue(ctx context.Context, e *OutboxEmail) error
	// ClaimPending returns up to limit pending emails, incrementing their
	// attempt count; rows are claimed with SKIP LOCKED so two workers
	// never send the same email.
	ClaimPending(ctx context.Context, limit int) ([]*OutboxEmail, error)
	MarkSent(ctx context.Context, id uuid.UUID, at time.Time) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, final bool) error
}

// WorkerRunRepository makes scheduled jobs idempotent across restarts
// and across machines: Claim succeeds for exactly one caller per
// (job, runKey).
type WorkerRunRepository interface {
	Claim(ctx context.Context, job, runKey string) (bool, error)
	Release(ctx context.Context, job, runKey string) error
}
