package dto

import (
	"fmt"
	"time"

	"propertymanagement/internal/domain"
)

// TerminateLeaseRequest backs the dedicated Terminate Lease action
// (R10) — distinct from a generic lease update, since termination
// always sets the unit Vacant and writes an audit entry.
// MoveOutInspectionAttachmentID is optional and, when set, creates a
// unit-level document auto-linked to this lease (R15).
type TerminateLeaseRequest struct {
	TerminationReason             string `json:"termination_reason" validate:"required,oneof=non_renewal eviction mutual resident_notice other"`
	TerminationNoticeDate         string `json:"termination_notice_date" validate:"required,datetime=2006-01-02"`
	MoveOutDate                   string `json:"move_out_date" validate:"omitempty,datetime=2006-01-02"`
	MoveOutInspectionAttachmentID string `json:"move_out_inspection_attachment_id" validate:"omitempty,uuid4"`
}

func (r TerminateLeaseRequest) ToDomain() (domain.TerminateLeaseInput, error) {
	noticeDate, err := time.Parse(leaseDateLayout, r.TerminationNoticeDate)
	if err != nil {
		return domain.TerminateLeaseInput{}, fmt.Errorf("termination_notice_date: %w", err)
	}
	input := domain.TerminateLeaseInput{
		TerminationReason:     domain.TerminationReason(r.TerminationReason),
		TerminationNoticeDate: noticeDate,
	}
	if moveOut, err := parseOptionalLeaseDate(r.MoveOutDate); err != nil {
		return domain.TerminateLeaseInput{}, fmt.Errorf("move_out_date: %w", err)
	} else {
		input.MoveOutDate = moveOut
	}
	return input, nil
}
