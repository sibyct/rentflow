package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/middleware"
	"propertymanagement/internal/transport/http/response"
)

type WorkOrderHandler struct {
	svc domain.WorkOrderService
}

func NewWorkOrderHandler(svc domain.WorkOrderService) *WorkOrderHandler {
	return &WorkOrderHandler{svc: svc}
}

// Create backs the property-scoped "new work order" flow — propertyId
// comes from the URL, never a client-supplied body field, same
// rationale as UnitHandler.Create.
func (h *WorkOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("create work order: %w", domain.ErrUnauthorized))
		return
	}

	propertyID, err := parsePropertyIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.CreateWorkOrderRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain(propertyID)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create work order: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	wo, err := h.svc.CreateWorkOrder(r.Context(), claims.UserID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewWorkOrderResponse(wo))
}

func (h *WorkOrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("get work order: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseWorkOrderIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	wo, err := h.svc.GetWorkOrder(r.Context(), id, claims.UserID, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewWorkOrderWithPropertyResponse(wo))
}

// List backs the global Maintenance page — every work order across
// every property the caller owns.
func (h *WorkOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list work orders: %w", domain.ErrUnauthorized))
		return
	}

	opts, err := parseWorkOrderListOptions(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	opts.PropertyAccess, _ = middleware.PropertyAccessFromContext(r.Context())

	orders, total, err := h.svc.ListWorkOrdersForOwner(r.Context(), claims.UserID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, dto.NewWorkOrderWithPropertyListResponse(orders), dto.WorkOrderListMeta{
		Total:  total,
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

func (h *WorkOrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("update work order: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseWorkOrderIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.UpdateWorkOrderRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update work order: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	wo, err := h.svc.UpdateWorkOrder(r.Context(), id, claims.UserID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewWorkOrderResponse(wo))
}

func (h *WorkOrderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("delete work order: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseWorkOrderIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	if err := h.svc.DeleteWorkOrder(r.Context(), id, claims.UserID, access); err != nil {
		response.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSummary backs the list page's clickable summary counters (Open,
// Overdue, Unassigned, Emergency).
func (h *WorkOrderHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("get work order summary: %w", domain.ErrUnauthorized))
		return
	}

	summary, err := h.svc.GetSummary(r.Context(), claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewWorkOrderSummaryResponse(summary))
}

// GetRecentActivity backs the Dashboard's activity feed — the most
// recent maintenance activity across every work order this owner has.
func (h *WorkOrderHandler) GetRecentActivity(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list recent activity: %w", domain.ErrUnauthorized))
		return
	}

	limit, offset := parsePagination(r)

	activity, err := h.svc.ListRecentActivity(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewMaintenanceActivityWithContextListResponse(activity))
}

func (h *WorkOrderHandler) BulkUpdateStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("bulk update work order status: %w", domain.ErrUnauthorized))
		return
	}

	var req dto.BulkUpdateWorkOrderStatusRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	ids, status, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("bulk update work order status: ids: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	n, err := h.svc.BulkUpdateStatus(r.Context(), claims.UserID, ids, status, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]int{"updated": n})
}

func (h *WorkOrderHandler) BulkReassign(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("bulk reassign work orders: %w", domain.ErrUnauthorized))
		return
	}

	var req dto.BulkReassignWorkOrdersRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	ids, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("bulk reassign work orders: ids: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	n, err := h.svc.BulkReassign(r.Context(), claims.UserID, ids, req.AssignedTo, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]int{"updated": n})
}

func (h *WorkOrderHandler) ListActivity(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list work order activity: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseWorkOrderIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	activity, err := h.svc.ListActivity(r.Context(), id, claims.UserID, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewMaintenanceActivityListResponse(activity))
}

func (h *WorkOrderHandler) AddNote(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("add work order note: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseWorkOrderIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.AddWorkOrderNoteRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	a, err := h.svc.AddNote(r.Context(), id, claims.UserID, req.Message, domain.MaintenanceVisibility(req.Visibility), access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewMaintenanceActivityResponse(a))
}

// parseWorkOrderListOptions mirrors parseLeaseListOptions/
// parseUnitListOptions — unrecognized filter/sort values are a genuine
// 400, not silently ignored.
func parseWorkOrderListOptions(r *http.Request) (domain.WorkOrderListOptions, error) {
	q := r.URL.Query()
	limit, offset := parsePagination(r)

	opts := domain.WorkOrderListOptions{
		Limit:  limit,
		Offset: offset,
		Filter: domain.WorkOrderListFilter{
			Search:     q.Get("search"),
			AssignedTo: q.Get("assigned_to"),
			Overdue:    q.Get("overdue") == "true",
			Unassigned: q.Get("unassigned") == "true",
		},
	}

	var verrs domain.ValidationErrors

	if v := q.Get("property_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			verrs = append(verrs, &domain.ValidationError{Field: "property_id", Message: "must be a valid id"})
		} else {
			opts.Filter.PropertyID = &id
		}
	}
	if v := q.Get("unit_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			verrs = append(verrs, &domain.ValidationError{Field: "unit_id", Message: "must be a valid id"})
		} else {
			opts.Filter.UnitID = &id
		}
	}
	if v := q.Get("vendor_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			verrs = append(verrs, &domain.ValidationError{Field: "vendor_id", Message: "must be a valid id"})
		} else {
			opts.Filter.VendorID = &id
		}
	}
	if v := q.Get("status"); v != "" {
		s := domain.WorkOrderStatus(v)
		if !s.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", v)})
		} else {
			opts.Filter.Status = &s
		}
	}
	if v := q.Get("priority"); v != "" {
		p := domain.WorkOrderPriority(v)
		if !p.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "priority", Message: fmt.Sprintf("unknown priority %q", v)})
		} else {
			opts.Filter.Priority = &p
		}
	}
	if v := q.Get("category"); v != "" {
		c := domain.WorkOrderCategory(v)
		if !c.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "category", Message: fmt.Sprintf("unknown category %q", v)})
		} else {
			opts.Filter.Category = &c
		}
	}
	if v := q.Get("sort"); v != "" {
		s := domain.WorkOrderSortKey(v)
		if !s.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "sort", Message: fmt.Sprintf("unknown sort key %q", v)})
		} else {
			opts.Sort = s
		}
	}
	if v := q.Get("order"); v != "" {
		switch v {
		case "asc":
			opts.SortDesc = false
		case "desc":
			opts.SortDesc = true
		default:
			verrs = append(verrs, &domain.ValidationError{Field: "order", Message: `must be "asc" or "desc"`})
		}
	}

	if len(verrs) > 0 {
		return domain.WorkOrderListOptions{}, fmt.Errorf("list work orders: %w", verrs)
	}
	return opts, nil
}

func parseWorkOrderIDParam(r *http.Request) (uuid.UUID, error) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("work order id %q: %w", idParam, domain.ErrInvalidInput)
	}
	return id, nil
}
