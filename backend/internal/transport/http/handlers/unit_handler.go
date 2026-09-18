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

type UnitHandler struct {
	svc domain.UnitService
}

func NewUnitHandler(svc domain.UnitService) *UnitHandler {
	return &UnitHandler{svc: svc}
}

func (h *UnitHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("create unit: %w", domain.ErrUnauthorized))
		return
	}

	propertyID, err := parsePropertyIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.CreateUnitRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input := req.ToDomain()
	input.PropertyID = propertyID

	u, err := h.svc.CreateUnit(r.Context(), claims.UserID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewUnitResponse(u))
}

// BulkCreate backs the "bulk add" spreadsheet-style flow for setting up
// a multi-unit building's units in one request instead of one per unit.
func (h *UnitHandler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("bulk create units: %w", domain.ErrUnauthorized))
		return
	}

	propertyID, err := parsePropertyIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.BulkCreateUnitsRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	units, err := h.svc.CreateUnitsBulk(r.Context(), claims.UserID, propertyID, req.ToDomain())
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewUnitListResponse(units))
}

// ListForOwner backs the global Units page (GET /api/v1/units) — every
// unit across every property the caller owns, unlike List below which
// is scoped to one property via the URL.
func (h *UnitHandler) ListForOwner(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list units: %w", domain.ErrUnauthorized))
		return
	}

	opts, err := parseUnitListOptions(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	units, total, err := h.svc.ListUnitsForOwner(r.Context(), claims.UserID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, dto.NewUnitWithPropertyListResponse(units), dto.UnitListMeta{
		Total:  total,
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

func (h *UnitHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list units: %w", domain.ErrUnauthorized))
		return
	}

	propertyID, err := parsePropertyIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	units, err := h.svc.ListUnitsByProperty(r.Context(), propertyID, claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewUnitListResponse(units))
}

func (h *UnitHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("get unit: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseUnitIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	u, err := h.svc.GetUnit(r.Context(), id, claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewUnitResponse(u))
}

func (h *UnitHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("update unit: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseUnitIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.UpdateUnitRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	u, err := h.svc.UpdateUnit(r.Context(), id, claims.UserID, req.ToDomain())
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewUnitResponse(u))
}

func (h *UnitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("delete unit: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseUnitIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	if err := h.svc.DeleteUnit(r.Context(), id, claims.UserID); err != nil {
		response.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseUnitListOptions reads the global Units page's filter/sort/
// pagination query params — see parsePropertyListOptions, which this
// mirrors exactly (unrecognized filter/sort values are a genuine 400,
// not silently ignored).
func parseUnitListOptions(r *http.Request) (domain.UnitListOptions, error) {
	q := r.URL.Query()
	limit, offset := parsePagination(r)

	opts := domain.UnitListOptions{
		Limit:  limit,
		Offset: offset,
		Filter: domain.UnitListFilter{Search: q.Get("search")},
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
	if v := q.Get("status"); v != "" {
		s := domain.UnitStatus(v)
		if !s.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", v)})
		} else {
			opts.Filter.Status = &s
		}
	}
	if v := q.Get("type"); v != "" {
		t := domain.UnitType(v)
		if !t.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "type", Message: fmt.Sprintf("unknown type %q", v)})
		} else {
			opts.Filter.Type = &t
		}
	}
	if v := q.Get("sort"); v != "" {
		s := domain.UnitSortKey(v)
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
		return domain.UnitListOptions{}, fmt.Errorf("list units: %w", verrs)
	}
	return opts, nil
}

func parsePropertyIDParam(r *http.Request) (uuid.UUID, error) {
	idParam := chi.URLParam(r, "propertyId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("property id %q: %w", idParam, domain.ErrInvalidInput)
	}
	return id, nil
}

func parseUnitIDParam(r *http.Request) (uuid.UUID, error) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("unit id %q: %w", idParam, domain.ErrInvalidInput)
	}
	return id, nil
}
