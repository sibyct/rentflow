package dto

import (
	"fmt"
	"time"

	"propertymanagement/internal/domain"
)

// CorrectTerminationRequest backs the "Correct termination details"
// action — a separate, explicitly-labeled action for fixing a
// data-entry mistake in an already-terminated lease's termination
// record, never a silent inline edit. Reason is mandatory: every
// correction must explain itself in the audit trail.
type CorrectTerminationRequest struct {
	TerminationReason     string `json:"termination_reason" validate:"required,oneof=non_renewal eviction mutual resident_notice other"`
	TerminationNoticeDate string `json:"termination_notice_date" validate:"required,datetime=2006-01-02"`
	MoveOutDate           string `json:"move_out_date" validate:"omitempty,datetime=2006-01-02"`
	Reason                string `json:"reason" validate:"required,max=500,noctrl"`
}

func (r *CorrectTerminationRequest) Sanitize() {
	r.Reason = sanitizeString(r.Reason)
}

func (r CorrectTerminationRequest) ToDomain() (domain.CorrectTerminationInput, error) {
	noticeDate, err := time.Parse(leaseDateLayout, r.TerminationNoticeDate)
	if err != nil {
		return domain.CorrectTerminationInput{}, fmt.Errorf("termination_notice_date: %w", err)
	}
	input := domain.CorrectTerminationInput{
		TerminationReason:     domain.TerminationReason(r.TerminationReason),
		TerminationNoticeDate: noticeDate,
		Reason:                r.Reason,
	}
	if moveOut, err := parseOptionalLeaseDate(r.MoveOutDate); err != nil {
		return domain.CorrectTerminationInput{}, fmt.Errorf("move_out_date: %w", err)
	} else {
		input.MoveOutDate = moveOut
	}
	return input, nil
}
