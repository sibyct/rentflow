package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

const leaseDateLayout = "2006-01-02"

type CreateLeaseRequest struct {
	LeaseType            string   `json:"lease_type" validate:"required,oneof=fixed month_to_month"`
	Status               string   `json:"status" validate:"omitempty,oneof=draft active terminated"`
	StartDate            string   `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate              string   `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
	MoveInDate           string   `json:"move_in_date" validate:"omitempty,datetime=2006-01-02"`
	MoveOutDate          string   `json:"move_out_date" validate:"omitempty,datetime=2006-01-02"`
	MonthlyRent          float64  `json:"monthly_rent" validate:"required,min=0"`
	SecurityDeposit      *float64 `json:"security_deposit" validate:"omitempty,min=0"`
	DepositStatus        string   `json:"deposit_status" validate:"omitempty,oneof=held partially_returned returned forfeited"`
	RentDueDay           *int     `json:"rent_due_day" validate:"omitempty,min=1,max=31"`
	LateFeeAmount        *float64 `json:"late_fee_amount" validate:"omitempty,min=0"`
	LateFeeGraceDays     *int     `json:"late_fee_grace_days" validate:"omitempty,min=0"`
	PrimaryResidentName  string   `json:"primary_resident_name" validate:"required,min=1,max=200,noctrl"`
	PrimaryResidentPhone string   `json:"primary_resident_phone" validate:"omitempty,max=50,noctrl"`
	PrimaryResidentEmail string   `json:"primary_resident_email" validate:"omitempty,max=200,noctrl,email"`
	CoResidents          []string `json:"co_residents" validate:"omitempty,max=10,dive,max=200,noctrl"`
	EmergencyContact     string   `json:"emergency_contact" validate:"omitempty,max=200,noctrl"`
	Notes                string   `json:"notes" validate:"omitempty,max=2000,noctrl"`
}

func (r *CreateLeaseRequest) Sanitize() {
	r.PrimaryResidentName = sanitizeString(r.PrimaryResidentName)
	r.PrimaryResidentPhone = sanitizeString(r.PrimaryResidentPhone)
	r.PrimaryResidentEmail = sanitizeString(r.PrimaryResidentEmail)
	r.EmergencyContact = sanitizeString(r.EmergencyContact)
	r.Notes = sanitizeString(r.Notes)
	for i, c := range r.CoResidents {
		r.CoResidents[i] = sanitizeString(c)
	}
}

// ToDomain takes unitID from the URL (see UnitHandler.Create's
// identical rationale) — a lease is always created in the context of
// the unit page the caller is already on.
func (r CreateLeaseRequest) ToDomain(unitID uuid.UUID) (domain.CreateLeaseInput, error) {
	input := domain.CreateLeaseInput{
		UnitID:               unitID,
		Type:                 domain.LeaseType(r.LeaseType),
		Status:               domain.LeaseStatus(r.Status),
		MonthlyRent:          r.MonthlyRent,
		SecurityDeposit:      r.SecurityDeposit,
		RentDueDay:           r.RentDueDay,
		LateFeeAmount:        r.LateFeeAmount,
		LateFeeGraceDays:     r.LateFeeGraceDays,
		PrimaryResidentName:  r.PrimaryResidentName,
		PrimaryResidentPhone: r.PrimaryResidentPhone,
		PrimaryResidentEmail: r.PrimaryResidentEmail,
		CoResidents:          r.CoResidents,
		EmergencyContact:     r.EmergencyContact,
		Notes:                r.Notes,
	}
	if r.DepositStatus != "" {
		d := domain.DepositStatus(r.DepositStatus)
		input.DepositStatus = &d
	}

	start, err := time.Parse(leaseDateLayout, r.StartDate)
	if err != nil {
		return domain.CreateLeaseInput{}, fmt.Errorf("start_date: %w", err)
	}
	input.StartDate = start

	if end, err := parseOptionalLeaseDate(r.EndDate); err != nil {
		return domain.CreateLeaseInput{}, fmt.Errorf("end_date: %w", err)
	} else {
		input.EndDate = end
	}
	if t, err := parseOptionalLeaseDate(r.MoveInDate); err != nil {
		return domain.CreateLeaseInput{}, fmt.Errorf("move_in_date: %w", err)
	} else {
		input.MoveInDate = t
	}
	if t, err := parseOptionalLeaseDate(r.MoveOutDate); err != nil {
		return domain.CreateLeaseInput{}, fmt.Errorf("move_out_date: %w", err)
	} else {
		input.MoveOutDate = t
	}

	return input, nil
}

type UpdateLeaseRequest struct {
	LeaseType             *string  `json:"lease_type" validate:"omitempty,oneof=fixed month_to_month"`
	Status                *string  `json:"status" validate:"omitempty,oneof=draft active terminated"`
	StartDate             *string  `json:"start_date" validate:"omitempty,datetime=2006-01-02"`
	EndDate               *string  `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
	MoveInDate            *string  `json:"move_in_date" validate:"omitempty,datetime=2006-01-02"`
	MoveOutDate           *string  `json:"move_out_date" validate:"omitempty,datetime=2006-01-02"`
	MonthlyRent           *float64 `json:"monthly_rent" validate:"omitempty,min=0"`
	SecurityDeposit       *float64 `json:"security_deposit" validate:"omitempty,min=0"`
	DepositStatus         *string  `json:"deposit_status" validate:"omitempty,oneof=held partially_returned returned forfeited"`
	RentDueDay            *int     `json:"rent_due_day" validate:"omitempty,min=1,max=31"`
	LateFeeAmount         *float64 `json:"late_fee_amount" validate:"omitempty,min=0"`
	LateFeeGraceDays      *int     `json:"late_fee_grace_days" validate:"omitempty,min=0"`
	PrimaryResidentName   *string  `json:"primary_resident_name" validate:"omitempty,min=1,max=200,noctrl"`
	PrimaryResidentPhone  *string  `json:"primary_resident_phone" validate:"omitempty,max=50,noctrl"`
	PrimaryResidentEmail  *string  `json:"primary_resident_email" validate:"omitempty,max=200,noctrl,email"`
	CoResidents           []string `json:"co_residents" validate:"omitempty,max=10,dive,max=200,noctrl"`
	EmergencyContact      *string  `json:"emergency_contact" validate:"omitempty,max=200,noctrl"`
	RenewalStatus         *string  `json:"renewal_status" validate:"omitempty,oneof=not_started offered accepted declined"`
	ProposedRent          *float64 `json:"proposed_rent" validate:"omitempty,min=0"`
	ProposedEndDate       *string  `json:"proposed_end_date" validate:"omitempty,datetime=2006-01-02"`
	OfferSentDate         *string  `json:"offer_sent_date" validate:"omitempty,datetime=2006-01-02"`
	TerminationReason     *string  `json:"termination_reason" validate:"omitempty,oneof=non_renewal eviction mutual resident_notice other"`
	TerminationNoticeDate *string  `json:"termination_notice_date" validate:"omitempty,datetime=2006-01-02"`
	Signed                *bool    `json:"signed"`
	SignedDate            *string  `json:"signed_date" validate:"omitempty,datetime=2006-01-02"`
	Notes                 *string  `json:"notes" validate:"omitempty,max=2000,noctrl"`
}

func (r *UpdateLeaseRequest) Sanitize() {
	if r.PrimaryResidentName != nil {
		*r.PrimaryResidentName = sanitizeString(*r.PrimaryResidentName)
	}
	if r.PrimaryResidentPhone != nil {
		*r.PrimaryResidentPhone = sanitizeString(*r.PrimaryResidentPhone)
	}
	if r.PrimaryResidentEmail != nil {
		*r.PrimaryResidentEmail = sanitizeString(*r.PrimaryResidentEmail)
	}
	if r.EmergencyContact != nil {
		*r.EmergencyContact = sanitizeString(*r.EmergencyContact)
	}
	if r.Notes != nil {
		*r.Notes = sanitizeString(*r.Notes)
	}
	for i, c := range r.CoResidents {
		r.CoResidents[i] = sanitizeString(c)
	}
}

func (r UpdateLeaseRequest) ToDomain() (domain.UpdateLeaseInput, error) {
	input := domain.UpdateLeaseInput{
		MonthlyRent:          r.MonthlyRent,
		SecurityDeposit:      r.SecurityDeposit,
		RentDueDay:           r.RentDueDay,
		LateFeeAmount:        r.LateFeeAmount,
		LateFeeGraceDays:     r.LateFeeGraceDays,
		PrimaryResidentName:  r.PrimaryResidentName,
		PrimaryResidentPhone: r.PrimaryResidentPhone,
		PrimaryResidentEmail: r.PrimaryResidentEmail,
		CoResidents:          r.CoResidents,
		EmergencyContact:     r.EmergencyContact,
		ProposedRent:         r.ProposedRent,
		Signed:               r.Signed,
		Notes:                r.Notes,
	}
	if r.LeaseType != nil {
		t := domain.LeaseType(*r.LeaseType)
		input.Type = &t
	}
	if r.Status != nil {
		s := domain.LeaseStatus(*r.Status)
		input.Status = &s
	}
	if r.DepositStatus != nil {
		d := domain.DepositStatus(*r.DepositStatus)
		input.DepositStatus = &d
	}
	if r.RenewalStatus != nil {
		rs := domain.RenewalStatus(*r.RenewalStatus)
		input.RenewalStatus = &rs
	}
	if r.TerminationReason != nil {
		tr := domain.TerminationReason(*r.TerminationReason)
		input.TerminationReason = &tr
	}

	var err error
	if input.ProposedEndDate, err = parseOptionalLeaseDate(derefString(r.ProposedEndDate)); err != nil {
		return domain.UpdateLeaseInput{}, fmt.Errorf("proposed_end_date: %w", err)
	}
	if input.OfferSentDate, err = parseOptionalLeaseDate(derefString(r.OfferSentDate)); err != nil {
		return domain.UpdateLeaseInput{}, fmt.Errorf("offer_sent_date: %w", err)
	}
	if input.StartDate, err = parseOptionalLeaseDate(derefString(r.StartDate)); err != nil {
		return domain.UpdateLeaseInput{}, fmt.Errorf("start_date: %w", err)
	}
	if input.EndDate, err = parseOptionalLeaseDate(derefString(r.EndDate)); err != nil {
		return domain.UpdateLeaseInput{}, fmt.Errorf("end_date: %w", err)
	}
	if input.MoveInDate, err = parseOptionalLeaseDate(derefString(r.MoveInDate)); err != nil {
		return domain.UpdateLeaseInput{}, fmt.Errorf("move_in_date: %w", err)
	}
	if input.MoveOutDate, err = parseOptionalLeaseDate(derefString(r.MoveOutDate)); err != nil {
		return domain.UpdateLeaseInput{}, fmt.Errorf("move_out_date: %w", err)
	}
	if input.TerminationNoticeDate, err = parseOptionalLeaseDate(derefString(r.TerminationNoticeDate)); err != nil {
		return domain.UpdateLeaseInput{}, fmt.Errorf("termination_notice_date: %w", err)
	}
	if input.SignedDate, err = parseOptionalLeaseDate(derefString(r.SignedDate)); err != nil {
		return domain.UpdateLeaseInput{}, fmt.Errorf("signed_date: %w", err)
	}

	return input, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseOptionalLeaseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(leaseDateLayout, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type LeaseResponse struct {
	ID        string `json:"id"`
	UnitID    string `json:"unit_id"`
	LeaseType string `json:"lease_type"`
	Status    string `json:"status"`
	// DisplayStatus is the richer, date-derived status a list/detail
	// view actually shows — see domain.Lease.DisplayStatus.
	DisplayStatus         string   `json:"display_status"`
	StartDate             string   `json:"start_date"`
	EndDate               string   `json:"end_date,omitempty"`
	MoveInDate            string   `json:"move_in_date,omitempty"`
	MoveOutDate           string   `json:"move_out_date,omitempty"`
	MonthlyRent           float64  `json:"monthly_rent"`
	SecurityDeposit       *float64 `json:"security_deposit,omitempty"`
	DepositStatus         string   `json:"deposit_status,omitempty"`
	RentDueDay            *int     `json:"rent_due_day,omitempty"`
	LateFeeAmount         *float64 `json:"late_fee_amount,omitempty"`
	LateFeeGraceDays      *int     `json:"late_fee_grace_days,omitempty"`
	PrimaryResidentName   string   `json:"primary_resident_name"`
	PrimaryResidentPhone  string   `json:"primary_resident_phone,omitempty"`
	PrimaryResidentEmail  string   `json:"primary_resident_email,omitempty"`
	CoResidents           []string `json:"co_residents"`
	EmergencyContact      string   `json:"emergency_contact,omitempty"`
	RenewalStatus         string   `json:"renewal_status"`
	ProposedRent          *float64 `json:"proposed_rent,omitempty"`
	ProposedEndDate       string   `json:"proposed_end_date,omitempty"`
	OfferSentDate         string   `json:"offer_sent_date,omitempty"`
	TerminationReason     string   `json:"termination_reason,omitempty"`
	TerminationNoticeDate string   `json:"termination_notice_date,omitempty"`
	RenewedIntoLeaseID    string   `json:"renewed_into_lease_id,omitempty"`
	RenewedFromLeaseID    string   `json:"renewed_from_lease_id,omitempty"`
	Signed                bool     `json:"signed"`
	SignedDate            string   `json:"signed_date,omitempty"`
	Notes                 string   `json:"notes,omitempty"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
}

func NewLeaseResponse(l *domain.Lease) LeaseResponse {
	resp := LeaseResponse{
		ID:                   l.ID.String(),
		UnitID:               l.UnitID.String(),
		LeaseType:            string(l.Type),
		Status:               string(l.Status),
		DisplayStatus:        string(l.DisplayStatus(time.Now().UTC())),
		StartDate:            l.StartDate.Format(leaseDateLayout),
		MonthlyRent:          l.MonthlyRent,
		SecurityDeposit:      l.SecurityDeposit,
		RentDueDay:           l.RentDueDay,
		LateFeeAmount:        l.LateFeeAmount,
		LateFeeGraceDays:     l.LateFeeGraceDays,
		PrimaryResidentName:  l.PrimaryResidentName,
		PrimaryResidentPhone: l.PrimaryResidentPhone,
		PrimaryResidentEmail: l.PrimaryResidentEmail,
		CoResidents:          l.CoResidents,
		EmergencyContact:     l.EmergencyContact,
		RenewalStatus:        string(l.RenewalStatus),
		ProposedRent:         l.ProposedRent,
		Signed:               l.Signed,
		Notes:                l.Notes,
		CreatedAt:            l.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            l.UpdatedAt.Format(time.RFC3339),
	}
	if l.EndDate != nil {
		resp.EndDate = l.EndDate.Format(leaseDateLayout)
	}
	if l.MoveInDate != nil {
		resp.MoveInDate = l.MoveInDate.Format(leaseDateLayout)
	}
	if l.MoveOutDate != nil {
		resp.MoveOutDate = l.MoveOutDate.Format(leaseDateLayout)
	}
	if l.DepositStatus != nil {
		resp.DepositStatus = string(*l.DepositStatus)
	}
	if l.ProposedEndDate != nil {
		resp.ProposedEndDate = l.ProposedEndDate.Format(leaseDateLayout)
	}
	if l.OfferSentDate != nil {
		resp.OfferSentDate = l.OfferSentDate.Format(leaseDateLayout)
	}
	if l.TerminationReason != nil {
		resp.TerminationReason = string(*l.TerminationReason)
	}
	if l.TerminationNoticeDate != nil {
		resp.TerminationNoticeDate = l.TerminationNoticeDate.Format(leaseDateLayout)
	}
	if l.RenewedIntoLeaseID != nil {
		resp.RenewedIntoLeaseID = l.RenewedIntoLeaseID.String()
	}
	if l.RenewedFromLeaseID != nil {
		resp.RenewedFromLeaseID = l.RenewedFromLeaseID.String()
	}
	if l.SignedDate != nil {
		resp.SignedDate = l.SignedDate.Format(leaseDateLayout)
	}
	if resp.CoResidents == nil {
		resp.CoResidents = []string{}
	}
	return resp
}

// LeaseWithUnitPropertyResponse is LeaseResponse plus the unit/property
// context a lease detail or list view needs — see
// UnitWithPropertyResponse's identical role for units.
type LeaseWithUnitPropertyResponse struct {
	LeaseResponse
	UnitName     string `json:"unit_name"`
	PropertyID   string `json:"property_id"`
	PropertyName string `json:"property_name"`
}

func NewLeaseWithUnitPropertyResponse(l *domain.LeaseWithUnitProperty) LeaseWithUnitPropertyResponse {
	return LeaseWithUnitPropertyResponse{
		LeaseResponse: NewLeaseResponse(&l.Lease),
		UnitName:      l.UnitName,
		PropertyID:    l.PropertyID.String(),
		PropertyName:  l.PropertyName,
	}
}

func NewLeaseWithUnitPropertyListResponse(leases []*domain.LeaseWithUnitProperty) []LeaseWithUnitPropertyResponse {
	out := make([]LeaseWithUnitPropertyResponse, len(leases))
	for i, l := range leases {
		out[i] = NewLeaseWithUnitPropertyResponse(l)
	}
	return out
}

type LeaseListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
