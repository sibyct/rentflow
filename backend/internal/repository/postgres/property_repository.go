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

type PropertyRepository struct {
	pool *pgxpool.Pool
}

func NewPropertyRepository(pool *pgxpool.Pool) *PropertyRepository {
	return &PropertyRepository{pool: pool}
}

var _ domain.PropertyRepository = (*PropertyRepository)(nil)

func (r *PropertyRepository) Create(ctx context.Context, p *domain.Property) error {
	const q = `
		INSERT INTO properties (id, address, unit_count, status, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, q, p.ID, p.Address, p.UnitCount, p.Status, p.OwnerID, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert property %s: %w", p.ID, err)
	}
	return nil
}

func (r *PropertyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	const q = `
		SELECT id, address, unit_count, status, owner_id, created_at, updated_at
		FROM properties
		WHERE id = $1`

	var p domain.Property
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.Address, &p.UnitCount, &p.Status, &p.OwnerID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get property %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get property %s: %w", id, err)
	}
	return &p, nil
}

func (r *PropertyRepository) List(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Property, int, error) {
	const q = `
		SELECT id, address, unit_count, status, owner_id, created_at, updated_at
		FROM properties
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, q, ownerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list properties: %w", err)
	}
	defer rows.Close()

	properties := make([]*domain.Property, 0)
	for rows.Next() {
		var p domain.Property
		if err := rows.Scan(&p.ID, &p.Address, &p.UnitCount, &p.Status, &p.OwnerID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan property row: %w", err)
		}
		properties = append(properties, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate property rows: %w", err)
	}

	const countQ = `SELECT COUNT(*) FROM properties WHERE owner_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQ, ownerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count properties: %w", err)
	}

	return properties, total, nil
}

func (r *PropertyRepository) Update(ctx context.Context, p *domain.Property) error {
	const q = `
		UPDATE properties
		SET address = $2, unit_count = $3, status = $4, updated_at = $5
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, p.ID, p.Address, p.UnitCount, p.Status, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update property %s: %w", p.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update property %s: %w", p.ID, domain.ErrNotFound)
	}
	return nil
}

func (r *PropertyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM properties WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete property %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete property %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

func (r *PropertyRepository) ExistsByOwnerAddress(ctx context.Context, ownerID uuid.UUID, address string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM properties WHERE owner_id = $1 AND address = $2)`

	var exists bool
	if err := r.pool.QueryRow(ctx, q, ownerID, address).Scan(&exists); err != nil {
		return false, fmt.Errorf("check existing property for owner %s at %q: %w", ownerID, address, err)
	}
	return exists, nil
}
