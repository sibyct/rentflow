package handlers

import (
	"fmt"
	"net/http"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/response"
)

// PropertyOwnerHandler and OwnerStatementHandler serve the Owner
// Statements screen: the owners statements are addressed to, and the
// statements themselves.
type PropertyOwnerHandler struct {
	svc domain.PropertyOwnerService
}

func NewPropertyOwnerHandler(svc domain.PropertyOwnerService) *PropertyOwnerHandler {
	return &PropertyOwnerHandler{svc: svc}
}

func (h *PropertyOwnerHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "list owners")
	if !ok {
		return
	}
	rows, err := h.svc.List(r.Context(), ownerID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewPropertyOwnerListResponse(rows))
}

func (h *PropertyOwnerHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "create owner")
	if !ok {
		return
	}
	var req dto.PropertyOwnerRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create owner: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	o, err := h.svc.Create(r.Context(), ownerID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewPropertyOwnerResponse(o))
}

func (h *PropertyOwnerHandler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "update owner")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.PropertyOwnerRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update owner: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	o, err := h.svc.Update(r.Context(), ownerID, id, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewPropertyOwnerResponse(o))
}

func (h *PropertyOwnerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "delete owner")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	if err := h.svc.Delete(r.Context(), ownerID, id); err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type OwnerStatementHandler struct {
	svc domain.OwnerStatementService
}

func NewOwnerStatementHandler(svc domain.OwnerStatementService) *OwnerStatementHandler {
	return &OwnerStatementHandler{svc: svc}
}

func (h *OwnerStatementHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "list statements")
	if !ok {
		return
	}
	var opts domain.OwnerStatementListOptions
	var err error
	opts.Limit, opts.Offset = parsePagination(r)
	if opts.PropertyID, err = optionalUUIDQuery(r, "property_id"); err != nil {
		response.WriteError(w, r, err)
		return
	}
	if s := r.URL.Query().Get("status"); s != "" {
		status := domain.StatementStatus(s)
		if !status.Valid() {
			response.WriteError(w, r, fmt.Errorf("status %q: %w", s, domain.ErrInvalidInput))
			return
		}
		opts.Status = &status
	}
	rows, total, err := h.svc.List(r.Context(), ownerID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, dto.NewStatementListResponse(rows), dto.AccountingListMeta{Total: total, Limit: opts.Limit, Offset: opts.Offset})
}

func (h *OwnerStatementHandler) Generate(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "generate statement")
	if !ok {
		return
	}
	var req dto.GenerateStatementRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	propertyID, start, end, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("generate statement: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	st, err := h.svc.Generate(r.Context(), ownerID, propertyID, start, end)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewStatementResponse(st, true))
}

func (h *OwnerStatementHandler) Get(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "get statement")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	st, err := h.svc.Get(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewStatementResponse(st, true))
}

func (h *OwnerStatementHandler) Send(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "send statement")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	st, err := h.svc.Send(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusAccepted, dto.NewStatementResponse(st, false))
}

func (h *OwnerStatementHandler) MarkSent(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "mark statement sent")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	st, err := h.svc.MarkSent(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewStatementResponse(st, false))
}

func (h *OwnerStatementHandler) MarkPaid(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "mark statement paid")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.MarkStatementPaidRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	in, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("mark statement paid: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	st, err := h.svc.MarkPaid(r.Context(), ownerID, id, in)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewStatementResponse(st, false))
}

// PDF streams the statement rendered from its frozen snapshot. The
// browser fetches it with the Authorization header, so the frontend
// downloads it via fetch + blob rather than a plain link.
func (h *OwnerStatementHandler) PDF(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "statement pdf")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	filename, data, err := h.svc.PDF(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
