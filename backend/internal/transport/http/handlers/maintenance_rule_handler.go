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

type MaintenanceRuleHandler struct {
	svc domain.RecurringRuleService
}

func NewMaintenanceRuleHandler(svc domain.RecurringRuleService) *MaintenanceRuleHandler {
	return &MaintenanceRuleHandler{svc: svc}
}

// Create backs the property-scoped "new recurring rule" flow —
// propertyId comes from the URL, same rationale as WorkOrderHandler.Create.
func (h *MaintenanceRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("create recurring rule: %w", domain.ErrUnauthorized))
		return
	}

	propertyID, err := parsePropertyIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.CreateRecurringRuleRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain(propertyID)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create recurring rule: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	rule, err := h.svc.CreateRule(r.Context(), claims.UserID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewRecurringRuleResponse(rule))
}

// List backs the Scheduled tab — every recurring rule across every
// property the caller owns.
func (h *MaintenanceRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list recurring rules: %w", domain.ErrUnauthorized))
		return
	}

	rules, err := h.svc.ListRulesForOwner(r.Context(), claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewRecurringRuleWithPropertyListResponse(rules))
}

func (h *MaintenanceRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("update recurring rule: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseMaintenanceRuleIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.UpdateRecurringRuleRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update recurring rule: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	rule, err := h.svc.UpdateRule(r.Context(), id, claims.UserID, input)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewRecurringRuleResponse(rule))
}

func (h *MaintenanceRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("delete recurring rule: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseMaintenanceRuleIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	if err := h.svc.DeleteRule(r.Context(), id, claims.UserID); err != nil {
		response.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GenerateNow is the manager-triggered stand-in for a background
// scheduler — see domain.RecurringRule's doc comment.
func (h *MaintenanceRuleHandler) GenerateNow(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("generate work order from recurring rule: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseMaintenanceRuleIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	wo, err := h.svc.GenerateNow(r.Context(), id, claims.UserID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewWorkOrderResponse(wo))
}

func parseMaintenanceRuleIDParam(r *http.Request) (uuid.UUID, error) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("recurring rule id %q: %w", idParam, domain.ErrInvalidInput)
	}
	return id, nil
}
