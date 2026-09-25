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

type UnitDocumentRepository struct {
	pool *pgxpool.Pool
}

func NewUnitDocumentRepository(pool *pgxpool.Pool) *UnitDocumentRepository {
	return &UnitDocumentRepository{pool: pool}
}

var _ domain.UnitDocumentRepository = (*UnitDocumentRepository)(nil)

func (r *UnitDocumentRepository) Create(ctx context.Context, d *domain.UnitDocument) error {
	const q = `
		INSERT INTO unit_documents (id, unit_id, attachment_id, category, uploaded_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	if _, err := r.pool.Exec(ctx, q, d.ID, d.UnitID, d.AttachmentID, d.Category, d.UploadedBy, d.CreatedAt); err != nil {
		return fmt.Errorf("create unit document %s: %w", d.ID, err)
	}
	return nil
}

const unitDocumentSelect = `
	SELECT ud.id, ud.unit_id, ud.attachment_id, ud.category, ud.uploaded_by, ud.created_at,
	       COALESCE(NULLIF(u.name, ''), u.email, '') AS uploaded_by_name,
	       a.filename, a.content_type, a.size_bytes
	FROM unit_documents ud
	JOIN attachments a ON a.id = ud.attachment_id
	JOIN users u ON u.id = ud.uploaded_by`

func scanUnitDocument(row rowScanner) (*domain.UnitDocument, error) {
	var d domain.UnitDocument
	if err := row.Scan(
		&d.ID, &d.UnitID, &d.AttachmentID, &d.Category, &d.UploadedBy, &d.CreatedAt,
		&d.UploadedByName, &d.Filename, &d.ContentType, &d.SizeBytes,
	); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *UnitDocumentRepository) ListByUnit(ctx context.Context, unitID uuid.UUID) ([]*domain.UnitDocument, error) {
	rows, err := r.pool.Query(ctx, unitDocumentSelect+` WHERE ud.unit_id = $1 ORDER BY ud.created_at DESC`, unitID)
	if err != nil {
		return nil, fmt.Errorf("list documents for unit %s: %w", unitID, err)
	}
	defer rows.Close()

	docs := make([]*domain.UnitDocument, 0)
	for rows.Next() {
		d, err := scanUnitDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan unit document row: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unit document rows: %w", err)
	}
	return docs, nil
}

func (r *UnitDocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.UnitDocument, error) {
	d, err := scanUnitDocument(r.pool.QueryRow(ctx, unitDocumentSelect+` WHERE ud.id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get unit document %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get unit document %s: %w", id, err)
	}
	return d, nil
}

func (r *UnitDocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM unit_documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete unit document %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete unit document %s: %w", id, domain.ErrNotFound)
	}
	return nil
}
