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

type PropertyHandler struct {
	svc domain.PropertyService
}

func NewPropertyHandler(svc domain.PropertyService) *PropertyHandler {
	return &PropertyHandler{svc: svc}
}

func (h *PropertyHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("create property: %w", domain.ErrUnauthorized))
		return
	}

	var req dto.CreatePropertyRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain(claims.UserID)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create property: onboard_date: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	p, err := h.svc.CreateProperty(r.Context(), input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewPropertyResponse(p))
}

func (h *PropertyHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("get property: %w", domain.ErrUnauthorized))
		return
	}

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("get property: id %q: %w", idParam, domain.ErrInvalidInput))
		return
	}

	p, err := h.svc.GetProperty(r.Context(), id, claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewPropertyResponse(p))
}

func (h *PropertyHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list properties: %w", domain.ErrUnauthorized))
		return
	}

	opts, err := parsePropertyListOptions(r, claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	properties, total, err := h.svc.ListProperties(r.Context(), opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, dto.NewPropertyListResponse(properties), dto.PropertyListMeta{
		Total:  total,
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

func (h *PropertyHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("update property: %w", domain.ErrUnauthorized))
		return
	}

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update property: id %q: %w", idParam, domain.ErrInvalidInput))
		return
	}

	var req dto.UpdatePropertyRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update property: onboard_date: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	p, err := h.svc.UpdateProperty(r.Context(), id, claims.UserID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewPropertyResponse(p))
}

func (h *PropertyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("delete property: %w", domain.ErrUnauthorized))
		return
	}

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("delete property: id %q: %w", idParam, domain.ErrInvalidInput))
		return
	}

	if err := h.svc.DeleteProperty(r.Context(), id, claims.UserID); err != nil {
		response.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BulkUpdateStatus backs the properties list's bulk-actions toolbar
// (e.g. "Archive" across a multi-row selection), which would otherwise
// be one round trip per row.
func (h *PropertyHandler) BulkUpdateStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("bulk update property status: %w", domain.ErrUnauthorized))
		return
	}

	var req dto.BulkUpdatePropertyStatusRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	ids, status, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("bulk update property status: ids: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	n, err := h.svc.BulkUpdateStatus(r.Context(), claims.UserID, ids, status)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]int{"updated": n})
}

// parsePropertyListOptions reads the properties list's filter/sort/
// pagination query params. Unlike parsePagination (which clamps
// out-of-range values rather than erroring, since those are almost
// always harmless off-by-ones), an unrecognized type/status/sort value
// here is treated as a genuine client error: silently ignoring it would
// make a typo'd filter look like "not filtered" instead of failing loud.
func parsePropertyListOptions(r *http.Request, ownerID uuid.UUID) (domain.PropertyListOptions, error) {
	q := r.URL.Query()
	limit, offset := parsePagination(r)

	opts := domain.PropertyListOptions{
		OwnerID: ownerID,
		Limit:   limit,
		Offset:  offset,
		Filter:  domain.PropertyListFilter{Search: q.Get("search")},
	}

	var verrs domain.ValidationErrors

	if v := q.Get("type"); v != "" {
		t := domain.PropertyType(v)
		if !t.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "type", Message: fmt.Sprintf("unknown type %q", v)})
		} else {
			opts.Filter.Type = &t
		}
	}
	if v := q.Get("status"); v != "" {
		s := domain.PropertyStatus(v)
		if !s.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", v)})
		} else {
			opts.Filter.Status = &s
		}
	}
	if v := q.Get("sort"); v != "" {
		s := domain.PropertySortKey(v)
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
		return domain.PropertyListOptions{}, fmt.Errorf("list properties: %w", verrs)
	}
	return opts, nil
}
