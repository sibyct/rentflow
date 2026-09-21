package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type WorkOrderRepository struct {
	pool *pgxpool.Pool
}

func NewWorkOrderRepository(pool *pgxpool.Pool) *WorkOrderRepository {
	return &WorkOrderRepository{pool: pool}
}

var _ domain.WorkOrderRepository = (*WorkOrderRepository)(nil)

const workOrderColumns = `
	id, property_id, unit_id, title, description, category, priority, status,
	reported_by, reported_by_contact, assigned_to, assigned_to_contact, vendor_id, rating, access_instructions,
	scheduled_start, scheduled_end, due_date, estimated_cost, actual_cost, photo_link, invoice_link,
	internal_notes, recurring_rule_id, completed_at, created_at, updated_at`

// qualifiedWorkOrderColumns is workOrderColumns aliased to "w." for
// ListForOwner's join to properties/units/vendors — all four tables
// have id/created_at/updated_at, so an unqualified SELECT across the
// join would be ambiguous.
const qualifiedWorkOrderColumns = `
	w.id, w.property_id, w.unit_id, w.title, w.description, w.category, w.priority, w.status,
	w.reported_by, w.reported_by_contact, w.assigned_to, w.assigned_to_contact, w.vendor_id, w.rating, w.access_instructions,
	w.scheduled_start, w.scheduled_end, w.due_date, w.estimated_cost, w.actual_cost, w.photo_link, w.invoice_link,
	w.internal_notes, w.recurring_rule_id, w.completed_at, w.created_at, w.updated_at`

func (r *WorkOrderRepository) Create(ctx context.Context, w *domain.WorkOrder) error {
	const q = `
		INSERT INTO work_orders (` + workOrderColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)`

	_, err := r.pool.Exec(ctx, q,
		w.ID, w.PropertyID, w.UnitID, w.Title, w.Description, w.Category, w.Priority, w.Status,
		w.ReportedBy, w.ReportedByContact, w.AssignedTo, w.AssignedToContact, w.VendorID, w.Rating, w.AccessInstructions,
		w.ScheduledStart, w.ScheduledEnd, w.DueDate, w.EstimatedCost, w.ActualCost, w.PhotoLink, w.InvoiceLink,
		w.InternalNotes, w.RecurringRuleID, w.CompletedAt, w.CreatedAt, w.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert work order %s: %w", w.ID, err)
	}
	return nil
}

func (r *WorkOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	const q = `SELECT ` + workOrderColumns + ` FROM work_orders WHERE id = $1`

	w, err := scanWorkOrder(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get work order %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get work order %s: %w", id, err)
	}
	return w, nil
}

// isOverdueSQL and priorityRankSQL back both filtering and the default
// sort — see domain.WorkOrder.IsOverdue and WorkOrderPriority for the
// Go-level equivalents this reproduces in SQL.
const isOverdueSQL = `(w.due_date IS NOT NULL AND w.due_date < CURRENT_DATE AND w.status NOT IN ('completed', 'cancelled'))`
const priorityRankSQL = `CASE w.priority WHEN 'emergency' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END`

var workOrderSortColumns = map[domain.WorkOrderSortKey]string{
	domain.WorkOrderSortPriority:  priorityRankSQL,
	domain.WorkOrderSortDueDate:   "w.due_date",
	domain.WorkOrderSortCreatedAt: "w.created_at",
	domain.WorkOrderSortStatus:    "w.status",
}

func (r *WorkOrderRepository) ListForOwner(ctx context.Context, opts domain.WorkOrderListOptions) ([]*domain.WorkOrderWithProperty, int, error) {
	where := []string{"p.owner_id = $1"}
	args := []any{opts.OwnerID}

	if opts.Filter.Search != "" {
		args = append(args, "%"+opts.Filter.Search+"%")
		where = append(where, fmt.Sprintf("(w.title ILIKE $%d OR w.description ILIKE $%d OR u.unit_name ILIKE $%d)", len(args), len(args), len(args)))
	}
	if opts.Filter.PropertyID != nil {
		args = append(args, *opts.Filter.PropertyID)
		where = append(where, fmt.Sprintf("w.property_id = $%d", len(args)))
	}
	if opts.Filter.Status != nil {
		args = append(args, *opts.Filter.Status)
		where = append(where, fmt.Sprintf("w.status = $%d", len(args)))
	}
	if opts.Filter.Priority != nil {
		args = append(args, *opts.Filter.Priority)
		where = append(where, fmt.Sprintf("w.priority = $%d", len(args)))
	}
	if opts.Filter.Category != nil {
		args = append(args, *opts.Filter.Category)
		where = append(where, fmt.Sprintf("w.category = $%d", len(args)))
	}
	if opts.Filter.AssignedTo != "" {
		args = append(args, opts.Filter.AssignedTo)
		where = append(where, fmt.Sprintf("w.assigned_to = $%d", len(args)))
	}
	if opts.Filter.VendorID != nil {
		args = append(args, *opts.Filter.VendorID)
		where = append(where, fmt.Sprintf("w.vendor_id = $%d", len(args)))
	}
	if opts.Filter.Overdue {
		where = append(where, isOverdueSQL)
	}
	if opts.Filter.Unassigned {
		where = append(where, "w.assigned_to = ''")
	}
	whereClause := strings.Join(where, " AND ")

	var orderBy string
	if opts.Sort == domain.WorkOrderSortDefault {
		// Spec default: overdue first, then priority descending, then
		// newest first — see domain.WorkOrderSortDefault.
		orderBy = fmt.Sprintf("%s DESC, %s DESC, w.created_at DESC", isOverdueSQL, priorityRankSQL)
	} else {
		sortColumn, ok := workOrderSortColumns[opts.Sort]
		if !ok {
			sortColumn = workOrderSortColumns[domain.WorkOrderSortCreatedAt]
		}
		sortDir := "ASC"
		if opts.SortDesc {
			sortDir = "DESC"
		}
		orderBy = fmt.Sprintf("%s %s, w.created_at DESC", sortColumn, sortDir)
	}

	args = append(args, opts.Limit, opts.Offset)
	q := fmt.Sprintf(
		`SELECT %s, p.name AS property_name, COALESCE(u.unit_name, '') AS unit_name, COALESCE(ven.company_name, '') AS vendor_name
		 FROM work_orders w
		 JOIN properties p ON p.id = w.property_id
		 LEFT JOIN units u ON u.id = w.unit_id
		 LEFT JOIN vendors ven ON ven.id = w.vendor_id
		 WHERE %s
		 ORDER BY %s, w.id ASC
		 LIMIT $%d OFFSET $%d`,
		qualifiedWorkOrderColumns, whereClause, orderBy, len(args)-1, len(args),
	)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list work orders for owner %s: %w", opts.OwnerID, err)
	}
	defer rows.Close()

	orders := make([]*domain.WorkOrderWithProperty, 0)
	for rows.Next() {
		w, err := scanWorkOrderWithProperty(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan work-order-with-property row: %w", err)
		}
		orders = append(orders, w)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate work-order-with-property rows: %w", err)
	}

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM work_orders w JOIN properties p ON p.id = w.property_id LEFT JOIN units u ON u.id = w.unit_id WHERE %s`, whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count work orders for owner %s: %w", opts.OwnerID, err)
	}

	return orders, total, nil
}

func (r *WorkOrderRepository) Update(ctx context.Context, w *domain.WorkOrder) error {
	const q = `
		UPDATE work_orders
		SET unit_id = $2, title = $3, description = $4, category = $5, priority = $6, status = $7,
			reported_by = $8, reported_by_contact = $9, assigned_to = $10, assigned_to_contact = $11,
			vendor_id = $12, rating = $13, access_instructions = $14, scheduled_start = $15, scheduled_end = $16,
			due_date = $17, estimated_cost = $18, actual_cost = $19, photo_link = $20, invoice_link = $21,
			internal_notes = $22, completed_at = $23, updated_at = $24
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		w.ID, w.UnitID, w.Title, w.Description, w.Category, w.Priority, w.Status,
		w.ReportedBy, w.ReportedByContact, w.AssignedTo, w.AssignedToContact,
		w.VendorID, w.Rating, w.AccessInstructions, w.ScheduledStart, w.ScheduledEnd, w.DueDate,
		w.EstimatedCost, w.ActualCost, w.PhotoLink, w.InvoiceLink,
		w.InternalNotes, w.CompletedAt, w.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update work order %s: %w", w.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update work order %s: %w", w.ID, domain.ErrNotFound)
	}
	return nil
}

func (r *WorkOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM work_orders WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete work order %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete work order %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

// bulkOwnerScope is the WHERE fragment BulkUpdateStatus/BulkReassign
// share: work_orders has no owner_id of its own, so ownership is
// enforced via a subquery into properties, same as every other bulk
// operation that isn't already scoped by a direct owner_id column.
const bulkOwnerScope = `id = ANY($2) AND property_id IN (SELECT id FROM properties WHERE owner_id = $3)`

func (r *WorkOrderRepository) BulkUpdateStatus(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, status domain.WorkOrderStatus) (int, error) {
	q := `UPDATE work_orders SET status = $1, updated_at = now() WHERE ` + bulkOwnerScope

	tag, err := r.pool.Exec(ctx, q, status, ids, ownerID)
	if err != nil {
		return 0, fmt.Errorf("bulk update status for %d work orders: %w", len(ids), err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *WorkOrderRepository) BulkReassign(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, assignedTo string) (int, error) {
	q := `UPDATE work_orders SET assigned_to = $1, status = CASE WHEN status = 'new' THEN 'assigned' ELSE status END, updated_at = now() WHERE ` + bulkOwnerScope

	tag, err := r.pool.Exec(ctx, q, assignedTo, ids, ownerID)
	if err != nil {
		return 0, fmt.Errorf("bulk reassign %d work orders: %w", len(ids), err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *WorkOrderRepository) GetSummary(ctx context.Context, ownerID uuid.UUID) (*domain.WorkOrderSummary, error) {
	const q = `
		SELECT
			COUNT(*) FILTER (WHERE w.status NOT IN ('completed', 'cancelled')) AS open_count,
			COUNT(*) FILTER (WHERE ` + isOverdueSQL + `) AS overdue_count,
			COUNT(*) FILTER (WHERE w.assigned_to = '' AND w.status NOT IN ('completed', 'cancelled')) AS unassigned_count,
			COUNT(*) FILTER (WHERE w.priority = 'emergency' AND w.status NOT IN ('completed', 'cancelled')) AS emergency_count,
			COUNT(*) FILTER (WHERE w.priority = 'high' AND w.status NOT IN ('completed', 'cancelled')) AS high_count,
			COUNT(*) FILTER (WHERE w.priority = 'medium' AND w.status NOT IN ('completed', 'cancelled')) AS medium_count,
			COUNT(*) FILTER (WHERE w.priority = 'low' AND w.status NOT IN ('completed', 'cancelled')) AS low_count
		FROM work_orders w
		JOIN properties p ON p.id = w.property_id
		WHERE p.owner_id = $1`

	s := &domain.WorkOrderSummary{}
	if err := r.pool.QueryRow(ctx, q, ownerID).Scan(&s.Open, &s.Overdue, &s.Unassigned, &s.Emergency, &s.High, &s.Medium, &s.Low); err != nil {
		return nil, fmt.Errorf("get work order summary for owner %s: %w", ownerID, err)
	}
	return s, nil
}

const maintenanceActivityColumns = `id, work_order_id, kind, visibility, message, old_value, new_value, created_at`

func (r *WorkOrderRepository) ListActivity(ctx context.Context, workOrderID uuid.UUID) ([]*domain.MaintenanceActivity, error) {
	const q = `SELECT ` + maintenanceActivityColumns + ` FROM maintenance_activity WHERE work_order_id = $1 ORDER BY created_at ASC, id ASC`

	rows, err := r.pool.Query(ctx, q, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("list activity for work order %s: %w", workOrderID, err)
	}
	defer rows.Close()

	activity := make([]*domain.MaintenanceActivity, 0)
	for rows.Next() {
		a, err := scanMaintenanceActivity(rows)
		if err != nil {
			return nil, fmt.Errorf("scan maintenance activity row: %w", err)
		}
		activity = append(activity, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate maintenance activity rows: %w", err)
	}
	return activity, nil
}

func (r *WorkOrderRepository) ListRecentActivityForOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.MaintenanceActivityWithContext, error) {
	const q = `
		SELECT ma.id, ma.work_order_id, ma.kind, ma.visibility, ma.message, ma.old_value, ma.new_value, ma.created_at,
			w.title, p.name
		FROM maintenance_activity ma
		JOIN work_orders w ON w.id = ma.work_order_id
		JOIN properties p ON p.id = w.property_id
		WHERE p.owner_id = $1
		ORDER BY ma.created_at DESC, ma.id DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, q, ownerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list recent activity for owner %s: %w", ownerID, err)
	}
	defer rows.Close()

	activity := make([]*domain.MaintenanceActivityWithContext, 0)
	for rows.Next() {
		a, err := scanMaintenanceActivityWithContext(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recent activity row: %w", err)
		}
		activity = append(activity, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent activity rows: %w", err)
	}
	return activity, nil
}

func (r *WorkOrderRepository) AddActivity(ctx context.Context, a *domain.MaintenanceActivity) error {
	const q = `INSERT INTO maintenance_activity (` + maintenanceActivityColumns + `) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, q, a.ID, a.WorkOrderID, a.Kind, a.Visibility, a.Message, a.OldValue, a.NewValue, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert maintenance activity %s: %w", a.ID, err)
	}
	return nil
}

func scanWorkOrder(row rowScanner) (*domain.WorkOrder, error) {
	var w domain.WorkOrder
	var unitID, vendorID uuid.NullUUID
	var rating sql.NullInt32
	var scheduledStart, scheduledEnd, dueDate, completedAt sql.NullTime
	var estimatedCost, actualCost sql.NullFloat64
	var recurringRuleID uuid.NullUUID

	err := row.Scan(
		&w.ID, &w.PropertyID, &unitID, &w.Title, &w.Description, &w.Category, &w.Priority, &w.Status,
		&w.ReportedBy, &w.ReportedByContact, &w.AssignedTo, &w.AssignedToContact, &vendorID, &rating, &w.AccessInstructions,
		&scheduledStart, &scheduledEnd, &dueDate, &estimatedCost, &actualCost, &w.PhotoLink, &w.InvoiceLink,
		&w.InternalNotes, &recurringRuleID, &completedAt, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	applyWorkOrderNullables(&w, unitID, vendorID, rating, scheduledStart, scheduledEnd, dueDate, completedAt, estimatedCost, actualCost, recurringRuleID)
	return &w, nil
}

func scanWorkOrderWithProperty(row rowScanner) (*domain.WorkOrderWithProperty, error) {
	var w domain.WorkOrderWithProperty
	var unitID, vendorID uuid.NullUUID
	var rating sql.NullInt32
	var scheduledStart, scheduledEnd, dueDate, completedAt sql.NullTime
	var estimatedCost, actualCost sql.NullFloat64
	var recurringRuleID uuid.NullUUID

	err := row.Scan(
		&w.ID, &w.PropertyID, &unitID, &w.Title, &w.Description, &w.Category, &w.Priority, &w.Status,
		&w.ReportedBy, &w.ReportedByContact, &w.AssignedTo, &w.AssignedToContact, &vendorID, &rating, &w.AccessInstructions,
		&scheduledStart, &scheduledEnd, &dueDate, &estimatedCost, &actualCost, &w.PhotoLink, &w.InvoiceLink,
		&w.InternalNotes, &recurringRuleID, &completedAt, &w.CreatedAt, &w.UpdatedAt,
		&w.PropertyName, &w.UnitName, &w.VendorName,
	)
	if err != nil {
		return nil, err
	}
	applyWorkOrderNullables(&w.WorkOrder, unitID, vendorID, rating, scheduledStart, scheduledEnd, dueDate, completedAt, estimatedCost, actualCost, recurringRuleID)
	return &w, nil
}

func applyWorkOrderNullables(
	w *domain.WorkOrder,
	unitID, vendorID uuid.NullUUID,
	rating sql.NullInt32,
	scheduledStart, scheduledEnd, dueDate, completedAt sql.NullTime,
	estimatedCost, actualCost sql.NullFloat64,
	recurringRuleID uuid.NullUUID,
) {
	if unitID.Valid {
		v := unitID.UUID
		w.UnitID = &v
	}
	if vendorID.Valid {
		v := vendorID.UUID
		w.VendorID = &v
	}
	if rating.Valid {
		v := int(rating.Int32)
		w.Rating = &v
	}
	if scheduledStart.Valid {
		w.ScheduledStart = &scheduledStart.Time
	}
	if scheduledEnd.Valid {
		w.ScheduledEnd = &scheduledEnd.Time
	}
	if dueDate.Valid {
		w.DueDate = &dueDate.Time
	}
	if completedAt.Valid {
		w.CompletedAt = &completedAt.Time
	}
	if estimatedCost.Valid {
		v := estimatedCost.Float64
		w.EstimatedCost = &v
	}
	if actualCost.Valid {
		v := actualCost.Float64
		w.ActualCost = &v
	}
	if recurringRuleID.Valid {
		v := recurringRuleID.UUID
		w.RecurringRuleID = &v
	}
}

func scanMaintenanceActivity(row rowScanner) (*domain.MaintenanceActivity, error) {
	var a domain.MaintenanceActivity
	var oldValue, newValue sql.NullString

	if err := row.Scan(&a.ID, &a.WorkOrderID, &a.Kind, &a.Visibility, &a.Message, &oldValue, &newValue, &a.CreatedAt); err != nil {
		return nil, err
	}
	if oldValue.Valid {
		a.OldValue = &oldValue.String
	}
	if newValue.Valid {
		a.NewValue = &newValue.String
	}
	return &a, nil
}

func scanMaintenanceActivityWithContext(row rowScanner) (*domain.MaintenanceActivityWithContext, error) {
	var a domain.MaintenanceActivityWithContext
	var oldValue, newValue sql.NullString

	if err := row.Scan(
		&a.ID, &a.WorkOrderID, &a.Kind, &a.Visibility, &a.Message, &oldValue, &newValue, &a.CreatedAt,
		&a.WorkOrderTitle, &a.PropertyName,
	); err != nil {
		return nil, err
	}
	if oldValue.Valid {
		a.OldValue = &oldValue.String
	}
	if newValue.Valid {
		a.NewValue = &newValue.String
	}
	return &a, nil
}
