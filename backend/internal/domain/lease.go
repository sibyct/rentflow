package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type LeaseType string

const (
	LeaseTypeFixed        LeaseType = "fixed"
	LeaseTypeMonthToMonth LeaseType = "month_to_month"
)

func (t LeaseType) Valid() bool {
	switch t {
	case LeaseTypeFixed, LeaseTypeMonthToMonth:
		return true
	default:
		return false
	}
}

// LeaseStatus is the small set of states a manager actually sets by
// taking an action. The richer states a lease list needs (upcoming,
// expiring soon, expired) are derived from dates, not stored — see
// Lease.DisplayStatus.
type LeaseStatus string

const (
	LeaseStatusDraft      LeaseStatus = "draft"
	LeaseStatusActive     LeaseStatus = "active"
	LeaseStatusTerminated LeaseStatus = "terminated"
)

func (s LeaseStatus) Valid() bool {
	switch s {
	case LeaseStatusDraft, LeaseStatusActive, LeaseStatusTerminated:
		return true
	default:
		return false
	}
}

// LeaseDisplayStatus is what a lease list/detail view actually shows —
// LeaseStatusActive fans out into upcoming/active/expiring_soon/expired
// depending on today's date relative to start_date/end_date. See
// Lease.DisplayStatus, the only place this is computed.
type LeaseDisplayStatus string

const (
	LeaseDisplayDraft        LeaseDisplayStatus = "draft"
	LeaseDisplayUpcoming     LeaseDisplayStatus = "upcoming"
	LeaseDisplayActive       LeaseDisplayStatus = "active"
	LeaseDisplayExpiringSoon LeaseDisplayStatus = "expiring_soon"
	LeaseDisplayExpired      LeaseDisplayStatus = "expired"
	LeaseDisplayTerminated   LeaseDisplayStatus = "terminated"
)

func (s LeaseDisplayStatus) Valid() bool {
	switch s {
	case LeaseDisplayDraft, LeaseDisplayUpcoming, LeaseDisplayActive, LeaseDisplayExpiringSoon, LeaseDisplayExpired, LeaseDisplayTerminated:
		return true
	default:
		return false
	}
}

// LeaseExpiringSoonDays is the window (from today) within which an
// active fixed-term lease's end_date makes it "expiring soon" rather
// than plain "active". A fixed, documented assumption — not user
// configurable yet.
const LeaseExpiringSoonDays = 30

type DepositStatus string

const (
	DepositStatusHeld              DepositStatus = "held"
	DepositStatusPartiallyReturned DepositStatus = "partially_returned"
	DepositStatusReturned          DepositStatus = "returned"
	DepositStatusForfeited         DepositStatus = "forfeited"
)

func (s DepositStatus) Valid() bool {
	switch s {
	case DepositStatusHeld, DepositStatusPartiallyReturned, DepositStatusReturned, DepositStatusForfeited:
		return true
	default:
		return false
	}
}

type RenewalStatus string

const (
	RenewalStatusNotStarted RenewalStatus = "not_started"
	RenewalStatusOffered    RenewalStatus = "offered"
	RenewalStatusAccepted   RenewalStatus = "accepted"
	RenewalStatusDeclined   RenewalStatus = "declined"
)

func (s RenewalStatus) Valid() bool {
	switch s {
	case RenewalStatusNotStarted, RenewalStatusOffered, RenewalStatusAccepted, RenewalStatusDeclined:
		return true
	default:
		return false
	}
}

type TerminationReason string

const (
	TerminationReasonNonRenewal     TerminationReason = "non_renewal"
	TerminationReasonEviction       TerminationReason = "eviction"
	TerminationReasonMutual         TerminationReason = "mutual"
	TerminationReasonResidentNotice TerminationReason = "resident_notice"
	TerminationReasonOther          TerminationReason = "other"
)

func (r TerminationReason) Valid() bool {
	switch r {
	case TerminationReasonNonRenewal, TerminationReasonEviction, TerminationReasonMutual, TerminationReasonResidentNotice, TerminationReasonOther:
		return true
	default:
		return false
	}
}

// Lease is the core business entity: a time-bound contract binding a
// unit to whoever is occupying it. It has no owner_id and no
// property_id of its own — ownership is reached by joining through
// UnitID to the unit's parent property, exactly like domain.Unit (see
// LeaseService), and PropertyID is likewise never stored redundantly
// here.
//
// Residents (PrimaryResidentName, CoResidents) are freeform text, not a
// foreign key: there is no Residents entity yet, so this mirrors
// Unit.TenantName's precedent rather than fabricating a link to
// something that doesn't exist.
type Lease struct {
	ID                    uuid.UUID
	UnitID                uuid.UUID
	Type                  LeaseType
	Status                LeaseStatus
	StartDate             time.Time
	EndDate               *time.Time
	MoveInDate            *time.Time
	MoveOutDate           *time.Time
	MonthlyRent           float64
	SecurityDeposit       *float64
	DepositStatus         *DepositStatus
	RentDueDay            *int
	LateFeeAmount         *float64
	LateFeeGraceDays      *int
	PrimaryResidentName   string
	PrimaryResidentPhone  string
	PrimaryResidentEmail  string
	CoResidents           []string
	EmergencyContact      string
	RenewalStatus         RenewalStatus
	TerminationReason     *TerminationReason
	TerminationNoticeDate *time.Time
	Signed                bool
	SignedDate            *time.Time
	Notes                 string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// DisplayStatus computes the richer status a lease list/detail view
// shows from Status plus today's date relative to StartDate/EndDate —
// see LeaseDisplayStatus's doc comment for why this isn't stored.
func (l *Lease) DisplayStatus(now time.Time) LeaseDisplayStatus {
	switch l.Status {
	case LeaseStatusDraft:
		return LeaseDisplayDraft
	case LeaseStatusTerminated:
		return LeaseDisplayTerminated
	}

	today := truncateToDate(now)
	start := truncateToDate(l.StartDate)
	if start.After(today) {
		return LeaseDisplayUpcoming
	}
	if l.EndDate != nil {
		end := truncateToDate(*l.EndDate)
		daysUntilEnd := int(end.Sub(today).Hours() / 24)
		if daysUntilEnd < 0 {
			return LeaseDisplayExpired
		}
		if daysUntilEnd <= LeaseExpiringSoonDays {
			return LeaseDisplayExpiringSoon
		}
	}
	return LeaseDisplayActive
}

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

type CreateLeaseInput struct {
	UnitID               uuid.UUID
	Type                 LeaseType
	Status               LeaseStatus
	StartDate            time.Time
	EndDate              *time.Time
	MoveInDate           *time.Time
	MoveOutDate          *time.Time
	MonthlyRent          float64
	SecurityDeposit      *float64
	DepositStatus        *DepositStatus
	RentDueDay           *int
	LateFeeAmount        *float64
	LateFeeGraceDays     *int
	PrimaryResidentName  string
	PrimaryResidentPhone string
	PrimaryResidentEmail string
	CoResidents          []string
	EmergencyContact     string
	Notes                string
}

// UpdateLeaseInput fields are all optional (nil = leave unchanged) —
// mirrors UpdatePropertyInput/UpdateUnitInput.
type UpdateLeaseInput struct {
	Type                  *LeaseType
	Status                *LeaseStatus
	StartDate             *time.Time
	EndDate               *time.Time
	MoveInDate            *time.Time
	MoveOutDate           *time.Time
	MonthlyRent           *float64
	SecurityDeposit       *float64
	DepositStatus         *DepositStatus
	RentDueDay            *int
	LateFeeAmount         *float64
	LateFeeGraceDays      *int
	PrimaryResidentName   *string
	PrimaryResidentPhone  *string
	PrimaryResidentEmail  *string
	CoResidents           []string
	EmergencyContact      *string
	RenewalStatus         *RenewalStatus
	TerminationReason     *TerminationReason
	TerminationNoticeDate *time.Time
	Signed                *bool
	SignedDate            *time.Time
	Notes                 *string
}

type LeaseSortKey string

const (
	LeaseSortStartDate   LeaseSortKey = "start_date"
	LeaseSortEndDate     LeaseSortKey = "end_date"
	LeaseSortMonthlyRent LeaseSortKey = "monthly_rent"
	LeaseSortStatus      LeaseSortKey = "status"
)

func (k LeaseSortKey) Valid() bool {
	switch k {
	case LeaseSortStartDate, LeaseSortEndDate, LeaseSortMonthlyRent, LeaseSortStatus:
		return true
	default:
		return false
	}
}

// LeaseListFilter narrows ListForOwner results. DisplayStatus filters
// on the *computed* status (see Lease.DisplayStatus) — the repository
// translates it into the equivalent date comparison in SQL, since the
// stored Status column alone can't distinguish upcoming/expiring_soon/
// expired.
type LeaseListFilter struct {
	Search        string
	PropertyID    *uuid.UUID
	UnitID        *uuid.UUID
	DisplayStatus *LeaseDisplayStatus
}

type LeaseListOptions struct {
	OwnerID        uuid.UUID
	PropertyAccess PropertyAccess
	Filter         LeaseListFilter
	Sort           LeaseSortKey
	SortDesc       bool
	Limit          int
	Offset         int
}

// LeaseWithUnitProperty decorates a Lease with its unit's and parent
// property's display names, for the portfolio-wide Leases list —
// mirrors domain.UnitWithProperty's role for the global Units page.
type LeaseWithUnitProperty struct {
	Lease
	UnitName     string
	PropertyID   uuid.UUID
	PropertyName string
}

// LeaseRepository is the port implemented by internal/repository/postgres.
type LeaseRepository interface {
	Create(ctx context.Context, l *Lease) error
	GetByID(ctx context.Context, id uuid.UUID) (*Lease, error)
	// ListForOwner enforces p.owner_id = opts.OwnerID in the query
	// itself (joining leases -> units -> properties), the same
	// row-security pattern UnitRepository.ListForOwner uses.
	ListForOwner(ctx context.Context, opts LeaseListOptions) ([]*LeaseWithUnitProperty, int, error)
	Update(ctx context.Context, l *Lease) error
	Delete(ctx context.Context, id uuid.UUID) error
	// HasActiveLease backs the one-active-lease-per-unit business rule
	// (also enforced by a DB partial unique index as a safety net — see
	// the migration) with a clean pre-check, the same pattern
	// PropertyRepository.ExistsByOwnerAddress and
	// UnitRepository.ExistsByPropertyUnitName use. excludeLeaseID lets
	// an update that keeps a lease active not collide with itself.
	HasActiveLease(ctx context.Context, unitID uuid.UUID, excludeLeaseID *uuid.UUID) (bool, error)
}

// LeaseService is the port implemented by internal/service and
// consumed by the HTTP transport layer. Every method resolves ownership
// by loading the lease's unit and then that unit's property, returning
// ErrNotFound (never ErrForbidden) when the lease exists but its
// property belongs to someone else — the same IDOR-safe contract as
// PropertyService and UnitService.
type LeaseService interface {
	CreateLease(ctx context.Context, ownerID uuid.UUID, input CreateLeaseInput, access PropertyAccess) (*Lease, error)
	// GetLease returns the unit/property-decorated shape (unlike Create,
	// whose caller already knows both from the URL it posted to) since a
	// lease detail view needs to show which unit and property it's for.
	GetLease(ctx context.Context, id, ownerID uuid.UUID, access PropertyAccess) (*LeaseWithUnitProperty, error)
	ListLeasesForOwner(ctx context.Context, ownerID uuid.UUID, opts LeaseListOptions) ([]*LeaseWithUnitProperty, int, error)
	UpdateLease(ctx context.Context, id, ownerID uuid.UUID, input UpdateLeaseInput, access PropertyAccess) (*Lease, error)
	DeleteLease(ctx context.Context, id, ownerID uuid.UUID, access PropertyAccess) error
}
