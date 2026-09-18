package dto

import (
	"fmt"
	"time"

	"propertymanagement/internal/domain"
)

const vendorDateLayout = "2006-01-02"

type CreateVendorRequest struct {
	CompanyName         string   `json:"company_name" validate:"required,min=1,max=200,noctrl"`
	Categories          []string `json:"categories" validate:"required,min=1,max=10,dive,oneof=plumbing electrical hvac appliance pest_control general other"`
	ContactPerson       string   `json:"contact_person" validate:"omitempty,max=200,noctrl"`
	Phone               string   `json:"phone" validate:"omitempty,max=40,noctrl"`
	Email               string   `json:"email" validate:"omitempty,max=200,noctrl"`
	Address             string   `json:"address" validate:"omitempty,max=500,noctrl"`
	ServesAllProperties bool     `json:"serves_all_properties"`
	PropertiesServed    []string `json:"properties_served" validate:"omitempty,max=500,dive,uuid4"`
	InsuranceExpiry     string   `json:"insurance_expiry" validate:"omitempty,datetime=2006-01-02"`
	LicenseNumber       string   `json:"license_number" validate:"omitempty,max=100,noctrl"`
	LicenseExpiry       string   `json:"license_expiry" validate:"omitempty,datetime=2006-01-02"`
	COILink             string   `json:"coi_link" validate:"omitempty,max=500,noctrl"`
	TaxDocLink          string   `json:"tax_doc_link" validate:"omitempty,max=500,noctrl"`
	RateType            string   `json:"rate_type" validate:"omitempty,oneof=hourly flat"`
	RateAmount          *float64 `json:"rate_amount" validate:"omitempty,min=0"`
	PaymentTerms        string   `json:"payment_terms" validate:"omitempty,oneof=net_15 net_30 net_45"`
	InternalNotes       string   `json:"internal_notes" validate:"omitempty,max=2000,noctrl"`
}

func (r *CreateVendorRequest) Sanitize() {
	r.CompanyName = sanitizeString(r.CompanyName)
	r.ContactPerson = sanitizeString(r.ContactPerson)
	r.Phone = sanitizeString(r.Phone)
	r.Email = sanitizeString(r.Email)
	r.Address = sanitizeString(r.Address)
	r.LicenseNumber = sanitizeString(r.LicenseNumber)
	r.COILink = sanitizeString(r.COILink)
	r.TaxDocLink = sanitizeString(r.TaxDocLink)
	r.InternalNotes = sanitizeString(r.InternalNotes)
}

func (r CreateVendorRequest) ToDomain() (domain.CreateVendorInput, error) {
	input := domain.CreateVendorInput{
		CompanyName:         r.CompanyName,
		Categories:          toCategories(r.Categories),
		ContactPerson:       r.ContactPerson,
		Phone:               r.Phone,
		Email:               r.Email,
		Address:             r.Address,
		ServesAllProperties: r.ServesAllProperties,
		LicenseNumber:       r.LicenseNumber,
		COILink:             r.COILink,
		TaxDocLink:          r.TaxDocLink,
		InternalNotes:       r.InternalNotes,
	}

	propertiesServed, err := parseUUIDs(r.PropertiesServed)
	if err != nil {
		return domain.CreateVendorInput{}, fmt.Errorf("properties_served: %w", err)
	}
	input.PropertiesServed = propertiesServed

	if r.RateType != "" {
		t := domain.VendorRateType(r.RateType)
		input.RateType = &t
	}
	input.RateAmount = r.RateAmount
	if r.PaymentTerms != "" {
		t := domain.VendorPaymentTerms(r.PaymentTerms)
		input.PaymentTerms = &t
	}

	var perr error
	if input.InsuranceExpiry, perr = parseOptionalDate(r.InsuranceExpiry, vendorDateLayout); perr != nil {
		return domain.CreateVendorInput{}, fmt.Errorf("insurance_expiry: %w", perr)
	}
	if input.LicenseExpiry, perr = parseOptionalDate(r.LicenseExpiry, vendorDateLayout); perr != nil {
		return domain.CreateVendorInput{}, fmt.Errorf("license_expiry: %w", perr)
	}

	return input, nil
}

func toCategories(ss []string) []domain.WorkOrderCategory {
	out := make([]domain.WorkOrderCategory, len(ss))
	for i, s := range ss {
		out[i] = domain.WorkOrderCategory(s)
	}
	return out
}

type UpdateVendorRequest struct {
	CompanyName         *string  `json:"company_name" validate:"omitempty,min=1,max=200,noctrl"`
	Categories          []string `json:"categories" validate:"omitempty,max=10,dive,oneof=plumbing electrical hvac appliance pest_control general other"`
	ContactPerson       *string  `json:"contact_person" validate:"omitempty,max=200,noctrl"`
	Phone               *string  `json:"phone" validate:"omitempty,max=40,noctrl"`
	Email               *string  `json:"email" validate:"omitempty,max=200,noctrl"`
	Address             *string  `json:"address" validate:"omitempty,max=500,noctrl"`
	ServesAllProperties *bool    `json:"serves_all_properties"`
	// PropertiesServed present (even as []) means "replace the set" —
	// mirrors Lease.CoResidents's nil-vs-empty distinction.
	PropertiesServed []string `json:"properties_served" validate:"omitempty,max=500,dive,uuid4"`
	InsuranceExpiry  *string  `json:"insurance_expiry" validate:"omitempty,datetime=2006-01-02"`
	LicenseNumber    *string  `json:"license_number" validate:"omitempty,max=100,noctrl"`
	LicenseExpiry    *string  `json:"license_expiry" validate:"omitempty,datetime=2006-01-02"`
	COILink          *string  `json:"coi_link" validate:"omitempty,max=500,noctrl"`
	TaxDocLink       *string  `json:"tax_doc_link" validate:"omitempty,max=500,noctrl"`
	RateType         *string  `json:"rate_type" validate:"omitempty,oneof=hourly flat"`
	RateAmount       *float64 `json:"rate_amount" validate:"omitempty,min=0"`
	PaymentTerms     *string  `json:"payment_terms" validate:"omitempty,oneof=net_15 net_30 net_45"`
	InternalNotes    *string  `json:"internal_notes" validate:"omitempty,max=2000,noctrl"`
	Active           *bool    `json:"active"`
}

func (r *UpdateVendorRequest) Sanitize() {
	trim := func(s *string) {
		if s != nil {
			*s = sanitizeString(*s)
		}
	}
	trim(r.CompanyName)
	trim(r.ContactPerson)
	trim(r.Phone)
	trim(r.Email)
	trim(r.Address)
	trim(r.LicenseNumber)
	trim(r.COILink)
	trim(r.TaxDocLink)
	trim(r.InternalNotes)
}

func (r UpdateVendorRequest) ToDomain() (domain.UpdateVendorInput, error) {
	input := domain.UpdateVendorInput{
		CompanyName:         r.CompanyName,
		ContactPerson:       r.ContactPerson,
		Phone:               r.Phone,
		Email:               r.Email,
		Address:             r.Address,
		ServesAllProperties: r.ServesAllProperties,
		RateAmount:          r.RateAmount,
		LicenseNumber:       r.LicenseNumber,
		COILink:             r.COILink,
		TaxDocLink:          r.TaxDocLink,
		InternalNotes:       r.InternalNotes,
		Active:              r.Active,
	}
	if r.Categories != nil {
		input.Categories = toCategories(r.Categories)
	}
	if r.PropertiesServed != nil {
		ids, err := parseUUIDs(r.PropertiesServed)
		if err != nil {
			return domain.UpdateVendorInput{}, fmt.Errorf("properties_served: %w", err)
		}
		input.PropertiesServed = ids
		input.PropertiesServedSet = true
	}
	if r.RateType != nil {
		t := domain.VendorRateType(*r.RateType)
		input.RateType = &t
	}
	if r.PaymentTerms != nil {
		t := domain.VendorPaymentTerms(*r.PaymentTerms)
		input.PaymentTerms = &t
	}

	var perr error
	if input.InsuranceExpiry, perr = parseOptionalDate(derefString(r.InsuranceExpiry), vendorDateLayout); perr != nil {
		return domain.UpdateVendorInput{}, fmt.Errorf("insurance_expiry: %w", perr)
	}
	if input.LicenseExpiry, perr = parseOptionalDate(derefString(r.LicenseExpiry), vendorDateLayout); perr != nil {
		return domain.UpdateVendorInput{}, fmt.Errorf("license_expiry: %w", perr)
	}

	return input, nil
}

type VendorResponse struct {
	ID                  string   `json:"id"`
	CompanyName         string   `json:"company_name"`
	Categories          []string `json:"categories"`
	ContactPerson       string   `json:"contact_person,omitempty"`
	Phone               string   `json:"phone,omitempty"`
	Email               string   `json:"email,omitempty"`
	Address             string   `json:"address,omitempty"`
	ServesAllProperties bool     `json:"serves_all_properties"`
	InsuranceExpiry     string   `json:"insurance_expiry,omitempty"`
	InsuranceStatus     string   `json:"insurance_status"`
	LicenseNumber       string   `json:"license_number,omitempty"`
	LicenseExpiry       string   `json:"license_expiry,omitempty"`
	COILink             string   `json:"coi_link,omitempty"`
	TaxDocLink          string   `json:"tax_doc_link,omitempty"`
	RateType            string   `json:"rate_type,omitempty"`
	RateAmount          *float64 `json:"rate_amount,omitempty"`
	PaymentTerms        string   `json:"payment_terms,omitempty"`
	InternalNotes       string   `json:"internal_notes,omitempty"`
	Active              bool     `json:"active"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

func NewVendorResponse(v *domain.Vendor) VendorResponse {
	categories := make([]string, len(v.Categories))
	for i, c := range v.Categories {
		categories[i] = string(c)
	}

	resp := VendorResponse{
		ID:                  v.ID.String(),
		CompanyName:         v.CompanyName,
		Categories:          categories,
		ContactPerson:       v.ContactPerson,
		Phone:               v.Phone,
		Email:               v.Email,
		Address:             v.Address,
		ServesAllProperties: v.ServesAllProperties,
		InsuranceStatus:     string(v.InsuranceStatus(time.Now().UTC())),
		LicenseNumber:       v.LicenseNumber,
		COILink:             v.COILink,
		TaxDocLink:          v.TaxDocLink,
		RateAmount:          v.RateAmount,
		InternalNotes:       v.InternalNotes,
		Active:              v.Active,
		CreatedAt:           v.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           v.UpdatedAt.Format(time.RFC3339),
	}
	if v.InsuranceExpiry != nil {
		resp.InsuranceExpiry = v.InsuranceExpiry.Format(vendorDateLayout)
	}
	if v.LicenseExpiry != nil {
		resp.LicenseExpiry = v.LicenseExpiry.Format(vendorDateLayout)
	}
	if v.RateType != nil {
		resp.RateType = string(*v.RateType)
	}
	if v.PaymentTerms != nil {
		resp.PaymentTerms = string(*v.PaymentTerms)
	}
	if resp.Categories == nil {
		resp.Categories = []string{}
	}
	return resp
}

// VendorWithStatsResponse is VendorResponse plus the computed numbers
// the list row and profile header show.
type VendorWithStatsResponse struct {
	VendorResponse
	OpenWorkOrders        int      `json:"open_work_orders"`
	AverageRating         *float64 `json:"average_rating,omitempty"`
	PropertiesServedCount int      `json:"properties_served_count"`
}

func NewVendorWithStatsResponse(v *domain.VendorWithStats) VendorWithStatsResponse {
	return VendorWithStatsResponse{
		VendorResponse:        NewVendorResponse(&v.Vendor),
		OpenWorkOrders:        v.OpenWorkOrders,
		AverageRating:         v.AverageRating,
		PropertiesServedCount: v.PropertiesServedCount,
	}
}

func NewVendorWithStatsListResponse(vendors []*domain.VendorWithStats) []VendorWithStatsResponse {
	out := make([]VendorWithStatsResponse, len(vendors))
	for i, v := range vendors {
		out[i] = NewVendorWithStatsResponse(v)
	}
	return out
}

type VendorListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type VendorPropertiesServedResponse struct {
	PropertyIDs []string `json:"property_ids"`
}

type VendorSpendSummaryResponse struct {
	ThisMonth  float64 `json:"this_month"`
	YearToDate float64 `json:"year_to_date"`
}

func NewVendorSpendSummaryResponse(s *domain.VendorSpendSummary) VendorSpendSummaryResponse {
	return VendorSpendSummaryResponse{ThisMonth: s.ThisMonth, YearToDate: s.YearToDate}
}
