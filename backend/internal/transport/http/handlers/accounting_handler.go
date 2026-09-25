package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/middleware"
	"propertymanagement/internal/transport/http/response"
)

// AccountingHandler serves the ledger-wide endpoints: the dashboard,
// the late-fee settings, the Rent Roll, and payment/void/audit actions
// that apply to any ledger row. Expenses and charges have their own
// handlers (they add create/edit forms of their own).
type AccountingHandler struct {
	ledger   domain.LedgerService
	rentRoll domain.RentRollService
}

func NewAccountingHandler(ledger domain.LedgerService, rentRoll domain.RentRollService) *AccountingHandler {
	return &AccountingHandler{ledger: ledger, rentRoll: rentRoll}
}

func claimsOrUnauthorized(w http.ResponseWriter, r *http.Request, op string) (uuid.UUID, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("%s: %w", op, domain.ErrUnauthorized))
		return uuid.UUID{}, false
	}
	return claims.UserID, true
}

func uuidURLParam(r *http.Request, name string) (uuid.UUID, error) {
	raw := chi.URLParam(r, name)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%s %q: %w", name, raw, domain.ErrInvalidInput)
	}
	return id, nil
}

func optionalUUIDQuery(r *http.Request, name string) (*uuid.UUID, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", name, raw, domain.ErrInvalidInput)
	}
	return &id, nil
}

// parseMonthQuery accepts "YYYY-MM" (what a month picker sends) or a
// full "YYYY-MM-DD" date, and returns the first day of that month.
func parseMonthQuery(r *http.Request, name string) (time.Time, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{"2006-01", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return domain.FirstOfMonth(t), nil
		}
	}
	return time.Time{}, fmt.Errorf("%s %q: %w", name, raw, domain.ErrInvalidInput)
}

func optionalDateQuery(r *http.Request, name string) (*time.Time, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", name, raw, domain.ErrInvalidInput)
	}
	return &t, nil
}

func sortDirection(r *http.Request) bool {
	return r.URL.Query().Get("order") == "desc"
}

func (h *AccountingHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "accounting dashboard")
	if !ok {
		return
	}
	period := domain.DashboardPeriod(r.URL.Query().Get("period"))
	if period == "" {
		period = domain.DashboardPeriodSixMo
	}
	d, err := h.ledger.Dashboard(r.Context(), ownerID, period)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewAccountingDashboardResponse(d))
}

func (h *AccountingHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "get accounting settings")
	if !ok {
		return
	}
	s, err := h.ledger.GetSettings(r.Context(), ownerID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewAccountingSettingsResponse(s))
}

func (h *AccountingHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "update accounting settings")
	if !ok {
		return
	}
	var req dto.UpdateAccountingSettingsRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	s, err := h.ledger.UpdateSettings(r.Context(), ownerID, req.ToDomain())
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewAccountingSettingsResponse(s))
}

func (h *AccountingHandler) ListRentRoll(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "list rent roll")
	if !ok {
		return
	}
	opts, err := parseRentRollOptions(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	rows, total, err := h.rentRoll.List(r.Context(), ownerID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, dto.NewRentRollListResponse(rows, time.Now().UTC()),
		dto.AccountingListMeta{Total: total, Limit: opts.Limit, Offset: opts.Offset})
}

func parseRentRollOptions(r *http.Request) (domain.RentRollOptions, error) {
	q := r.URL.Query()
	var opts domain.RentRollOptions
	var err error

	opts.Limit, opts.Offset = parsePagination(r)
	if opts.PropertyID, err = optionalUUIDQuery(r, "property_id"); err != nil {
		return opts, err
	}
	if opts.LeaseID, err = optionalUUIDQuery(r, "lease_id"); err != nil {
		return opts, err
	}
	if s := q.Get("status"); s != "" {
		status := domain.PaymentStatus(s)
		if !status.Valid() {
			return opts, fmt.Errorf("status %q: %w", s, domain.ErrInvalidInput)
		}
		opts.Status = &status
	}
	if opts.PeriodFrom, err = parseMonthQuery(r, "from"); err != nil {
		return opts, err
	}
	if opts.PeriodTo, err = parseMonthQuery(r, "to"); err != nil {
		return opts, err
	}
	opts.Sort = domain.RentRollSortKey(q.Get("sort"))
	opts.SortDesc = sortDirection(r)
	return opts, nil
}

func (h *AccountingHandler) GenerateRent(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "generate rent")
	if !ok {
		return
	}
	var req dto.GenerateRentRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	period, err := time.Parse("2006-01-02", req.Period)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("generate rent: period: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	res, err := h.rentRoll.GenerateForPeriod(r.Context(), ownerID, period)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.GenerateResultResponse{RentCreated: res.RentCreated, LateFeeCreated: res.LateFeeCreated})
}

// RecordLeasePayment is the Rent Roll "Record Payment" action.
func (h *AccountingHandler) RecordLeasePayment(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "record lease payment")
	if !ok {
		return
	}
	leaseID, err := uuidURLParam(r, "leaseId")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.RecordPaymentRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("record lease payment: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	payments, err := h.ledger.RecordLeasePayment(r.Context(), ownerID, leaseID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewPaymentListResponse(payments))
}

func (h *AccountingHandler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "record payment")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	var req dto.RecordPaymentRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("record payment: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	row, err := h.ledger.RecordPayment(r.Context(), ownerID, id, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewTransactionResponse(row, time.Now().UTC()))
}

func (h *AccountingHandler) ListPayments(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "list payments")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	payments, err := h.ledger.ListPayments(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewPaymentListResponse(payments))
}

func (h *AccountingHandler) VoidTransaction(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "void transaction")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	if err := h.ledger.VoidTransaction(r.Context(), ownerID, id); err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AccountingHandler) VoidPayment(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "void payment")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	if err := h.ledger.VoidPayment(r.Context(), ownerID, id); err != nil {
		response.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AccountingHandler) TransactionAudit(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := claimsOrUnauthorized(w, r, "transaction audit")
	if !ok {
		return
	}
	id, err := uuidURLParam(r, "id")
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	entries, err := h.ledger.ListTransactionAudit(r.Context(), ownerID, id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewAuditListResponse(entries))
}
