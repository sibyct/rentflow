package handlers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/response"
)

type BankAccountHandler struct {
	svc domain.BankAccountService
}

func NewBankAccountHandler(svc domain.BankAccountService) *BankAccountHandler {
	return &BankAccountHandler{svc: svc}
}

func (h *BankAccountHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "list bank accounts")
	if !ok {
		return
	}
	rows, err := h.svc.List(r.Context(), ownerID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewBankAccountRowListResponse(rows))
}

func (h *BankAccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "create bank account")
	if !ok {
		return
	}
	var req dto.BankAccountRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create bank account: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	a, err := h.svc.Create(r.Context(), ownerID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewBankAccountResponse(a))
}

func (h *BankAccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "update bank account")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.BankAccountRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update bank account: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	a, err := h.svc.Update(r.Context(), ownerID, id, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewBankAccountResponse(a))
}

func (h *BankAccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "delete bank account")
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

func (h *BankAccountHandler) Reconciliation(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "reconciliation")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	rec, err := h.svc.Reconciliation(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewReconciliationResponse(rec))
}

func (h *BankAccountHandler) ListLines(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "list statement lines")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	limit, offset := parsePagination(r)
	opts := domain.StatementLineListOptions{BankAccountID: id, Filter: domain.StatementLineFilter(r.URL.Query().Get("filter")), Limit: limit, Offset: offset}
	if opts.Filter != domain.StatementLineAll && opts.Filter != domain.StatementLineMatched && opts.Filter != domain.StatementLineUnmatched {
		response.WriteError(w, r, fmt.Errorf("filter %q: %w", opts.Filter, domain.ErrInvalidInput))
		return
	}
	lines, total, err := h.svc.ListStatementLines(r.Context(), ownerID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, dto.NewStatementLineListResponse(lines), dto.AccountingListMeta{Total: total, Limit: opts.Limit, Offset: opts.Offset})
}

func (h *BankAccountHandler) AddLines(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "add statement lines")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.AddStatementLinesRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	lines, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("add statement lines: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	added, err := h.svc.AddStatementLines(r.Context(), ownerID, id, domain.StatementLineSource(req.Source), lines)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]int{"added": added, "skipped": len(lines) - added})
}

func (h *BankAccountHandler) lineParams(w http.ResponseWriter, r *http.Request) (accountID, lineID uuid.UUID, ok bool) {
	accountID, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return uuid.UUID{}, uuid.UUID{}, false
	}
	lineID, err = uuidURLParam(r, "lineId")
	if err != nil {
		response.WriteError(w, r, err)
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return accountID, lineID, true
}

func (h *BankAccountHandler) DeleteLine(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "delete statement line")
	if !ok {
		return
	}
	accountID, lineID, ok := h.lineParams(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteStatementLine(r.Context(), ownerID, accountID, lineID); err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BankAccountHandler) Match(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "match statement line")
	if !ok {
		return
	}
	accountID, lineID, ok := h.lineParams(w, r)
	if !ok {
		return
	}
	var req dto.MatchRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	paymentID, err := uuid.Parse(req.PaymentID)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("payment_id: %w", domain.ErrInvalidInput))
		return
	}
	if err := h.svc.Match(r.Context(), ownerID, accountID, lineID, paymentID); err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BankAccountHandler) Unmatch(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "unmatch statement line")
	if !ok {
		return
	}
	accountID, lineID, ok := h.lineParams(w, r)
	if !ok {
		return
	}
	if err := h.svc.Unmatch(r.Context(), ownerID, accountID, lineID); err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
