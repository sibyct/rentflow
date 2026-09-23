package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type DepositRepository struct {
	pool *pgxpool.Pool
}

func NewDepositRepository(pool *pgxpool.Pool) *DepositRepository {
	return &DepositRepository{pool: pool}
}

var _ domain.DepositRepository = (*DepositRepository)(nil)

func (r *DepositRepository) Ensure(ctx context.Context, d *domain.SecurityDeposit) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO security_deposits (id, owner_id, lease_id, property_id, unit_id, amount_cents, collected_on, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (lease_id) DO UPDATE SET amount_cents = EXCLUDED.amount_cents, updated_at = EXCLUDED.updated_at
		WHERE security_deposits.settled_at IS NULL AND security_deposits.forfeited_at IS NULL
		  AND (SELECT COALESCE(SUM(amount_cents), 0) FROM deposit_deductions WHERE deposit_id = security_deposits.id) <= EXCLUDED.amount_cents`,
		d.ID, d.OwnerID, d.LeaseID, d.PropertyID, d.UnitID, d.AmountCents, d.CollectedOn, d.CreatedAt)
	if err != nil {
		return fmt.Errorf("ensure deposit for lease %s: %w", d.LeaseID, err)
	}
	return nil
}

func (r *DepositRepository) EnsureMissingForOwner(ctx context.Context, ownerID uuid.UUID) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO security_deposits (id, owner_id, lease_id, property_id, unit_id, amount_cents, collected_on)
		SELECT gen_random_uuid(), p.owner_id, l.id, p.id, u.id, ROUND(l.security_deposit * 100)::BIGINT, l.start_date
		FROM leases l JOIN units u ON u.id = l.unit_id JOIN properties p ON p.id = u.property_id
		WHERE p.owner_id = $1 AND l.security_deposit IS NOT NULL AND l.security_deposit > 0
		ON CONFLICT (lease_id) DO NOTHING`, ownerID)
	if err != nil {
		return 0, fmt.Errorf("backfill deposits for owner %s: %w", ownerID, err)
	}
	return int(tag.RowsAffected()), nil
}

const depositSelect = `
	SELECT d.id, d.owner_id, d.lease_id, d.property_id, d.unit_id, d.amount_cents, d.collected_on, d.held_in_account_id,
	       d.settled_at, d.refund_cents, d.refund_method, d.refunded_on, d.forfeited_at, d.created_at, d.updated_at,
	       pr.name, u.unit_name, l.primary_resident_name, COALESCE(ba.nickname, ''),
	       COALESCE((SELECT SUM(x.amount_cents) FROM deposit_deductions x WHERE x.deposit_id = d.id), 0)::bigint
	FROM security_deposits d
	JOIN properties pr ON pr.id = d.property_id
	JOIN units u ON u.id = d.unit_id
	JOIN leases l ON l.id = d.lease_id
	LEFT JOIN bank_accounts ba ON ba.id = d.held_in_account_id`

func scanDepositRow(row rowScanner) (*domain.DepositRow, error) {
	var d domain.DepositRow
	var held uuid.NullUUID
	var settled, refunded, forfeited sql.NullTime
	if err := row.Scan(&d.ID, &d.OwnerID, &d.LeaseID, &d.PropertyID, &d.UnitID, &d.AmountCents, &d.CollectedOn, &held,
		&settled, &d.RefundCents, &d.RefundMethod, &refunded, &forfeited, &d.CreatedAt, &d.UpdatedAt,
		&d.PropertyName, &d.UnitName, &d.TenantName, &d.HeldInAccount, &d.DeductionsCents); err != nil {
		return nil, err
	}
	d.HeldInAccountID = nullUUIDPtr(held)
	d.SettledAt = nullTimePtr(settled)
	d.RefundedOn = nullTimePtr(refunded)
	d.ForfeitedAt = nullTimePtr(forfeited)
	return &d, nil
}

func (r *DepositRepository) loadDeductions(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]*domain.DepositDeduction, error) {
	out := map[uuid.UUID][]*domain.DepositDeduction{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, deposit_id, description, amount_cents, attachment_id, created_at
		FROM deposit_deductions WHERE deposit_id = ANY($1) ORDER BY created_at`, ids)
	if err != nil {
		return nil, fmt.Errorf("load deposit deductions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var x domain.DepositDeduction
		var att uuid.NullUUID
		if err := rows.Scan(&x.ID, &x.DepositID, &x.Description, &x.AmountCents, &att, &x.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan deduction: %w", err)
		}
		x.AttachmentID = nullUUIDPtr(att)
		out[x.DepositID] = append(out[x.DepositID], &x)
	}
	return out, rows.Err()
}

func (r *DepositRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.DepositRow, error) {
	d, err := scanDepositRow(r.pool.QueryRow(ctx, depositSelect+` WHERE d.id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get deposit %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get deposit %s: %w", id, err)
	}
	ded, err := r.loadDeductions(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	d.Deductions = ded[id]
	return d, nil
}

// depositStatusWhere mirrors SecurityDeposit.Status in SQL (hardcoded
// fragments, never client input).
func depositStatusWhere(s domain.SecurityDepositStatus) string {
	switch s {
	case domain.SecurityDepositForfeited:
		return "d.forfeited_at IS NOT NULL"
	case domain.SecurityDepositHeld:
		return "d.forfeited_at IS NULL AND d.settled_at IS NULL"
	case domain.SecurityDepositFullyRefunded:
		return "d.forfeited_at IS NULL AND d.settled_at IS NOT NULL AND d.refund_cents >= d.amount_cents"
	default: // partially refunded
		return "d.forfeited_at IS NULL AND d.settled_at IS NOT NULL AND d.refund_cents < d.amount_cents"
	}
}

func (r *DepositRepository) List(ctx context.Context, opts domain.DepositListOptions) ([]*domain.DepositRow, int, error) {
	where := []string{"d.owner_id = $1"}
	args := []any{opts.OwnerID}
	if opts.PropertyID != nil {
		args = append(args, *opts.PropertyID)
		where = append(where, fmt.Sprintf("d.property_id = $%d", len(args)))
	}
	if opts.Status != nil {
		where = append(where, "("+depositStatusWhere(*opts.Status)+")")
	}
	if opts.Search != "" {
		args = append(args, "%"+opts.Search+"%")
		where = append(where, fmt.Sprintf("(l.primary_resident_name ILIKE $%d OR u.unit_name ILIKE $%d OR pr.name ILIKE $%d)", len(args), len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM security_deposits d JOIN properties pr ON pr.id = d.property_id JOIN units u ON u.id = d.unit_id JOIN leases l ON l.id = d.lease_id WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count deposits: %w", err)
	}

	sortCol := map[domain.DepositSortKey]string{
		domain.DepositSortCollectedOn: "d.collected_on",
		domain.DepositSortAmount:      "d.amount_cents",
		domain.DepositSortProperty:    "pr.name",
	}[opts.Sort]
	if sortCol == "" {
		sortCol = "d.collected_on"
	}
	dir := "ASC"
	if opts.SortDesc {
		dir = "DESC"
	}
	args = append(args, opts.Limit, opts.Offset)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`%s WHERE %s ORDER BY %s %s, d.created_at DESC LIMIT $%d OFFSET $%d`,
		depositSelect, whereSQL, sortCol, dir, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list deposits: %w", err)
	}
	defer rows.Close()

	var out []*domain.DepositRow
	var ids []uuid.UUID
	for rows.Next() {
		d, err := scanDepositRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan deposit: %w", err)
		}
		out = append(out, d)
		ids = append(ids, d.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	ded, err := r.loadDeductions(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for _, d := range out {
		d.Deductions = ded[d.ID]
	}
	return out, total, nil
}

func (r *DepositRepository) SetHeldInAccount(ctx context.Context, id uuid.UUID, accountID *uuid.UUID, at time.Time, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin set deposit account: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE security_deposits SET held_in_account_id = $2, updated_at = $3 WHERE id = $1`, id, accountID, at)
	if err != nil {
		return fmt.Errorf("set deposit %s account: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set deposit %s account: %w", id, domain.ErrNotFound)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// lockDeposit selects the deposit row FOR UPDATE and returns what the
// invariant checks need.
func lockDeposit(ctx context.Context, tx pgx.Tx, id uuid.UUID) (amount, refund, deductions int64, open bool, err error) {
	var settled, forfeited sql.NullTime
	err = tx.QueryRow(ctx, `SELECT amount_cents, refund_cents, settled_at, forfeited_at FROM security_deposits WHERE id = $1 FOR UPDATE`, id).
		Scan(&amount, &refund, &settled, &forfeited)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, 0, false, fmt.Errorf("lock deposit %s: %w", id, domain.ErrNotFound)
		}
		return 0, 0, 0, false, fmt.Errorf("lock deposit %s: %w", id, err)
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount_cents), 0)::bigint FROM deposit_deductions WHERE deposit_id = $1`, id).Scan(&deductions); err != nil {
		return 0, 0, 0, false, fmt.Errorf("sum deductions %s: %w", id, err)
	}
	return amount, refund, deductions, !settled.Valid && !forfeited.Valid, nil
}

func (r *DepositRepository) AddDeduction(ctx context.Context, d *domain.DepositDeduction, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add deduction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	amount, _, deductions, open, err := lockDeposit(ctx, tx, d.DepositID)
	if err != nil {
		return err
	}
	if !open {
		return fmt.Errorf("add deduction to %s: deposit already settled: %w", d.DepositID, domain.ErrConflict)
	}
	if deductions+d.AmountCents > amount {
		return fmt.Errorf("add deduction to %s: %w", d.DepositID, domain.ValidationErrors{{Field: "amount_cents", Message: "deductions would exceed the deposit amount"}})
	}
	if _, err := tx.Exec(ctx, `INSERT INTO deposit_deductions (id, deposit_id, description, amount_cents, attachment_id, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		d.ID, d.DepositID, d.Description, d.AmountCents, d.AttachmentID, d.CreatedAt); err != nil {
		return fmt.Errorf("insert deduction: %w", err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *DepositRepository) RemoveDeduction(ctx context.Context, depositID, deductionID uuid.UUID, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin remove deduction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, _, _, open, err := lockDeposit(ctx, tx, depositID)
	if err != nil {
		return err
	}
	if !open {
		return fmt.Errorf("remove deduction from %s: deposit already settled: %w", depositID, domain.ErrConflict)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM deposit_deductions WHERE id = $1 AND deposit_id = $2`, deductionID, depositID)
	if err != nil {
		return fmt.Errorf("delete deduction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("remove deduction %s: %w", deductionID, domain.ErrNotFound)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *DepositRepository) Settle(ctx context.Context, id uuid.UUID, refundCents int64, method string, refundedOn, at time.Time, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin settle deposit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	amount, _, deductions, open, err := lockDeposit(ctx, tx, id)
	if err != nil {
		return err
	}
	if !open {
		return fmt.Errorf("settle deposit %s: already settled: %w", id, domain.ErrConflict)
	}
	if refundCents+deductions > amount {
		return fmt.Errorf("settle deposit %s: %w", id, domain.ValidationErrors{{Field: "refund_cents", Message: "refund plus deductions cannot exceed the deposit amount"}})
	}
	if _, err := tx.Exec(ctx, `UPDATE security_deposits SET settled_at = $2, refund_cents = $3, refund_method = $4, refunded_on = $5, updated_at = $2 WHERE id = $1`,
		id, at, refundCents, method, refundedOn); err != nil {
		return fmt.Errorf("settle deposit %s: %w", id, err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *DepositRepository) Forfeit(ctx context.Context, id uuid.UUID, at time.Time, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin forfeit deposit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, _, _, open, err := lockDeposit(ctx, tx, id)
	if err != nil {
		return err
	}
	if !open {
		return fmt.Errorf("forfeit deposit %s: already settled: %w", id, domain.ErrConflict)
	}
	if _, err := tx.Exec(ctx, `UPDATE security_deposits SET forfeited_at = $2, updated_at = $2 WHERE id = $1`, id, at); err != nil {
		return fmt.Errorf("forfeit deposit %s: %w", id, err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
