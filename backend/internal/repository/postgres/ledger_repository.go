package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type LedgerRepository struct {
	pool *pgxpool.Pool
}

func NewLedgerRepository(pool *pgxpool.Pool) *LedgerRepository {
	return &LedgerRepository{pool: pool}
}

var _ domain.LedgerRepository = (*LedgerRepository)(nil)

// execer is the slice of pgx that both *pgxpool.Pool and pgx.Tx satisfy,
// so audit/insert helpers work inside or outside a transaction.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

const txColumns = `
	id, owner_id, property_id, unit_id, kind, type, charge_type, category, amount_cents,
	incurred_on, due_on, period, lease_id, work_order_id, vendor_id, vendor_name, description,
	tax_deductible, is_recurring, recurrence_frequency, recurrence_parent_id, attachment_id,
	source, voided_at, created_by, created_at, updated_by, updated_at`

func qualifiedTxColumns(alias string) string {
	cols := strings.Split(strings.ReplaceAll(strings.TrimSpace(txColumns), "\n", " "), ",")
	for i, c := range cols {
		cols[i] = alias + "." + strings.TrimSpace(c)
	}
	return strings.Join(cols, ", ")
}

// paymentLateral aggregates a ledger row's live (un-voided) payments.
// Every read model joins it so status filters can reference
// pay.paid_cents directly.
const paymentLateral = `
	LEFT JOIN LATERAL (
		SELECT COALESCE(SUM(p.amount_cents), 0)::bigint AS paid_cents, MAX(p.paid_on) AS last_paid_on
		FROM transaction_payments p
		WHERE p.transaction_id = t.id AND p.voided_at IS NULL
	) pay ON true`

const txRowFrom = `
	FROM transactions t
	JOIN properties pr ON pr.id = t.property_id
	LEFT JOIN units u ON u.id = t.unit_id
	LEFT JOIN leases l ON l.id = t.lease_id
	LEFT JOIN vendors v ON v.id = t.vendor_id
	LEFT JOIN work_orders w ON w.id = t.work_order_id` + paymentLateral

var txRowSelect = `SELECT ` + qualifiedTxColumns("t") + `,
	pay.paid_cents, pay.last_paid_on,
	pr.name, COALESCE(u.unit_name, ''), COALESCE(l.primary_resident_name, ''),
	COALESCE(v.company_name, ''), COALESCE(w.title, '')` + txRowFrom

func strPtr[T ~string](p *T) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func nullUUIDPtr(n uuid.NullUUID) *uuid.UUID {
	if !n.Valid {
		return nil
	}
	id := n.UUID
	return &id
}

func nullTimePtr(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	t := n.Time
	return &t
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func insertAudit(ctx context.Context, q execer, a domain.AuditEntry) error {
	changes, err := json.Marshal(a.Changes)
	if err != nil {
		return fmt.Errorf("marshal audit changes: %w", err)
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO accounting_audit_log (id, owner_id, entity_type, entity_id, actor_id, action, changes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		a.ID, a.OwnerID, a.EntityType, a.EntityID, a.ActorID, a.Action, changes, a.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert audit entry: %w", err)
	}
	return nil
}

func insertTransaction(ctx context.Context, q execer, t *domain.Transaction) (int64, error) {
	tag, err := q.Exec(ctx, `
		INSERT INTO transactions (`+txColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
		ON CONFLICT DO NOTHING`,
		t.ID, t.OwnerID, t.PropertyID, t.UnitID, string(t.Kind), string(t.Type), strPtr(t.ChargeType), strPtr(t.Category), t.AmountCents,
		t.IncurredOn, t.DueOn, t.Period, t.LeaseID, t.WorkOrderID, t.VendorID, t.VendorName, t.Description,
		t.TaxDeductible, t.IsRecurring, strPtr(t.RecurrenceFrequency), t.RecurrenceParentID, t.AttachmentID,
		string(t.Source), t.VoidedAt, t.CreatedBy, t.CreatedAt, t.UpdatedBy, t.UpdatedAt,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func insertPayment(ctx context.Context, q execer, p *domain.TransactionPayment) error {
	_, err := q.Exec(ctx, `
		INSERT INTO transaction_payments (id, transaction_id, amount_cents, paid_on, method, reference, bank_account_id, voided_at, recorded_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		p.ID, p.TransactionID, p.AmountCents, p.PaidOn, p.Method, p.Reference, p.BankAccountID, p.VoidedAt, p.RecordedBy, p.CreatedAt,
	)
	return err
}

// CreateTransaction inserts a ledger row, an optional first payment
// (the expense form's "Date paid") and the audit entries in one
// transaction. A collision with an idempotency index (a second rent row
// for the same lease and month, a second expense for one work order)
// returns domain.ErrAlreadyExists.
func (r *LedgerRepository) CreateTransaction(ctx context.Context, t *domain.Transaction, payment *domain.TransactionPayment, audits []domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create transaction %s: %w", t.ID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Plain INSERT semantics: a conflict must surface, not be swallowed.
	inserted, err := insertTransaction(ctx, tx, t)
	if err != nil {
		return fmt.Errorf("insert transaction %s: %w", t.ID, err)
	}
	if inserted == 0 {
		return fmt.Errorf("insert transaction %s: %w", t.ID, domain.ErrAlreadyExists)
	}
	if payment != nil {
		if err := insertPayment(ctx, tx, payment); err != nil {
			return fmt.Errorf("insert payment for transaction %s: %w", t.ID, err)
		}
	}
	for _, a := range audits {
		if err := insertAudit(ctx, tx, a); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create transaction %s: %w", t.ID, err)
	}
	return nil
}

func (r *LedgerRepository) InsertGenerated(ctx context.Context, ts []*domain.Transaction, actorID uuid.UUID) ([]*domain.Transaction, error) {
	if len(ts) == 0 {
		return nil, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin insert generated: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var inserted []*domain.Transaction
	for _, t := range ts {
		n, err := insertTransaction(ctx, tx, t)
		if err != nil {
			return nil, fmt.Errorf("insert generated transaction %s: %w", t.ID, err)
		}
		if n == 0 {
			continue // already generated — the idempotency index did its job
		}
		inserted = append(inserted, t)
		audit := domain.NewAuditEntry(t.OwnerID, domain.AuditEntityTransaction, t.ID, actorID, "generated",
			map[string]domain.FieldChange{
				"type":         {New: string(t.Type)},
				"source":       {New: string(t.Source)},
				"amount_cents": {New: t.AmountCents},
			}, t.CreatedAt)
		if err := insertAudit(ctx, tx, audit); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit insert generated: %w", err)
	}
	return inserted, nil
}

func (r *LedgerRepository) UpdateTransaction(ctx context.Context, t *domain.Transaction, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update transaction %s: %w", t.ID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE transactions SET
			property_id = $2, unit_id = $3, charge_type = $4, category = $5, amount_cents = $6,
			incurred_on = $7, due_on = $8, vendor_id = $9, vendor_name = $10, description = $11,
			tax_deductible = $12, is_recurring = $13, recurrence_frequency = $14, attachment_id = $15,
			updated_by = $16, updated_at = $17
		WHERE id = $1 AND voided_at IS NULL`,
		t.ID, t.PropertyID, t.UnitID, strPtr(t.ChargeType), strPtr(t.Category), t.AmountCents,
		t.IncurredOn, t.DueOn, t.VendorID, t.VendorName, t.Description,
		t.TaxDeductible, t.IsRecurring, strPtr(t.RecurrenceFrequency), t.AttachmentID,
		t.UpdatedBy, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update transaction %s: %w", t.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update transaction %s: %w", t.ID, domain.ErrNotFound)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update transaction %s: %w", t.ID, err)
	}
	return nil
}

func (r *LedgerRepository) VoidTransaction(ctx context.Context, id uuid.UUID, at time.Time, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin void transaction %s: %w", id, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE transactions SET voided_at = $2, updated_at = $2, updated_by = $3 WHERE id = $1 AND voided_at IS NULL`,
		id, at, audit.ActorID)
	if err != nil {
		return fmt.Errorf("void transaction %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("void transaction %s: %w", id, domain.ErrNotFound)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit void transaction %s: %w", id, err)
	}
	return nil
}

// AddPayments writes every payment (and its audit entry) in one
// transaction, so a lump-sum lease payment allocated across several rows
// is all-or-nothing.
func (r *LedgerRepository) AddPayments(ctx context.Context, payments []*domain.TransactionPayment, audits []domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add payments: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, p := range payments {
		if err := insertPayment(ctx, tx, p); err != nil {
			return fmt.Errorf("insert payment %s: %w", p.ID, err)
		}
	}
	for _, a := range audits {
		if err := insertAudit(ctx, tx, a); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit add payments: %w", err)
	}
	return nil
}

func (r *LedgerRepository) VoidPayment(ctx context.Context, id uuid.UUID, at time.Time, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin void payment %s: %w", id, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE transaction_payments SET voided_at = $2 WHERE id = $1 AND voided_at IS NULL`, id, at)
	if err != nil {
		return fmt.Errorf("void payment %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("void payment %s: %w", id, domain.ErrNotFound)
	}
	// A reconciled payment can no longer back its bank line.
	if _, err := tx.Exec(ctx, `UPDATE bank_statement_lines SET matched_payment_id = NULL, matched_at = NULL WHERE matched_payment_id = $1`, id); err != nil {
		return fmt.Errorf("unmatch voided payment %s: %w", id, err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit void payment %s: %w", id, err)
	}
	return nil
}

func (r *LedgerRepository) GetTransaction(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	t, err := scanTransaction(r.pool.QueryRow(ctx, `SELECT `+txColumns+` FROM transactions WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get transaction %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get transaction %s: %w", id, err)
	}
	return t, nil
}

func (r *LedgerRepository) GetTransactionRow(ctx context.Context, id uuid.UUID) (*domain.TransactionRow, error) {
	row, err := scanTransactionRow(r.pool.QueryRow(ctx, txRowSelect+` WHERE t.id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get transaction %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get transaction %s: %w", id, err)
	}
	return row, nil
}

func (r *LedgerRepository) GetTransactionRowByWorkOrder(ctx context.Context, workOrderID uuid.UUID) (*domain.TransactionRow, error) {
	row, err := scanTransactionRow(r.pool.QueryRow(ctx, txRowSelect+` WHERE t.work_order_id = $1 AND t.source = 'work_order'`, workOrderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get expense for work order %s: %w", workOrderID, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get expense for work order %s: %w", workOrderID, err)
	}
	return row, nil
}

func (r *LedgerRepository) GetActiveLeaseIDForUnit(ctx context.Context, unitID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, `SELECT id FROM leases WHERE unit_id = $1 AND status = 'active'`, unitID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("get active lease for unit %s: %w", unitID, domain.ErrNotFound)
		}
		return uuid.Nil, fmt.Errorf("get active lease for unit %s: %w", unitID, err)
	}
	return id, nil
}

func (r *LedgerRepository) BankAccountOwnedBy(ctx context.Context, id, ownerID uuid.UUID) (bool, error) {
	var ok bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM bank_accounts WHERE id = $1 AND owner_id = $2)`, id, ownerID).Scan(&ok); err != nil {
		return false, fmt.Errorf("check bank account ownership: %w", err)
	}
	return ok, nil
}

func (r *LedgerRepository) AttachmentOwnedBy(ctx context.Context, id, ownerID uuid.UUID) (bool, error) {
	var ok bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM attachments WHERE id = $1 AND owner_id = $2 AND status = 'ready')`, id, ownerID).Scan(&ok); err != nil {
		return false, fmt.Errorf("check attachment ownership: %w", err)
	}
	return ok, nil
}

const paymentColumns = `id, transaction_id, amount_cents, paid_on, method, reference, bank_account_id, voided_at, recorded_by, created_at`

func scanPayment(row rowScanner) (*domain.TransactionPayment, error) {
	var p domain.TransactionPayment
	var bank uuid.NullUUID
	var voided sql.NullTime
	if err := row.Scan(&p.ID, &p.TransactionID, &p.AmountCents, &p.PaidOn, &p.Method, &p.Reference, &bank, &voided, &p.RecordedBy, &p.CreatedAt); err != nil {
		return nil, err
	}
	p.BankAccountID = nullUUIDPtr(bank)
	p.VoidedAt = nullTimePtr(voided)
	return &p, nil
}

func (r *LedgerRepository) GetPayment(ctx context.Context, id uuid.UUID) (*domain.TransactionPayment, error) {
	p, err := scanPayment(r.pool.QueryRow(ctx, `SELECT `+paymentColumns+` FROM transaction_payments WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get payment %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get payment %s: %w", id, err)
	}
	return p, nil
}

func (r *LedgerRepository) ListPayments(ctx context.Context, transactionID uuid.UUID) ([]*domain.TransactionPayment, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+paymentColumns+` FROM transaction_payments WHERE transaction_id = $1 ORDER BY paid_on, created_at`, transactionID)
	if err != nil {
		return nil, fmt.Errorf("list payments for %s: %w", transactionID, err)
	}
	defer rows.Close()

	var out []*domain.TransactionPayment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *LedgerRepository) ListOpenIncomeForLease(ctx context.Context, leaseID uuid.UUID) ([]*domain.TransactionRow, error) {
	q := txRowSelect + `
		WHERE t.lease_id = $1 AND t.kind = 'income' AND t.voided_at IS NULL AND pay.paid_cents < t.amount_cents
		ORDER BY t.due_on, CASE t.type WHEN 'rent' THEN 0 WHEN 'late_fee' THEN 1 ELSE 2 END, t.created_at`
	rows, err := r.pool.Query(ctx, q, leaseID)
	if err != nil {
		return nil, fmt.Errorf("list open income for lease %s: %w", leaseID, err)
	}
	defer rows.Close()

	var out []*domain.TransactionRow
	for rows.Next() {
		row, err := scanTransactionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction row: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// receivableStatusWhere / expenseStatusWhere translate the computed
// statuses into SQL over the pay lateral and (for charges) a zero grace
// period — see domain.ReceivableStatus / TransactionRow.ExpenseStatus.
// Fully hardcoded fragments, never built from client input.
func chargeStatusWhere(s domain.PaymentStatus) string {
	switch s {
	case domain.PaymentStatusPaid:
		return "pay.paid_cents >= t.amount_cents"
	case domain.PaymentStatusLate:
		return "pay.paid_cents < t.amount_cents AND CURRENT_DATE > t.due_on"
	case domain.PaymentStatusPartial:
		return "pay.paid_cents > 0 AND pay.paid_cents < t.amount_cents AND CURRENT_DATE <= t.due_on"
	default: // unpaid
		return "pay.paid_cents = 0 AND CURRENT_DATE <= t.due_on"
	}
}

func expenseStatusWhere(s domain.ExpenseStatus) string {
	switch s {
	case domain.ExpenseStatusPaid:
		return "pay.paid_cents >= t.amount_cents"
	case domain.ExpenseStatusOverdue:
		return "pay.paid_cents < t.amount_cents AND t.due_on IS NOT NULL AND t.due_on < CURRENT_DATE"
	default: // unpaid
		return "pay.paid_cents < t.amount_cents AND (t.due_on IS NULL OR t.due_on >= CURRENT_DATE)"
	}
}

func (r *LedgerRepository) ListExpenses(ctx context.Context, opts domain.ExpenseListOptions) ([]*domain.TransactionRow, int, error) {
	where := []string{"t.owner_id = $1", "t.kind = 'expense'", "t.voided_at IS NULL"}
	args := []any{opts.OwnerID}

	if opts.PropertyID != nil {
		args = append(args, *opts.PropertyID)
		where = append(where, fmt.Sprintf("t.property_id = $%d", len(args)))
	}
	if opts.Category != nil {
		args = append(args, string(*opts.Category))
		where = append(where, fmt.Sprintf("t.category = $%d", len(args)))
	}
	if opts.VendorID != nil {
		args = append(args, *opts.VendorID)
		where = append(where, fmt.Sprintf("t.vendor_id = $%d", len(args)))
	}
	if opts.Status != nil {
		where = append(where, "("+expenseStatusWhere(*opts.Status)+")")
	}
	if opts.From != nil {
		args = append(args, *opts.From)
		where = append(where, fmt.Sprintf("t.incurred_on >= $%d", len(args)))
	}
	if opts.To != nil {
		args = append(args, *opts.To)
		where = append(where, fmt.Sprintf("t.incurred_on <= $%d", len(args)))
	}
	if opts.Search != "" {
		args = append(args, "%"+opts.Search+"%")
		where = append(where, fmt.Sprintf("(t.description ILIKE $%d OR t.vendor_name ILIKE $%d OR v.company_name ILIKE $%d OR pr.name ILIKE $%d)", len(args), len(args), len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) `+txRowFrom+` WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count expenses: %w", err)
	}

	sortCol := map[domain.ExpenseSortKey]string{
		domain.ExpenseSortIncurredOn: "t.incurred_on",
		domain.ExpenseSortAmount:     "t.amount_cents",
		domain.ExpenseSortProperty:   "pr.name",
		domain.ExpenseSortCategory:   "t.category",
	}[opts.Sort]
	if sortCol == "" {
		sortCol = "t.incurred_on"
	}
	dir := "ASC"
	if opts.SortDesc {
		dir = "DESC"
	}
	args = append(args, opts.Limit, opts.Offset)
	q := fmt.Sprintf(`%s WHERE %s ORDER BY %s %s, t.created_at DESC LIMIT $%d OFFSET $%d`, txRowSelect, whereSQL, sortCol, dir, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	var out []*domain.TransactionRow
	for rows.Next() {
		row, err := scanTransactionRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan expense: %w", err)
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func (r *LedgerRepository) ListCharges(ctx context.Context, opts domain.ChargeListOptions) ([]*domain.TransactionRow, int, error) {
	where := []string{"t.owner_id = $1", "t.type = 'charge'", "t.voided_at IS NULL"}
	args := []any{opts.OwnerID}
	if opts.PropertyID != nil {
		args = append(args, *opts.PropertyID)
		where = append(where, fmt.Sprintf("t.property_id = $%d", len(args)))
	}
	if opts.Status != nil {
		where = append(where, "("+chargeStatusWhere(*opts.Status)+")")
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) `+txRowFrom+` WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count charges: %w", err)
	}

	args = append(args, opts.Limit, opts.Offset)
	q := fmt.Sprintf(`%s WHERE %s ORDER BY t.due_on DESC, t.created_at DESC LIMIT $%d OFFSET $%d`, txRowSelect, whereSQL, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list charges: %w", err)
	}
	defer rows.Close()

	var out []*domain.TransactionRow
	for rows.Next() {
		row, err := scanTransactionRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan charge: %w", err)
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

// rentRollStatusWhere reproduces domain.ReceivableStatus over the
// aggregated rent-roll columns (billed_cents, paid_cents, due_on,
// grace_days).
func rentRollStatusWhere(s domain.PaymentStatus) string {
	switch s {
	case domain.PaymentStatusPaid:
		return "billed_cents - paid_cents <= 0"
	case domain.PaymentStatusLate:
		return "billed_cents - paid_cents > 0 AND CURRENT_DATE > due_on + grace_days"
	case domain.PaymentStatusPartial:
		return "billed_cents - paid_cents > 0 AND paid_cents > 0 AND CURRENT_DATE <= due_on + grace_days"
	default: // unpaid
		return "billed_cents - paid_cents > 0 AND paid_cents = 0 AND CURRENT_DATE <= due_on + grace_days"
	}
}

func (r *LedgerRepository) ListRentRoll(ctx context.Context, opts domain.RentRollOptions) ([]*domain.RentRollRow, int, error) {
	args := []any{opts.OwnerID, opts.PeriodFrom, opts.PeriodTo}
	propertyFilter := ""
	if opts.PropertyID != nil {
		args = append(args, *opts.PropertyID)
		propertyFilter = fmt.Sprintf("WHERE pr.id = $%d", len(args))
	}
	statusFilter := ""
	if opts.Status != nil {
		statusFilter = "WHERE " + rentRollStatusWhere(*opts.Status)
	}

	sortExpr := map[domain.RentRollSortKey]string{
		domain.RentRollSortProperty:    "property_name %s, unit_name %[1]s",
		domain.RentRollSortUnit:        "unit_name %s",
		domain.RentRollSortDueOn:       "due_on %s",
		domain.RentRollSortOutstanding: "(billed_cents - paid_cents) %s",
	}[opts.Sort]
	if sortExpr == "" {
		sortExpr = "property_name %s, unit_name %[1]s"
	}
	dir := "ASC"
	if opts.SortDesc {
		dir = "DESC"
	}
	orderBy := fmt.Sprintf(sortExpr, dir)

	args = append(args, opts.Limit, opts.Offset)
	q := fmt.Sprintf(`
		WITH tx AS (
			SELECT t.lease_id, t.period, t.type, t.amount_cents, t.due_on,
			       pay.paid_cents, pay.last_paid_on
			FROM transactions t`+paymentLateral+`
			WHERE t.owner_id = $1 AND t.kind = 'income' AND t.voided_at IS NULL
			  AND t.period >= $2 AND t.period <= $3
		), agg AS (
			SELECT lease_id, period,
			       COALESCE(SUM(amount_cents) FILTER (WHERE type = 'rent'), 0)::bigint AS rent_cents,
			       COALESCE(MIN(due_on) FILTER (WHERE type = 'rent'), MIN(due_on)) AS due_on,
			       SUM(amount_cents)::bigint AS billed_cents,
			       SUM(paid_cents)::bigint AS paid_cents,
			       MAX(last_paid_on) AS last_paid_on
			FROM tx GROUP BY lease_id, period
		), rr AS (
			SELECT l.id AS lease_id, pr.id AS property_id, u.id AS unit_id, pr.name AS property_name,
			       u.unit_name, l.primary_resident_name AS tenant_name,
			       agg.period, agg.rent_cents, agg.due_on, agg.billed_cents, agg.paid_cents, agg.last_paid_on,
			       COALESCE(l.late_fee_grace_days, s.grace_days, 5) AS grace_days
			FROM agg
			JOIN leases l ON l.id = agg.lease_id
			JOIN units u ON u.id = l.unit_id
			JOIN properties pr ON pr.id = u.property_id
			LEFT JOIN accounting_settings s ON s.owner_id = $1
			%s
		)
		SELECT lease_id, property_id, unit_id, property_name, unit_name, tenant_name,
		       period, rent_cents, due_on, billed_cents, paid_cents, last_paid_on, grace_days,
		       COUNT(*) OVER() AS total
		FROM rr %s
		ORDER BY %s, period DESC, lease_id
		LIMIT $%d OFFSET $%d`, propertyFilter, statusFilter, orderBy, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list rent roll: %w", err)
	}
	defer rows.Close()

	var out []*domain.RentRollRow
	total := 0
	for rows.Next() {
		var row domain.RentRollRow
		var last sql.NullTime
		if err := rows.Scan(&row.LeaseID, &row.PropertyID, &row.UnitID, &row.PropertyName, &row.UnitName, &row.TenantName,
			&row.Period, &row.RentCents, &row.DueOn, &row.BilledCents, &row.PaidCents, &last, &row.GraceDays, &total); err != nil {
			return nil, 0, fmt.Errorf("scan rent roll row: %w", err)
		}
		row.LastPaidOn = nullTimePtr(last)
		out = append(out, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// total comes from COUNT(*) OVER(), so it is 0 for a page past the
	// end of the result set — callers only ever page forward from a
	// non-empty first page, so that edge is harmless.
	return out, total, nil
}

func (r *LedgerRepository) ListLeasesToBill(ctx context.Context, ownerID uuid.UUID, period time.Time) ([]*domain.LeaseBillingInfo, error) {
	first := domain.FirstOfMonth(period)
	last := domain.LastOfMonth(period)
	rows, err := r.pool.Query(ctx, `
		SELECT l.id, p.owner_id, p.id, u.id, ROUND(l.monthly_rent * 100)::bigint, l.rent_due_day
		FROM leases l
		JOIN units u ON u.id = l.unit_id
		JOIN properties p ON p.id = u.property_id
		WHERE p.owner_id = $1 AND l.status = 'active' AND l.monthly_rent > 0
		  AND l.start_date <= $3 AND (l.end_date IS NULL OR l.end_date >= $2)`,
		ownerID, first, last)
	if err != nil {
		return nil, fmt.Errorf("list leases to bill: %w", err)
	}
	defer rows.Close()

	var out []*domain.LeaseBillingInfo
	for rows.Next() {
		var b domain.LeaseBillingInfo
		var dueDay sql.NullInt32
		if err := rows.Scan(&b.LeaseID, &b.OwnerID, &b.PropertyID, &b.UnitID, &b.MonthlyRentCents, &dueDay); err != nil {
			return nil, fmt.Errorf("scan lease billing info: %w", err)
		}
		if dueDay.Valid {
			d := int(dueDay.Int32)
			b.RentDueDay = &d
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *LedgerRepository) ListLateFeeCandidates(ctx context.Context, ownerID uuid.UUID, today time.Time, defaultGraceDays int) ([]*domain.LateFeeCandidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.owner_id, t.lease_id, t.property_id, t.unit_id, t.period, t.amount_cents, t.due_on,
		       COALESCE(l.late_fee_grace_days, $3),
		       CASE WHEN l.late_fee_amount IS NOT NULL THEN ROUND(l.late_fee_amount * 100)::bigint END
		FROM transactions t
		JOIN leases l ON l.id = t.lease_id`+paymentLateral+`
		WHERE t.owner_id = $1 AND t.type = 'rent' AND t.voided_at IS NULL
		  AND pay.paid_cents < t.amount_cents
		  AND $2::date > t.due_on + COALESCE(l.late_fee_grace_days, $3)
		  -- Only this and last month's rent: turning late fees on (or a first
		  -- run after an outage) must not retroactively fee old arrears.
		  AND t.period >= (date_trunc('month', $2::date) - INTERVAL '1 month')::date
		  AND NOT EXISTS (
			SELECT 1 FROM transactions f
			WHERE f.lease_id = t.lease_id AND f.period = t.period AND f.type = 'late_fee'
		  )`,
		ownerID, today, defaultGraceDays)
	if err != nil {
		return nil, fmt.Errorf("list late fee candidates: %w", err)
	}
	defer rows.Close()

	var out []*domain.LateFeeCandidate
	for rows.Next() {
		var c domain.LateFeeCandidate
		var unit uuid.NullUUID
		var leaseFee sql.NullInt64
		if err := rows.Scan(&c.RentTransactionID, &c.OwnerID, &c.LeaseID, &c.PropertyID, &unit, &c.Period, &c.RentCents, &c.DueOn, &c.GraceDays, &leaseFee); err != nil {
			return nil, fmt.Errorf("scan late fee candidate: %w", err)
		}
		if unit.Valid {
			c.UnitID = unit.UUID
		}
		if leaseFee.Valid {
			v := leaseFee.Int64
			c.LeaseLateFeeCents = &v
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *LedgerRepository) ListRecurringTemplates(ctx context.Context, ownerID uuid.UUID) ([]*domain.Transaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+txColumns+` FROM transactions
		WHERE owner_id = $1 AND kind = 'expense' AND is_recurring AND recurrence_parent_id IS NULL AND voided_at IS NULL
		  AND recurrence_frequency IS NOT NULL`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list recurring templates: %w", err)
	}
	defer rows.Close()

	var out []*domain.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recurring template: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *LedgerRepository) ListOwnerIDsWithActiveLeases(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT p.owner_id
		FROM leases l JOIN units u ON u.id = l.unit_id JOIN properties p ON p.id = u.property_id
		WHERE l.status = 'active'`)
	if err != nil {
		return nil, fmt.Errorf("list owner ids with active leases: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan owner id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *LedgerRepository) ListAccountIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT owner_id FROM properties`)
	if err != nil {
		return nil, fmt.Errorf("list account ids: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan account id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *LedgerRepository) GetSettings(ctx context.Context, ownerID uuid.UUID) (*domain.AccountingSettings, error) {
	var s domain.AccountingSettings
	var kind string
	err := r.pool.QueryRow(ctx, `
		SELECT owner_id, late_fee_kind, late_fee_value, grace_days, default_rent_due_day, updated_at
		FROM accounting_settings WHERE owner_id = $1`, ownerID).
		Scan(&s.OwnerID, &kind, &s.LateFeeValue, &s.GraceDays, &s.DefaultRentDueDay, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DefaultAccountingSettings(ownerID), nil
		}
		return nil, fmt.Errorf("get accounting settings %s: %w", ownerID, err)
	}
	s.LateFeeKind = domain.LateFeeKind(kind)
	return &s, nil
}

func (r *LedgerRepository) UpsertSettings(ctx context.Context, s *domain.AccountingSettings, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert settings: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO accounting_settings (owner_id, late_fee_kind, late_fee_value, grace_days, default_rent_due_day, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (owner_id) DO UPDATE SET
			late_fee_kind = EXCLUDED.late_fee_kind, late_fee_value = EXCLUDED.late_fee_value,
			grace_days = EXCLUDED.grace_days, default_rent_due_day = EXCLUDED.default_rent_due_day,
			updated_at = EXCLUDED.updated_at`,
		s.OwnerID, string(s.LateFeeKind), s.LateFeeValue, s.GraceDays, s.DefaultRentDueDay, s.UpdatedAt); err != nil {
		return fmt.Errorf("upsert accounting settings %s: %w", s.OwnerID, err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert settings: %w", err)
	}
	return nil
}

func (r *LedgerRepository) ListAudit(ctx context.Context, ownerID uuid.UUID, entityType string, entityID uuid.UUID) ([]*domain.AuditEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_id, entity_type, entity_id, actor_id, action, changes, created_at
		FROM accounting_audit_log
		WHERE owner_id = $1 AND entity_type = $2 AND entity_id = $3
		ORDER BY created_at, id`, ownerID, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("list audit for %s %s: %w", entityType, entityID, err)
	}
	defer rows.Close()

	var out []*domain.AuditEntry
	for rows.Next() {
		var a domain.AuditEntry
		var raw []byte
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.EntityType, &a.EntityID, &a.ActorID, &a.Action, &raw, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		if err := json.Unmarshal(raw, &a.Changes); err != nil {
			return nil, fmt.Errorf("unmarshal audit changes: %w", err)
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *LedgerRepository) DashboardTotals(ctx context.Context, ownerID uuid.UUID, today time.Time) (*domain.DashboardTotals, error) {
	monthStart, monthEnd := domain.FirstOfMonth(today), domain.LastOfMonth(today)
	var d domain.DashboardTotals

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(pay.amount_cents) FILTER (WHERE t.kind = 'income'), 0)::bigint,
			COALESCE(SUM(pay.amount_cents) FILTER (WHERE t.kind = 'expense'), 0)::bigint
		FROM transaction_payments pay
		JOIN transactions t ON t.id = pay.transaction_id
		WHERE t.owner_id = $1 AND t.voided_at IS NULL AND pay.voided_at IS NULL
		  AND pay.paid_on >= $2 AND pay.paid_on <= $3`, ownerID, monthStart, monthEnd).
		Scan(&d.CollectedCents, &d.ExpensesPaidCents); err != nil {
		return nil, fmt.Errorf("dashboard cash totals: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_cents), 0)::bigint
		FROM transactions
		WHERE owner_id = $1 AND kind = 'income' AND voided_at IS NULL AND due_on >= $2 AND due_on <= $3`,
		ownerID, monthStart, monthEnd).Scan(&d.ExpectedCents); err != nil {
		return nil, fmt.Errorf("dashboard expected: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(t.amount_cents - pay.paid_cents), 0)::bigint
		FROM transactions t`+paymentLateral+`
		WHERE t.owner_id = $1 AND t.kind = 'income' AND t.voided_at IS NULL
		  AND t.due_on <= $2 AND pay.paid_cents < t.amount_cents`, ownerID, today).
		Scan(&d.OutstandingCents); err != nil {
		return nil, fmt.Errorf("dashboard outstanding: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(t.amount_cents - pay.paid_cents), 0)::bigint, COUNT(*)::int,
		       COUNT(*) FILTER (WHERE t.due_on < $2)::int
		FROM transactions t`+paymentLateral+`
		WHERE t.owner_id = $1 AND t.kind = 'expense' AND t.voided_at IS NULL
		  AND t.due_on IS NOT NULL AND t.due_on <= $2::date + 30 AND pay.paid_cents < t.amount_cents`, ownerID, today).
		Scan(&d.UpcomingExpenseCents, &d.UpcomingExpenseCount, &d.OverdueExpenseCount); err != nil {
		return nil, fmt.Errorf("dashboard upcoming expenses: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM transactions t
		JOIN leases l ON l.id = t.lease_id
		LEFT JOIN accounting_settings s ON s.owner_id = t.owner_id`+paymentLateral+`
		WHERE t.owner_id = $1 AND t.type = 'rent' AND t.voided_at IS NULL AND pay.paid_cents < t.amount_cents
		  AND $2::date > t.due_on + COALESCE(l.late_fee_grace_days, s.grace_days, 5)`, ownerID, today).
		Scan(&d.LateRentCount); err != nil {
		return nil, fmt.Errorf("dashboard late rent: %w", err)
	}
	return &d, nil
}

// DashboardSeries buckets received income and paid expenses by
// date_trunc(bucket). bucket is 'week' or 'month' — callers pass a
// constant, never client input.
func (r *LedgerRepository) DashboardSeries(ctx context.Context, ownerID uuid.UUID, from, to time.Time, bucket string) ([]domain.SeriesPoint, error) {
	if bucket != "week" && bucket != "month" {
		return nil, fmt.Errorf("dashboard series: unsupported bucket %q: %w", bucket, domain.ErrInvalidInput)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT date_trunc($4::text, pay.paid_on::timestamp)::date AS bucket,
		       COALESCE(SUM(pay.amount_cents) FILTER (WHERE t.kind = 'income'), 0)::bigint,
		       COALESCE(SUM(pay.amount_cents) FILTER (WHERE t.kind = 'expense'), 0)::bigint
		FROM transaction_payments pay
		JOIN transactions t ON t.id = pay.transaction_id
		WHERE t.owner_id = $1 AND t.voided_at IS NULL AND pay.voided_at IS NULL
		  AND pay.paid_on >= $2 AND pay.paid_on <= $3
		GROUP BY 1 ORDER BY 1`, ownerID, from, to, bucket)
	if err != nil {
		return nil, fmt.Errorf("dashboard series: %w", err)
	}
	defer rows.Close()

	var out []domain.SeriesPoint
	for rows.Next() {
		var p domain.SeriesPoint
		if err := rows.Scan(&p.Bucket, &p.IncomeCents, &p.ExpenseCents); err != nil {
			return nil, fmt.Errorf("scan series point: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanTransaction(row rowScanner) (*domain.Transaction, error) {
	var t domain.Transaction
	var kind, typ, source string
	var chargeType, category, recurrence sql.NullString
	var unitID, leaseID, workOrderID, vendorID, parentID, attachmentID, updatedBy uuid.NullUUID
	var dueOn, period, voidedAt sql.NullTime
	if err := row.Scan(
		&t.ID, &t.OwnerID, &t.PropertyID, &unitID, &kind, &typ, &chargeType, &category, &t.AmountCents,
		&t.IncurredOn, &dueOn, &period, &leaseID, &workOrderID, &vendorID, &t.VendorName, &t.Description,
		&t.TaxDeductible, &t.IsRecurring, &recurrence, &parentID, &attachmentID,
		&source, &voidedAt, &t.CreatedBy, &t.CreatedAt, &updatedBy, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	applyTransactionNullables(&t, kind, typ, source, chargeType, category, recurrence,
		unitID, leaseID, workOrderID, vendorID, parentID, attachmentID, updatedBy, dueOn, period, voidedAt)
	return &t, nil
}

func scanTransactionRow(row rowScanner) (*domain.TransactionRow, error) {
	var r domain.TransactionRow
	t := &r.Transaction
	var kind, typ, source string
	var chargeType, category, recurrence sql.NullString
	var unitID, leaseID, workOrderID, vendorID, parentID, attachmentID, updatedBy uuid.NullUUID
	var dueOn, period, voidedAt, lastPaid sql.NullTime
	if err := row.Scan(
		&t.ID, &t.OwnerID, &t.PropertyID, &unitID, &kind, &typ, &chargeType, &category, &t.AmountCents,
		&t.IncurredOn, &dueOn, &period, &leaseID, &workOrderID, &vendorID, &t.VendorName, &t.Description,
		&t.TaxDeductible, &t.IsRecurring, &recurrence, &parentID, &attachmentID,
		&source, &voidedAt, &t.CreatedBy, &t.CreatedAt, &updatedBy, &t.UpdatedAt,
		&r.PaidCents, &lastPaid, &r.PropertyName, &r.UnitName, &r.TenantName, &r.VendorCompany, &r.WorkOrderTitle,
	); err != nil {
		return nil, err
	}
	applyTransactionNullables(t, kind, typ, source, chargeType, category, recurrence,
		unitID, leaseID, workOrderID, vendorID, parentID, attachmentID, updatedBy, dueOn, period, voidedAt)
	r.LastPaidOn = nullTimePtr(lastPaid)
	return &r, nil
}

func applyTransactionNullables(
	t *domain.Transaction,
	kind, typ, source string,
	chargeType, category, recurrence sql.NullString,
	unitID, leaseID, workOrderID, vendorID, parentID, attachmentID, updatedBy uuid.NullUUID,
	dueOn, period, voidedAt sql.NullTime,
) {
	t.Kind = domain.TransactionKind(kind)
	t.Type = domain.TransactionType(typ)
	t.Source = domain.TransactionSource(source)
	if chargeType.Valid {
		c := domain.ChargeType(chargeType.String)
		t.ChargeType = &c
	}
	if category.Valid {
		c := domain.ExpenseCategory(category.String)
		t.Category = &c
	}
	if recurrence.Valid {
		f := domain.RecurrenceFrequency(recurrence.String)
		t.RecurrenceFrequency = &f
	}
	t.UnitID = nullUUIDPtr(unitID)
	t.LeaseID = nullUUIDPtr(leaseID)
	t.WorkOrderID = nullUUIDPtr(workOrderID)
	t.VendorID = nullUUIDPtr(vendorID)
	t.RecurrenceParentID = nullUUIDPtr(parentID)
	t.AttachmentID = nullUUIDPtr(attachmentID)
	t.UpdatedBy = nullUUIDPtr(updatedBy)
	t.DueOn = nullTimePtr(dueOn)
	t.Period = nullTimePtr(period)
	t.VoidedAt = nullTimePtr(voidedAt)
}
