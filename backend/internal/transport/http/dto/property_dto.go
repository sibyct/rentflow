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

const propertyDateLayout = "2006-01-02"

type CreatePropertyRequest struct {
	Name          string   `json:"name" validate:"required,min=1,max=200,noctrl"`
	Type          string   `json:"type" validate:"required,oneof=residential_single_unit residential_multi_unit commercial mixed_use"`
	AddressLine1  string   `json:"address_line1" validate:"required,min=1,max=255,noctrl"`
	AddressLine2  string   `json:"address_line2" validate:"omitempty,max=255,noctrl"`
	City          string   `json:"city" validate:"omitempty,max=120,noctrl"`
	StateProvince string   `json:"state_province" validate:"omitempty,max=120,noctrl"`
	PostalCode    string   `json:"postal_code" validate:"omitempty,max=20,noctrl"`
	Country       string   `json:"country" validate:"omitempty,max=120,noctrl"`
	Units         int      `json:"units" validate:"required,min=1,max=100000"`
	Ownership     string   `json:"ownership" validate:"omitempty,oneof=owned managed"`
	OwnerName     string   `json:"owner_name" validate:"omitempty,max=200,noctrl"`
	YearBuilt     *int     `json:"year_built" validate:"omitempty,min=1800,max=2100"`
	OnboardDate   string   `json:"onboard_date" validate:"omitempty,datetime=2006-01-02"`
	Amenities     []string `json:"amenities" validate:"omitempty,max=32,dive,max=40,noctrl"`
	Notes         string   `json:"notes" validate:"omitempty,max=2000,noctrl"`
	Status        string   `json:"status" validate:"omitempty,oneof=onboarding active archived"`
}

// Sanitize trims and strips control characters from free-text fields
// before validation runs (see Sanitizable). Enum-constrained, numeric,
// and date fields have no free text to normalize.
func (r *CreatePropertyRequest) Sanitize() {
	r.Name = sanitizeString(r.Name)
	r.AddressLine1 = sanitizeString(r.AddressLine1)
	r.AddressLine2 = sanitizeString(r.AddressLine2)
	r.City = sanitizeString(r.City)
	r.StateProvince = sanitizeString(r.StateProvince)
	r.PostalCode = sanitizeString(r.PostalCode)
	r.Country = sanitizeString(r.Country)
	r.OwnerName = sanitizeString(r.OwnerName)
	r.Notes = sanitizeString(r.Notes)
	for i, a := range r.Amenities {
		r.Amenities[i] = sanitizeString(a)
	}
}

func (r CreatePropertyRequest) ToDomain(ownerID uuid.UUID) (domain.CreatePropertyInput, error) {
	input := domain.CreatePropertyInput{
		Name:          r.Name,
		Type:          domain.PropertyType(r.Type),
		AddressLine1:  r.AddressLine1,
		AddressLine2:  r.AddressLine2,
		City:          r.City,
		StateProvince: r.StateProvince,
		PostalCode:    r.PostalCode,
		Country:       r.Country,
		Units:         r.Units,
		OwnerName:     r.OwnerName,
		YearBuilt:     r.YearBuilt,
		Amenities:     r.Amenities,
		Notes:         r.Notes,
		Status:        domain.PropertyStatus(r.Status),
		OwnerID:       ownerID,
	}
	if r.Ownership != "" {
		o := domain.PropertyOwnership(r.Ownership)
		input.Ownership = &o
	}
	if r.OnboardDate != "" {
		t, err := time.Parse(propertyDateLayout, r.OnboardDate)
		if err != nil {
			return domain.CreatePropertyInput{}, err
		}
		input.OnboardDate = &t
	}
	return input, nil
}

type UpdatePropertyRequest struct {
	Name          *string  `json:"name" validate:"omitempty,min=1,max=200,noctrl"`
	Type          *string  `json:"type" validate:"omitempty,oneof=residential_single_unit residential_multi_unit commercial mixed_use"`
	AddressLine1  *string  `json:"address_line1" validate:"omitempty,min=1,max=255,noctrl"`
	AddressLine2  *string  `json:"address_line2" validate:"omitempty,max=255,noctrl"`
	City          *string  `json:"city" validate:"omitempty,max=120,noctrl"`
	StateProvince *string  `json:"state_province" validate:"omitempty,max=120,noctrl"`
	PostalCode    *string  `json:"postal_code" validate:"omitempty,max=20,noctrl"`
	Country       *string  `json:"country" validate:"omitempty,max=120,noctrl"`
	Units         *int     `json:"units" validate:"omitempty,min=1,max=100000"`
	Ownership     *string  `json:"ownership" validate:"omitempty,oneof=owned managed"`
	OwnerName     *string  `json:"owner_name" validate:"omitempty,max=200,noctrl"`
	YearBuilt     *int     `json:"year_built" validate:"omitempty,min=1800,max=2100"`
	OnboardDate   *string  `json:"onboard_date" validate:"omitempty,datetime=2006-01-02"`
	Amenities     []string `json:"amenities" validate:"omitempty,max=32,dive,max=40,noctrl"`
	Notes         *string  `json:"notes" validate:"omitempty,max=2000,noctrl"`
	Status        *string  `json:"status" validate:"omitempty,oneof=onboarding active archived"`
}

func (r *UpdatePropertyRequest) Sanitize() {
	if r.Name != nil {
		*r.Name = sanitizeString(*r.Name)
	}
	if r.AddressLine1 != nil {
		*r.AddressLine1 = sanitizeString(*r.AddressLine1)
	}
	if r.AddressLine2 != nil {
		*r.AddressLine2 = sanitizeString(*r.AddressLine2)
	}
	if r.City != nil {
		*r.City = sanitizeString(*r.City)
	}
	if r.StateProvince != nil {
		*r.StateProvince = sanitizeString(*r.StateProvince)
	}
	if r.PostalCode != nil {
		*r.PostalCode = sanitizeString(*r.PostalCode)
	}
	if r.Country != nil {
		*r.Country = sanitizeString(*r.Country)
	}
	if r.OwnerName != nil {
		*r.OwnerName = sanitizeString(*r.OwnerName)
	}
	if r.Notes != nil {
		*r.Notes = sanitizeString(*r.Notes)
	}
	for i, a := range r.Amenities {
		r.Amenities[i] = sanitizeString(a)
	}
}

func (r UpdatePropertyRequest) ToDomain() (domain.UpdatePropertyInput, error) {
	input := domain.UpdatePropertyInput{
		Name:          r.Name,
		AddressLine1:  r.AddressLine1,
		AddressLine2:  r.AddressLine2,
		City:          r.City,
		StateProvince: r.StateProvince,
		PostalCode:    r.PostalCode,
		Country:       r.Country,
		Units:         r.Units,
		OwnerName:     r.OwnerName,
		YearBuilt:     r.YearBuilt,
		Amenities:     r.Amenities,
		Notes:         r.Notes,
	}
	if r.Type != nil {
		t := domain.PropertyType(*r.Type)
		input.Type = &t
	}
	if r.Ownership != nil {
		o := domain.PropertyOwnership(*r.Ownership)
		input.Ownership = &o
	}
	if r.Status != nil {
		s := domain.PropertyStatus(*r.Status)
		input.Status = &s
	}
	if r.OnboardDate != nil {
		t, err := time.Parse(propertyDateLayout, *r.OnboardDate)
		if err != nil {
			return domain.UpdatePropertyInput{}, err
		}
		input.OnboardDate = &t
	}
	return input, nil
}

type BulkUpdatePropertyStatusRequest struct {
	IDs    []string `json:"ids" validate:"required,min=1,max=200,dive,uuid4"`
	Status string   `json:"status" validate:"required,oneof=onboarding active archived"`
}

func (r BulkUpdatePropertyStatusRequest) ToDomain() ([]uuid.UUID, domain.PropertyStatus, error) {
	ids := make([]uuid.UUID, len(r.IDs))
	for i, s := range r.IDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, "", err
		}
		ids[i] = id
	}
	return ids, domain.PropertyStatus(r.Status), nil
}

type PropertyResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Address       string   `json:"address"`
	AddressLine1  string   `json:"address_line1"`
	AddressLine2  string   `json:"address_line2,omitempty"`
	City          string   `json:"city,omitempty"`
	StateProvince string   `json:"state_province,omitempty"`
	PostalCode    string   `json:"postal_code,omitempty"`
	Country       string   `json:"country,omitempty"`
	Units         int      `json:"units"`
	Ownership     string   `json:"ownership,omitempty"`
	OwnerName     string   `json:"owner_name,omitempty"`
	YearBuilt     *int     `json:"year_built,omitempty"`
	OnboardDate   string   `json:"onboard_date,omitempty"`
	Amenities     []string `json:"amenities"`
	Notes         string   `json:"notes,omitempty"`
	Status        string   `json:"status"`
	// OccupancyPct and CollectedThisMonth mirror what the frontend's mock
	// list shows, but there's no units/leases/payments feature yet to
	// compute them from — hardcoded to 0 until that exists, rather than
	// storing a number nothing keeps up to date.
	OccupancyPct       int    `json:"occupancy_pct"`
	CollectedThisMonth int    `json:"collected_this_month"`
	OwnerID            string `json:"owner_id"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type PropertyListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func NewPropertyResponse(p *domain.Property) PropertyResponse {
	resp := PropertyResponse{
		ID:            p.ID.String(),
		Name:          p.Name,
		Type:          string(p.Type),
		Address:       p.FormattedAddress(),
		AddressLine1:  p.AddressLine1,
		AddressLine2:  p.AddressLine2,
		City:          p.City,
		StateProvince: p.StateProvince,
		PostalCode:    p.PostalCode,
		Country:       p.Country,
		Units:         p.Units,
		OwnerName:     p.OwnerName,
		YearBuilt:     p.YearBuilt,
		Amenities:     p.Amenities,
		Notes:         p.Notes,
		Status:        string(p.Status),
		OwnerID:       p.OwnerID.String(),
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt.Format(time.RFC3339),
	}
	if p.Ownership != nil {
		resp.Ownership = string(*p.Ownership)
	}
	if p.OnboardDate != nil {
		resp.OnboardDate = p.OnboardDate.Format(propertyDateLayout)
	}
	if resp.Amenities == nil {
		resp.Amenities = []string{}
	}
	return resp
}

func NewPropertyListResponse(properties []*domain.Property) []PropertyResponse {
	out := make([]PropertyResponse, len(properties))
	for i, p := range properties {
		out[i] = NewPropertyResponse(p)
	}
	return out
}
