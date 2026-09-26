package dto

import (
	"fmt"
	"time"

	"propertymanagement/internal/domain"
)

// ChangeRentRequest is the only way a client can alter an active
// lease's effective rent — see R3/R4. EffectiveDate must be today or
// later unless IsCorrection is set (R6).
type ChangeRentRequest struct {
	Amount        float64 `json:"amount" validate:"required,min=0"`
	EffectiveDate string  `json:"effective_date" validate:"required,datetime=2006-01-02"`
	Reason        string  `json:"reason" validate:"omitempty,max=500,noctrl"`
	IsCorrection  bool    `json:"is_correction"`
}

func (r *ChangeRentRequest) Sanitize() {
	r.Reason = sanitizeString(r.Reason)
}

func (r ChangeRentRequest) ToDomain() (domain.ChangeRentInput, error) {
	effective, err := time.Parse(leaseDateLayout, r.EffectiveDate)
	if err != nil {
		return domain.ChangeRentInput{}, fmt.Errorf("effective_date: %w", err)
	}
	return domain.ChangeRentInput{
		Amount:        r.Amount,
		EffectiveDate: effective,
		Reason:        r.Reason,
		IsCorrection:  r.IsCorrection,
	}, nil
}

type LeaseRentHistoryResponse struct {
	ID            string  `json:"id"`
	LeaseID       string  `json:"lease_id"`
	Amount        float64 `json:"amount"`
	EffectiveDate string  `json:"effective_date"`
	Reason        string  `json:"reason,omitempty"`
	IsCorrection  bool    `json:"is_correction"`
	CreatedBy     string  `json:"created_by"`
	CreatedAt     string  `json:"created_at"`
}

func NewLeaseRentHistoryListResponse(entries []*domain.LeaseRentHistoryEntry) []LeaseRentHistoryResponse {
	out := make([]LeaseRentHistoryResponse, len(entries))
	for i, e := range entries {
		out[i] = LeaseRentHistoryResponse{
			ID:            e.ID.String(),
			LeaseID:       e.LeaseID.String(),
			Amount:        e.Amount,
			EffectiveDate: e.EffectiveDate.Format(leaseDateLayout),
			Reason:        e.Reason,
			IsCorrection:  e.IsCorrection,
			CreatedBy:     e.CreatedBy.String(),
			CreatedAt:     e.CreatedAt.Format(time.RFC3339),
		}
	}
	return out
}
