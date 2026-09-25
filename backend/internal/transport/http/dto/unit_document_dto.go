package dto

import (
	"time"

	"propertymanagement/internal/domain"
)

// AddUnitDocumentRequest only ever receives an already-confirmed
// attachment id — the upload itself goes through the generic
// presign/confirm attachment flow first, same as a work order's
// photo/invoice attachment fields.
type AddUnitDocumentRequest struct {
	AttachmentID string `json:"attachment_id" validate:"required,uuid4"`
	Category     string `json:"category" validate:"omitempty,oneof=inspection manual photo other"`
}

func (r AddUnitDocumentRequest) ToDomain() (attachmentID string, category domain.UnitDocumentCategory) {
	return r.AttachmentID, domain.UnitDocumentCategory(r.Category)
}

type UnitDocumentResponse struct {
	ID             string    `json:"id"`
	UnitID         string    `json:"unit_id"`
	AttachmentID   string    `json:"attachment_id"`
	Category       string    `json:"category"`
	UploadedBy     string    `json:"uploaded_by"`
	UploadedByName string    `json:"uploaded_by_name"`
	Filename       string    `json:"filename"`
	ContentType    string    `json:"content_type"`
	SizeBytes      int64     `json:"size_bytes"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewUnitDocumentResponse(d *domain.UnitDocument) UnitDocumentResponse {
	return UnitDocumentResponse{
		ID:             d.ID.String(),
		UnitID:         d.UnitID.String(),
		AttachmentID:   d.AttachmentID.String(),
		Category:       string(d.Category),
		UploadedBy:     d.UploadedBy.String(),
		UploadedByName: d.UploadedByName,
		Filename:       d.Filename,
		ContentType:    d.ContentType,
		SizeBytes:      d.SizeBytes,
		CreatedAt:      d.CreatedAt,
	}
}

func NewUnitDocumentListResponse(docs []*domain.UnitDocument) []UnitDocumentResponse {
	out := make([]UnitDocumentResponse, len(docs))
	for i, d := range docs {
		out[i] = NewUnitDocumentResponse(d)
	}
	return out
}
