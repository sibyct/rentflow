package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// WorkOrderService resolves ownership via the work order's own
// PropertyID — one hop, unlike Lease/Unit which resolve through a
// parent, since work_orders.property_id is a direct column.
type WorkOrderService struct {
	repo         domain.WorkOrderRepository
	unitRepo     domain.UnitRepository
	propertyRepo domain.PropertyRepository
	vendorRepo   domain.VendorRepository
	log          *slog.Logger
}

func NewWorkOrderService(repo domain.WorkOrderRepository, unitRepo domain.UnitRepository, propertyRepo domain.PropertyRepository, vendorRepo domain.VendorRepository, log *slog.Logger) *WorkOrderService {
	return &WorkOrderService{repo: repo, unitRepo: unitRepo, propertyRepo: propertyRepo, vendorRepo: vendorRepo, log: log}
}

var _ domain.WorkOrderService = (*WorkOrderService)(nil)

func (s *WorkOrderService) CreateWorkOrder(ctx context.Context, ownerID uuid.UUID, input domain.CreateWorkOrderInput) (*domain.WorkOrder, error) {
	if _, err := s.requireOwnedProperty(ctx, input.PropertyID, ownerID); err != nil {
		return nil, fmt.Errorf("create work order: %w", err)
	}
	if err := s.validateUnitBelongsToProperty(ctx, input.UnitID, input.PropertyID); err != nil {
		return nil, fmt.Errorf("create work order: %w", err)
	}
	vendor, err := s.resolveVendor(ctx, input.VendorID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("create work order: %w", err)
	}

	if input.Status == "" {
		input.Status = domain.WorkOrderStatusNew
	}
	if verrs := validateWorkOrder(input.Title, input.Category, input.Priority, input.Status); len(verrs) > 0 {
		return nil, fmt.Errorf("create work order: %w", verrs)
	}
	// Enforced only at creation, not on every later edit: re-checking
	// this on update would make it impossible to touch an emergency
	// work order at all once its original 24-hour window has simply
	// passed, which is the common case for one still open a day later.
	if input.Priority == domain.WorkOrderPriorityEmergency {
		if verr := validateEmergencySLA(input.DueDate, time.Now().UTC()); verr != nil {
			return nil, fmt.Errorf("create work order: %w", domain.ValidationErrors{verr})
		}
	}

	now := time.Now().UTC()
	w := &domain.WorkOrder{
		ID:                 uuid.New(),
		PropertyID:         input.PropertyID,
		UnitID:             input.UnitID,
		Title:              input.Title,
		Description:        input.Description,
		Category:           input.Category,
		Priority:           input.Priority,
		Status:             input.Status,
		ReportedBy:         input.ReportedBy,
		ReportedByContact:  input.ReportedByContact,
		AssignedTo:         input.AssignedTo,
		AssignedToContact:  input.AssignedToContact,
		VendorID:           input.VendorID,
		AccessInstructions: input.AccessInstructions,
		ScheduledStart:     input.ScheduledStart,
		ScheduledEnd:       input.ScheduledEnd,
		DueDate:            input.DueDate,
		EstimatedCost:      input.EstimatedCost,
		ActualCost:         input.ActualCost,
		PhotoLink:          input.PhotoLink,
		InvoiceLink:        input.InvoiceLink,
		InternalNotes:      input.InternalNotes,
		RecurringRuleID:    input.RecurringRuleID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if input.Priority == "" {
		w.Priority = domain.WorkOrderPriorityMedium
	}
	// AssignedTo is a display label; a chosen vendor is always the
	// source of truth for it while VendorID is set — see
	// AssignedTo/AssignedToContact's doc comment on WorkOrder.
	if vendor != nil {
		w.AssignedTo = vendor.CompanyName
		w.AssignedToContact = vendor.Phone
	}

	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("create work order: %w", err)
	}

	statusStr := string(w.Status)
	s.logActivity(ctx, w.ID, domain.MaintenanceActivityKindStatusChange, "Work order created", nil, &statusStr)

	return w, nil
}

func (s *WorkOrderService) GetWorkOrder(ctx context.Context, id, ownerID uuid.UUID) (*domain.WorkOrderWithProperty, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get work order %s: %w", id, err)
	}
	p, err := s.requireOwnedProperty(ctx, w.PropertyID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("get work order %s: %w", id, err)
	}

	result := &domain.WorkOrderWithProperty{WorkOrder: *w, PropertyName: p.Name}
	if w.UnitID != nil {
		if u, err := s.unitRepo.GetByID(ctx, *w.UnitID); err == nil {
			result.UnitName = u.UnitName
		}
	}
	return result, nil
}

func (s *WorkOrderService) ListWorkOrdersForOwner(ctx context.Context, ownerID uuid.UUID, opts domain.WorkOrderListOptions) ([]*domain.WorkOrderWithProperty, int, error) {
	opts.OwnerID = ownerID
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	orders, total, err := s.repo.ListForOwner(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list work orders for owner %s: %w", ownerID, err)
	}
	return orders, total, nil
}

func (s *WorkOrderService) UpdateWorkOrder(ctx context.Context, id, ownerID uuid.UUID, input domain.UpdateWorkOrderInput) (*domain.WorkOrder, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update work order %s: %w", id, err)
	}
	if _, err := s.requireOwnedProperty(ctx, w.PropertyID, ownerID); err != nil {
		return nil, fmt.Errorf("update work order %s: %w", id, err)
	}

	var verrs domain.ValidationErrors

	if input.UnitIDSet {
		if err := s.validateUnitBelongsToProperty(ctx, input.UnitID, w.PropertyID); err != nil {
			var fieldErrs domain.ValidationErrors
			if !errors.As(err, &fieldErrs) {
				return nil, fmt.Errorf("update work order %s: %w", id, err)
			}
			verrs = append(verrs, fieldErrs...)
		} else {
			w.UnitID = input.UnitID
		}
	}
	if input.Title != nil {
		if *input.Title == "" {
			verrs = append(verrs, &domain.ValidationError{Field: "title", Message: "cannot be empty"})
		} else {
			w.Title = *input.Title
		}
	}
	if input.Description != nil {
		w.Description = *input.Description
	}
	if input.Category != nil {
		if !input.Category.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "category", Message: fmt.Sprintf("unknown category %q", *input.Category)})
		} else {
			w.Category = *input.Category
		}
	}
	if input.Priority != nil {
		if !input.Priority.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "priority", Message: fmt.Sprintf("unknown priority %q", *input.Priority)})
		} else {
			w.Priority = *input.Priority
		}
	}

	var oldStatus domain.WorkOrderStatus
	statusChanged := false
	if input.Status != nil {
		if !input.Status.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", *input.Status)})
		} else if *input.Status != w.Status {
			oldStatus = w.Status
			w.Status = *input.Status
			statusChanged = true
		}
	}
	if input.ReportedBy != nil {
		w.ReportedBy = *input.ReportedBy
	}
	if input.ReportedByContact != nil {
		w.ReportedByContact = *input.ReportedByContact
	}
	if input.AssignedTo != nil {
		w.AssignedTo = *input.AssignedTo
	}
	if input.AssignedToContact != nil {
		w.AssignedToContact = *input.AssignedToContact
	}
	if input.VendorIDSet {
		vendor, err := s.resolveVendor(ctx, input.VendorID, ownerID)
		if err != nil {
			return nil, fmt.Errorf("update work order %s: %w", id, err)
		}
		w.VendorID = input.VendorID
		if vendor != nil {
			// AssignedTo/AssignedToContact stay the vendor's display
			// info while a vendor is chosen — see CreateWorkOrder's
			// identical rationale.
			w.AssignedTo = vendor.CompanyName
			w.AssignedToContact = vendor.Phone
		}
	}
	if input.Rating != nil {
		if *input.Rating < 1 || *input.Rating > 5 {
			verrs = append(verrs, &domain.ValidationError{Field: "rating", Message: "must be between 1 and 5"})
		} else {
			w.Rating = input.Rating
		}
	}
	if input.AccessInstructions != nil {
		w.AccessInstructions = *input.AccessInstructions
	}
	if input.ScheduledStart != nil {
		w.ScheduledStart = input.ScheduledStart
	}
	if input.ScheduledEnd != nil {
		w.ScheduledEnd = input.ScheduledEnd
	}
	if input.DueDate != nil {
		w.DueDate = input.DueDate
	}
	if input.EstimatedCost != nil {
		if *input.EstimatedCost < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "estimated_cost", Message: "cannot be negative"})
		} else {
			w.EstimatedCost = input.EstimatedCost
		}
	}
	if input.ActualCost != nil {
		if *input.ActualCost < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "actual_cost", Message: "cannot be negative"})
		} else {
			w.ActualCost = input.ActualCost
		}
	}
	if input.PhotoLink != nil {
		w.PhotoLink = *input.PhotoLink
	}
	if input.InvoiceLink != nil {
		w.InvoiceLink = *input.InvoiceLink
	}
	if input.InternalNotes != nil {
		w.InternalNotes = *input.InternalNotes
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("update work order %s: %w", id, verrs)
	}

	if statusChanged {
		now := time.Now().UTC()
		if w.Status == domain.WorkOrderStatusCompleted {
			w.CompletedAt = &now
		} else {
			w.CompletedAt = nil
		}
	}

	w.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, w); err != nil {
		return nil, fmt.Errorf("update work order %s: %w", id, err)
	}

	if statusChanged {
		oldStr, newStr := string(oldStatus), string(w.Status)
		s.logActivity(ctx, w.ID, domain.MaintenanceActivityKindStatusChange, fmt.Sprintf("Status changed from %s to %s", oldStatus, w.Status), &oldStr, &newStr)
	}

	return w, nil
}

func (s *WorkOrderService) DeleteWorkOrder(ctx context.Context, id, ownerID uuid.UUID) error {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete work order %s: %w", id, err)
	}
	if _, err := s.requireOwnedProperty(ctx, w.PropertyID, ownerID); err != nil {
		return fmt.Errorf("delete work order %s: %w", id, err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete work order %s: %w", id, err)
	}
	return nil
}

func (s *WorkOrderService) GetSummary(ctx context.Context, ownerID uuid.UUID) (*domain.WorkOrderSummary, error) {
	summary, err := s.repo.GetSummary(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("get work order summary for owner %s: %w", ownerID, err)
	}
	return summary, nil
}

func (s *WorkOrderService) BulkUpdateStatus(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, status domain.WorkOrderStatus) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("bulk update work order status: %w", domain.ValidationErrors{
			{Field: "ids", Message: "must include at least one work order id"},
		})
	}
	if !status.Valid() {
		return 0, fmt.Errorf("bulk update work order status: %w", domain.ValidationErrors{
			{Field: "status", Message: fmt.Sprintf("unknown status %q", status)},
		})
	}

	// Bulk actions don't write per-row activity entries — see
	// UpdateWorkOrder for the detailed timeline a single-work-order edit
	// gets; this is a convenience shortcut for acting on many rows at
	// once, not a substitute for it.
	n, err := s.repo.BulkUpdateStatus(ctx, ownerID, ids, status)
	if err != nil {
		return 0, fmt.Errorf("bulk update work order status: %w", err)
	}
	return n, nil
}

func (s *WorkOrderService) BulkReassign(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, assignedTo string) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("bulk reassign work orders: %w", domain.ValidationErrors{
			{Field: "ids", Message: "must include at least one work order id"},
		})
	}

	n, err := s.repo.BulkReassign(ctx, ownerID, ids, assignedTo)
	if err != nil {
		return 0, fmt.Errorf("bulk reassign work orders: %w", err)
	}
	return n, nil
}

func (s *WorkOrderService) ListActivity(ctx context.Context, workOrderID, ownerID uuid.UUID) ([]*domain.MaintenanceActivity, error) {
	w, err := s.repo.GetByID(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("list activity for work order %s: %w", workOrderID, err)
	}
	if _, err := s.requireOwnedProperty(ctx, w.PropertyID, ownerID); err != nil {
		return nil, fmt.Errorf("list activity for work order %s: %w", workOrderID, err)
	}

	activity, err := s.repo.ListActivity(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("list activity for work order %s: %w", workOrderID, err)
	}
	return activity, nil
}

func (s *WorkOrderService) ListRecentActivity(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.MaintenanceActivityWithContext, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	activity, err := s.repo.ListRecentActivityForOwner(ctx, ownerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list recent activity for owner %s: %w", ownerID, err)
	}
	return activity, nil
}

func (s *WorkOrderService) AddNote(ctx context.Context, workOrderID, ownerID uuid.UUID, message string, visibility domain.MaintenanceVisibility) (*domain.MaintenanceActivity, error) {
	w, err := s.repo.GetByID(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("add note to work order %s: %w", workOrderID, err)
	}
	if _, err := s.requireOwnedProperty(ctx, w.PropertyID, ownerID); err != nil {
		return nil, fmt.Errorf("add note to work order %s: %w", workOrderID, err)
	}

	if message == "" {
		return nil, fmt.Errorf("add note to work order %s: %w", workOrderID, domain.ValidationErrors{
			{Field: "message", Message: "cannot be empty"},
		})
	}
	if !visibility.Valid() {
		return nil, fmt.Errorf("add note to work order %s: %w", workOrderID, domain.ValidationErrors{
			{Field: "visibility", Message: fmt.Sprintf("unknown visibility %q", visibility)},
		})
	}

	a := &domain.MaintenanceActivity{
		ID:          uuid.New(),
		WorkOrderID: workOrderID,
		Kind:        domain.MaintenanceActivityKindNote,
		Visibility:  visibility,
		Message:     message,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.repo.AddActivity(ctx, a); err != nil {
		return nil, fmt.Errorf("add note to work order %s: %w", workOrderID, err)
	}
	return a, nil
}

// requireOwnedProperty loads propertyID and returns domain.ErrNotFound
// (not ErrForbidden) if it belongs to someone else — the same
// IDOR-safe contract as PropertyService/UnitService/LeaseService.
func (s *WorkOrderService) requireOwnedProperty(ctx context.Context, propertyID, ownerID uuid.UUID) (*domain.Property, error) {
	p, err := s.propertyRepo.GetByID(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	if p.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

// validateUnitBelongsToProperty is nil-safe (a work order may be
// property-wide, with no unit at all) and guards against a
// cross-property unit_id — e.g. a stale ID from a different property's
// page.
func (s *WorkOrderService) validateUnitBelongsToProperty(ctx context.Context, unitID *uuid.UUID, propertyID uuid.UUID) error {
	if unitID == nil {
		return nil
	}
	u, err := s.unitRepo.GetByID(ctx, *unitID)
	if err != nil {
		return err
	}
	if u.PropertyID != propertyID {
		return domain.ValidationErrors{{Field: "unit_id", Message: "does not belong to this property"}}
	}
	return nil
}

// resolveVendor is nil-safe (a work order may have no vendor, assigned
// only to freeform internal staff) and guards against a vendor_id that
// exists but belongs to someone else, same IDOR-safe pattern as
// requireOwnedProperty.
func (s *WorkOrderService) resolveVendor(ctx context.Context, vendorID *uuid.UUID, ownerID uuid.UUID) (*domain.Vendor, error) {
	if vendorID == nil {
		return nil, nil
	}
	v, err := s.vendorRepo.GetByID(ctx, *vendorID)
	if err != nil {
		return nil, err
	}
	if v.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return v, nil
}

func (s *WorkOrderService) logActivity(ctx context.Context, workOrderID uuid.UUID, kind domain.MaintenanceActivityKind, message string, oldValue, newValue *string) {
	a := &domain.MaintenanceActivity{
		ID:          uuid.New(),
		WorkOrderID: workOrderID,
		Kind:        kind,
		Visibility:  domain.MaintenanceVisibilityInternal,
		Message:     message,
		OldValue:    oldValue,
		NewValue:    newValue,
		CreatedAt:   time.Now().UTC(),
	}
	// Best-effort: a failure to log history shouldn't fail the request
	// that already succeeded (the work order itself was created/updated
	// fine) — mirrors PropertyService's best-effort cache writes.
	if err := s.repo.AddActivity(ctx, a); err != nil {
		s.log.WarnContext(ctx, "failed to log maintenance activity", "error", err, "work_order_id", workOrderID)
	}
}

func validateWorkOrder(title string, category domain.WorkOrderCategory, priority domain.WorkOrderPriority, status domain.WorkOrderStatus) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if title == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "title", Message: "is required"})
	}
	if !category.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "category", Message: fmt.Sprintf("unknown category %q", category)})
	}
	if priority != "" && !priority.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "priority", Message: fmt.Sprintf("unknown priority %q", priority)})
	}
	if !status.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", status)})
	}
	return verrs
}

func validateEmergencySLA(dueDate *time.Time, now time.Time) *domain.ValidationError {
	if dueDate == nil {
		return &domain.ValidationError{Field: "due_date", Message: fmt.Sprintf("is required for emergency priority (within %d hours)", domain.EmergencySLAHours)}
	}
	deadline := now.Add(domain.EmergencySLAHours * time.Hour)
	if dueDate.After(deadline) {
		return &domain.ValidationError{Field: "due_date", Message: fmt.Sprintf("must be within %d hours for emergency priority", domain.EmergencySLAHours)}
	}
	return nil
}
