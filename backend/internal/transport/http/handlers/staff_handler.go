package handlers

import (
	"fmt"
	"net/http"
	"time"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/middleware"
	"propertymanagement/internal/transport/http/response"
)

// StaffHandler serves the authenticated staff-management endpoints
// (/staff/*), gated by middleware.RequireAccountAdmin. InviteHandler
// (below) serves the public accept-invite endpoints an invite email
// links to, before the recipient has a password to authenticate with.
type StaffHandler struct {
	svc domain.StaffService
}

func NewStaffHandler(svc domain.StaffService) *StaffHandler {
	return &StaffHandler{svc: svc}
}

func (h *StaffHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list staff: %w", domain.ErrUnauthorized))
		return
	}
	members, err := h.svc.List(r.Context(), claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewStaffMemberListResponse(members, time.Now().UTC(), claims.ActorID))
}

func (h *StaffHandler) Invite(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("invite staff: %w", domain.ErrUnauthorized))
		return
	}
	var req dto.InviteStaffRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("invite staff: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	result, err := h.svc.Invite(r.Context(), claims.UserID, claims.ActorID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	now := time.Now().UTC()
	if result.Conflict != nil {
		conflict := dto.NewStaffMemberResponse(result.Conflict, now, claims.ActorID)
		response.JSON(w, http.StatusConflict, dto.InviteResultResponse{Conflict: &conflict})
		return
	}
	member := dto.NewStaffMemberResponse(result.Created, now, claims.ActorID)
	response.JSON(w, http.StatusCreated, dto.InviteResultResponse{Member: &member, InviteURL: result.InviteURL})
}

func (h *StaffHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("update staff: %w", domain.ErrUnauthorized))
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.UpdateStaffRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update staff: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	member, err := h.svc.Update(r.Context(), claims.UserID, claims.ActorID, id, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewStaffMemberResponse(member, time.Now().UTC(), claims.ActorID))
}

func (h *StaffHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("deactivate staff: %w", domain.ErrUnauthorized))
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	if err := h.svc.Deactivate(r.Context(), claims.UserID, claims.ActorID, id); err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StaffHandler) Reactivate(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("reactivate staff: %w", domain.ErrUnauthorized))
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	if err := h.svc.Reactivate(r.Context(), claims.UserID, claims.ActorID, id); err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StaffHandler) ResendInvite(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("resend invite: %w", domain.ErrUnauthorized))
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	url, err := h.svc.ResendInvite(r.Context(), claims.UserID, claims.ActorID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"invite_url": url})
}

func (h *StaffHandler) AuditLog(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("staff audit log: %w", domain.ErrUnauthorized))
		return
	}
	limit, offset := parsePagination(r)
	entries, total, err := h.svc.AuditLog(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, dto.NewStaffAuditListResponse(entries), dto.AccountingListMeta{Total: total, Limit: limit, Offset: offset})
}

// InviteHandler serves the two unauthenticated endpoints an invite
// email links to — reached before the recipient has any credentials.
type InviteHandler struct {
	svc domain.StaffService
}

func NewInviteHandler(svc domain.StaffService) *InviteHandler {
	return &InviteHandler{svc: svc}
}

func (h *InviteHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	lookup, err := h.svc.LookupInvite(r.Context(), token)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewInviteLookupResponse(lookup))
}

func (h *InviteHandler) Accept(w http.ResponseWriter, r *http.Request) {
	var req dto.AcceptInviteRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	if err := h.svc.AcceptInvite(r.Context(), domain.AcceptInviteInput{Token: req.Token, Password: req.Password}); err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
