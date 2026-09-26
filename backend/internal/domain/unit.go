package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UnitType string

const (
	UnitTypeStudio          UnitType = "studio"
	UnitTypeOneBed          UnitType = "1br"
	UnitTypeTwoBed          UnitType = "2br"
	UnitTypeThreeBedPlus    UnitType = "3br_plus"
	UnitTypeCommercialSuite UnitType = "commercial_suite"
	UnitTypeOther           UnitType = "other"
)

func (t UnitType) Valid() bool {
	switch t {
	case UnitTypeStudio, UnitTypeOneBed, UnitTypeTwoBed, UnitTypeThreeBedPlus, UnitTypeCommercialSuite, UnitTypeOther:
		return true
	default:
		return false
	}
}

type UnitFurnished string

const (
	UnitFurnishedUnfurnished UnitFurnished = "unfurnished"
	UnitFurnishedFurnished   UnitFurnished = "furnished"
	UnitFurnishedPartial     UnitFurnished = "partial"
)

func (f UnitFurnished) Valid() bool {
	switch f {
	case UnitFurnishedUnfurnished, UnitFurnishedFurnished, UnitFurnishedPartial:
		return true
	default:
		return false
	}
}

type UnitStatus string

const (
	UnitStatusVacant      UnitStatus = "vacant"
	UnitStatusOccupied    UnitStatus = "occupied"
	UnitStatusMaintenance UnitStatus = "maintenance"
	UnitStatusOffMarket   UnitStatus = "off_market"
)

func (s UnitStatus) Valid() bool {
	switch s {
	case UnitStatusVacant, UnitStatusOccupied, UnitStatusMaintenance, UnitStatusOffMarket:
		return true
	default:
		return false
	}
}

// Unit is the rentable thing inside a Property. It has no JSON or SQL
// knowledge — see dto and repository/postgres respectively. It is
// reached only through its parent property: there is no owner_id here,
// so every caller must check the parent property's OwnerID before
// trusting a unit_id (see UnitService).
type Unit struct {
	ID         uuid.UUID
	PropertyID uuid.UUID
	UnitName   string
	Floor      string
	Type       UnitType
	Bedrooms   *int
	Bathrooms  *float64
	Sqft       *int
	Furnished  *UnitFurnished
	Status     UnitStatus
	MarketRent *float64
	// CurrentRent, whenever this Unit came from a repository read, is
	// the *effective* current rent: the unit's active lease's
	// MonthlyRent when one exists, falling back to the raw stored
	// units.current_rent value otherwise (see UnitRepository's
	// unitColumnsForRead). A caller constructing or writing a Unit
	// directly (Create/Update) is instead setting the raw fallback
	// column — the two only diverge while a unit has an active lease.
	CurrentRent     *float64
	SecurityDeposit *float64
	RentDueDay      *int
	TenantName      string
	Notes           string
	// VacatedAt is when this unit most recently became vacant — nil
	// means it isn't currently vacant, or became vacant before this
	// field existed. Stamped by UnitService whenever Status transitions
	// to/from UnitStatusVacant (see UpdateUnit), not derived from
	// leases: a unit's Status is a manually-set field, independent of
	// whether it has an active lease.
	VacatedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateUnitInput struct {
	PropertyID      uuid.UUID
	UnitName        string
	Floor           string
	Type            UnitType
	Bedrooms        *int
	Bathrooms       *float64
	Sqft            *int
	Furnished       *UnitFurnished
	Status          UnitStatus
	MarketRent      *float64
	CurrentRent     *float64
	SecurityDeposit *float64
	RentDueDay      *int
	TenantName      string
	Notes           string
}

// UpdateUnitInput fields are all optional (nil = leave unchanged),
// mirroring UpdatePropertyInput — like YearBuilt there, a field that's
// itself nullable (Bedrooms, Furnished, ...) can be set but not
// explicitly cleared back to NULL once it has a value. That's an
// existing limitation of this codebase's update-input pattern, not one
// specific to units.
type UpdateUnitInput struct {
	UnitName        *string
	Floor           *string
	Type            *UnitType
	Bedrooms        *int
	Bathrooms       *float64
	Sqft            *int
	Furnished       *UnitFurnished
	Status          *UnitStatus
	MarketRent      *float64
	CurrentRent     *float64
	SecurityDeposit *float64
	RentDueDay      *int
	TenantName      *string
	Notes           *string
}

type UnitSortKey string

const (
	UnitSortRent         UnitSortKey = "rent"
	UnitSortStatus       UnitSortKey = "status"
	UnitSortPropertyName UnitSortKey = "property_name"
)

func (k UnitSortKey) Valid() bool {
	switch k {
	case UnitSortRent, UnitSortStatus, UnitSortPropertyName:
		return true
	default:
		return false
	}
}

// UnitListFilter narrows ListForOwner results. Zero values (empty
// Search, nil PropertyID/Status/Type) mean "don't filter on this" —
// mirrors PropertyListFilter.
type UnitListFilter struct {
	Search     string
	PropertyID *uuid.UUID
	Status     *UnitStatus
	Type       *UnitType
}

type UnitListOptions struct {
	OwnerID        uuid.UUID
	PropertyAccess PropertyAccess
	Filter         UnitListFilter
	Sort           UnitSortKey
	SortDesc       bool
	Limit          int
	Offset         int
}

// UnitWithProperty decorates a Unit with its parent property's display
// name, for the portfolio-wide Units list (see
// UnitRepository.ListForOwner) — the property-scoped list
// (ListByProperty) has no need for this, since the property is already
// known from the page the caller is on.
type UnitWithProperty struct {
	Unit
	PropertyName string
}

// PropertyUnitStats is the aggregate rollup a property's stat cards are
// computed from — see UnitRepository.GetPropertyUnitStats. TotalCollected
// sums each occupied unit's effective current rent (its active lease's
// MonthlyRent when one exists, else the unit's own current_rent — see
// Unit.CurrentRent): there's no payments feature yet to read actual
// received payments from, so this approximates what should be collected
// this month rather than reporting real receipts.
type PropertyUnitStats struct {
	PropertyID     uuid.UUID
	UnitCount      int
	OccupiedCount  int
	OccupancyPct   int
	TotalCollected float64
}

// UnitDocumentCategory is a freeform organizational tag on a unit
// document — TEXT + CHECK, not a native Postgres enum, matching every
// other status/category column in this codebase.
type UnitDocumentCategory string

const (
	UnitDocumentCategoryInspection UnitDocumentCategory = "inspection"
	UnitDocumentCategoryManual     UnitDocumentCategory = "manual"
	UnitDocumentCategoryPhoto      UnitDocumentCategory = "photo"
	UnitDocumentCategoryOther      UnitDocumentCategory = "other"
)

func (c UnitDocumentCategory) Valid() bool {
	switch c {
	case UnitDocumentCategoryInspection, UnitDocumentCategoryManual, UnitDocumentCategoryPhoto, UnitDocumentCategoryOther:
		return true
	default:
		return false
	}
}

// UnitDocument is a unit-level file — an inspection report, a manual, a
// photo — distinct from any lease-linked attachment (none exist today;
// see the Unit Detail page's Documents tab). UploadedByName/Filename/
// ContentType/SizeBytes are joined in for a list response; a create
// request only ever needs AttachmentID.
type UnitDocument struct {
	ID             uuid.UUID
	UnitID         uuid.UUID
	AttachmentID   uuid.UUID
	Category       UnitDocumentCategory
	UploadedBy     uuid.UUID
	UploadedByName string
	Filename       string
	ContentType    string
	SizeBytes      int64
	CreatedAt      time.Time
}

// UnitDocumentRepository is the port implemented by internal/repository/postgres.
type UnitDocumentRepository interface {
	Create(ctx context.Context, d *UnitDocument) error
	// ListByUnit joins attachments for Filename/ContentType/SizeBytes and
	// users for UploadedByName, newest first.
	ListByUnit(ctx context.Context, unitID uuid.UUID) ([]*UnitDocument, error)
	// GetByID is used only to resolve a document's UnitID before an
	// ownership check — see UnitService.DeleteDocument.
	GetByID(ctx context.Context, id uuid.UUID) (*UnitDocument, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// UnitRepository is the port implemented by internal/repository/postgres.
type UnitRepository interface {
	Create(ctx context.Context, u *Unit) error
	CreateMany(ctx context.Context, units []*Unit) error
	GetByID(ctx context.Context, id uuid.UUID) (*Unit, error)
	ListByProperty(ctx context.Context, propertyID uuid.UUID) ([]*Unit, error)
	// ListForOwner is the portfolio-wide equivalent of ListByProperty,
	// joined to properties so it can filter/sort/search across every
	// property an owner has — see the global Units page. It enforces
	// p.owner_id = opts.OwnerID in the query itself (like
	// PropertyRepository.List does for owner_id), not as a post-hoc
	// check on returned rows, so a caller can never accidentally see
	// another owner's units by getting the filter wrong upstream.
	ListForOwner(ctx context.Context, opts UnitListOptions) ([]*UnitWithProperty, int, error)
	Update(ctx context.Context, u *Unit) error
	Delete(ctx context.Context, id uuid.UUID) error
	// ExistsByPropertyUnitName backs the same kind of pre-insert business
	// check as PropertyRepository.ExistsByOwnerAddress: whether unitName
	// is a duplicate depends on what's already stored for this property.
	ExistsByPropertyUnitName(ctx context.Context, propertyID uuid.UUID, unitName string) (bool, error)
	GetPropertyUnitStats(ctx context.Context, propertyID uuid.UUID) (*PropertyUnitStats, error)
	// GetPropertyUnitStatsBulk is the batched form GetPropertyUnitStats
	// used by a properties list response, so decorating a page of results
	// costs one query instead of one per row.
	GetPropertyUnitStatsBulk(ctx context.Context, propertyIDs []uuid.UUID) (map[uuid.UUID]*PropertyUnitStats, error)
}

// UnitService is the port implemented by internal/service and consumed
// by the HTTP transport layer. Every method takes ownerID and resolves
// it against the unit's parent property, returning ErrNotFound (never
// ErrForbidden) when the unit exists but its property belongs to
// someone else — the same IDOR-safe contract as PropertyService.
type UnitService interface {
	CreateUnit(ctx context.Context, ownerID uuid.UUID, input CreateUnitInput, access PropertyAccess) (*Unit, error)
	// CreateUnitsBulk backs the "bulk add" spreadsheet-style flow: all
	// rows are validated up front and inserted together, so a mistake on
	// row 6 doesn't leave rows 1-5 committed and 7-10 not.
	CreateUnitsBulk(ctx context.Context, ownerID, propertyID uuid.UUID, inputs []CreateUnitInput, access PropertyAccess) ([]*Unit, error)
	GetUnit(ctx context.Context, id, ownerID uuid.UUID, access PropertyAccess) (*Unit, error)
	ListUnitsByProperty(ctx context.Context, propertyID, ownerID uuid.UUID, access PropertyAccess) ([]*Unit, error)
	// ListUnitsForOwner backs the global Units page — every unit across
	// every property ownerID owns, filterable/sortable/searchable. See
	// UnitRepository.ListForOwner for where the ownership scoping
	// actually happens.
	ListUnitsForOwner(ctx context.Context, ownerID uuid.UUID, opts UnitListOptions) ([]*UnitWithProperty, int, error)
	UpdateUnit(ctx context.Context, id, ownerID uuid.UUID, input UpdateUnitInput, access PropertyAccess) (*Unit, error)
	DeleteUnit(ctx context.Context, id, ownerID uuid.UUID, access PropertyAccess) error
	GetPropertyUnitStats(ctx context.Context, propertyID, ownerID uuid.UUID, access PropertyAccess) (*PropertyUnitStats, error)
	// GetPropertyUnitStatsBulk takes no ownerID: it exists to decorate a
	// properties list response that the caller (PropertyHandler.List) has
	// already scoped to the caller's own properties via
	// PropertyListOptions.OwnerID, so re-checking ownership per id here
	// would be redundant.
	GetPropertyUnitStatsBulk(ctx context.Context, propertyIDs []uuid.UUID) (map[uuid.UUID]*PropertyUnitStats, error)

	// ListDocuments, AddDocument, and DeleteDocument back the Unit Detail
	// page's Documents tab — folded into UnitService rather than a
	// separate service, same as GetPropertyUnitStats, since a unit
	// document has no meaning independent of its unit.
	ListDocuments(ctx context.Context, unitID, ownerID uuid.UUID, access PropertyAccess) ([]*UnitDocument, error)
	// AddDocument's uploadedBy is the actor actually signed in
	// (claims.ActorID), not the account owner — see AuthClaims's doc
	// comment on UserID vs ActorID.
	AddDocument(ctx context.Context, unitID, ownerID, uploadedBy, attachmentID uuid.UUID, category UnitDocumentCategory, access PropertyAccess) (*UnitDocument, error)
	DeleteDocument(ctx context.Context, unitID, documentID, ownerID uuid.UUID, access PropertyAccess) error
}
