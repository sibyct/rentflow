package dto

import (
	"time"

	"propertymanagement/internal/domain"
)

type CreateUnitRequest struct {
	UnitName        string   `json:"unit_name" validate:"required,min=1,max=100,noctrl"`
	Floor           string   `json:"floor" validate:"omitempty,max=50,noctrl"`
	UnitType        string   `json:"unit_type" validate:"required,oneof=studio 1br 2br 3br_plus commercial_suite other"`
	Bedrooms        *int     `json:"bedrooms" validate:"omitempty,min=0,max=20"`
	Bathrooms       *float64 `json:"bathrooms" validate:"omitempty,min=0,max=20"`
	Sqft            *int     `json:"sqft" validate:"omitempty,min=1,max=1000000"`
	Furnished       string   `json:"furnished" validate:"omitempty,oneof=unfurnished furnished partial"`
	Status          string   `json:"status" validate:"omitempty,oneof=vacant occupied maintenance off_market"`
	MarketRent      *float64 `json:"market_rent" validate:"omitempty,min=0"`
	CurrentRent     *float64 `json:"current_rent" validate:"omitempty,min=0"`
	SecurityDeposit *float64 `json:"security_deposit" validate:"omitempty,min=0"`
	RentDueDay      *int     `json:"rent_due_day" validate:"omitempty,min=1,max=31"`
	TenantName      string   `json:"tenant_name" validate:"omitempty,max=200,noctrl"`
	Notes           string   `json:"notes" validate:"omitempty,max=2000,noctrl"`
}

func (r *CreateUnitRequest) Sanitize() {
	r.UnitName = sanitizeString(r.UnitName)
	r.Floor = sanitizeString(r.Floor)
	r.TenantName = sanitizeString(r.TenantName)
	r.Notes = sanitizeString(r.Notes)
}

func (r CreateUnitRequest) ToDomain() domain.CreateUnitInput {
	input := domain.CreateUnitInput{
		UnitName:        r.UnitName,
		Floor:           r.Floor,
		Type:            domain.UnitType(r.UnitType),
		Bedrooms:        r.Bedrooms,
		Bathrooms:       r.Bathrooms,
		Sqft:            r.Sqft,
		Status:          domain.UnitStatus(r.Status),
		MarketRent:      r.MarketRent,
		CurrentRent:     r.CurrentRent,
		SecurityDeposit: r.SecurityDeposit,
		RentDueDay:      r.RentDueDay,
		TenantName:      r.TenantName,
		Notes:           r.Notes,
	}
	if r.Furnished != "" {
		f := domain.UnitFurnished(r.Furnished)
		input.Furnished = &f
	}
	return input
}

// BulkCreateUnitsRequest backs the spreadsheet-style "bulk add" flow —
// PropertyID comes from the URL, not the body, same as CreateUnitRequest.
type BulkCreateUnitsRequest struct {
	Units []CreateUnitRequest `json:"units" validate:"required,min=1,max=200,dive"`
}

func (r *BulkCreateUnitsRequest) Sanitize() {
	for i := range r.Units {
		r.Units[i].Sanitize()
	}
}

func (r BulkCreateUnitsRequest) ToDomain() []domain.CreateUnitInput {
	inputs := make([]domain.CreateUnitInput, len(r.Units))
	for i, u := range r.Units {
		inputs[i] = u.ToDomain()
	}
	return inputs
}

type UpdateUnitRequest struct {
	UnitName        *string  `json:"unit_name" validate:"omitempty,min=1,max=100,noctrl"`
	Floor           *string  `json:"floor" validate:"omitempty,max=50,noctrl"`
	UnitType        *string  `json:"unit_type" validate:"omitempty,oneof=studio 1br 2br 3br_plus commercial_suite other"`
	Bedrooms        *int     `json:"bedrooms" validate:"omitempty,min=0,max=20"`
	Bathrooms       *float64 `json:"bathrooms" validate:"omitempty,min=0,max=20"`
	Sqft            *int     `json:"sqft" validate:"omitempty,min=1,max=1000000"`
	Furnished       *string  `json:"furnished" validate:"omitempty,oneof=unfurnished furnished partial"`
	Status          *string  `json:"status" validate:"omitempty,oneof=vacant occupied maintenance off_market"`
	MarketRent      *float64 `json:"market_rent" validate:"omitempty,min=0"`
	CurrentRent     *float64 `json:"current_rent" validate:"omitempty,min=0"`
	SecurityDeposit *float64 `json:"security_deposit" validate:"omitempty,min=0"`
	RentDueDay      *int     `json:"rent_due_day" validate:"omitempty,min=1,max=31"`
	TenantName      *string  `json:"tenant_name" validate:"omitempty,max=200,noctrl"`
	Notes           *string  `json:"notes" validate:"omitempty,max=2000,noctrl"`
}

func (r *UpdateUnitRequest) Sanitize() {
	if r.UnitName != nil {
		*r.UnitName = sanitizeString(*r.UnitName)
	}
	if r.Floor != nil {
		*r.Floor = sanitizeString(*r.Floor)
	}
	if r.TenantName != nil {
		*r.TenantName = sanitizeString(*r.TenantName)
	}
	if r.Notes != nil {
		*r.Notes = sanitizeString(*r.Notes)
	}
}

func (r UpdateUnitRequest) ToDomain() domain.UpdateUnitInput {
	input := domain.UpdateUnitInput{
		UnitName:        r.UnitName,
		Floor:           r.Floor,
		Bedrooms:        r.Bedrooms,
		Bathrooms:       r.Bathrooms,
		Sqft:            r.Sqft,
		MarketRent:      r.MarketRent,
		CurrentRent:     r.CurrentRent,
		SecurityDeposit: r.SecurityDeposit,
		RentDueDay:      r.RentDueDay,
		TenantName:      r.TenantName,
		Notes:           r.Notes,
	}
	if r.UnitType != nil {
		t := domain.UnitType(*r.UnitType)
		input.Type = &t
	}
	if r.Furnished != nil {
		f := domain.UnitFurnished(*r.Furnished)
		input.Furnished = &f
	}
	if r.Status != nil {
		s := domain.UnitStatus(*r.Status)
		input.Status = &s
	}
	return input
}

type UnitResponse struct {
	ID              string   `json:"id"`
	PropertyID      string   `json:"property_id"`
	UnitName        string   `json:"unit_name"`
	Floor           string   `json:"floor,omitempty"`
	UnitType        string   `json:"unit_type"`
	Bedrooms        *int     `json:"bedrooms,omitempty"`
	Bathrooms       *float64 `json:"bathrooms,omitempty"`
	Sqft            *int     `json:"sqft,omitempty"`
	Furnished       string   `json:"furnished,omitempty"`
	Status          string   `json:"status"`
	MarketRent      *float64 `json:"market_rent,omitempty"`
	CurrentRent     *float64 `json:"current_rent,omitempty"`
	SecurityDeposit *float64 `json:"security_deposit,omitempty"`
	RentDueDay      *int     `json:"rent_due_day,omitempty"`
	TenantName      string   `json:"tenant_name,omitempty"`
	Notes           string   `json:"notes,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

func NewUnitResponse(u *domain.Unit) UnitResponse {
	resp := UnitResponse{
		ID:              u.ID.String(),
		PropertyID:      u.PropertyID.String(),
		UnitName:        u.UnitName,
		Floor:           u.Floor,
		UnitType:        string(u.Type),
		Bedrooms:        u.Bedrooms,
		Bathrooms:       u.Bathrooms,
		Sqft:            u.Sqft,
		Status:          string(u.Status),
		MarketRent:      u.MarketRent,
		CurrentRent:     u.CurrentRent,
		SecurityDeposit: u.SecurityDeposit,
		RentDueDay:      u.RentDueDay,
		TenantName:      u.TenantName,
		Notes:           u.Notes,
		CreatedAt:       u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       u.UpdatedAt.Format(time.RFC3339),
	}
	if u.Furnished != nil {
		resp.Furnished = string(*u.Furnished)
	}
	return resp
}

func NewUnitListResponse(units []*domain.Unit) []UnitResponse {
	out := make([]UnitResponse, len(units))
	for i, u := range units {
		out[i] = NewUnitResponse(u)
	}
	return out
}

// UnitWithPropertyResponse is UnitResponse plus the parent property's
// name — the shape the global Units page needs for its clickable
// Property column that a property-scoped unit list never has to carry.
type UnitWithPropertyResponse struct {
	UnitResponse
	PropertyName string `json:"property_name"`
}

func NewUnitWithPropertyResponse(u *domain.UnitWithProperty) UnitWithPropertyResponse {
	return UnitWithPropertyResponse{
		UnitResponse: NewUnitResponse(&u.Unit),
		PropertyName: u.PropertyName,
	}
}

func NewUnitWithPropertyListResponse(units []*domain.UnitWithProperty) []UnitWithPropertyResponse {
	out := make([]UnitWithPropertyResponse, len(units))
	for i, u := range units {
		out[i] = NewUnitWithPropertyResponse(u)
	}
	return out
}

type UnitListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type PropertyUnitStatsResponse struct {
	UnitCount      int     `json:"unit_count"`
	OccupiedCount  int     `json:"occupied_count"`
	OccupancyPct   int     `json:"occupancy_pct"`
	TotalCollected float64 `json:"total_collected"`
}

func NewPropertyUnitStatsResponse(s *domain.PropertyUnitStats) PropertyUnitStatsResponse {
	return PropertyUnitStatsResponse{
		UnitCount:      s.UnitCount,
		OccupiedCount:  s.OccupiedCount,
		OccupancyPct:   s.OccupancyPct,
		TotalCollected: s.TotalCollected,
	}
}
