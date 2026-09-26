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
	ID                   uuid.UUID
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
	RenewalStatus        RenewalStatus
	// ProposedRent, ProposedEndDate, and OfferSentDate hold a renewal
	// offer's terms while RenewalStatus is Offered — kept separate from
	// MonthlyRent/EndDate so a proposal can never affect the active
	// lease's real terms until it's Accepted and GenerateRenewalLease
	// turns it into an actual new Lease row.
	ProposedRent          *float64
	ProposedEndDate       *time.Time
	OfferSentDate         *time.Time
	TerminationReason     *TerminationReason
	TerminationNoticeDate *time.Time
	// RenewedIntoLeaseID/RenewedFromLeaseID link a renewal's two lease
	// records: the original points at the new lease it was renewed
	// into, and the new lease points back at the one it renewed. Set
	// atomically by GenerateRenewalLease on both rows.
	RenewedIntoLeaseID *uuid.UUID
	RenewedFromLeaseID *uuid.UUID
	Signed             bool
	SignedDate         *time.Time
	Notes              string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// MonthlyRent, whenever this Lease came from a repository read, is the
// *effective* rent as of today: the latest lease_rent_history row whose
// effective_date has arrived, falling back to the raw stored
// leases.monthly_rent column when no history row applies yet (see
// LeaseRepository's qualifiedLeaseColumns/leaseColumnsForRead). Once a
// lease has ever gone active, leases.monthly_rent itself is frozen at
// its creation-time value — see LeaseService.ChangeRent and
// LeaseService.UpdateLease's rejection of direct MonthlyRent edits on a
// non-draft lease. A caller constructing a Lease directly (Create) is
// setting that raw creation-time value.

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
// mirrors UpdatePropertyInput/UpdateUnitInput. MonthlyRent is only
// honored by UpdateLease while the lease is still a draft — once it has
// ever gone active, a non-nil MonthlyRent is rejected; use ChangeRent
// instead (see LeaseService.UpdateLease).
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
	ProposedRent          *float64
	ProposedEndDate       *time.Time
	OfferSentDate         *time.Time
	TerminationReason     *TerminationReason
	TerminationNoticeDate *time.Time
	Signed                *bool
	SignedDate            *time.Time
	Notes                 *string
}

// ChangeRentInput backs LeaseService.ChangeRent — the only way to alter
// an active lease's effective rent (see R3/R4 in the business rules
// this implements). It is never an overwrite: every call appends a new
// lease_rent_history row.
type ChangeRentInput struct {
	Amount float64
	// EffectiveDate must be today or later unless IsCorrection is true —
	// a rent change is never backdated except as an explicitly-flagged
	// data-entry correction.
	EffectiveDate time.Time
	Reason        string
	IsCorrection  bool
}

// LeaseRentHistoryEntry is one dated rent amendment on a lease — see
// ChangeRentInput and the lease_rent_history migration's comment for
// why this is append-only.
type LeaseRentHistoryEntry struct {
	ID            uuid.UUID
	LeaseID       uuid.UUID
	Amount        float64
	EffectiveDate time.Time
	Reason        string
	IsCorrection  bool
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
}

// GenerateRenewalLeaseInput backs LeaseService.GenerateRenewalLease,
// called once a lease's RenewalStatus is Accepted. Fields default to
// the source lease's ProposedRent/ProposedEndDate when left zero/nil,
// but can be overridden at generation time.
type GenerateRenewalLeaseInput struct {
	StartDate       time.Time
	EndDate         *time.Time
	MonthlyRent     *float64
	SecurityDeposit *float64
	RentDueDay      *int
}

// TerminateLeaseInput backs LeaseService.TerminateLease — the dedicated
// action for ending a lease early/on schedule, distinct from a generic
// UpdateLease(Status: terminated) call so the unit-vacancy and
// audit-log side effects always happen together (see R10/R18).
type TerminateLeaseInput struct {
	TerminationReason     TerminationReason
	TerminationNoticeDate time.Time
	MoveOutDate           *time.Time
	// MoveOutInspectionAttachmentID, if set, creates a unit-level
	// document automatically linked to this lease (RelatedLeaseID) and
	// flagged IsAutomated — see R15's example of an automated lease
	// action auto-tagging a document.
	MoveOutInspectionAttachmentID *uuid.UUID
}

// CorrectTerminationInput backs LeaseService.CorrectTermination — a
// separate, explicitly-labeled action for fixing a data-entry mistake
// in an already-terminated lease's termination details, distinct from
// ordinary editing: once a lease is Terminated, its termination
// reason/notice date/move-out date render read-only on the form (they
// are the historical record of how the lease ended), and Reason here
// is mandatory so every correction is self-explaining in the audit
// trail, never a silent inline edit.
type CorrectTerminationInput struct {
	TerminationReason     TerminationReason
	TerminationNoticeDate time.Time
	MoveOutDate           *time.Time
	Reason                string
}

// LeaseAuditAction names the lease lifecycle events R18 requires an
// audit trail for. Unlike accounting_audit_log's polymorphic
// entity_type/entity_id pair, lease_audit_log is keyed directly by
// lease_id, since every action here always has exactly one lease as
// its subject.
const (
	LeaseAuditActionRentChanged          = "rent_changed"
	LeaseAuditActionRenewed              = "renewed"
	LeaseAuditActionTerminated           = "terminated"
	LeaseAuditActionTerminationCorrected = "termination_corrected"
)

// LeaseAuditEntry mirrors AuditEntry's/StaffAuditEntry's shape for the
// separate lease_audit_log table — its own table per the precedent
// staff audit already set of not sharing accounting's table.
type LeaseAuditEntry struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	LeaseID   uuid.UUID
	ActorID   uuid.UUID
	Action    string
	Changes   map[string]FieldChange
	CreatedAt time.Time
}

func NewLeaseAuditEntry(ownerID, leaseID, actorID uuid.UUID, action string, changes map[string]FieldChange, now time.Time) LeaseAuditEntry {
	if changes == nil {
		changes = map[string]FieldChange{}
	}
	return LeaseAuditEntry{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		LeaseID:   leaseID,
		ActorID:   actorID,
		Action:    action,
		Changes:   changes,
		CreatedAt: now,
	}
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
	// Update never writes monthly_rent (see leaseColumnsForRead's doc
	// comment) — that column is frozen after creation except through
	// UpdateMonthlyRent (draft leases only) or AppendRentChange (active
	// leases, via lease_rent_history).
	Update(ctx context.Context, l *Lease) error
	// UpdateMonthlyRent writes the raw monthly_rent column directly —
	// only ever called for a still-draft lease, which has no rent
	// history yet (see LeaseService.UpdateLease's rejection of a direct
	// MonthlyRent edit on any lease that has gone active).
	UpdateMonthlyRent(ctx context.Context, leaseID uuid.UUID, amount float64) error
	Delete(ctx context.Context, id uuid.UUID) error
	// HasActiveLease backs the one-active-lease-per-unit business rule
	// (also enforced by a DB partial unique index as a safety net — see
	// the migration) with a clean pre-check, the same pattern
	// PropertyRepository.ExistsByOwnerAddress and
	// UnitRepository.ExistsByPropertyUnitName use. excludeLeaseID lets
	// an update that keeps a lease active not collide with itself.
	HasActiveLease(ctx context.Context, unitID uuid.UUID, excludeLeaseID *uuid.UUID) (bool, error)

	// ListRentHistory returns a lease's rent amendments, most recent
	// effective date first — see LeaseRentHistoryEntry.
	ListRentHistory(ctx context.Context, leaseID uuid.UUID) ([]*LeaseRentHistoryEntry, error)
	// AppendRentChange inserts entry and audit in one transaction —
	// never an overwrite of leases.monthly_rent, following the same
	// "repository method takes the audit entry, writes atomically"
	// convention as LedgerRepository.CreateTransaction.
	AppendRentChange(ctx context.Context, entry *LeaseRentHistoryEntry, audit LeaseAuditEntry) error
	// CreateRenewal atomically inserts newLease, retires the source
	// lease (status -> terminated, both RenewedInto/FromLeaseID links
	// set), and writes audit — see LeaseService.GenerateRenewalLease.
	CreateRenewal(ctx context.Context, newLease *Lease, sourceLeaseID uuid.UUID, audit LeaseAuditEntry) error
	// Terminate atomically sets l's status to terminated and unit's
	// status to vacant (stamping VacatedAt), writes audit, and — when
	// autoDocument is non-nil — inserts the auto-linked unit_documents
	// row, all in one transaction. See LeaseService.TerminateLease.
	Terminate(ctx context.Context, l *Lease, unit *Unit, audit LeaseAuditEntry, autoDocument *UnitDocument) error
	// CorrectTermination writes l's (already-corrected in memory)
	// termination fields and audit in one transaction — no unit
	// involvement, unlike Terminate, since the unit's vacancy was
	// already settled by the original termination.
	CorrectTermination(ctx context.Context, l *Lease, audit LeaseAuditEntry) error
	// ListAudit returns a lease's audit trail, most recent first.
	ListAudit(ctx context.Context, leaseID uuid.UUID) ([]*LeaseAuditEntry, error)
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

	ListRentHistory(ctx context.Context, id, ownerID uuid.UUID, access PropertyAccess) ([]*LeaseRentHistoryEntry, error)
	// ChangeRent's actorID is the signed-in staff user (claims.ActorID),
	// not the account owner — same distinction UnitService.AddDocument's
	// uploadedBy makes.
	ChangeRent(ctx context.Context, id, ownerID, actorID uuid.UUID, input ChangeRentInput, access PropertyAccess) (*Lease, error)
	// GenerateRenewalLease turns an Accepted renewal into a new, real
	// Lease record — see R7/R8/R9.
	GenerateRenewalLease(ctx context.Context, id, ownerID, actorID uuid.UUID, input GenerateRenewalLeaseInput, access PropertyAccess) (*Lease, error)
	// TerminateLease is the dedicated action for ending a lease, distinct
	// from a generic UpdateLease(Status: terminated) call — see R10/R18.
	TerminateLease(ctx context.Context, id, ownerID, actorID uuid.UUID, input TerminateLeaseInput, access PropertyAccess) (*Lease, error)
	// CorrectTermination fixes a data-entry mistake in an already-
	// terminated lease's termination details — never a silent inline
	// edit; Reason is mandatory and every correction is audit-logged.
	CorrectTermination(ctx context.Context, id, ownerID, actorID uuid.UUID, input CorrectTerminationInput, access PropertyAccess) (*Lease, error)
	ListAudit(ctx context.Context, id, ownerID uuid.UUID, access PropertyAccess) ([]*LeaseAuditEntry, error)

	// ListDocuments, AddDocument, and DeleteDocument back a lease's
	// Documents tab — lease-specific documents live only here, never
	// duplicated onto the unit's UnitDocument list (see R14).
	ListDocuments(ctx context.Context, leaseID, ownerID uuid.UUID, access PropertyAccess) ([]*LeaseDocument, error)
	AddDocument(ctx context.Context, leaseID, ownerID, uploadedBy, attachmentID uuid.UUID, category LeaseDocumentCategory, access PropertyAccess) (*LeaseDocument, error)
	DeleteDocument(ctx context.Context, leaseID, documentID, ownerID uuid.UUID, access PropertyAccess) error
}

// LeaseDocumentCategory mirrors UnitDocumentCategory's role for
// lease-level documents — TEXT + CHECK, not a native Postgres enum.
type LeaseDocumentCategory string

const (
	LeaseDocumentCategoryAgreement  LeaseDocumentCategory = "lease_agreement"
	LeaseDocumentCategoryInspection LeaseDocumentCategory = "inspection"
	LeaseDocumentCategoryNotice     LeaseDocumentCategory = "notice"
	LeaseDocumentCategoryRenewal    LeaseDocumentCategory = "renewal"
	LeaseDocumentCategoryOther      LeaseDocumentCategory = "other"
)

func (c LeaseDocumentCategory) Valid() bool {
	switch c {
	case LeaseDocumentCategoryAgreement, LeaseDocumentCategoryInspection, LeaseDocumentCategoryNotice, LeaseDocumentCategoryRenewal, LeaseDocumentCategoryOther:
		return true
	default:
		return false
	}
}

// LeaseDocument is a lease-level file — a signed agreement, an addendum,
// a notice — mirroring UnitDocument's shape exactly. It lives only on
// its originating lease's Documents tab; it is never duplicated onto
// that lease's unit-level Documents tab (see R14). IsAutomated marks a
// document a lease action (e.g. TerminateLease) created on its own,
// rather than one a staff member manually uploaded.
type LeaseDocument struct {
	ID             uuid.UUID
	LeaseID        uuid.UUID
	AttachmentID   uuid.UUID
	Category       LeaseDocumentCategory
	UploadedBy     uuid.UUID
	UploadedByName string
	IsAutomated    bool
	Filename       string
	ContentType    string
	SizeBytes      int64
	CreatedAt      time.Time
}

// LeaseDocumentRepository is the port implemented by
// internal/repository/postgres, mirroring UnitDocumentRepository.
type LeaseDocumentRepository interface {
	Create(ctx context.Context, d *LeaseDocument) error
	ListByLease(ctx context.Context, leaseID uuid.UUID) ([]*LeaseDocument, error)
	GetByID(ctx context.Context, id uuid.UUID) (*LeaseDocument, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
