// Package dto holds HTTP request/response shapes. These are intentionally
// separate from domain types: JSON tags, validation tags, and wire-format
// concerns (e.g. RFC3339 timestamps as strings) never leak into the
// domain layer.
package dto

import (
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

type CreatePropertyRequest struct {
	Address   string `json:"address" validate:"required,min=3,max=255,noctrl"`
	UnitCount int    `json:"unit_count" validate:"required,min=1,max=100000"`
	Status    string `json:"status" validate:"omitempty,oneof=active inactive maintenance"`
}

// Sanitize trims and strips control characters from free-text fields
// before validation runs (see Sanitizable). UnitCount and Status are
// already exact/enum-constrained values with no free text to normalize.
func (r *CreatePropertyRequest) Sanitize() {
	r.Address = sanitizeString(r.Address)
}

type UpdatePropertyRequest struct {
	Address   *string `json:"address" validate:"omitempty,min=3,max=255,noctrl"`
	UnitCount *int    `json:"unit_count" validate:"omitempty,min=1,max=100000"`
	Status    *string `json:"status" validate:"omitempty,oneof=active inactive maintenance"`
}

func (r *UpdatePropertyRequest) Sanitize() {
	if r.Address != nil {
		trimmed := sanitizeString(*r.Address)
		r.Address = &trimmed
	}
}

type PropertyResponse struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	UnitCount int    `json:"unit_count"`
	Status    string `json:"status"`
	OwnerID   string `json:"owner_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type PropertyListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func NewPropertyResponse(p *domain.Property) PropertyResponse {
	return PropertyResponse{
		ID:        p.ID.String(),
		Address:   p.Address,
		UnitCount: p.UnitCount,
		Status:    string(p.Status),
		OwnerID:   p.OwnerID.String(),
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
	}
}

func NewPropertyListResponse(properties []*domain.Property) []PropertyResponse {
	out := make([]PropertyResponse, len(properties))
	for i, p := range properties {
		out[i] = NewPropertyResponse(p)
	}
	return out
}

func (r CreatePropertyRequest) ToDomain(ownerID uuid.UUID) domain.CreatePropertyInput {
	return domain.CreatePropertyInput{
		Address:   r.Address,
		UnitCount: r.UnitCount,
		Status:    domain.PropertyStatus(r.Status),
		OwnerID:   ownerID,
	}
}

func (r UpdatePropertyRequest) ToDomain() domain.UpdatePropertyInput {
	input := domain.UpdatePropertyInput{
		Address:   r.Address,
		UnitCount: r.UnitCount,
	}
	if r.Status != nil {
		status := domain.PropertyStatus(*r.Status)
		input.Status = &status
	}
	return input
}
