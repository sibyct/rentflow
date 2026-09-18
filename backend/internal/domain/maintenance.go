package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type WorkOrderCategory string

const (
	WorkOrderCategoryPlumbing    WorkOrderCategory = "plumbing"
	WorkOrderCategoryElectrical  WorkOrderCategory = "electrical"
	WorkOrderCategoryHVAC        WorkOrderCategory = "hvac"
	WorkOrderCategoryAppliance   WorkOrderCategory = "appliance"
	WorkOrderCategoryPestControl WorkOrderCategory = "pest_control"
	WorkOrderCategoryGeneral     WorkOrderCategory = "general"
	WorkOrderCategoryOther       WorkOrderCategory = "other"
)

func (c WorkOrderCategory) Valid() bool {
	switch c {
	case WorkOrderCategoryPlumbing, WorkOrderCategoryElectrical, WorkOrderCategoryHVAC, WorkOrderCategoryAppliance,
		WorkOrderCategoryPestControl, WorkOrderCategoryGeneral, WorkOrderCategoryOther:
		return true
	default:
		return false
	}
}

type WorkOrderPriority string

const (
	WorkOrderPriorityLow       WorkOrderPriority = "low"
	WorkOrderPriorityMedium    WorkOrderPriority = "medium"
	WorkOrderPriorityHigh      WorkOrderPriority = "high"
	WorkOrderPriorityEmergency WorkOrderPriority = "emergency"
)

func (p WorkOrderPriority) Valid() bool {
	switch p {
	case WorkOrderPriorityLow, WorkOrderPriorityMedium, WorkOrderPriorityHigh, WorkOrderPriorityEmergency:
		return true
	default:
		return false
	}
}

// EmergencySLAHours is how soon a due date must fall, from creation,
// for an emergency-priority work order — enforced in
// WorkOrderService.CreateWorkOrder, not just the frontend form.
const EmergencySLAHours = 24

type WorkOrderStatus string

const (
	WorkOrderStatusNew        WorkOrderStatus = "new"
	WorkOrderStatusAssigned   WorkOrderStatus = "assigned"
	WorkOrderStatusInProgress WorkOrderStatus = "in_progress"
	WorkOrderStatusOnHold     WorkOrderStatus = "on_hold"
	WorkOrderStatusCompleted  WorkOrderStatus = "completed"
	WorkOrderStatusCancelled  WorkOrderStatus = "cancelled"
)

func (s WorkOrderStatus) Valid() bool {
	switch s {
	case WorkOrderStatusNew, WorkOrderStatusAssigned, WorkOrderStatusInProgress, WorkOrderStatusOnHold,
		WorkOrderStatusCompleted, WorkOrderStatusCancelled:
		return true
	default:
		return false
	}
}

// IsOpen reports whether a work order still needs attention — used for
// both the "Open"/"Overdue" summary counters and IsOverdue below.
func (s WorkOrderStatus) IsOpen() bool {
	return s != WorkOrderStatusCompleted && s != WorkOrderStatusCancelled
}

// WorkOrder is the core business entity. It has no JSON or SQL
// knowledge — see dto and repository/postgres respectively.
//
// ReportedBy/AssignedTo are freeform text, not a foreign key to a
// tenant or vendor/staff record: neither entity exists in this
// codebase yet, so this mirrors Lease.PrimaryResidentName's precedent
// rather than fabricating a link. PhotoLink/InvoiceLink are likewise
// freeform URLs, not real file uploads — there is no file-storage
// feature here either.
type WorkOrder struct {
	ID                 uuid.UUID
	PropertyID         uuid.UUID
	UnitID             *uuid.UUID
	Title              string
	Description        string
	Category           WorkOrderCategory
	Priority           WorkOrderPriority
	Status             WorkOrderStatus
	ReportedBy         string
	ReportedByContact  string
	AssignedTo         string
	AssignedToContact  string
	AccessInstructions string
	ScheduledStart     *time.Time
	ScheduledEnd       *time.Time
	DueDate            *time.Time
	EstimatedCost      *float64
	ActualCost         *float64
	PhotoLink          string
	InvoiceLink        string
	InternalNotes      string
	RecurringRuleID    *uuid.UUID
	CompletedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// IsOverdue reports whether DueDate has passed and the work order is
// still open — a completed/cancelled work order is never "overdue",
// regardless of how late it finished.
func (w *WorkOrder) IsOverdue(now time.Time) bool {
	if w.DueDate == nil || !w.Status.IsOpen() {
		return false
	}
	return truncateToDate(*w.DueDate).Before(truncateToDate(now))
}

type CreateWorkOrderInput struct {
	PropertyID         uuid.UUID
	UnitID             *uuid.UUID
	Title              string
	Description        string
	Category           WorkOrderCategory
	Priority           WorkOrderPriority
	Status             WorkOrderStatus
	ReportedBy         string
	ReportedByContact  string
	AssignedTo         string
	AssignedToContact  string
	AccessInstructions string
	ScheduledStart     *time.Time
	ScheduledEnd       *time.Time
	DueDate            *time.Time
	EstimatedCost      *float64
	ActualCost         *float64
	PhotoLink          string
	InvoiceLink        string
	InternalNotes      string
	RecurringRuleID    *uuid.UUID
}

// UpdateWorkOrderInput fields are all optional (nil = leave unchanged)
// — mirrors UpdateLeaseInput/UpdateUnitInput.
type UpdateWorkOrderInput struct {
	UnitID             *uuid.UUID
	UnitIDSet          bool // UnitID is itself nullable, so "clear it back to property-wide" needs its own flag
	Title              *string
	Description        *string
	Category           *WorkOrderCategory
	Priority           *WorkOrderPriority
	Status             *WorkOrderStatus
	ReportedBy         *string
	ReportedByContact  *string
	AssignedTo         *string
	AssignedToContact  *string
	AccessInstructions *string
	ScheduledStart     *time.Time
	ScheduledEnd       *time.Time
	DueDate            *time.Time
	EstimatedCost      *float64
	ActualCost         *float64
	PhotoLink          *string
	InvoiceLink        *string
	InternalNotes      *string
}

type WorkOrderSortKey string

const (
	WorkOrderSortDefault   WorkOrderSortKey = "" // overdue first, then priority desc, then newest first — see WorkOrderRepository.ListForOwner
	WorkOrderSortPriority  WorkOrderSortKey = "priority"
	WorkOrderSortDueDate   WorkOrderSortKey = "due_date"
	WorkOrderSortCreatedAt WorkOrderSortKey = "created_at"
	WorkOrderSortStatus    WorkOrderSortKey = "status"
)

func (k WorkOrderSortKey) Valid() bool {
	switch k {
	case WorkOrderSortDefault, WorkOrderSortPriority, WorkOrderSortDueDate, WorkOrderSortCreatedAt, WorkOrderSortStatus:
		return true
	default:
		return false
	}
}

// WorkOrderListFilter narrows ListForOwner results. Zero values (empty
// Search, nil pointers) mean "don't filter on this" — mirrors
// LeaseListFilter/UnitListFilter. Overdue and Unassigned are computed
// filters (see WorkOrderRepository.ListForOwner) backing the summary
// counters' click-to-filter behavior.
type WorkOrderListFilter struct {
	Search     string
	PropertyID *uuid.UUID
	Status     *WorkOrderStatus
	Priority   *WorkOrderPriority
	Category   *WorkOrderCategory
	AssignedTo string
	Overdue    bool
	Unassigned bool
}

type WorkOrderListOptions struct {
	OwnerID  uuid.UUID
	Filter   WorkOrderListFilter
	Sort     WorkOrderSortKey
	SortDesc bool
	Limit    int
	Offset   int
}

// WorkOrderWithProperty decorates a WorkOrder with its property's (and,
// if set, unit's) display name — mirrors UnitWithProperty/
// LeaseWithUnitProperty's role for their own portfolio-wide lists.
type WorkOrderWithProperty struct {
	WorkOrder
	PropertyName string
	UnitName     string // empty when WorkOrder.UnitID is nil (a property-wide work order)
}

// WorkOrderSummary backs the list page's clickable summary counters.
type WorkOrderSummary struct {
	Open       int
	Overdue    int
	Unassigned int
	Emergency  int
}

type MaintenanceActivityKind string

const (
	MaintenanceActivityKindNote         MaintenanceActivityKind = "note"
	MaintenanceActivityKindStatusChange MaintenanceActivityKind = "status_change"
)

type MaintenanceVisibility string

const (
	MaintenanceVisibilityInternal      MaintenanceVisibility = "internal"
	MaintenanceVisibilityTenantVisible MaintenanceVisibility = "tenant_visible"
)

func (v MaintenanceVisibility) Valid() bool {
	switch v {
	case MaintenanceVisibilityInternal, MaintenanceVisibilityTenantVisible:
		return true
	default:
		return false
	}
}

// MaintenanceActivity is one entry in a work order's unified timeline —
// see the migration's table comment for why notes and the auto-inserted
// status history share one table instead of three.
type MaintenanceActivity struct {
	ID          uuid.UUID
	WorkOrderID uuid.UUID
	Kind        MaintenanceActivityKind
	Visibility  MaintenanceVisibility
	Message     string
	OldValue    *string
	NewValue    *string
	CreatedAt   time.Time
}

// WorkOrderRepository is the port implemented by internal/repository/postgres.
type WorkOrderRepository interface {
	Create(ctx context.Context, w *WorkOrder) error
	GetByID(ctx context.Context, id uuid.UUID) (*WorkOrder, error)
	// ListForOwner enforces p.owner_id = opts.OwnerID in the query
	// itself, the same row-security pattern every other portfolio-wide
	// list in this codebase uses.
	ListForOwner(ctx context.Context, opts WorkOrderListOptions) ([]*WorkOrderWithProperty, int, error)
	Update(ctx context.Context, w *WorkOrder) error
	Delete(ctx context.Context, id uuid.UUID) error
	BulkUpdateStatus(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, status WorkOrderStatus) (int, error)
	BulkReassign(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, assignedTo string) (int, error)
	GetSummary(ctx context.Context, ownerID uuid.UUID) (*WorkOrderSummary, error)
	ListActivity(ctx context.Context, workOrderID uuid.UUID) ([]*MaintenanceActivity, error)
	AddActivity(ctx context.Context, a *MaintenanceActivity) error
}

// WorkOrderService is the port implemented by internal/service and
// consumed by the HTTP transport layer. Every method resolves ownership
// via the work order's own PropertyID — one hop, unlike Lease/Unit
// which need two, since work_orders.property_id is a direct column.
type WorkOrderService interface {
	CreateWorkOrder(ctx context.Context, ownerID uuid.UUID, input CreateWorkOrderInput) (*WorkOrder, error)
	GetWorkOrder(ctx context.Context, id, ownerID uuid.UUID) (*WorkOrderWithProperty, error)
	ListWorkOrdersForOwner(ctx context.Context, ownerID uuid.UUID, opts WorkOrderListOptions) ([]*WorkOrderWithProperty, int, error)
	UpdateWorkOrder(ctx context.Context, id, ownerID uuid.UUID, input UpdateWorkOrderInput) (*WorkOrder, error)
	DeleteWorkOrder(ctx context.Context, id, ownerID uuid.UUID) error
	GetSummary(ctx context.Context, ownerID uuid.UUID) (*WorkOrderSummary, error)
	BulkUpdateStatus(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, status WorkOrderStatus) (int, error)
	BulkReassign(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, assignedTo string) (int, error)
	ListActivity(ctx context.Context, workOrderID, ownerID uuid.UUID) ([]*MaintenanceActivity, error)
	AddNote(ctx context.Context, workOrderID, ownerID uuid.UUID, message string, visibility MaintenanceVisibility) (*MaintenanceActivity, error)
}

type RecurringRuleFrequencyUnit string

const (
	RecurringRuleFrequencyDays   RecurringRuleFrequencyUnit = "days"
	RecurringRuleFrequencyWeeks  RecurringRuleFrequencyUnit = "weeks"
	RecurringRuleFrequencyMonths RecurringRuleFrequencyUnit = "months"
)

func (u RecurringRuleFrequencyUnit) Valid() bool {
	switch u {
	case RecurringRuleFrequencyDays, RecurringRuleFrequencyWeeks, RecurringRuleFrequencyMonths:
		return true
	default:
		return false
	}
}

// RecurringRule is a template for preventive maintenance (e.g. "HVAC
// filter change every 3 months") that generates a real WorkOrder each
// time it's due. There is no background scheduler in this codebase, so
// generation is an explicit, manager-triggered action
// (RecurringRuleService.GenerateNow) rather than automatic — a real
// deployment would call the same method from a cron job once one
// exists; this exposes the same operation without inventing scheduler
// infrastructure that isn't there.
type RecurringRule struct {
	ID                uuid.UUID
	PropertyID        uuid.UUID
	UnitID            *uuid.UUID
	Title             string
	Description       string
	Category          WorkOrderCategory
	FrequencyInterval int
	FrequencyUnit     RecurringRuleFrequencyUnit
	NextDueDate       time.Time
	Active            bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NextOccurrence advances NextDueDate by one frequency step, called
// after GenerateNow successfully creates a work order from this rule.
func (r *RecurringRule) NextOccurrence() time.Time {
	switch r.FrequencyUnit {
	case RecurringRuleFrequencyDays:
		return r.NextDueDate.AddDate(0, 0, r.FrequencyInterval)
	case RecurringRuleFrequencyWeeks:
		return r.NextDueDate.AddDate(0, 0, 7*r.FrequencyInterval)
	default: // RecurringRuleFrequencyMonths
		return r.NextDueDate.AddDate(0, r.FrequencyInterval, 0)
	}
}

type CreateRecurringRuleInput struct {
	PropertyID        uuid.UUID
	UnitID            *uuid.UUID
	Title             string
	Description       string
	Category          WorkOrderCategory
	FrequencyInterval int
	FrequencyUnit     RecurringRuleFrequencyUnit
	NextDueDate       time.Time
}

type UpdateRecurringRuleInput struct {
	UnitID            *uuid.UUID
	UnitIDSet         bool
	Title             *string
	Description       *string
	Category          *WorkOrderCategory
	FrequencyInterval *int
	FrequencyUnit     *RecurringRuleFrequencyUnit
	NextDueDate       *time.Time
	Active            *bool
}

// RecurringRuleWithProperty decorates a RecurringRule with its
// property's (and, if set, unit's) display name — mirrors
// WorkOrderWithProperty.
type RecurringRuleWithProperty struct {
	RecurringRule
	PropertyName string
	UnitName     string
}

// RecurringRuleRepository is the port implemented by internal/repository/postgres.
type RecurringRuleRepository interface {
	Create(ctx context.Context, r *RecurringRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*RecurringRule, error)
	ListForOwner(ctx context.Context, ownerID uuid.UUID) ([]*RecurringRuleWithProperty, error)
	Update(ctx context.Context, r *RecurringRule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// RecurringRuleService is the port implemented by internal/service and
// consumed by the HTTP transport layer.
type RecurringRuleService interface {
	CreateRule(ctx context.Context, ownerID uuid.UUID, input CreateRecurringRuleInput) (*RecurringRule, error)
	ListRulesForOwner(ctx context.Context, ownerID uuid.UUID) ([]*RecurringRuleWithProperty, error)
	UpdateRule(ctx context.Context, id, ownerID uuid.UUID, input UpdateRecurringRuleInput) (*RecurringRule, error)
	DeleteRule(ctx context.Context, id, ownerID uuid.UUID) error
	GenerateNow(ctx context.Context, id, ownerID uuid.UUID) (*WorkOrder, error)
}
