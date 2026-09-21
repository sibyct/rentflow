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

type UnitRepository struct {
	pool *pgxpool.Pool
}

func NewUnitRepository(pool *pgxpool.Pool) *UnitRepository {
	return &UnitRepository{pool: pool}
}

var _ domain.UnitRepository = (*UnitRepository)(nil)

const unitColumns = `
	id, property_id, unit_name, floor, unit_type, bedrooms, bathrooms, sqft, furnished,
	status, market_rent, current_rent, security_deposit, rent_due_day, tenant_name, notes,
	vacated_at, created_at, updated_at`

// unitColumnsForRead is unitColumns aliased to "u." with current_rent
// replaced by the *effective* current rent: the unit's active lease's
// monthly_rent when one exists, falling back to the unit's own raw
// current_rent otherwise. Every read query in this file uses this (with
// a LEFT JOIN to leases) rather than unitColumns, so "current rent"
// always reflects what's actually being charged rather than a
// manually-entered value that can silently drift out of sync with the
// signed lease. Writes (Create/Update) still target the raw
// units.current_rent column directly — that column remains the
// fallback a manager can set for a unit with no formal lease on file.
const unitColumnsForRead = `
	u.id, u.property_id, u.unit_name, u.floor, u.unit_type, u.bedrooms, u.bathrooms, u.sqft, u.furnished,
	u.status, u.market_rent, COALESCE(l.monthly_rent, u.current_rent) AS current_rent, u.security_deposit,
	u.rent_due_day, u.tenant_name, u.notes, u.vacated_at, u.created_at, u.updated_at`

// activeLeaseJoin is the LEFT JOIN every read query pairs with
// unitColumnsForRead. LEFT (not INNER) so a unit with no lease at all
// still returns a row; at most one row matches per unit because of the
// partial unique index on leases(unit_id) WHERE status = 'active', so
// this can never multiply a unit into more than one result row.
const activeLeaseJoin = `LEFT JOIN leases l ON l.unit_id = u.id AND l.status = 'active'`

func (r *UnitRepository) Create(ctx context.Context, u *domain.Unit) error {
	const q = `
		INSERT INTO units (` + unitColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

	_, err := r.pool.Exec(ctx, q,
		u.ID, u.PropertyID, u.UnitName, u.Floor, u.Type, u.Bedrooms, u.Bathrooms, u.Sqft, furnishedArg(u.Furnished),
		u.Status, u.MarketRent, u.CurrentRent, u.SecurityDeposit, u.RentDueDay, u.TenantName, u.Notes,
		u.VacatedAt, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert unit %s: %w", u.ID, err)
	}
	return nil
}

// CreateMany backs the bulk-add flow. It runs inside a single
// transaction so a batch either lands completely or not at all — unlike
// the rest of this repository, which has no other call site that writes
// more than one row as a single logical operation.
func (r *UnitRepository) CreateMany(ctx context.Context, units []*domain.Unit) error {
	if len(units) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bulk unit insert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO units (` + unitColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

	for _, u := range units {
		if _, err := tx.Exec(ctx, q,
			u.ID, u.PropertyID, u.UnitName, u.Floor, u.Type, u.Bedrooms, u.Bathrooms, u.Sqft, furnishedArg(u.Furnished),
			u.Status, u.MarketRent, u.CurrentRent, u.SecurityDeposit, u.RentDueDay, u.TenantName, u.Notes,
			u.VacatedAt, u.CreatedAt, u.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert unit %s in bulk batch: %w", u.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit bulk unit insert: %w", err)
	}
	return nil
}

func (r *UnitRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Unit, error) {
	const q = `SELECT ` + unitColumnsForRead + ` FROM units u ` + activeLeaseJoin + ` WHERE u.id = $1`

	u, err := scanUnit(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get unit %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get unit %s: %w", id, err)
	}
	return u, nil
}

func (r *UnitRepository) ListByProperty(ctx context.Context, propertyID uuid.UUID) ([]*domain.Unit, error) {
	const q = `SELECT ` + unitColumnsForRead + ` FROM units u ` + activeLeaseJoin + ` WHERE u.property_id = $1 ORDER BY u.unit_name ASC, u.id ASC`

	rows, err := r.pool.Query(ctx, q, propertyID)
	if err != nil {
		return nil, fmt.Errorf("list units for property %s: %w", propertyID, err)
	}
	defer rows.Close()

	units := make([]*domain.Unit, 0)
	for rows.Next() {
		u, err := scanUnit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan unit row: %w", err)
		}
		units = append(units, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unit rows: %w", err)
	}
	return units, nil
}

// qualifiedUnitColumns is unitColumnsForRead's shape for ListForOwner's
// additional join to properties: units, properties, and leases all have
// id/created_at/updated_at, so an unqualified SELECT across the join
// would be ambiguous. current_rent is the same effective-rent
// computation unitColumnsForRead uses — see its doc comment.
const qualifiedUnitColumns = `
	u.id, u.property_id, u.unit_name, u.floor, u.unit_type, u.bedrooms, u.bathrooms, u.sqft, u.furnished,
	u.status, u.market_rent, COALESCE(l.monthly_rent, u.current_rent) AS current_rent, u.security_deposit,
	u.rent_due_day, u.tenant_name, u.notes, u.vacated_at, u.created_at, u.updated_at`

// unitSortColumns allow-lists ListForOwner's sortable columns — never
// interpolate a client-supplied string directly into an ORDER BY clause
// (see propertySortColumns). Rent sorts by the same effective-rent
// computation the selected column uses (active lease's rent, falling
// back to the unit's own current_rent, falling back to market_rent for
// a vacant unit with neither) — see unitColumnsForRead.
var unitSortColumns = map[domain.UnitSortKey]string{
	domain.UnitSortRent:         "COALESCE(l.monthly_rent, u.current_rent, u.market_rent)",
	domain.UnitSortStatus:       "u.status",
	domain.UnitSortPropertyName: "p.name",
}

func (r *UnitRepository) ListForOwner(ctx context.Context, opts domain.UnitListOptions) ([]*domain.UnitWithProperty, int, error) {
	// p.owner_id = $1 is the actual row-security boundary: every other
	// filter below narrows further, but this one is what makes it
	// impossible for the query to return a unit belonging to a property
	// this owner doesn't own, regardless of what else the caller passes.
	where := []string{"p.owner_id = $1"}
	args := []any{opts.OwnerID}

	if opts.Filter.Search != "" {
		args = append(args, "%"+opts.Filter.Search+"%")
		where = append(where, fmt.Sprintf("(u.unit_name ILIKE $%d OR p.name ILIKE $%d OR u.tenant_name ILIKE $%d)", len(args), len(args), len(args)))
	}
	if opts.Filter.PropertyID != nil {
		args = append(args, *opts.Filter.PropertyID)
		where = append(where, fmt.Sprintf("u.property_id = $%d", len(args)))
	}
	if opts.Filter.Status != nil {
		args = append(args, *opts.Filter.Status)
		where = append(where, fmt.Sprintf("u.status = $%d", len(args)))
	}
	if opts.Filter.Type != nil {
		args = append(args, *opts.Filter.Type)
		where = append(where, fmt.Sprintf("u.unit_type = $%d", len(args)))
	}
	whereClause := strings.Join(where, " AND ")

	sortColumn, ok := unitSortColumns[opts.Sort]
	if !ok {
		sortColumn = unitSortColumns[domain.UnitSortPropertyName]
	}
	sortDir := "ASC"
	if opts.SortDesc {
		sortDir = "DESC"
	}

	args = append(args, opts.Limit, opts.Offset)
	q := fmt.Sprintf(
		`SELECT %s, p.name AS property_name
		 FROM units u
		 JOIN properties p ON p.id = u.property_id
		 %s
		 WHERE %s
		 ORDER BY %s %s, u.unit_name ASC, u.id ASC
		 LIMIT $%d OFFSET $%d`,
		qualifiedUnitColumns, activeLeaseJoin, whereClause, sortColumn, sortDir, len(args)-1, len(args),
	)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list units for owner %s: %w", opts.OwnerID, err)
	}
	defer rows.Close()

	units := make([]*domain.UnitWithProperty, 0)
	for rows.Next() {
		u, err := scanUnitWithProperty(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan unit-with-property row: %w", err)
		}
		units = append(units, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate unit-with-property rows: %w", err)
	}

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM units u JOIN properties p ON p.id = u.property_id WHERE %s`, whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count units for owner %s: %w", opts.OwnerID, err)
	}

	return units, total, nil
}

func (r *UnitRepository) Update(ctx context.Context, u *domain.Unit) error {
	const q = `
		UPDATE units
		SET unit_name = $2, floor = $3, unit_type = $4, bedrooms = $5, bathrooms = $6, sqft = $7,
			furnished = $8, status = $9, market_rent = $10, current_rent = $11, security_deposit = $12,
			rent_due_day = $13, tenant_name = $14, notes = $15, vacated_at = $16, updated_at = $17
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		u.ID, u.UnitName, u.Floor, u.Type, u.Bedrooms, u.Bathrooms, u.Sqft, furnishedArg(u.Furnished),
		u.Status, u.MarketRent, u.CurrentRent, u.SecurityDeposit, u.RentDueDay, u.TenantName, u.Notes, u.VacatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update unit %s: %w", u.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update unit %s: %w", u.ID, domain.ErrNotFound)
	}
	return nil
}

func (r *UnitRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM units WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete unit %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete unit %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

func (r *UnitRepository) ExistsByPropertyUnitName(ctx context.Context, propertyID uuid.UUID, unitName string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM units WHERE property_id = $1 AND unit_name = $2)`

	var exists bool
	if err := r.pool.QueryRow(ctx, q, propertyID, unitName).Scan(&exists); err != nil {
		return false, fmt.Errorf("check existing unit for property %s named %q: %w", propertyID, unitName, err)
	}
	return exists, nil
}

func (r *UnitRepository) GetPropertyUnitStats(ctx context.Context, propertyID uuid.UUID) (*domain.PropertyUnitStats, error) {
	const q = `
		SELECT
			COUNT(*) AS unit_count,
			COUNT(*) FILTER (WHERE u.status = 'occupied') AS occupied_count,
			COALESCE(SUM(COALESCE(l.monthly_rent, u.current_rent)) FILTER (WHERE u.status = 'occupied'), 0) AS total_collected
		FROM units u ` + activeLeaseJoin + `
		WHERE u.property_id = $1`

	stats := &domain.PropertyUnitStats{PropertyID: propertyID}
	if err := r.pool.QueryRow(ctx, q, propertyID).Scan(&stats.UnitCount, &stats.OccupiedCount, &stats.TotalCollected); err != nil {
		return nil, fmt.Errorf("get unit stats for property %s: %w", propertyID, err)
	}
	stats.OccupancyPct = occupancyPct(stats.OccupiedCount, stats.UnitCount)
	return stats, nil
}

func (r *UnitRepository) GetPropertyUnitStatsBulk(ctx context.Context, propertyIDs []uuid.UUID) (map[uuid.UUID]*domain.PropertyUnitStats, error) {
	result := make(map[uuid.UUID]*domain.PropertyUnitStats, len(propertyIDs))
	if len(propertyIDs) == 0 {
		return result, nil
	}

	const q = `
		SELECT
			u.property_id,
			COUNT(*) AS unit_count,
			COUNT(*) FILTER (WHERE u.status = 'occupied') AS occupied_count,
			COALESCE(SUM(COALESCE(l.monthly_rent, u.current_rent)) FILTER (WHERE u.status = 'occupied'), 0) AS total_collected
		FROM units u ` + activeLeaseJoin + `
		WHERE u.property_id = ANY($1)
		GROUP BY u.property_id`

	rows, err := r.pool.Query(ctx, q, propertyIDs)
	if err != nil {
		return nil, fmt.Errorf("bulk get unit stats for %d properties: %w", len(propertyIDs), err)
	}
	defer rows.Close()

	for rows.Next() {
		stats := &domain.PropertyUnitStats{}
		if err := rows.Scan(&stats.PropertyID, &stats.UnitCount, &stats.OccupiedCount, &stats.TotalCollected); err != nil {
			return nil, fmt.Errorf("scan bulk unit stats row: %w", err)
		}
		stats.OccupancyPct = occupancyPct(stats.OccupiedCount, stats.UnitCount)
		result[stats.PropertyID] = stats
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bulk unit stats rows: %w", err)
	}
	return result, nil
}

func occupancyPct(occupied, total int) int {
	if total <= 0 {
		return 0
	}
	return int(float64(occupied) / float64(total) * 100)
}

func scanUnit(row rowScanner) (*domain.Unit, error) {
	var u domain.Unit
	var floor sql.NullString
	var bedrooms sql.NullInt32
	var bathrooms sql.NullFloat64
	var sqft sql.NullInt32
	var furnished sql.NullString
	var marketRent sql.NullFloat64
	var currentRent sql.NullFloat64
	var securityDeposit sql.NullFloat64
	var rentDueDay sql.NullInt32
	var vacatedAt sql.NullTime

	err := row.Scan(
		&u.ID, &u.PropertyID, &u.UnitName, &floor, &u.Type, &bedrooms, &bathrooms, &sqft, &furnished,
		&u.Status, &marketRent, &currentRent, &securityDeposit, &rentDueDay, &u.TenantName, &u.Notes,
		&vacatedAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if floor.Valid {
		u.Floor = floor.String
	}
	if bedrooms.Valid {
		v := int(bedrooms.Int32)
		u.Bedrooms = &v
	}
	if bathrooms.Valid {
		v := bathrooms.Float64
		u.Bathrooms = &v
	}
	if sqft.Valid {
		v := int(sqft.Int32)
		u.Sqft = &v
	}
	if furnished.Valid {
		f := domain.UnitFurnished(furnished.String)
		u.Furnished = &f
	}
	if marketRent.Valid {
		v := marketRent.Float64
		u.MarketRent = &v
	}
	if currentRent.Valid {
		v := currentRent.Float64
		u.CurrentRent = &v
	}
	if securityDeposit.Valid {
		v := securityDeposit.Float64
		u.SecurityDeposit = &v
	}
	if rentDueDay.Valid {
		v := int(rentDueDay.Int32)
		u.RentDueDay = &v
	}
	if vacatedAt.Valid {
		v := vacatedAt.Time
		u.VacatedAt = &v
	}

	return &u, nil
}

// scanUnitWithProperty is scanUnit plus the one extra property_name
// column ListForOwner's join selects — duplicated rather than built on
// top of scanUnit, since a rowScanner's Scan call must list every
// destination in one shot and there's no clean way to append to it
// after the fact.
func scanUnitWithProperty(row rowScanner) (*domain.UnitWithProperty, error) {
	var u domain.UnitWithProperty
	var floor sql.NullString
	var bedrooms sql.NullInt32
	var bathrooms sql.NullFloat64
	var sqft sql.NullInt32
	var furnished sql.NullString
	var marketRent sql.NullFloat64
	var currentRent sql.NullFloat64
	var securityDeposit sql.NullFloat64
	var rentDueDay sql.NullInt32
	var vacatedAt sql.NullTime

	err := row.Scan(
		&u.ID, &u.PropertyID, &u.UnitName, &floor, &u.Type, &bedrooms, &bathrooms, &sqft, &furnished,
		&u.Status, &marketRent, &currentRent, &securityDeposit, &rentDueDay, &u.TenantName, &u.Notes,
		&vacatedAt, &u.CreatedAt, &u.UpdatedAt, &u.PropertyName,
	)
	if err != nil {
		return nil, err
	}

	if floor.Valid {
		u.Floor = floor.String
	}
	if bedrooms.Valid {
		v := int(bedrooms.Int32)
		u.Bedrooms = &v
	}
	if bathrooms.Valid {
		v := bathrooms.Float64
		u.Bathrooms = &v
	}
	if sqft.Valid {
		v := int(sqft.Int32)
		u.Sqft = &v
	}
	if furnished.Valid {
		f := domain.UnitFurnished(furnished.String)
		u.Furnished = &f
	}
	if marketRent.Valid {
		v := marketRent.Float64
		u.MarketRent = &v
	}
	if currentRent.Valid {
		v := currentRent.Float64
		u.CurrentRent = &v
	}
	if securityDeposit.Valid {
		v := securityDeposit.Float64
		u.SecurityDeposit = &v
	}
	if rentDueDay.Valid {
		v := int(rentDueDay.Int32)
		u.RentDueDay = &v
	}
	if vacatedAt.Valid {
		v := vacatedAt.Time
		u.VacatedAt = &v
	}

	return &u, nil
}

// furnishedArg converts the domain's *UnitFurnished to *string for the
// query argument — see property_repository.go's ownershipArg for why
// this indirection through *string (rather than a pointer to the named
// type) is the reliable path through pgx's argument encoding.
func furnishedArg(f *domain.UnitFurnished) *string {
	if f == nil {
		return nil
	}
	s := string(*f)
	return &s
}
