package dto

import (
	"time"

	"propertymanagement/internal/domain"
)

// AddLeaseDocumentRequest mirrors AddUnitDocumentRequest — the upload
// itself goes through the generic presign/confirm attachment flow
// first. Lease-specific documents live only on their originating
// lease's Documents tab (R14).
type AddLeaseDocumentRequest struct {
	AttachmentID string `json:"attachment_id" validate:"required,uuid4"`
	Category     string `json:"category" validate:"omitempty,oneof=lease_agreement inspection notice renewal other"`
}

func (r AddLeaseDocumentRequest) ToDomain() (attachmentID string, category domain.LeaseDocumentCategory) {
	return r.AttachmentID, domain.LeaseDocumentCategory(r.Category)
}

type LeaseDocumentResponse struct {
	ID             string    `json:"id"`
	LeaseID        string    `json:"lease_id"`
	AttachmentID   string    `json:"attachment_id"`
	Category       string    `json:"category"`
	UploadedBy     string    `json:"uploaded_by"`
	UploadedByName string    `json:"uploaded_by_name"`
	IsAutomated    bool      `json:"is_automated"`
	Filename       string    `json:"filename"`
	ContentType    string    `json:"content_type"`
	SizeBytes      int64     `json:"size_bytes"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewLeaseDocumentResponse(d *domain.LeaseDocument) LeaseDocumentResponse {
	return LeaseDocumentResponse{
		ID:             d.ID.String(),
		LeaseID:        d.LeaseID.String(),
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
}

func NewLeaseDocumentListResponse(docs []*domain.LeaseDocument) []LeaseDocumentResponse {
	out := make([]LeaseDocumentResponse, len(docs))
	for i, d := range docs {
		out[i] = NewLeaseDocumentResponse(d)
	}
	return out
}
