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

type VendorHandler struct {
	svc domain.VendorService
}

func NewVendorHandler(svc domain.VendorService) *VendorHandler {
	return &VendorHandler{svc: svc}
}

func (h *VendorHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("create vendor: %w", domain.ErrUnauthorized))
		return
	}

	var req dto.CreateVendorRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create vendor: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	v, err := h.svc.CreateVendor(r.Context(), claims.UserID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewVendorResponse(v))
}

func (h *VendorHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("get vendor: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseVendorIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	v, err := h.svc.GetVendor(r.Context(), id, claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewVendorWithStatsResponse(v))
}

// List backs the Vendors list page — every vendor the caller owns.
func (h *VendorHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list vendors: %w", domain.ErrUnauthorized))
		return
	}

	opts, err := parseVendorListOptions(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	vendors, total, err := h.svc.ListVendorsForOwner(r.Context(), claims.UserID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, dto.NewVendorWithStatsListResponse(vendors), dto.VendorListMeta{
		Total:  total,
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

// ListForCategory backs the Vendor Selector inside the work order
// drawer — active vendors whose trades include the given category.
func (h *VendorHandler) ListForCategory(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list vendors for category: %w", domain.ErrUnauthorized))
		return
	}

	categoryParam := r.URL.Query().Get("category")
	category := domain.WorkOrderCategory(categoryParam)
	if !category.Valid() {
		response.WriteError(w, r, fmt.Errorf("list vendors for category: category %q: %w", categoryParam, domain.ErrInvalidInput))
		return
	}

	vendors, err := h.svc.ListVendorsForCategory(r.Context(), claims.UserID, category)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewVendorWithStatsListResponse(vendors))
}

func (h *VendorHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("update vendor: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseVendorIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.UpdateVendorRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update vendor: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	v, err := h.svc.UpdateVendor(r.Context(), id, claims.UserID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewVendorResponse(v))
}

func (h *VendorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("delete vendor: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseVendorIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	if err := h.svc.DeleteVendor(r.Context(), id, claims.UserID); err != nil {
		response.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPropertiesServed backs the vendor form's Service Scope section on
// edit — the current properties_served selection for a vendor that
// doesn't serve all properties.
func (h *VendorHandler) GetPropertiesServed(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("get vendor properties served: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseVendorIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	ids, err := h.svc.GetPropertiesServed(r.Context(), id, claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	propertyIDs := make([]string, len(ids))
	for i, id := range ids {
		propertyIDs[i] = id.String()
	}
	response.JSON(w, http.StatusOK, dto.VendorPropertiesServedResponse{PropertyIDs: propertyIDs})
}

func (h *VendorHandler) GetSpendSummary(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("get vendor spend summary: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseVendorIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	summary, err := h.svc.GetSpendSummary(r.Context(), id, claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewVendorSpendSummaryResponse(summary))
}

func parseVendorListOptions(r *http.Request) (domain.VendorListOptions, error) {
	q := r.URL.Query()
	limit, offset := parsePagination(r)

	opts := domain.VendorListOptions{
		Limit:  limit,
		Offset: offset,
		Filter: domain.VendorListFilter{Search: q.Get("search")},
	}

	var verrs domain.ValidationErrors

	if v := q.Get("category"); v != "" {
		c := domain.WorkOrderCategory(v)
		if !c.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "category", Message: fmt.Sprintf("unknown category %q", v)})
		} else {
			opts.Filter.Category = &c
		}
	}
	if v := q.Get("active"); v != "" {
		switch v {
		case "true":
			b := true
			opts.Filter.Active = &b
		case "false":
			b := false
			opts.Filter.Active = &b
		default:
			verrs = append(verrs, &domain.ValidationError{Field: "active", Message: `must be "true" or "false"`})
		}
	}
	if v := q.Get("insurance_status"); v != "" {
		s := domain.InsuranceStatus(v)
		switch s {
		case domain.InsuranceStatusValid, domain.InsuranceStatusExpiringSoon, domain.InsuranceStatusExpired, domain.InsuranceStatusUnknown:
			opts.Filter.InsuranceStatus = &s
		default:
			verrs = append(verrs, &domain.ValidationError{Field: "insurance_status", Message: fmt.Sprintf("unknown insurance status %q", v)})
		}
	}
	if v := q.Get("sort"); v != "" {
		s := domain.VendorSortKey(v)
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
		return domain.VendorListOptions{}, fmt.Errorf("list vendors: %w", verrs)
	}
	return opts, nil
}

func parseVendorIDParam(r *http.Request) (uuid.UUID, error) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("vendor id %q: %w", idParam, domain.ErrInvalidInput)
	}
	return id, nil
}
