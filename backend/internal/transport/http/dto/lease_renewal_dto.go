package dto

import (
	"fmt"

	"propertymanagement/internal/domain"
)

// GenerateRenewalRequest turns an Accepted renewal into a real new
// Lease record (R7). Every field but StartDate defaults to the source
// lease's stored ProposedRent/ProposedEndDate/etc. when omitted — see
// LeaseService.GenerateRenewalLease.
type GenerateRenewalRequest struct {
	StartDate       string   `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate         string   `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
	MonthlyRent     *float64 `json:"monthly_rent" validate:"omitempty,min=0"`
	SecurityDeposit *float64 `json:"security_deposit" validate:"omitempty,min=0"`
	RentDueDay      *int     `json:"rent_due_day" validate:"omitempty,min=1,max=31"`
}

func (r GenerateRenewalRequest) ToDomain() (domain.GenerateRenewalLeaseInput, error) {
	start, err := parseOptionalLeaseDate(r.StartDate)
	if err != nil {
		return domain.GenerateRenewalLeaseInput{}, fmt.Errorf("start_date: %w", err)
	}
	input := domain.GenerateRenewalLeaseInput{
		MonthlyRent:     r.MonthlyRent,
		SecurityDeposit: r.SecurityDeposit,
		RentDueDay:      r.RentDueDay,
	}
	if start != nil {
		input.StartDate = *start
	}
	if end, err := parseOptionalLeaseDate(r.EndDate); err != nil {
		return domain.GenerateRenewalLeaseInput{}, fmt.Errorf("end_date: %w", err)
	} else {
		input.EndDate = end
	}
	return input, nil
}
