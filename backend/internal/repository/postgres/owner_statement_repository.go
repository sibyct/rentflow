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
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type OwnerStatementRepository struct {
	pool *pgxpool.Pool
}

func NewOwnerStatementRepository(pool *pgxpool.Pool) *OwnerStatementRepository {
	return &OwnerStatementRepository{pool: pool}
}

var _ domain.OwnerStatementRepository = (*OwnerStatementRepository)(nil)

func (r *OwnerStatementRepository) CreateOrReplaceDraft(ctx context.Context, s *domain.OwnerStatement, audit domain.AuditEntry) error {
	snapshot, err := json.Marshal(s.Snapshot)
	if err != nil {
		return fmt.Errorf("marshal statement snapshot: %w", err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create statement: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var existingID uuid.UUID
	var existingStatus string
	err = tx.QueryRow(ctx, `SELECT id, status FROM owner_statements WHERE property_id = $1 AND period_start = $2 AND period_end = $3 FOR UPDATE`,
		s.PropertyID, s.PeriodStart, s.PeriodEnd).Scan(&existingID, &existingStatus)
	switch {
	case err == nil:
		if existingStatus != string(domain.StatementStatusDraft) {
			return fmt.Errorf("statement for this period is already %s and cannot be regenerated: %w", existingStatus, domain.ErrConflict)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM owner_statements WHERE id = $1`, existingID); err != nil {
			return fmt.Errorf("replace draft statement %s: %w", existingID, err)
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return fmt.Errorf("check existing statement: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO owner_statements (id, owner_id, property_owner_id, property_id, period_start, period_end, status, snapshot,
			income_cents, expenses_cents, fee_cents, net_payout_cents, generated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		s.ID, s.OwnerID, s.PropertyOwnerID, s.PropertyID, s.PeriodStart, s.PeriodEnd, string(s.Status), snapshot,
		s.Snapshot.IncomeCents, s.Snapshot.ExpensesCents, s.Snapshot.FeeCents, s.Snapshot.NetPayoutCents, s.GeneratedAt); err != nil {
		return fmt.Errorf("insert statement %s: %w", s.ID, err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const statementSelect = `
	SELECT s.id, s.owner_id, s.property_owner_id, s.property_id, s.period_start, s.period_end, s.status, s.snapshot,
	       s.generated_at, s.sent_at, s.paid_cents, s.paid_on, s.pay_method, s.pay_reference,
	       pr.name, COALESCE(s.snapshot->>'owner_name', ''),
	       EXISTS (SELECT 1 FROM email_outbox e WHERE e.statement_id = s.id AND e.status = 'pending')
	FROM owner_statements s JOIN properties pr ON pr.id = s.property_id`

func scanStatementRow(row rowScanner) (*domain.OwnerStatementRow, error) {
	var out domain.OwnerStatementRow
	var status string
	var owner uuid.NullUUID
	var raw []byte
	var sent, paidOn sql.NullTime
	var paidCents sql.NullInt64
	if err := row.Scan(&out.ID, &out.OwnerID, &owner, &out.PropertyID, &out.PeriodStart, &out.PeriodEnd, &status, &raw,
		&out.GeneratedAt, &sent, &paidCents, &paidOn, &out.PayMethod, &out.PayReference,
		&out.PropertyName, &out.OwnerName, &out.EmailQueued); err != nil {
		return nil, err
	}
	out.Status = domain.StatementStatus(status)
	out.PropertyOwnerID = nullUUIDPtr(owner)
	out.SentAt = nullTimePtr(sent)
	out.PaidOn = nullTimePtr(paidOn)
	if paidCents.Valid {
		v := paidCents.Int64
		out.PaidCents = &v
	}
	if err := json.Unmarshal(raw, &out.Snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal statement snapshot: %w", err)
	}
	return &out, nil
}

func (r *OwnerStatementRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.OwnerStatementRow, error) {
	row, err := scanStatementRow(r.pool.QueryRow(ctx, statementSelect+` WHERE s.id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get statement %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get statement %s: %w", id, err)
	}
	return row, nil
}

func (r *OwnerStatementRepository) Exists(ctx context.Context, propertyID uuid.UUID, start, end time.Time) (bool, error) {
	var ok bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM owner_statements WHERE property_id = $1 AND period_start = $2 AND period_end = $3)`,
		propertyID, start, end).Scan(&ok); err != nil {
		return false, fmt.Errorf("check statement exists: %w", err)
	}
	return ok, nil
}

func (r *OwnerStatementRepository) List(ctx context.Context, opts domain.OwnerStatementListOptions) ([]*domain.OwnerStatementRow, int, error) {
	where := []string{"s.owner_id = $1"}
	args := []any{opts.OwnerID}
	if opts.PropertyID != nil {
		args = append(args, *opts.PropertyID)
		where = append(where, fmt.Sprintf("s.property_id = $%d", len(args)))
	}
	if opts.Status != nil {
		args = append(args, string(*opts.Status))
		where = append(where, fmt.Sprintf("s.status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM owner_statements s WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count statements: %w", err)
	}
	args = append(args, opts.Limit, opts.Offset)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`%s WHERE %s ORDER BY s.period_start DESC, s.generated_at DESC LIMIT $%d OFFSET $%d`,
		statementSelect, whereSQL, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list statements: %w", err)
	}
	defer rows.Close()

	var out []*domain.OwnerStatementRow
	for rows.Next() {
		row, err := scanStatementRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan statement: %w", err)
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func (r *OwnerStatementRepository) MarkSent(ctx context.Context, id uuid.UUID, at time.Time, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin mark statement sent: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE owner_statements SET status = 'sent', sent_at = $2 WHERE id = $1 AND status = 'draft'`, id, at)
	if err != nil {
		return fmt.Errorf("mark statement %s sent: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		// Already sent or paid (a re-delivery, or a race): nothing to do,
		// unless the statement doesn't exist at all.
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM owner_statements WHERE id = $1)`, id).Scan(&exists); err != nil {
			return fmt.Errorf("check statement %s: %w", id, err)
		}
		if !exists {
			return fmt.Errorf("mark statement %s sent: %w", id, domain.ErrNotFound)
		}
		return nil
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *OwnerStatementRepository) MarkPaid(ctx context.Context, id uuid.UUID, in domain.MarkStatementPaidInput, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin mark statement paid: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE owner_statements SET status = 'paid', paid_cents = $2, paid_on = $3, pay_method = $4, pay_reference = $5
		WHERE id = $1 AND status IN ('draft', 'sent')`, id, in.AmountCents, in.PaidOn, in.Method, in.Reference)
	if err != nil {
		return fmt.Errorf("mark statement %s paid: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mark statement %s paid: already paid: %w", id, domain.ErrConflict)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *OwnerStatementRepository) CollectPeriod(ctx context.Context, ownerID, propertyID uuid.UUID, start, end time.Time) ([]domain.StatementLineItem, []domain.StatementLineItem, error) {
	collect := func(query string) ([]domain.StatementLineItem, error) {
		rows, err := r.pool.Query(ctx, query, ownerID, propertyID, start, end)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []domain.StatementLineItem{}
		for rows.Next() {
			var item domain.StatementLineItem
			var on time.Time
			if err := rows.Scan(&on, &item.Description, &item.AmountCents); err != nil {
				return nil, err
			}
			item.Date = on.Format("2006-01-02")
			items = append(items, item)
		}
		return items, rows.Err()
	}

	income, err := collect(`
		SELECT pay.paid_on,
		       COALESCE(NULLIF(t.description, ''), t.type) || COALESCE(' — ' || u.unit_name, '') || COALESCE(' (' || l.primary_resident_name || ')', ''),
		       pay.amount_cents
		FROM transaction_payments pay
		JOIN transactions t ON t.id = pay.transaction_id
		LEFT JOIN units u ON u.id = t.unit_id
		LEFT JOIN leases l ON l.id = t.lease_id
		WHERE t.owner_id = $1 AND t.property_id = $2 AND t.kind = 'income' AND t.voided_at IS NULL AND pay.voided_at IS NULL
		  AND pay.paid_on >= $3 AND pay.paid_on <= $4
		ORDER BY pay.paid_on, t.created_at, pay.created_at`)
	if err != nil {
		return nil, nil, fmt.Errorf("collect statement income: %w", err)
	}
	expenses, err := collect(`
		SELECT pay.paid_on,
		       COALESCE(NULLIF(t.description, ''), REPLACE(COALESCE(t.category, 'expense'), '_', ' ')) ||
		       COALESCE(' — ' || NULLIF(COALESCE(v.company_name, t.vendor_name), ''), ''),
		       pay.amount_cents
		FROM transaction_payments pay
		JOIN transactions t ON t.id = pay.transaction_id
		LEFT JOIN vendors v ON v.id = t.vendor_id
		WHERE t.owner_id = $1 AND t.property_id = $2 AND t.kind = 'expense' AND t.voided_at IS NULL AND pay.voided_at IS NULL
		  AND pay.paid_on >= $3 AND pay.paid_on <= $4
		ORDER BY pay.paid_on, t.created_at, pay.created_at`)
	if err != nil {
		return nil, nil, fmt.Errorf("collect statement expenses: %w", err)
	}
	return income, expenses, nil
}
