package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

const recurringRuleDateLayout = "2006-01-02"

type CreateRecurringRuleRequest struct {
	UnitID            *string `json:"unit_id" validate:"omitempty,uuid4"`
	Title             string  `json:"title" validate:"required,min=1,max=200,noctrl"`
	Description       string  `json:"description" validate:"omitempty,max=2000,noctrl"`
	Category          string  `json:"category" validate:"required,oneof=plumbing electrical hvac appliance pest_control general other"`
	FrequencyInterval int     `json:"frequency_interval" validate:"required,min=1,max=365"`
	FrequencyUnit     string  `json:"frequency_unit" validate:"required,oneof=days weeks months"`
	NextDueDate       string  `json:"next_due_date" validate:"required,datetime=2006-01-02"`
}

func (r *CreateRecurringRuleRequest) Sanitize() {
	r.Title = sanitizeString(r.Title)
	r.Description = sanitizeString(r.Description)
}

// ToDomain takes propertyID from the URL — a recurring rule is always
// created in the context of the property page the caller is already
// on, same rationale as CreateWorkOrderRequest.ToDomain.
func (r CreateRecurringRuleRequest) ToDomain(propertyID uuid.UUID) (domain.CreateRecurringRuleInput, error) {
	input := domain.CreateRecurringRuleInput{
		PropertyID:        propertyID,
		Title:             r.Title,
		Description:       r.Description,
		Category:          domain.WorkOrderCategory(r.Category),
		FrequencyInterval: r.FrequencyInterval,
		FrequencyUnit:     domain.RecurringRuleFrequencyUnit(r.FrequencyUnit),
	}
	if r.UnitID != nil && *r.UnitID != "" {
		id, err := uuid.Parse(*r.UnitID)
		if err != nil {
			return domain.CreateRecurringRuleInput{}, fmt.Errorf("unit_id: %w", err)
		}
		input.UnitID = &id
	}

	dueDate, err := time.Parse(recurringRuleDateLayout, r.NextDueDate)
	if err != nil {
		return domain.CreateRecurringRuleInput{}, fmt.Errorf("next_due_date: %w", err)
	}
	input.NextDueDate = dueDate

	return input, nil
}

// UpdateRecurringRuleRequest.UnitID uses the same empty-string-means-
// "clear" sentinel as UpdateWorkOrderRequest.UnitID.
type UpdateRecurringRuleRequest struct {
	UnitID            *string `json:"unit_id"`
	Title             *string `json:"title" validate:"omitempty,min=1,max=200,noctrl"`
	Description       *string `json:"description" validate:"omitempty,max=2000,noctrl"`
	Category          *string `json:"category" validate:"omitempty,oneof=plumbing electrical hvac appliance pest_control general other"`
	FrequencyInterval *int    `json:"frequency_interval" validate:"omitempty,min=1,max=365"`
	FrequencyUnit     *string `json:"frequency_unit" validate:"omitempty,oneof=days weeks months"`
	NextDueDate       *string `json:"next_due_date" validate:"omitempty,datetime=2006-01-02"`
	Active            *bool   `json:"active"`
}

func (r *UpdateRecurringRuleRequest) Sanitize() {
	if r.Title != nil {
		*r.Title = sanitizeString(*r.Title)
	}
	if r.Description != nil {
		*r.Description = sanitizeString(*r.Description)
	}
}

func (r UpdateRecurringRuleRequest) ToDomain() (domain.UpdateRecurringRuleInput, error) {
	input := domain.UpdateRecurringRuleInput{
		Title:             r.Title,
		Description:       r.Description,
		FrequencyInterval: r.FrequencyInterval,
		Active:            r.Active,
	}
	if r.Category != nil {
		c := domain.WorkOrderCategory(*r.Category)
		input.Category = &c
	}
	if r.FrequencyUnit != nil {
		u := domain.RecurringRuleFrequencyUnit(*r.FrequencyUnit)
		input.FrequencyUnit = &u
	}
	if r.UnitID != nil {
		input.UnitIDSet = true
		if *r.UnitID != "" {
			id, err := uuid.Parse(*r.UnitID)
			if err != nil {
				return domain.UpdateRecurringRuleInput{}, fmt.Errorf("unit_id: %w", err)
			}
			input.UnitID = &id
		}
	}
	if r.NextDueDate != nil {
		t, err := time.Parse(recurringRuleDateLayout, *r.NextDueDate)
		if err != nil {
			return domain.UpdateRecurringRuleInput{}, fmt.Errorf("next_due_date: %w", err)
		}
		input.NextDueDate = &t
	}

	return input, nil
}

type RecurringRuleResponse struct {
	ID                string `json:"id"`
	PropertyID        string `json:"property_id"`
	UnitID            string `json:"unit_id,omitempty"`
	Title             string `json:"title"`
	Description       string `json:"description,omitempty"`
	Category          string `json:"category"`
	FrequencyInterval int    `json:"frequency_interval"`
	FrequencyUnit     string `json:"frequency_unit"`
	NextDueDate       string `json:"next_due_date"`
	Active            bool   `json:"active"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

func NewRecurringRuleResponse(r *domain.RecurringRule) RecurringRuleResponse {
	resp := RecurringRuleResponse{
		ID:                r.ID.String(),
		PropertyID:        r.PropertyID.String(),
		Title:             r.Title,
		Description:       r.Description,
		Category:          string(r.Category),
		FrequencyInterval: r.FrequencyInterval,
		FrequencyUnit:     string(r.FrequencyUnit),
		NextDueDate:       r.NextDueDate.Format(recurringRuleDateLayout),
		Active:            r.Active,
		CreatedAt:         r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         r.UpdatedAt.Format(time.RFC3339),
	}
	if r.UnitID != nil {
		resp.UnitID = r.UnitID.String()
	}
	return resp
}

// RecurringRuleWithPropertyResponse is RecurringRuleResponse plus the
// property/unit context the Scheduled tab needs — mirrors
// WorkOrderWithPropertyResponse.
type RecurringRuleWithPropertyResponse struct {
	RecurringRuleResponse
	PropertyName string `json:"property_name"`
	UnitName     string `json:"unit_name,omitempty"`
}

func NewRecurringRuleWithPropertyResponse(r *domain.RecurringRuleWithProperty) RecurringRuleWithPropertyResponse {
	return RecurringRuleWithPropertyResponse{
		RecurringRuleResponse: NewRecurringRuleResponse(&r.RecurringRule),
		PropertyName:          r.PropertyName,
		UnitName:              r.UnitName,
	}
}

func NewRecurringRuleWithPropertyListResponse(rules []*domain.RecurringRuleWithProperty) []RecurringRuleWithPropertyResponse {
	out := make([]RecurringRuleWithPropertyResponse, len(rules))
	for i, r := range rules {
		out[i] = NewRecurringRuleWithPropertyResponse(r)
	}
	return out
}
