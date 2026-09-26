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

// LeaseDocumentRepository mirrors UnitDocumentRepository's shape
// exactly — lease-level documents live in their own table (R14), never
// duplicated into unit_documents.
type LeaseDocumentRepository struct {
	pool *pgxpool.Pool
}

func NewLeaseDocumentRepository(pool *pgxpool.Pool) *LeaseDocumentRepository {
	return &LeaseDocumentRepository{pool: pool}
}

var _ domain.LeaseDocumentRepository = (*LeaseDocumentRepository)(nil)

func (r *LeaseDocumentRepository) Create(ctx context.Context, d *domain.LeaseDocument) error {
	const q = `
		INSERT INTO lease_documents (id, lease_id, attachment_id, category, uploaded_by, is_automated, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err := r.pool.Exec(ctx, q, d.ID, d.LeaseID, d.AttachmentID, d.Category, d.UploadedBy, d.IsAutomated, d.CreatedAt); err != nil {
		return fmt.Errorf("create lease document %s: %w", d.ID, err)
	}
	return nil
}

const leaseDocumentSelect = `
	SELECT ld.id, ld.lease_id, ld.attachment_id, ld.category, ld.uploaded_by, ld.is_automated, ld.created_at,
	       COALESCE(NULLIF(u.name, ''), u.email, '') AS uploaded_by_name,
	       a.filename, a.content_type, a.size_bytes
	FROM lease_documents ld
	JOIN attachments a ON a.id = ld.attachment_id
	JOIN users u ON u.id = ld.uploaded_by`

func scanLeaseDocument(row rowScanner) (*domain.LeaseDocument, error) {
	var d domain.LeaseDocument
	if err := row.Scan(
		&d.ID, &d.LeaseID, &d.AttachmentID, &d.Category, &d.UploadedBy, &d.IsAutomated, &d.CreatedAt,
		&d.UploadedByName, &d.Filename, &d.ContentType, &d.SizeBytes,
	); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *LeaseDocumentRepository) ListByLease(ctx context.Context, leaseID uuid.UUID) ([]*domain.LeaseDocument, error) {
	rows, err := r.pool.Query(ctx, leaseDocumentSelect+` WHERE ld.lease_id = $1 ORDER BY ld.created_at DESC`, leaseID)
	if err != nil {
		return nil, fmt.Errorf("list documents for lease %s: %w", leaseID, err)
	}
	defer rows.Close()

	docs := make([]*domain.LeaseDocument, 0)
	for rows.Next() {
		d, err := scanLeaseDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan lease document row: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lease document rows: %w", err)
	}
	return docs, nil
}

func (r *LeaseDocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.LeaseDocument, error) {
	d, err := scanLeaseDocument(r.pool.QueryRow(ctx, leaseDocumentSelect+` WHERE ld.id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get lease document %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get lease document %s: %w", id, err)
	}
	return d, nil
}

func (r *LeaseDocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM lease_documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete lease document %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete lease document %s: %w", id, domain.ErrNotFound)
	}
	return nil
}
