package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SecurityDepositStatus is derived, never stored: Held until it is
// settled; Forfeited if forfeited; otherwise Fully Refunded when the
// whole amount went back and Partially Refunded when anything was kept
// (deductions) or nothing returned.
type SecurityDepositStatus string

const (
	SecurityDepositHeld              SecurityDepositStatus = "held"
	SecurityDepositPartiallyRefunded SecurityDepositStatus = "partially_refunded"
	SecurityDepositFullyRefunded     SecurityDepositStatus = "fully_refunded"
	SecurityDepositForfeited         SecurityDepositStatus = "forfeited"
)

func (s SecurityDepositStatus) Valid() bool {
	switch s {
	case SecurityDepositHeld, SecurityDepositPartiallyRefunded, SecurityDepositFullyRefunded, SecurityDepositForfeited:
		return true
	default:
		return false
	}
}

type SecurityDeposit struct {
	ID              uuid.UUID
	OwnerID         uuid.UUID
	LeaseID         uuid.UUID
	PropertyID      uuid.UUID
	UnitID          uuid.UUID
	AmountCents     int64
	CollectedOn     time.Time
	HeldInAccountID *uuid.UUID
	SettledAt       *time.Time
	RefundCents     int64
	RefundMethod    string
	RefundedOn      *time.Time
	ForfeitedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (d *SecurityDeposit) Status() SecurityDepositStatus {
	switch {
	case d.ForfeitedAt != nil:
		return SecurityDepositForfeited
	case d.SettledAt == nil:
		return SecurityDepositHeld
	case d.RefundCents >= d.AmountCents:
		return SecurityDepositFullyRefunded
	default:
		return SecurityDepositPartiallyRefunded
	}
}

// Open reports whether the deposit can still be changed.
func (d *SecurityDeposit) Open() bool { return d.SettledAt == nil && d.ForfeitedAt == nil }

type DepositDeduction struct {
	ID           uuid.UUID
	DepositID    uuid.UUID
	Description  string
	AmountCents  int64
	AttachmentID *uuid.UUID
	CreatedAt    time.Time
}

// DepositRow is a deposit plus the display context and its deductions.
type DepositRow struct {
	SecurityDeposit
	PropertyName    string
	UnitName        string
	TenantName      string
	HeldInAccount   string
	DeductionsCents int64
	Deductions      []*DepositDeduction
}

type DepositSortKey string

const (
	DepositSortCollectedOn DepositSortKey = "collected_on"
	DepositSortAmount      DepositSortKey = "amount"
	DepositSortProperty    DepositSortKey = "property"
)

type DepositListOptions struct {
	OwnerID    uuid.UUID
	PropertyID *uuid.UUID
	Status     *SecurityDepositStatus
	Search     string
	Sort       DepositSortKey
	SortDesc   bool
	Limit      int
	Offset     int
}

type DepositRepository interface {
	// Ensure inserts the lease's deposit, or — when it already exists,
	// is still open, and the new amount still covers its deductions —
	// updates the amount. Idempotent by lease.
	Ensure(ctx context.Context, d *SecurityDeposit) error
	// EnsureMissingForOwner backfills a deposit for every lease that
	// carries a deposit amount but has none yet, and returns how many.
	EnsureMissingForOwner(ctx context.Context, ownerID uuid.UUID) (int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*DepositRow, error)
	List(ctx context.Context, opts DepositListOptions) ([]*DepositRow, int, error)
	SetHeldInAccount(ctx context.Context, id uuid.UUID, accountID *uuid.UUID, at time.Time, audit AuditEntry) error
	// AddDeduction / RemoveDeduction / Settle / Forfeit lock the deposit
	// row and re-check the invariants (open; refund + deductions never
	// exceed the amount) inside one transaction, so two concurrent edits
	// cannot together overspend it.
	AddDeduction(ctx context.Context, d *DepositDeduction, audit AuditEntry) error
	RemoveDeduction(ctx context.Context, depositID, deductionID uuid.UUID, audit AuditEntry) error
	Settle(ctx context.Context, id uuid.UUID, refundCents int64, method string, refundedOn time.Time, at time.Time, audit AuditEntry) error
	Forfeit(ctx context.Context, id uuid.UUID, at time.Time, audit AuditEntry) error
}

type AddDeductionInput struct {
	Description  string
	AmountCents  int64
	AttachmentID *uuid.UUID
}

type SettleDepositInput struct {
	RefundCents int64
	Method      string
	RefundedOn  time.Time
}

// DepositEnsurer is the narrow port LeaseService depends on to keep a
// lease's deposit record in step (best-effort, like the accounting hook
// on work orders).
type DepositEnsurer interface {
	EnsureForLease(ctx context.Context, ownerID uuid.UUID, lease *Lease, propertyID uuid.UUID) error
}

type DepositService interface {
	DepositEnsurer
	Get(ctx context.Context, ownerID, id uuid.UUID) (*DepositRow, error)
	List(ctx context.Context, ownerID uuid.UUID, opts DepositListOptions) ([]*DepositRow, int, error)
	SetHeldInAccount(ctx context.Context, ownerID, id uuid.UUID, accountID *uuid.UUID) (*DepositRow, error)
	AddDeduction(ctx context.Context, ownerID, id uuid.UUID, input AddDeductionInput) (*DepositRow, error)
	RemoveDeduction(ctx context.Context, ownerID, id, deductionID uuid.UUID) (*DepositRow, error)
	Settle(ctx context.Context, ownerID, id uuid.UUID, input SettleDepositInput) (*DepositRow, error)
	Forfeit(ctx context.Context, ownerID, id uuid.UUID) (*DepositRow, error)
}
