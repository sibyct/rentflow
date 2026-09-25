package domain

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PropertyStatus string

const (
	// PropertyStatusOnboarding is the default for a newly created
	// property — it hasn't been fully set up (units, leases) yet.
	PropertyStatusOnboarding PropertyStatus = "onboarding"
	PropertyStatusActive     PropertyStatus = "active"
	PropertyStatusArchived   PropertyStatus = "archived"
)

func (s PropertyStatus) Valid() bool {
	switch s {
	case PropertyStatusOnboarding, PropertyStatusActive, PropertyStatusArchived:
		return true
	default:
		return false
	}
}

type PropertyType string

const (
	PropertyTypeResidentialSingleUnit PropertyType = "residential_single_unit"
	PropertyTypeResidentialMultiUnit  PropertyType = "residential_multi_unit"
	PropertyTypeCommercial            PropertyType = "commercial"
	PropertyTypeMixedUse              PropertyType = "mixed_use"
)

func (t PropertyType) Valid() bool {
	switch t {
	case PropertyTypeResidentialSingleUnit, PropertyTypeResidentialMultiUnit, PropertyTypeCommercial, PropertyTypeMixedUse:
		return true
	default:
		return false
	}
}

// PropertyOwnership records whether the account holder owns a property
// directly or manages it on behalf of someone else (OwnerName). It's
// distinct from Property.OwnerID, which is the internal account the
// property record belongs to, not a business fact about the property.
type PropertyOwnership string

const (
	PropertyOwnershipOwned   PropertyOwnership = "owned"
	PropertyOwnershipManaged PropertyOwnership = "managed"
)

func (o PropertyOwnership) Valid() bool {
	switch o {
	case PropertyOwnershipOwned, PropertyOwnershipManaged:
		return true
	default:
		return false
	}
}

// ValidAmenities is the fixed set of amenity tags the product supports.
// Kept in sync with frontend/src/features/properties/mock/propertyRows.ts
// (AMENITIES) — there's no amenities-management feature yet, so both
// sides hardcode the same list.
var ValidAmenities = map[string]bool{
	"Parking":      true,
	"Laundry":      true,
	"Pool":         true,
	"Elevator":     true,
	"Pet-friendly": true,
	"Gym":          true,
	"Storage":      true,
	"EV charging":  true,
}

// Property is the core business entity. It has no JSON tags, no SQL
// knowledge, and no HTTP knowledge — those concerns live in the DTO and
// repository layers respectively.
type Property struct {
	ID            uuid.UUID
	Name          string
	Type          PropertyType
	AddressLine1  string
	AddressLine2  string
	City          string
	StateProvince string
	PostalCode    string
	Country       string
	Units         int
	Ownership     *PropertyOwnership
	OwnerName     string
	YearBuilt     *int
	OnboardDate   *time.Time
	Amenities     []string
	Notes         string
	Status        PropertyStatus
	OwnerID       uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// FormattedAddress composes the structured address fields into the
// single display line the properties list shows (e.g. "214 Willow Creek
// Rd, Austin, TX 78745"), rather than storing that string redundantly
// alongside its components.
func (p *Property) FormattedAddress() string {
	cityLine := p.City
	stateZip := strings.TrimSpace(p.StateProvince + " " + p.PostalCode)
	switch {
	case cityLine != "" && stateZip != "":
		cityLine += ", " + stateZip
	case cityLine == "":
		cityLine = stateZip
	}

	parts := make([]string, 0, 3)
	for _, part := range []string{p.AddressLine1, p.AddressLine2, cityLine} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

type CreatePropertyInput struct {
	Name          string
	Type          PropertyType
	AddressLine1  string
	AddressLine2  string
	City          string
	StateProvince string
	PostalCode    string
	Country       string
	Units         int
	Ownership     *PropertyOwnership
	OwnerName     string
	YearBuilt     *int
	OnboardDate   *time.Time
	Amenities     []string
	Notes         string
	Status        PropertyStatus
	OwnerID       uuid.UUID
}

// UpdatePropertyInput fields are all optional (nil = leave unchanged).
// Amenities is the one exception: nil means unchanged, but a non-nil
// empty slice means "clear the list" — the JSON decoder already
// distinguishes an absent field (nil) from `"amenities": []` (empty,
// non-nil), so this falls out naturally.
type UpdatePropertyInput struct {
	Name          *string
	Type          *PropertyType
	AddressLine1  *string
	AddressLine2  *string
	City          *string
	StateProvince *string
	PostalCode    *string
	Country       *string
	Units         *int
	Ownership     *PropertyOwnership
	OwnerName     *string
	YearBuilt     *int
	OnboardDate   *time.Time
	Amenities     []string
	Notes         *string
	Status        *PropertyStatus
}

type PropertySortKey string

const (
	PropertySortName      PropertySortKey = "name"
	PropertySortType      PropertySortKey = "type"
	PropertySortUnits     PropertySortKey = "units"
	PropertySortStatus    PropertySortKey = "status"
	PropertySortCreatedAt PropertySortKey = "created_at"
)

func (k PropertySortKey) Valid() bool {
	switch k {
	case PropertySortName, PropertySortType, PropertySortUnits, PropertySortStatus, PropertySortCreatedAt:
		return true
	default:
		return false
	}
}

// PropertyListFilter narrows List/ListProperties results. Zero values
// (empty Search, nil Type/Status) mean "don't filter on this".
type PropertyListFilter struct {
	Search string
	Type   *PropertyType
	Status *PropertyStatus
}

type PropertyListOptions struct {
	OwnerID        uuid.UUID
	PropertyAccess PropertyAccess
	Filter         PropertyListFilter
	Sort           PropertySortKey
	SortDesc       bool
	Limit          int
	Offset         int
}

// PropertyRepository is the port implemented by internal/repository/postgres.
type PropertyRepository interface {
	Create(ctx context.Context, p *Property) error
	GetByID(ctx context.Context, id uuid.UUID) (*Property, error)
	List(ctx context.Context, opts PropertyListOptions) ([]*Property, int, error)
	Update(ctx context.Context, p *Property) error
	Delete(ctx context.Context, id uuid.UUID) error
	// BulkUpdateStatus applies status to every property in ids owned by
	// ownerID and within access's scope, ignoring ids that don't exist,
	// belong to someone else, or fall outside access, and returns how
	// many rows were actually changed.
	BulkUpdateStatus(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, status PropertyStatus, access PropertyAccess) (int, error)
	// ExistsByOwnerAddress reports whether owner already has a property
	// at addressLine1. It backs a business rule (see PropertyService)
	// that can't be expressed as a struct validation tag: whether the
	// address is a duplicate depends on what's already stored for that
	// owner, not on the shape of the input alone.
	ExistsByOwnerAddress(ctx context.Context, ownerID uuid.UUID, addressLine1 string) (bool, error)
}

// PropertyService is the port implemented by internal/service and consumed
// by the HTTP transport layer.
type PropertyService interface {
	CreateProperty(ctx context.Context, input CreatePropertyInput) (*Property, error)
	// GetProperty, UpdateProperty, and DeleteProperty all take ownerID
	// and access and return ErrNotFound (not ErrForbidden) when the
	// property exists but belongs to someone else or falls outside
	// access's scope — an authenticated user should not be able to
	// distinguish "not yours"/"not in your scope" from "doesn't exist"
	// by probing IDs.
	GetProperty(ctx context.Context, id, ownerID uuid.UUID, access PropertyAccess) (*Property, error)
	ListProperties(ctx context.Context, opts PropertyListOptions) ([]*Property, int, error)
	UpdateProperty(ctx context.Context, id, ownerID uuid.UUID, input UpdatePropertyInput, access PropertyAccess) (*Property, error)
	DeleteProperty(ctx context.Context, id, ownerID uuid.UUID, access PropertyAccess) error
	BulkUpdateStatus(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, status PropertyStatus, access PropertyAccess) (int, error)
}
