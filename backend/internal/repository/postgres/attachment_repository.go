package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type AttachmentRepository struct {
	pool *pgxpool.Pool
}

func NewAttachmentRepository(pool *pgxpool.Pool) *AttachmentRepository {
	return &AttachmentRepository{pool: pool}
}

var _ domain.AttachmentRepository = (*AttachmentRepository)(nil)

func (r *AttachmentRepository) Create(ctx context.Context, a *domain.Attachment) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO attachments (id, owner_id, object_key, filename, content_type, size_bytes, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		a.ID, a.OwnerID, a.ObjectKey, a.Filename, a.ContentType, a.SizeBytes, string(a.Status), a.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert attachment %s: %w", a.ID, err)
	}
	return nil
}

func (r *AttachmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	var a domain.Attachment
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, object_key, filename, content_type, size_bytes, status, created_at
		FROM attachments WHERE id = $1`, id).
		Scan(&a.ID, &a.OwnerID, &a.ObjectKey, &a.Filename, &a.ContentType, &a.SizeBytes, &status, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get attachment %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get attachment %s: %w", id, err)
	}
	a.Status = domain.AttachmentStatus(status)
	return &a, nil
}

func (r *AttachmentRepository) MarkReady(ctx context.Context, id uuid.UUID, sizeBytes int64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE attachments SET status = 'ready', size_bytes = $2 WHERE id = $1`, id, sizeBytes)
	if err != nil {
		return fmt.Errorf("mark attachment %s ready: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mark attachment %s ready: %w", id, domain.ErrNotFound)
	}
	return nil
}
