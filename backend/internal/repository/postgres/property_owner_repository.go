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

type PropertyOwnerRepository struct {
	pool *pgxpool.Pool
}

func NewPropertyOwnerRepository(pool *pgxpool.Pool) *PropertyOwnerRepository {
	return &PropertyOwnerRepository{pool: pool}
}

var _ domain.PropertyOwnerRepository = (*PropertyOwnerRepository)(nil)

const propertyOwnerColumns = `po.id, po.owner_id, po.name, po.email, po.phone, po.management_fee_bps, po.auto_statements, po.created_at, po.updated_at`

func scanPropertyOwner(row rowScanner) (*domain.PropertyOwner, error) {
	var o domain.PropertyOwner
	if err := row.Scan(&o.ID, &o.OwnerID, &o.Name, &o.Email, &o.Phone, &o.ManagementFeeBps, &o.AutoStatements, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, err
	}
	return &o, nil
}

// linkProperties makes propertyIDs — and only those — point at owner id,
// restricted to properties the account owns. Inside the caller's tx.
func linkProperties(ctx context.Context, tx pgx.Tx, accountID, id uuid.UUID, propertyIDs []uuid.UUID) error {
	if _, err := tx.Exec(ctx, `UPDATE properties SET property_owner_id = NULL WHERE property_owner_id = $1 AND NOT (id = ANY($2))`, id, propertyIDs); err != nil {
		return fmt.Errorf("unlink properties from owner %s: %w", id, err)
	}
	if len(propertyIDs) == 0 {
		return nil
	}
	tag, err := tx.Exec(ctx, `UPDATE properties SET property_owner_id = $1 WHERE owner_id = $2 AND id = ANY($3)`, id, accountID, propertyIDs)
	if err != nil {
		return fmt.Errorf("link properties to owner %s: %w", id, err)
	}
	if int(tag.RowsAffected()) != len(propertyIDs) {
		return fmt.Errorf("link properties to owner %s: %w", id, domain.ValidationErrors{{Field: "property_ids", Message: "contains an unknown property"}})
	}
	return nil
}

func (r *PropertyOwnerRepository) Create(ctx context.Context, o *domain.PropertyOwner, propertyIDs []uuid.UUID, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create property owner: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO property_owners (id, owner_id, name, email, phone, management_fee_bps, auto_statements, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		o.ID, o.OwnerID, o.Name, o.Email, o.Phone, o.ManagementFeeBps, o.AutoStatements, o.CreatedAt, o.UpdatedAt); err != nil {
		return fmt.Errorf("insert property owner %s: %w", o.ID, err)
	}
	if err := linkProperties(ctx, tx, o.OwnerID, o.ID, propertyIDs); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PropertyOwnerRepository) Update(ctx context.Context, o *domain.PropertyOwner, propertyIDs []uuid.UUID, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update property owner: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE property_owners SET name = $2, email = $3, phone = $4, management_fee_bps = $5, auto_statements = $6, updated_at = $7
		WHERE id = $1`, o.ID, o.Name, o.Email, o.Phone, o.ManagementFeeBps, o.AutoStatements, o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update property owner %s: %w", o.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update property owner %s: %w", o.ID, domain.ErrNotFound)
	}
	if err := linkProperties(ctx, tx, o.OwnerID, o.ID, propertyIDs); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PropertyOwnerRepository) Delete(ctx context.Context, id uuid.UUID, audit domain.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete property owner: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// properties.property_owner_id and owner_statements.property_owner_id
	// are ON DELETE SET NULL: statements keep their frozen snapshot.
	tag, err := tx.Exec(ctx, `DELETE FROM property_owners WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete property owner %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete property owner %s: %w", id, domain.ErrNotFound)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const propertyOwnerRowSelect = `
	SELECT ` + propertyOwnerColumns + `,
	       ARRAY(SELECT p.id::text FROM properties p WHERE p.property_owner_id = po.id ORDER BY p.name),
	       ARRAY(SELECT p.name FROM properties p WHERE p.property_owner_id = po.id ORDER BY p.name)
	FROM property_owners po`

func scanPropertyOwnerRow(row rowScanner) (*domain.PropertyOwnerRow, error) {
	var out domain.PropertyOwnerRow
	var ids, names []string
	if err := row.Scan(&out.ID, &out.OwnerID, &out.Name, &out.Email, &out.Phone, &out.ManagementFeeBps, &out.AutoStatements,
		&out.CreatedAt, &out.UpdatedAt, &ids, &names); err != nil {
		return nil, err
	}
	for _, s := range ids {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("parse property id %q: %w", s, err)
		}
		out.PropertyIDs = append(out.PropertyIDs, id)
	}
	out.PropertyNames = names
	return &out, nil
}

func (r *PropertyOwnerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PropertyOwnerRow, error) {
	row, err := scanPropertyOwnerRow(r.pool.QueryRow(ctx, propertyOwnerRowSelect+` WHERE po.id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get property owner %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get property owner %s: %w", id, err)
	}
	return row, nil
}

func (r *PropertyOwnerRepository) ListForOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.PropertyOwnerRow, error) {
	rows, err := r.pool.Query(ctx, propertyOwnerRowSelect+` WHERE po.owner_id = $1 ORDER BY po.name`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list property owners: %w", err)
	}
	defer rows.Close()

	var out []*domain.PropertyOwnerRow
	for rows.Next() {
		row, err := scanPropertyOwnerRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan property owner: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PropertyOwnerRepository) GetForProperty(ctx context.Context, propertyID uuid.UUID) (*domain.PropertyOwner, error) {
	o, err := scanPropertyOwner(r.pool.QueryRow(ctx, `
		SELECT `+propertyOwnerColumns+` FROM property_owners po JOIN properties p ON p.property_owner_id = po.id WHERE p.id = $1`, propertyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("owner for property %s: %w", propertyID, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("owner for property %s: %w", propertyID, err)
	}
	return o, nil
}

func (r *PropertyOwnerRepository) ListAutoStatementTargets(ctx context.Context, accountID uuid.UUID) ([]*domain.AutoStatementTarget, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+propertyOwnerColumns+`, p.id
		FROM property_owners po JOIN properties p ON p.property_owner_id = po.id
		WHERE po.owner_id = $1 AND po.auto_statements ORDER BY po.name, p.name`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list auto statement targets: %w", err)
	}
	defer rows.Close()

	var out []*domain.AutoStatementTarget
	for rows.Next() {
		var o domain.PropertyOwner
		var propertyID uuid.UUID
		if err := rows.Scan(&o.ID, &o.OwnerID, &o.Name, &o.Email, &o.Phone, &o.ManagementFeeBps, &o.AutoStatements, &o.CreatedAt, &o.UpdatedAt, &propertyID); err != nil {
			return nil, fmt.Errorf("scan auto statement target: %w", err)
		}
		out = append(out, &domain.AutoStatementTarget{Owner: &o, PropertyID: propertyID})
	}
	return out, rows.Err()
}
