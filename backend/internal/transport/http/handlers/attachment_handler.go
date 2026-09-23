package handlers

import (
	"net/http"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/response"
)

type AttachmentHandler struct {
	svc domain.AttachmentService
}

func NewAttachmentHandler(svc domain.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{svc: svc}
}

func (h *AttachmentHandler) Presign(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "presign attachment")
	if !ok {
		return
	}
	var req dto.PresignAttachmentRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	up, err := h.svc.Presign(r.Context(), ownerID, req.Filename, req.ContentType, req.SizeBytes)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewPresignedUploadResponse(up))
}

func (h *AttachmentHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "confirm attachment")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	a, err := h.svc.Confirm(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewAttachmentResponse(a))
}

func (h *AttachmentHandler) DownloadURL(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "attachment url")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	url, filename, err := h.svc.DownloadURL(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.AttachmentURLResponse{URL: url, Filename: filename})
}
