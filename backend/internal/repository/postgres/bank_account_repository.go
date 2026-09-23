package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type BankAccountRepository struct {
	pool *pgxpool.Pool
}

func NewBankAccountRepository(pool *pgxpool.Pool) *BankAccountRepository {
	return &BankAccountRepository{pool: pool}
}

var _ domain.BankAccountRepository = (*BankAccountRepository)(nil)

const bankAccountColumns = `id, owner_id, nickname, bank_name, account_type, property_id, balance_cents, balance_as_of, last4, provider, COALESCE(external_account_id, ''), created_at, updated_at`

const bankAccountColumnsA = `a.id, a.owner_id, a.nickname, a.bank_name, a.account_type, a.property_id, a.balance_cents, a.balance_as_of, a.last4, a.provider, COALESCE(a.external_account_id, ''), a.created_at, a.updated_at`

func scanBankAccount(row rowScanner) (*domain.BankAccount, error) {
	var a domain.BankAccount
	var typ string
	var property uuid.NullUUID
	var asOf sql.NullTime
	if err := row.Scan(&a.ID, &a.OwnerID, &a.Nickname, &a.BankName, &typ, &property, &a.BalanceCents, &asOf,
		&a.Last4, &a.Provider, &a.ExternalAccountID, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.Type = domain.BankAccountType(typ)
	a.PropertyID = nullUUIDPtr(property)
	a.BalanceAsOf = nullTimePtr(asOf)
	return &a, nil
}

func (r *BankAccountRepository) Create(ctx context.Context, a *domain.BankAccount, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create bank account: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO bank_accounts (id, owner_id, nickname, bank_name, account_type, property_id, balance_cents, balance_as_of, last4, provider, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		a.ID, a.OwnerID, a.Nickname, a.BankName, string(a.Type), a.PropertyID, a.BalanceCents, a.BalanceAsOf, a.Last4, a.Provider, a.CreatedAt, a.UpdatedAt); err != nil {
		return fmt.Errorf("insert bank account %s: %w", a.ID, err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create bank account: %w", err)
	}
	return nil
}

func (r *BankAccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BankAccount, error) {
	a, err := scanBankAccount(r.pool.QueryRow(ctx, `SELECT `+bankAccountColumns+` FROM bank_accounts WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get bank account %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get bank account %s: %w", id, err)
	}
	return a, nil
}

func (r *BankAccountRepository) Update(ctx context.Context, a *domain.BankAccount, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update bank account: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE bank_accounts SET nickname = $2, bank_name = $3, account_type = $4, property_id = $5,
			balance_cents = $6, balance_as_of = $7, last4 = $8, updated_at = $9
		WHERE id = $1`,
		a.ID, a.Nickname, a.BankName, string(a.Type), a.PropertyID, a.BalanceCents, a.BalanceAsOf, a.Last4, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update bank account %s: %w", a.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update bank account %s: %w", a.ID, domain.ErrNotFound)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update bank account: %w", err)
	}
	return nil
}

func (r *BankAccountRepository) Delete(ctx context.Context, id uuid.UUID, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete bank account: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Payments and deposits that named this account keep their rows
	// (ON DELETE SET NULL); statement lines go with the account.
	tag, err := tx.Exec(ctx, `DELETE FROM bank_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete bank account %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete bank account %s: %w", id, domain.ErrNotFound)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete bank account: %w", err)
	}
	return nil
}

func (r *BankAccountRepository) ListForOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.BankAccountRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+bankAccountColumnsA+`, COALESCE(p.name, ''),
		  (SELECT COUNT(*) FROM bank_statement_lines l WHERE l.bank_account_id = a.id AND l.matched_payment_id IS NULL)::int,
		  (SELECT COUNT(*) FROM transaction_payments pay JOIN transactions t ON t.id = pay.transaction_id
		    WHERE pay.voided_at IS NULL AND t.voided_at IS NULL AND pay.bank_account_id = a.id
		      AND NOT EXISTS (SELECT 1 FROM bank_statement_lines m WHERE m.matched_payment_id = pay.id))::int
		FROM bank_accounts a LEFT JOIN properties p ON p.id = a.property_id
		WHERE a.owner_id = $1 ORDER BY a.nickname, a.created_at`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list bank accounts: %w", err)
	}
	defer rows.Close()

	var out []*domain.BankAccountRow
	for rows.Next() {
		var row domain.BankAccountRow
		var typ string
		var property uuid.NullUUID
		var asOf sql.NullTime
		if err := rows.Scan(&row.ID, &row.OwnerID, &row.Nickname, &row.BankName, &typ, &property, &row.BalanceCents, &asOf,
			&row.Last4, &row.Provider, &row.ExternalAccountID, &row.CreatedAt, &row.UpdatedAt,
			&row.PropertyName, &row.UnmatchedLineCount, &row.UnmatchedPaymentCount); err != nil {
			return nil, fmt.Errorf("scan bank account row: %w", err)
		}
		row.Type = domain.BankAccountType(typ)
		row.PropertyID = nullUUIDPtr(property)
		row.BalanceAsOf = nullTimePtr(asOf)
		out = append(out, &row)
	}
	return out, rows.Err()
}

const lineColumns = `id, owner_id, bank_account_id, posted_on, description, amount_cents, source, COALESCE(external_id, ''), matched_payment_id, matched_at, created_at`

func scanLine(row rowScanner) (*domain.StatementLine, error) {
	var l domain.StatementLine
	var source string
	var matched uuid.NullUUID
	var matchedAt sql.NullTime
	if err := row.Scan(&l.ID, &l.OwnerID, &l.BankAccountID, &l.PostedOn, &l.Description, &l.AmountCents, &source, &l.ExternalID, &matched, &matchedAt, &l.CreatedAt); err != nil {
		return nil, err
	}
	l.Source = domain.StatementLineSource(source)
	l.MatchedPaymentID = nullUUIDPtr(matched)
	l.MatchedAt = nullTimePtr(matchedAt)
	return &l, nil
}

func (r *BankAccountRepository) InsertLines(ctx context.Context, lines []*domain.StatementLine) (int, error) {
	if len(lines) == 0 {
		return 0, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin insert statement lines: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	added := 0
	for _, l := range lines {
		var external any
		if l.ExternalID != "" {
			external = l.ExternalID
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO bank_statement_lines (id, owner_id, bank_account_id, posted_on, description, amount_cents, source, external_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT DO NOTHING`,
			l.ID, l.OwnerID, l.BankAccountID, l.PostedOn, l.Description, l.AmountCents, string(l.Source), external, l.CreatedAt)
		if err != nil {
			return 0, fmt.Errorf("insert statement line: %w", err)
		}
		added += int(tag.RowsAffected())
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit insert statement lines: %w", err)
	}
	return added, nil
}

func (r *BankAccountRepository) GetLine(ctx context.Context, id uuid.UUID) (*domain.StatementLine, error) {
	l, err := scanLine(r.pool.QueryRow(ctx, `SELECT `+lineColumns+` FROM bank_statement_lines WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get statement line %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get statement line %s: %w", id, err)
	}
	return l, nil
}

func (r *BankAccountRepository) DeleteLine(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM bank_statement_lines WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete statement line %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete statement line %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

func (r *BankAccountRepository) ListLines(ctx context.Context, opts domain.StatementLineListOptions) ([]*domain.StatementLine, int, error) {
	where := "owner_id = $1 AND bank_account_id = $2"
	switch opts.Filter {
	case domain.StatementLineMatched:
		where += " AND matched_payment_id IS NOT NULL"
	case domain.StatementLineUnmatched:
		where += " AND matched_payment_id IS NULL"
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM bank_statement_lines WHERE `+where, opts.OwnerID, opts.BankAccountID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count statement lines: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT `+lineColumns+` FROM bank_statement_lines WHERE `+where+`
		ORDER BY posted_on DESC, created_at DESC LIMIT $3 OFFSET $4`, opts.OwnerID, opts.BankAccountID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list statement lines: %w", err)
	}
	defer rows.Close()

	var out []*domain.StatementLine
	for rows.Next() {
		l, err := scanLine(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan statement line: %w", err)
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}

func (r *BankAccountRepository) ListUnmatchedPayments(ctx context.Context, ownerID, accountID uuid.UUID) ([]*domain.ReconcilablePayment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pay.id, pay.transaction_id,
		       CASE WHEN t.kind = 'income' THEN pay.amount_cents ELSE -pay.amount_cents END,
		       pay.paid_on, pay.method, pay.reference,
		       CASE WHEN t.kind = 'income'
		            THEN pr.name || COALESCE(' · ' || u.unit_name, '') || COALESCE(' — ' || l.primary_resident_name, '')
		            ELSE pr.name || COALESCE(' · ' || NULLIF(t.description, ''), '') END
		FROM transaction_payments pay
		JOIN transactions t ON t.id = pay.transaction_id
		JOIN properties pr ON pr.id = t.property_id
		LEFT JOIN units u ON u.id = t.unit_id
		LEFT JOIN leases l ON l.id = t.lease_id
		WHERE t.owner_id = $1 AND pay.voided_at IS NULL AND t.voided_at IS NULL
		  AND (pay.bank_account_id = $2 OR pay.bank_account_id IS NULL)
		  AND NOT EXISTS (SELECT 1 FROM bank_statement_lines m WHERE m.matched_payment_id = pay.id)
		ORDER BY pay.paid_on DESC, pay.created_at DESC
		LIMIT 500`, ownerID, accountID)
	if err != nil {
		return nil, fmt.Errorf("list unmatched payments: %w", err)
	}
	defer rows.Close()

	var out []*domain.ReconcilablePayment
	for rows.Next() {
		var p domain.ReconcilablePayment
		if err := rows.Scan(&p.PaymentID, &p.TransactionID, &p.AmountCents, &p.PaidOn, &p.Method, &p.Reference, &p.Description); err != nil {
			return nil, fmt.Errorf("scan unmatched payment: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *BankAccountRepository) Match(ctx context.Context, lineID, paymentID uuid.UUID, at time.Time, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin match: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE bank_statement_lines SET matched_payment_id = $2, matched_at = $3 WHERE id = $1 AND matched_payment_id IS NULL`, lineID, paymentID, at)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("match line %s: payment already matched: %w", lineID, domain.ErrConflict)
		}
		return fmt.Errorf("match line %s: %w", lineID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("match line %s: already matched: %w", lineID, domain.ErrConflict)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE transaction_payments SET bank_account_id = (SELECT bank_account_id FROM bank_statement_lines WHERE id = $1)
		WHERE id = $2 AND bank_account_id IS NULL`, lineID, paymentID); err != nil {
		return fmt.Errorf("assign bank account to payment %s: %w", paymentID, err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit match: %w", err)
	}
	return nil
}

func (r *BankAccountRepository) Unmatch(ctx context.Context, lineID uuid.UUID, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin unmatch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE bank_statement_lines SET matched_payment_id = NULL, matched_at = NULL WHERE id = $1 AND matched_payment_id IS NOT NULL`, lineID)
	if err != nil {
		return fmt.Errorf("unmatch line %s: %w", lineID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("unmatch line %s: %w", lineID, domain.ErrNotFound)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit unmatch: %w", err)
	}
	return nil
}

func (r *BankAccountRepository) PaymentSignedAmount(ctx context.Context, paymentID, ownerID uuid.UUID) (int64, bool, error) {
	var cents int64
	err := r.pool.QueryRow(ctx, `
		SELECT CASE WHEN t.kind = 'income' THEN pay.amount_cents ELSE -pay.amount_cents END
		FROM transaction_payments pay JOIN transactions t ON t.id = pay.transaction_id
		WHERE pay.id = $1 AND t.owner_id = $2 AND pay.voided_at IS NULL AND t.voided_at IS NULL`, paymentID, ownerID).Scan(&cents)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("payment signed amount %s: %w", paymentID, err)
	}
	return cents, true, nil
}

func (r *BankAccountRepository) LineCounts(ctx context.Context, accountID uuid.UUID) (int, int, int64, error) {
	var matched, unmatched int
	var unmatchedCents int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE matched_payment_id IS NOT NULL)::int,
		       COUNT(*) FILTER (WHERE matched_payment_id IS NULL)::int,
		       COALESCE(SUM(amount_cents) FILTER (WHERE matched_payment_id IS NULL), 0)::bigint
		FROM bank_statement_lines WHERE bank_account_id = $1`, accountID).Scan(&matched, &unmatched, &unmatchedCents)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("statement line counts %s: %w", accountID, err)
	}
	return matched, unmatched, unmatchedCents, nil
}
