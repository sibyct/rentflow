package handlers

import (
	"fmt"
	"net/http"
	"time"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/response"
)

type ChargeHandler struct {
	svc domain.ChargeService
}

func NewChargeHandler(svc domain.ChargeService) *ChargeHandler {
	return &ChargeHandler{svc: svc}
}

func (h *ChargeHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "create charge")
	if !ok {
		return
	}
	var req dto.CreateChargeRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create charge: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	row, err := h.svc.Create(r.Context(), ownerID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewTransactionResponse(row, time.Now().UTC()))
}

func (h *ChargeHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "list charges")
	if !ok {
		return
	}
	var opts domain.ChargeListOptions
	var err error
	opts.Limit, opts.Offset = parsePagination(r)
	if opts.PropertyID, err = optionalUUIDQuery(r, "property_id"); err != nil {
		response.WriteError(w, r, err)
		return
	}
	if s := r.URL.Query().Get("status"); s != "" {
		status := domain.PaymentStatus(s)
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
	response.JSONWithMeta(w, http.StatusOK, dto.NewTransactionListResponse(rows, time.Now().UTC()),
		dto.AccountingListMeta{Total: total, Limit: opts.Limit, Offset: opts.Offset})
}
