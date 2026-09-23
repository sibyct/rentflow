package handlers

import (
	"fmt"
	"net/http"
	"time"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/response"
)

type ExpenseHandler struct {
	svc domain.ExpenseService
}

func NewExpenseHandler(svc domain.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{svc: svc}
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "create expense")
	if !ok {
		return
	}
	var req dto.ExpenseRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToCreateDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create expense: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	row, err := h.svc.Create(r.Context(), ownerID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewTransactionResponse(row, time.Now().UTC()))
}

func (h *ExpenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "get expense")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	row, err := h.svc.Get(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewTransactionResponse(row, time.Now().UTC()))
}

func (h *ExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "update expense")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.ExpenseRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToUpdateDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update expense: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	row, err := h.svc.Update(r.Context(), ownerID, id, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewTransactionResponse(row, time.Now().UTC()))
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "list expenses")
	if !ok {
		return
	}
	opts, err := parseExpenseListOptions(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	rows, total, err := h.svc.List(r.Context(), ownerID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, dto.NewTransactionListResponse(rows, time.Now().UTC()),
		dto.AccountingListMeta{Total: total, Limit: opts.Limit, Offset: opts.Offset})
}

func parseExpenseListOptions(r *http.Request) (domain.ExpenseListOptions, error) {
	q := r.URL.Query()
	var opts domain.ExpenseListOptions
	var err error

	opts.Limit, opts.Offset = parsePagination(r)
	opts.Search = q.Get("search")
	if opts.PropertyID, err = optionalUUIDQuery(r, "property_id"); err != nil {
		return opts, err
	}
	if opts.VendorID, err = optionalUUIDQuery(r, "vendor_id"); err != nil {
		return opts, err
	}
	if c := q.Get("category"); c != "" {
		cat := domain.ExpenseCategory(c)
		if !cat.Valid() {
			return opts, fmt.Errorf("category %q: %w", c, domain.ErrInvalidInput)
		}
		opts.Category = &cat
	}
	if s := q.Get("status"); s != "" {
		status := domain.ExpenseStatus(s)
		if !status.Valid() {
			return opts, fmt.Errorf("status %q: %w", s, domain.ErrInvalidInput)
		}
		opts.Status = &status
	}
	if opts.From, err = optionalDateQuery(r, "from"); err != nil {
		return opts, err
	}
	if opts.To, err = optionalDateQuery(r, "to"); err != nil {
		return opts, err
	}
	opts.Sort = domain.ExpenseSortKey(q.Get("sort"))
	opts.SortDesc = sortDirection(r)
	return opts, nil
}
