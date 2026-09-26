package dto

import (
	"time"

	"propertymanagement/internal/domain"
)

// AddUnitDocumentRequest only ever receives an already-confirmed
// attachment id — the upload itself goes through the generic
// presign/confirm attachment flow first, same as a work order's
// photo/invoice attachment fields. RelatedLeaseID is optional (R15): a
// unit-level document may reference a specific lease on this unit.
type AddUnitDocumentRequest struct {
	AttachmentID   string `json:"attachment_id" validate:"required,uuid4"`
	Category       string `json:"category" validate:"omitempty,oneof=inspection manual photo other"`
	RelatedLeaseID string `json:"related_lease_id" validate:"omitempty,uuid4"`
}

func (r AddUnitDocumentRequest) ToDomain() (attachmentID string, category domain.UnitDocumentCategory, relatedLeaseID string) {
	return r.AttachmentID, domain.UnitDocumentCategory(r.Category), r.RelatedLeaseID
}

type UnitDocumentResponse struct {
	ID             string    `json:"id"`
	UnitID         string    `json:"unit_id"`
	AttachmentID   string    `json:"attachment_id"`
	Category       string    `json:"category"`
	UploadedBy     string    `json:"uploaded_by"`
	UploadedByName string    `json:"uploaded_by_name"`
	RelatedLeaseID string    `json:"related_lease_id,omitempty"`
	IsAutomated    bool      `json:"is_automated"`
	Filename       string    `json:"filename"`
	ContentType    string    `json:"content_type"`
	SizeBytes      int64     `json:"size_bytes"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewUnitDocumentResponse(d *domain.UnitDocument) UnitDocumentResponse {
	resp := UnitDocumentResponse{
		ID:             d.ID.String(),
		UnitID:         d.UnitID.String(),
		AttachmentID:   d.AttachmentID.String(),
		Category:       string(d.Category),
		UploadedBy:     d.UploadedBy.String(),
		UploadedByName: d.UploadedByName,
		IsAutomated:    d.IsAutomated,
		Filename:       d.Filename,
		ContentType:    d.ContentType,
		SizeBytes:      d.SizeBytes,
		CreatedAt:      d.CreatedAt,
	}
	if d.RelatedLeaseID != nil {
		resp.RelatedLeaseID = d.RelatedLeaseID.String()
	}
	return resp
}

func NewUnitDocumentListResponse(docs []*domain.UnitDocument) []UnitDocumentResponse {
	out := make([]UnitDocumentResponse, len(docs))
	for i, d := range docs {
		out[i] = NewUnitDocumentResponse(d)
	}
	return out
}
