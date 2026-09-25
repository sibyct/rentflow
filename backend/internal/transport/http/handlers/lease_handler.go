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

type LeaseHandler struct {
	svc domain.LeaseService
}

func NewLeaseHandler(svc domain.LeaseService) *LeaseHandler {
	return &LeaseHandler{svc: svc}
}

// Create backs the "unit -> terms" lease-creation flow — unitId comes
// from the URL, never a client-supplied body field, same rationale as
// UnitHandler.Create taking propertyId from the URL.
func (h *LeaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("create lease: %w", domain.ErrUnauthorized))
		return
	}

	unitID, err := parseUnitIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.CreateLeaseRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain(unitID)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create lease: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.CreateLease(r.Context(), claims.UserID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewLeaseResponse(l))
}

func (h *LeaseHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("get lease: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.GetLease(r.Context(), id, claims.UserID, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseWithUnitPropertyResponse(l))
}

// List backs the global Leases page — every lease across every
// property the caller owns.
func (h *LeaseHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list leases: %w", domain.ErrUnauthorized))
		return
	}

	opts, err := parseLeaseListOptions(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	opts.PropertyAccess, _ = middleware.PropertyAccessFromContext(r.Context())

	leases, total, err := h.svc.ListLeasesForOwner(r.Context(), claims.UserID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, dto.NewLeaseWithUnitPropertyListResponse(leases), dto.LeaseListMeta{
		Total:  total,
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

func (h *LeaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("update lease: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.UpdateLeaseRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update lease: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.UpdateLease(r.Context(), id, claims.UserID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseResponse(l))
}

func (h *LeaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("delete lease: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	if err := h.svc.DeleteLease(r.Context(), id, claims.UserID, access); err != nil {
		response.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseLeaseListOptions mirrors parseUnitListOptions — unrecognized
// filter/sort values are a genuine 400, not silently ignored.
func parseLeaseListOptions(r *http.Request) (domain.LeaseListOptions, error) {
	q := r.URL.Query()
	limit, offset := parsePagination(r)

	opts := domain.LeaseListOptions{
		Limit:  limit,
		Offset: offset,
		Filter: domain.LeaseListFilter{Search: q.Get("search")},
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
	if v := q.Get("status"); v != "" {
		s := domain.LeaseDisplayStatus(v)
		if !s.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", v)})
		} else {
			opts.Filter.DisplayStatus = &s
		}
	}
	if v := q.Get("sort"); v != "" {
		s := domain.LeaseSortKey(v)
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
		return domain.LeaseListOptions{}, fmt.Errorf("list leases: %w", verrs)
	}
	return opts, nil
}

func parseLeaseIDParam(r *http.Request) (uuid.UUID, error) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("lease id %q: %w", idParam, domain.ErrInvalidInput)
	}
	return id, nil
}
