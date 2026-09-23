package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type EmailOutboxRepository struct {
	pool *pgxpool.Pool
}

func NewEmailOutboxRepository(pool *pgxpool.Pool) *EmailOutboxRepository {
	return &EmailOutboxRepository{pool: pool}
}

var _ domain.EmailOutboxRepository = (*EmailOutboxRepository)(nil)

func (r *EmailOutboxRepository) Enqueue(ctx context.Context, e *domain.OutboxEmail) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO email_outbox (id, owner_id, to_email, subject, body_html, statement_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.ID, e.OwnerID, e.To, e.Subject, e.HTML, e.StatementID, e.CreatedAt)
	if err != nil {
		return fmt.Errorf("enqueue email %s: %w", e.ID, err)
	}
	return nil
}

func (r *EmailOutboxRepository) ClaimPending(ctx context.Context, limit int) ([]*domain.OutboxEmail, error) {
	rows, err := r.pool.Query(ctx, `
		WITH c AS (
			SELECT id FROM email_outbox WHERE status = 'pending' ORDER BY created_at LIMIT $1 FOR UPDATE SKIP LOCKED
		)
		UPDATE email_outbox e SET attempts = e.attempts + 1 FROM c WHERE e.id = c.id
		RETURNING e.id, e.owner_id, e.to_email, e.subject, e.body_html, e.statement_id, e.attempts, e.created_at`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim pending emails: %w", err)
	}
	defer rows.Close()

	var out []*domain.OutboxEmail
	for rows.Next() {
		var e domain.OutboxEmail
		var statement uuid.NullUUID
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.To, &e.Subject, &e.HTML, &statement, &e.Attempts, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox email: %w", err)
		}
		e.StatementID = nullUUIDPtr(statement)
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *EmailOutboxRepository) MarkSent(ctx context.Context, id uuid.UUID, at time.Time) error {
	if _, err := r.pool.Exec(ctx, `UPDATE email_outbox SET status = 'sent', sent_at = $2, last_error = '' WHERE id = $1`, id, at); err != nil {
		return fmt.Errorf("mark email %s sent: %w", id, err)
	}
	return nil
}

func (r *EmailOutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, reason string, final bool) error {
	if len(reason) > 500 {
		reason = reason[:500]
	}
	status := "pending"
	if final {
		status = "failed"
	}
	if _, err := r.pool.Exec(ctx, `UPDATE email_outbox SET status = $2, last_error = $3 WHERE id = $1`, id, status, reason); err != nil {
		return fmt.Errorf("mark email %s failed: %w", id, err)
	}
	return nil
}

type WorkerRunRepository struct {
	pool *pgxpool.Pool
}

func NewWorkerRunRepository(pool *pgxpool.Pool) *WorkerRunRepository {
	return &WorkerRunRepository{pool: pool}
}

var _ domain.WorkerRunRepository = (*WorkerRunRepository)(nil)

func (r *WorkerRunRepository) Claim(ctx context.Context, job, runKey string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `INSERT INTO worker_runs (job, run_key) VALUES ($1, $2) ON CONFLICT DO NOTHING`, job, runKey)
	if err != nil {
		return false, fmt.Errorf("claim worker run %s/%s: %w", job, runKey, err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *WorkerRunRepository) Release(ctx context.Context, job, runKey string) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM worker_runs WHERE job = $1 AND run_key = $2`, job, runKey); err != nil {
		return fmt.Errorf("release worker run %s/%s: %w", job, runKey, err)
	}
	return nil
}
