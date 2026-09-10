package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PropertyStatus string

const (
	PropertyStatusActive      PropertyStatus = "active"
	PropertyStatusInactive    PropertyStatus = "inactive"
	PropertyStatusMaintenance PropertyStatus = "maintenance"
)

func (s PropertyStatus) Valid() bool {
	switch s {
	case PropertyStatusActive, PropertyStatusInactive, PropertyStatusMaintenance:
		return true
	default:
		return false
	}
}

// Property is the core business entity. It has no JSON tags, no SQL
// knowledge, and no HTTP knowledge — those concerns live in the DTO and
// repository layers respectively.
type Property struct {
	ID        uuid.UUID
	Address   string
	UnitCount int
	Status    PropertyStatus
	OwnerID   uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreatePropertyInput struct {
	Address   string
	UnitCount int
	Status    PropertyStatus
	OwnerID   uuid.UUID
}

type UpdatePropertyInput struct {
	Address   *string
	UnitCount *int
	Status    *PropertyStatus
}

// PropertyRepository is the port implemented by internal/repository/postgres.
type PropertyRepository interface {
	Create(ctx context.Context, p *Property) error
	GetByID(ctx context.Context, id uuid.UUID) (*Property, error)
	List(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*Property, int, error)
	Update(ctx context.Context, p *Property) error
	Delete(ctx context.Context, id uuid.UUID) error
	// ExistsByOwnerAddress reports whether owner already has a property
	// at address. It backs a business rule (see PropertyService) that
	// can't be expressed as a struct validation tag: whether the
	// address is a duplicate depends on what's already stored for that
	// owner, not on the shape of the input alone.
	ExistsByOwnerAddress(ctx context.Context, ownerID uuid.UUID, address string) (bool, error)
}

// PropertyService is the port implemented by internal/service and consumed
// by the HTTP transport layer.
type PropertyService interface {
	CreateProperty(ctx context.Context, input CreatePropertyInput) (*Property, error)
	GetProperty(ctx context.Context, id uuid.UUID) (*Property, error)
	ListProperties(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*Property, int, error)
	UpdateProperty(ctx context.Context, id uuid.UUID, input UpdatePropertyInput) (*Property, error)
	DeleteProperty(ctx context.Context, id uuid.UUID) error
}
