package handlers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/response"
)

type DepositHandler struct {
	svc domain.DepositService
}

func NewDepositHandler(svc domain.DepositService) *DepositHandler {
	return &DepositHandler{svc: svc}
}

func (h *DepositHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "list deposits")
	if !ok {
		return
	}
	var opts domain.DepositListOptions
	var err error
	opts.Limit, opts.Offset = parsePagination(r)
	opts.Search = r.URL.Query().Get("search")
	if opts.PropertyID, err = optionalUUIDQuery(r, "property_id"); err != nil {
		response.WriteError(w, r, err)
		return
	}
	if s := r.URL.Query().Get("status"); s != "" {
		status := domain.SecurityDepositStatus(s)
		if !status.Valid() {
			response.WriteError(w, r, fmt.Errorf("status %q: %w", s, domain.ErrInvalidInput))
			return
		}
		opts.Status = &status
	}
	opts.Sort = domain.DepositSortKey(r.URL.Query().Get("sort"))
	opts.SortDesc = sortDirection(r)

	rows, total, err := h.svc.List(r.Context(), ownerID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, dto.NewDepositListResponse(rows), dto.AccountingListMeta{Total: total, Limit: opts.Limit, Offset: opts.Offset})
}

func (h *DepositHandler) Get(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "get deposit")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	d, err := h.svc.Get(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewDepositResponse(d))
}

func (h *DepositHandler) SetAccount(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "set deposit account")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.SetHeldInAccountRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	var accountID *uuid.UUID
	if req.AccountID != "" {
		parsed, err := uuid.Parse(req.AccountID)
		if err != nil {
			response.WriteError(w, r, fmt.Errorf("account_id: %w", domain.ErrInvalidInput))
			return
		}
		accountID = &parsed
	}
	d, err := h.svc.SetHeldInAccount(r.Context(), ownerID, id, accountID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewDepositResponse(d))
}

func (h *DepositHandler) AddDeduction(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "add deduction")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.AddDeductionRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("add deduction: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	d, err := h.svc.AddDeduction(r.Context(), ownerID, id, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewDepositResponse(d))
}

func (h *DepositHandler) RemoveDeduction(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "remove deduction")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	deductionID, err := uuidURLParam(r, "deductionId")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	d, err := h.svc.RemoveDeduction(r.Context(), ownerID, id, deductionID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewDepositResponse(d))
}

func (h *DepositHandler) Settle(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "settle deposit")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.SettleDepositRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("settle deposit: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	d, err := h.svc.Settle(r.Context(), ownerID, id, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewDepositResponse(d))
}

func (h *DepositHandler) Forfeit(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "forfeit deposit")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	d, err := h.svc.Forfeit(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewDepositResponse(d))
}
