package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

const propertyColumns = `
	id, name, type, address_line1, address_line2, city, state_province, postal_code, country,
	units, ownership, owner_name, year_built, onboard_date, amenities, notes, status, owner_id,
	created_at, updated_at`

func (r *PropertyRepository) Create(ctx context.Context, p *domain.Property) error {
	const q = `
		INSERT INTO properties (` + propertyColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`

	_, err := r.pool.Exec(ctx, q,
		p.ID, p.Name, p.Type, p.AddressLine1, p.AddressLine2, p.City, p.StateProvince, p.PostalCode, p.Country,
		p.Units, ownershipArg(p.Ownership), p.OwnerName, p.YearBuilt, p.OnboardDate, p.Amenities, p.Notes, p.Status, p.OwnerID,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert property %s: %w", p.ID, err)
	}
	return nil
}

func (r *PropertyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	const q = `SELECT ` + propertyColumns + ` FROM properties WHERE id = $1`

	p, err := scanProperty(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get property %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get property %s: %w", id, err)
	}
	return p, nil
}

// propertySortColumns allow-lists the columns List can sort by, mapping
// the domain's PropertySortKey to the actual SQL column — never
// interpolate a client-supplied string directly into an ORDER BY clause.
var propertySortColumns = map[domain.PropertySortKey]string{
	domain.PropertySortName:      "name",
	domain.PropertySortType:      "type",
	domain.PropertySortUnits:     "units",
	domain.PropertySortStatus:    "status",
	domain.PropertySortCreatedAt: "created_at",
}

func (r *PropertyRepository) List(ctx context.Context, opts domain.PropertyListOptions) ([]*domain.Property, int, error) {
	where := []string{"owner_id = $1"}
	args := []any{opts.OwnerID}

	if opts.Filter.Search != "" {
		args = append(args, "%"+opts.Filter.Search+"%")
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR address_line1 ILIKE $%d OR city ILIKE $%d)", len(args), len(args), len(args)))
	}
	if opts.Filter.Type != nil {
		args = append(args, *opts.Filter.Type)
		where = append(where, fmt.Sprintf("type = $%d", len(args)))
	}
	if opts.Filter.Status != nil {
		args = append(args, *opts.Filter.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	whereClause := strings.Join(where, " AND ")

	sortColumn, ok := propertySortColumns[opts.Sort]
	if !ok {
		sortColumn = propertySortColumns[domain.PropertySortCreatedAt]
	}
	sortDir := "ASC"
	if opts.SortDesc {
		sortDir = "DESC"
	}

	args = append(args, opts.Limit, opts.Offset)
	q := fmt.Sprintf(
		`SELECT %s FROM properties WHERE %s ORDER BY %s %s, id ASC LIMIT $%d OFFSET $%d`,
		propertyColumns, whereClause, sortColumn, sortDir, len(args)-1, len(args),
	)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list properties: %w", err)
	}
	defer rows.Close()

	properties := make([]*domain.Property, 0)
	for rows.Next() {
		p, err := scanProperty(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan property row: %w", err)
		}
		properties = append(properties, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate property rows: %w", err)
	}

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM properties WHERE %s`, whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count properties: %w", err)
	}

	return properties, total, nil
}

func (r *PropertyRepository) Update(ctx context.Context, p *domain.Property) error {
	const q = `
		UPDATE properties
		SET name = $2, type = $3, address_line1 = $4, address_line2 = $5, city = $6, state_province = $7,
			postal_code = $8, country = $9, units = $10, ownership = $11, owner_name = $12, year_built = $13,
			onboard_date = $14, amenities = $15, notes = $16, status = $17, updated_at = $18
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		p.ID, p.Name, p.Type, p.AddressLine1, p.AddressLine2, p.City, p.StateProvince,
		p.PostalCode, p.Country, p.Units, ownershipArg(p.Ownership), p.OwnerName, p.YearBuilt,
		p.OnboardDate, p.Amenities, p.Notes, p.Status, p.UpdatedAt,
	)
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

func (r *PropertyRepository) BulkUpdateStatus(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, status domain.PropertyStatus) (int, error) {
	const q = `UPDATE properties SET status = $1, updated_at = now() WHERE owner_id = $2 AND id = ANY($3)`

	tag, err := r.pool.Exec(ctx, q, status, ownerID, ids)
	if err != nil {
		return 0, fmt.Errorf("bulk update status for %d properties: %w", len(ids), err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *PropertyRepository) ExistsByOwnerAddress(ctx context.Context, ownerID uuid.UUID, addressLine1 string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM properties WHERE owner_id = $1 AND address_line1 = $2)`

	var exists bool
	if err := r.pool.QueryRow(ctx, q, ownerID, addressLine1).Scan(&exists); err != nil {
		return false, fmt.Errorf("check existing property for owner %s at %q: %w", ownerID, addressLine1, err)
	}
	return exists, nil
}

// rowScanner is satisfied by both pgx.Row (QueryRow) and pgx.Rows
// (Query, via its embedded Scan), so scanProperty works for both List
// and GetByID without duplicating the column list.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanProperty centralizes the nullable-column handling: ownership,
// year_built, and onboard_date are all NULL-able, and pgx's Scan needs
// database/sql's Null* wrapper types (rather than scanning directly into
// domain's *string/*int/*time.Time) to read a NULL column reliably.
func scanProperty(row rowScanner) (*domain.Property, error) {
	var p domain.Property
	var ownership sql.NullString
	var yearBuilt sql.NullInt32
	var onboardDate sql.NullTime

	err := row.Scan(
		&p.ID, &p.Name, &p.Type, &p.AddressLine1, &p.AddressLine2, &p.City, &p.StateProvince, &p.PostalCode, &p.Country,
		&p.Units, &ownership, &p.OwnerName, &yearBuilt, &onboardDate, &p.Amenities, &p.Notes, &p.Status, &p.OwnerID,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if ownership.Valid {
		o := domain.PropertyOwnership(ownership.String)
		p.Ownership = &o
	}
	if yearBuilt.Valid {
		y := int(yearBuilt.Int32)
		p.YearBuilt = &y
	}
	if onboardDate.Valid {
		p.OnboardDate = &onboardDate.Time
	}

	return &p, nil
}

// ownershipArg converts the domain's *PropertyOwnership to *string for
// the query argument: pgx's argument encoding has native, well-tested
// support for nil-able *string (NULL on nil), whereas a pointer to a
// named string type takes a less certain path through its codec.
func ownershipArg(o *domain.PropertyOwnership) *string {
	if o == nil {
		return nil
	}
	s := string(*o)
	return &s
}
