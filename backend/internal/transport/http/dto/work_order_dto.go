package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

const workOrderDateLayout = "2006-01-02"
const workOrderDateTimeLayout = time.RFC3339

type CreateWorkOrderRequest struct {
	UnitID             *string  `json:"unit_id" validate:"omitempty,uuid4"`
	Title              string   `json:"title" validate:"required,min=1,max=200,noctrl"`
	Description        string   `json:"description" validate:"omitempty,max=4000,noctrl"`
	Category           string   `json:"category" validate:"required,oneof=plumbing electrical hvac appliance pest_control general other"`
	Priority           string   `json:"priority" validate:"omitempty,oneof=low medium high emergency"`
	Status             string   `json:"status" validate:"omitempty,oneof=new assigned in_progress on_hold completed cancelled"`
	ReportedBy         string   `json:"reported_by" validate:"omitempty,max=200,noctrl"`
	ReportedByContact  string   `json:"reported_by_contact" validate:"omitempty,max=200,noctrl"`
	AssignedTo         string   `json:"assigned_to" validate:"omitempty,max=200,noctrl"`
	AssignedToContact  string   `json:"assigned_to_contact" validate:"omitempty,max=200,noctrl"`
	VendorID           *string  `json:"vendor_id" validate:"omitempty,uuid4"`
	AccessInstructions string   `json:"access_instructions" validate:"omitempty,max=1000,noctrl"`
	ScheduledStart     string   `json:"scheduled_start" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	ScheduledEnd       string   `json:"scheduled_end" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	DueDate            string   `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	EstimatedCost      *float64 `json:"estimated_cost" validate:"omitempty,min=0"`
	ActualCost         *float64 `json:"actual_cost" validate:"omitempty,min=0"`
	PhotoLink          string   `json:"photo_link" validate:"omitempty,max=500,noctrl"`
	InvoiceLink        string   `json:"invoice_link" validate:"omitempty,max=500,noctrl"`
	InternalNotes      string   `json:"internal_notes" validate:"omitempty,max=2000,noctrl"`
}

func (r *CreateWorkOrderRequest) Sanitize() {
	r.Title = sanitizeString(r.Title)
	r.Description = sanitizeString(r.Description)
	r.ReportedBy = sanitizeString(r.ReportedBy)
	r.ReportedByContact = sanitizeString(r.ReportedByContact)
	r.AssignedTo = sanitizeString(r.AssignedTo)
	r.AssignedToContact = sanitizeString(r.AssignedToContact)
	r.AccessInstructions = sanitizeString(r.AccessInstructions)
	r.PhotoLink = sanitizeString(r.PhotoLink)
	r.InvoiceLink = sanitizeString(r.InvoiceLink)
	r.InternalNotes = sanitizeString(r.InternalNotes)
}

// ToDomain takes propertyID from the URL — a work order is always
// created in the context of the property (or unit) page the caller is
// already on, same rationale as UnitHandler.Create.
func (r CreateWorkOrderRequest) ToDomain(propertyID uuid.UUID) (domain.CreateWorkOrderInput, error) {
	input := domain.CreateWorkOrderInput{
		PropertyID:         propertyID,
		Title:              r.Title,
		Description:        r.Description,
		Category:           domain.WorkOrderCategory(r.Category),
		Priority:           domain.WorkOrderPriority(r.Priority),
		Status:             domain.WorkOrderStatus(r.Status),
		ReportedBy:         r.ReportedBy,
		ReportedByContact:  r.ReportedByContact,
		AssignedTo:         r.AssignedTo,
		AssignedToContact:  r.AssignedToContact,
		AccessInstructions: r.AccessInstructions,
		EstimatedCost:      r.EstimatedCost,
		ActualCost:         r.ActualCost,
		PhotoLink:          r.PhotoLink,
		InvoiceLink:        r.InvoiceLink,
		InternalNotes:      r.InternalNotes,
	}

	if r.UnitID != nil && *r.UnitID != "" {
		id, err := uuid.Parse(*r.UnitID)
		if err != nil {
			return domain.CreateWorkOrderInput{}, fmt.Errorf("unit_id: %w", err)
		}
		input.UnitID = &id
	}
	if r.VendorID != nil && *r.VendorID != "" {
		id, err := uuid.Parse(*r.VendorID)
		if err != nil {
			return domain.CreateWorkOrderInput{}, fmt.Errorf("vendor_id: %w", err)
		}
		input.VendorID = &id
	}

	var err error
	if input.ScheduledStart, err = parseOptionalDateTime(r.ScheduledStart); err != nil {
		return domain.CreateWorkOrderInput{}, fmt.Errorf("scheduled_start: %w", err)
	}
	if input.ScheduledEnd, err = parseOptionalDateTime(r.ScheduledEnd); err != nil {
		return domain.CreateWorkOrderInput{}, fmt.Errorf("scheduled_end: %w", err)
	}
	if input.DueDate, err = parseOptionalDate(r.DueDate, workOrderDateLayout); err != nil {
		return domain.CreateWorkOrderInput{}, fmt.Errorf("due_date: %w", err)
	}

	return input, nil
}

// UpdateWorkOrderRequest.UnitID distinguishes "leave unchanged" (field
// omitted, nil) from "clear back to property-wide" (field present as
// an empty string) from "reassign to a different unit" (a real UUID
// string) — a plain *string can't otherwise tell "omitted" from
// "explicit null" apart (see UpdateUnitInput's identical limitation),
// so this uses an empty string as the "clear" sentinel instead.
type UpdateWorkOrderRequest struct {
	UnitID            *string `json:"unit_id" validate:"omitempty"`
	Title             *string `json:"title" validate:"omitempty,min=1,max=200,noctrl"`
	Description       *string `json:"description" validate:"omitempty,max=4000,noctrl"`
	Category          *string `json:"category" validate:"omitempty,oneof=plumbing electrical hvac appliance pest_control general other"`
	Priority          *string `json:"priority" validate:"omitempty,oneof=low medium high emergency"`
	Status            *string `json:"status" validate:"omitempty,oneof=new assigned in_progress on_hold completed cancelled"`
	ReportedBy        *string `json:"reported_by" validate:"omitempty,max=200,noctrl"`
	ReportedByContact *string `json:"reported_by_contact" validate:"omitempty,max=200,noctrl"`
	AssignedTo        *string `json:"assigned_to" validate:"omitempty,max=200,noctrl"`
	AssignedToContact *string `json:"assigned_to_contact" validate:"omitempty,max=200,noctrl"`
	// VendorID uses the same empty-string-means-"clear" sentinel as
	// UpdateWorkOrderRequest.UnitID.
	VendorID           *string  `json:"vendor_id"`
	Rating             *int     `json:"rating" validate:"omitempty,min=1,max=5"`
	AccessInstructions *string  `json:"access_instructions" validate:"omitempty,max=1000,noctrl"`
	ScheduledStart     *string  `json:"scheduled_start" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	ScheduledEnd       *string  `json:"scheduled_end" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	DueDate            *string  `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	EstimatedCost      *float64 `json:"estimated_cost" validate:"omitempty,min=0"`
	ActualCost         *float64 `json:"actual_cost" validate:"omitempty,min=0"`
	PhotoLink          *string  `json:"photo_link" validate:"omitempty,max=500,noctrl"`
	InvoiceLink        *string  `json:"invoice_link" validate:"omitempty,max=500,noctrl"`
	InternalNotes      *string  `json:"internal_notes" validate:"omitempty,max=2000,noctrl"`
}

func (r *UpdateWorkOrderRequest) Sanitize() {
	trim := func(s *string) {
		if s != nil {
			*s = sanitizeString(*s)
		}
	}
	trim(r.Title)
	trim(r.Description)
	trim(r.ReportedBy)
	trim(r.ReportedByContact)
	trim(r.AssignedTo)
	trim(r.AssignedToContact)
	trim(r.AccessInstructions)
	trim(r.PhotoLink)
	trim(r.InvoiceLink)
	trim(r.InternalNotes)
}

func (r UpdateWorkOrderRequest) ToDomain() (domain.UpdateWorkOrderInput, error) {
	input := domain.UpdateWorkOrderInput{
		Title:              r.Title,
		Description:        r.Description,
		ReportedBy:         r.ReportedBy,
		ReportedByContact:  r.ReportedByContact,
		AssignedTo:         r.AssignedTo,
		AssignedToContact:  r.AssignedToContact,
		Rating:             r.Rating,
		AccessInstructions: r.AccessInstructions,
		EstimatedCost:      r.EstimatedCost,
		ActualCost:         r.ActualCost,
		PhotoLink:          r.PhotoLink,
		InvoiceLink:        r.InvoiceLink,
		InternalNotes:      r.InternalNotes,
	}
	if r.VendorID != nil {
		input.VendorIDSet = true
		if *r.VendorID != "" {
			id, err := uuid.Parse(*r.VendorID)
			if err != nil {
				return domain.UpdateWorkOrderInput{}, fmt.Errorf("vendor_id: %w", err)
			}
			input.VendorID = &id
		}
	}
	if r.Category != nil {
		c := domain.WorkOrderCategory(*r.Category)
		input.Category = &c
	}
	if r.Priority != nil {
		p := domain.WorkOrderPriority(*r.Priority)
		input.Priority = &p
	}
	if r.Status != nil {
		s := domain.WorkOrderStatus(*r.Status)
		input.Status = &s
	}
	if r.UnitID != nil {
		input.UnitIDSet = true
		if *r.UnitID != "" {
			id, err := uuid.Parse(*r.UnitID)
			if err != nil {
				return domain.UpdateWorkOrderInput{}, fmt.Errorf("unit_id: %w", err)
			}
			input.UnitID = &id
		}
	}

	var err error
	if input.ScheduledStart, err = parseOptionalDateTime(derefString(r.ScheduledStart)); err != nil {
		return domain.UpdateWorkOrderInput{}, fmt.Errorf("scheduled_start: %w", err)
	}
	if input.ScheduledEnd, err = parseOptionalDateTime(derefString(r.ScheduledEnd)); err != nil {
		return domain.UpdateWorkOrderInput{}, fmt.Errorf("scheduled_end: %w", err)
	}
	if input.DueDate, err = parseOptionalDate(derefString(r.DueDate), workOrderDateLayout); err != nil {
		return domain.UpdateWorkOrderInput{}, fmt.Errorf("due_date: %w", err)
	}

	return input, nil
}

func parseOptionalDateTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(workOrderDateTimeLayout, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func parseOptionalDate(s, layout string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(layout, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type AddWorkOrderNoteRequest struct {
	Message    string `json:"message" validate:"required,min=1,max=2000,noctrl"`
	Visibility string `json:"visibility" validate:"required,oneof=internal tenant_visible"`
}

func (r *AddWorkOrderNoteRequest) Sanitize() {
	r.Message = sanitizeString(r.Message)
}

type BulkUpdateWorkOrderStatusRequest struct {
	IDs    []string `json:"ids" validate:"required,min=1,max=200,dive,uuid4"`
	Status string   `json:"status" validate:"required,oneof=new assigned in_progress on_hold completed cancelled"`
}

func (r BulkUpdateWorkOrderStatusRequest) ToDomain() ([]uuid.UUID, domain.WorkOrderStatus, error) {
	ids, err := parseUUIDs(r.IDs)
	if err != nil {
		return nil, "", err
	}
	return ids, domain.WorkOrderStatus(r.Status), nil
}

type BulkReassignWorkOrdersRequest struct {
	IDs        []string `json:"ids" validate:"required,min=1,max=200,dive,uuid4"`
	AssignedTo string   `json:"assigned_to" validate:"required,min=1,max=200,noctrl"`
}

func (r *BulkReassignWorkOrdersRequest) Sanitize() {
	r.AssignedTo = sanitizeString(r.AssignedTo)
}

func (r BulkReassignWorkOrdersRequest) ToDomain() ([]uuid.UUID, error) {
	return parseUUIDs(r.IDs)
}

func parseUUIDs(ss []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(ss))
	for i, s := range ss {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

type WorkOrderResponse struct {
	ID                 string   `json:"id"`
	PropertyID         string   `json:"property_id"`
	UnitID             string   `json:"unit_id,omitempty"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	Category           string   `json:"category"`
	Priority           string   `json:"priority"`
	Status             string   `json:"status"`
	IsOverdue          bool     `json:"is_overdue"`
	ReportedBy         string   `json:"reported_by,omitempty"`
	ReportedByContact  string   `json:"reported_by_contact,omitempty"`
	AssignedTo         string   `json:"assigned_to,omitempty"`
	AssignedToContact  string   `json:"assigned_to_contact,omitempty"`
	VendorID           string   `json:"vendor_id,omitempty"`
	Rating             *int     `json:"rating,omitempty"`
	AccessInstructions string   `json:"access_instructions,omitempty"`
	ScheduledStart     string   `json:"scheduled_start,omitempty"`
	ScheduledEnd       string   `json:"scheduled_end,omitempty"`
	DueDate            string   `json:"due_date,omitempty"`
	EstimatedCost      *float64 `json:"estimated_cost,omitempty"`
	ActualCost         *float64 `json:"actual_cost,omitempty"`
	PhotoLink          string   `json:"photo_link,omitempty"`
	InvoiceLink        string   `json:"invoice_link,omitempty"`
	InternalNotes      string   `json:"internal_notes,omitempty"`
	RecurringRuleID    string   `json:"recurring_rule_id,omitempty"`
	CompletedAt        string   `json:"completed_at,omitempty"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

func NewWorkOrderResponse(w *domain.WorkOrder) WorkOrderResponse {
	resp := WorkOrderResponse{
		ID:                 w.ID.String(),
		PropertyID:         w.PropertyID.String(),
		Title:              w.Title,
		Description:        w.Description,
		Category:           string(w.Category),
		Priority:           string(w.Priority),
		Status:             string(w.Status),
		IsOverdue:          w.IsOverdue(time.Now().UTC()),
		ReportedBy:         w.ReportedBy,
		ReportedByContact:  w.ReportedByContact,
		AssignedTo:         w.AssignedTo,
		AssignedToContact:  w.AssignedToContact,
		Rating:             w.Rating,
		AccessInstructions: w.AccessInstructions,
		EstimatedCost:      w.EstimatedCost,
		ActualCost:         w.ActualCost,
		PhotoLink:          w.PhotoLink,
		InvoiceLink:        w.InvoiceLink,
		InternalNotes:      w.InternalNotes,
		CreatedAt:          w.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          w.UpdatedAt.Format(time.RFC3339),
	}
	if w.UnitID != nil {
		resp.UnitID = w.UnitID.String()
	}
	if w.VendorID != nil {
		resp.VendorID = w.VendorID.String()
	}
	if w.ScheduledStart != nil {
		resp.ScheduledStart = w.ScheduledStart.Format(workOrderDateTimeLayout)
	}
	if w.ScheduledEnd != nil {
		resp.ScheduledEnd = w.ScheduledEnd.Format(workOrderDateTimeLayout)
	}
	if w.DueDate != nil {
		resp.DueDate = w.DueDate.Format(workOrderDateLayout)
	}
	if w.RecurringRuleID != nil {
		resp.RecurringRuleID = w.RecurringRuleID.String()
	}
	if w.CompletedAt != nil {
		resp.CompletedAt = w.CompletedAt.Format(time.RFC3339)
	}
	return resp
}

// WorkOrderWithPropertyResponse is WorkOrderResponse plus the property/
// unit context the global Maintenance page and a work order's own
// detail view need — mirrors LeaseWithUnitPropertyResponse.
type WorkOrderWithPropertyResponse struct {
	WorkOrderResponse
	PropertyName string `json:"property_name"`
	UnitName     string `json:"unit_name,omitempty"`
	VendorName   string `json:"vendor_name,omitempty"`
}

func NewWorkOrderWithPropertyResponse(w *domain.WorkOrderWithProperty) WorkOrderWithPropertyResponse {
	return WorkOrderWithPropertyResponse{
		WorkOrderResponse: NewWorkOrderResponse(&w.WorkOrder),
		PropertyName:      w.PropertyName,
		UnitName:          w.UnitName,
		VendorName:        w.VendorName,
	}
}

func NewWorkOrderWithPropertyListResponse(orders []*domain.WorkOrderWithProperty) []WorkOrderWithPropertyResponse {
	out := make([]WorkOrderWithPropertyResponse, len(orders))
	for i, w := range orders {
		out[i] = NewWorkOrderWithPropertyResponse(w)
	}
	return out
}

type WorkOrderListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type WorkOrderSummaryResponse struct {
	Open       int `json:"open"`
	Overdue    int `json:"overdue"`
	Unassigned int `json:"unassigned"`
	Emergency  int `json:"emergency"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
}

func NewWorkOrderSummaryResponse(s *domain.WorkOrderSummary) WorkOrderSummaryResponse {
	return WorkOrderSummaryResponse{
		Open: s.Open, Overdue: s.Overdue, Unassigned: s.Unassigned, Emergency: s.Emergency,
		High: s.High, Medium: s.Medium, Low: s.Low,
	}
}

type MaintenanceActivityResponse struct {
	ID          string `json:"id"`
	WorkOrderID string `json:"work_order_id"`
	Kind        string `json:"kind"`
	Visibility  string `json:"visibility"`
	Message     string `json:"message"`
	OldValue    string `json:"old_value,omitempty"`
	NewValue    string `json:"new_value,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func NewMaintenanceActivityResponse(a *domain.MaintenanceActivity) MaintenanceActivityResponse {
	resp := MaintenanceActivityResponse{
		ID:          a.ID.String(),
		WorkOrderID: a.WorkOrderID.String(),
		Kind:        string(a.Kind),
		Visibility:  string(a.Visibility),
		Message:     a.Message,
		CreatedAt:   a.CreatedAt.Format(time.RFC3339),
	}
	if a.OldValue != nil {
		resp.OldValue = *a.OldValue
	}
	if a.NewValue != nil {
		resp.NewValue = *a.NewValue
	}
	return resp
}

func NewMaintenanceActivityListResponse(activity []*domain.MaintenanceActivity) []MaintenanceActivityResponse {
	out := make([]MaintenanceActivityResponse, len(activity))
	for i, a := range activity {
		out[i] = NewMaintenanceActivityResponse(a)
	}
	return out
}

// MaintenanceActivityWithContextResponse is MaintenanceActivityResponse
// plus the work order/property context a portfolio-wide feed needs
// (see WorkOrderService.ListRecentActivity) that the per-work-order
// activity panel doesn't, since that page already knows which work
// order it's looking at.
type MaintenanceActivityWithContextResponse struct {
	MaintenanceActivityResponse
	WorkOrderTitle string `json:"work_order_title"`
	PropertyName   string `json:"property_name"`
}

func NewMaintenanceActivityWithContextResponse(a *domain.MaintenanceActivityWithContext) MaintenanceActivityWithContextResponse {
	return MaintenanceActivityWithContextResponse{
		MaintenanceActivityResponse: NewMaintenanceActivityResponse(&a.MaintenanceActivity),
		WorkOrderTitle:              a.WorkOrderTitle,
		PropertyName:                a.PropertyName,
	}
}

func NewMaintenanceActivityWithContextListResponse(activity []*domain.MaintenanceActivityWithContext) []MaintenanceActivityWithContextResponse {
	out := make([]MaintenanceActivityWithContextResponse, len(activity))
	for i, a := range activity {
		out[i] = NewMaintenanceActivityWithContextResponse(a)
	}
	return out
}
