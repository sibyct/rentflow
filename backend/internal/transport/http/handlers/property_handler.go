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

	p, err := h.svc.CreateProperty(r.Context(), req.ToDomain(claims.UserID))
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewPropertyResponse(p))
}

func (h *PropertyHandler) Get(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("get property: id %q: %w", idParam, domain.ErrInvalidInput))
		return
	}

	p, err := h.svc.GetProperty(r.Context(), id)
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

	limit, offset := parsePagination(r)

	properties, total, err := h.svc.ListProperties(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, dto.NewPropertyListResponse(properties), dto.PropertyListMeta{
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *PropertyHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	p, err := h.svc.UpdateProperty(r.Context(), id, req.ToDomain())
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewPropertyResponse(p))
}

func (h *PropertyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("delete property: id %q: %w", idParam, domain.ErrInvalidInput))
		return
	}

	if err := h.svc.DeleteProperty(r.Context(), id); err != nil {
		response.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
