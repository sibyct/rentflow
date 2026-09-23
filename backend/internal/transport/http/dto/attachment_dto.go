package dto

import (
	"propertymanagement/internal/domain"
)

type PresignAttachmentRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=300,noctrl"`
	ContentType string `json:"content_type" validate:"required,max=100"`
	SizeBytes   int64  `json:"size_bytes" validate:"required,gt=0"`
}

func (r *PresignAttachmentRequest) Sanitize() { r.Filename = sanitizeString(r.Filename) }

type AttachmentResponse struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Status      string `json:"status"`
}

func NewAttachmentResponse(a *domain.Attachment) AttachmentResponse {
	return AttachmentResponse{
		ID:          a.ID.String(),
		Filename:    a.Filename,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		Status:      string(a.Status),
	}
}

// PresignedUploadResponse is a presigned POST: send multipart/form-data
// to upload_url with every entry of fields first, and the file last.
type PresignedUploadResponse struct {
	Attachment AttachmentResponse `json:"attachment"`
	UploadURL  string             `json:"upload_url"`
	Fields     map[string]string  `json:"fields"`
}

func NewPresignedUploadResponse(p *domain.PresignedUpload) PresignedUploadResponse {
	return PresignedUploadResponse{
		Attachment: NewAttachmentResponse(p.Attachment),
		UploadURL:  p.URL,
		Fields:     p.Fields,
	}
}

type AttachmentURLResponse struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
}
