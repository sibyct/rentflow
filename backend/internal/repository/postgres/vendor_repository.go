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

type VendorRepository struct {
	pool *pgxpool.Pool
}

func NewVendorRepository(pool *pgxpool.Pool) *VendorRepository {
	return &VendorRepository{pool: pool}
}

var _ domain.VendorRepository = (*VendorRepository)(nil)

const vendorColumns = `
	id, owner_id, company_name, categories, contact_person, phone, email, address,
	serves_all_properties, insurance_expiry, license_number, license_expiry, coi_link, tax_doc_link,
	rate_type, rate_amount, payment_terms, internal_notes, active, created_at, updated_at`

// Create runs inside a transaction when propertiesServed is non-empty,
// so a vendor scoped to specific properties never lands with a partial
// vendor_properties set — same rationale as UnitRepository.CreateMany.
func (r *VendorRepository) Create(ctx context.Context, v *domain.Vendor, propertiesServed []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create vendor %s: %w", v.ID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO vendors (` + vendorColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`

	if _, err := tx.Exec(ctx, q,
		v.ID, v.OwnerID, v.CompanyName, categoryArgs(v.Categories), v.ContactPerson, v.Phone, v.Email, v.Address,
		v.ServesAllProperties, v.InsuranceExpiry, v.LicenseNumber, v.LicenseExpiry, v.COILink, v.TaxDocLink,
		rateTypeArg(v.RateType), v.RateAmount, paymentTermsArg(v.PaymentTerms), v.InternalNotes, v.Active, v.CreatedAt, v.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert vendor %s: %w", v.ID, err)
	}

	for _, propertyID := range propertiesServed {
		if _, err := tx.Exec(ctx, `INSERT INTO vendor_properties (vendor_id, property_id) VALUES ($1, $2)`, v.ID, propertyID); err != nil {
			return fmt.Errorf("insert vendor_properties for vendor %s: %w", v.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create vendor %s: %w", v.ID, err)
	}
	return nil
}

func (r *VendorRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Vendor, error) {
	const q = `SELECT ` + vendorColumns + ` FROM vendors WHERE id = $1`

	v, err := scanVendor(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get vendor %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get vendor %s: %w", id, err)
	}
	return v, nil
}

func (r *VendorRepository) GetByIDWithStats(ctx context.Context, id uuid.UUID) (*domain.VendorWithStats, error) {
	q := fmt.Sprintf(`SELECT %s, %s FROM vendors v WHERE v.id = $1`, qualifiedVendorColumns("v"), vendorStatsSelect)

	v, err := scanVendorWithStats(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get vendor %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get vendor %s: %w", id, err)
	}
	return v, nil
}

// insuranceStatusWhere translates a domain.InsuranceStatus filter into
// the SQL that reproduces Vendor.InsuranceStatus's logic — see
// lease_repository.go's displayStatusWhere for the identical rationale
// (the raw column alone can't express the richer, date-derived status).
func insuranceStatusWhere(s domain.InsuranceStatus) string {
	switch s {
	case domain.InsuranceStatusUnknown:
		return "v.insurance_expiry IS NULL"
	case domain.InsuranceStatusExpired:
		return "v.insurance_expiry IS NOT NULL AND v.insurance_expiry < CURRENT_DATE"
	case domain.InsuranceStatusExpiringSoon:
		return fmt.Sprintf(
			"v.insurance_expiry IS NOT NULL AND v.insurance_expiry >= CURRENT_DATE AND v.insurance_expiry <= CURRENT_DATE + INTERVAL '%d days'",
			domain.InsuranceExpiringSoonDays,
		)
	default: // domain.InsuranceStatusValid
		return fmt.Sprintf("v.insurance_expiry IS NOT NULL AND v.insurance_expiry > CURRENT_DATE + INTERVAL '%d days'", domain.InsuranceExpiringSoonDays)
	}
}

const vendorStatsSelect = `
	(SELECT COUNT(*) FROM work_orders w WHERE w.vendor_id = v.id AND w.status NOT IN ('completed', 'cancelled')) AS open_work_orders,
	(SELECT AVG(rating)::float8 FROM work_orders w WHERE w.vendor_id = v.id AND w.rating IS NOT NULL) AS average_rating,
	(SELECT COUNT(*) FROM vendor_properties vp WHERE vp.vendor_id = v.id) AS properties_served_count`

func (r *VendorRepository) ListForOwner(ctx context.Context, opts domain.VendorListOptions) ([]*domain.VendorWithStats, int, error) {
	where := []string{"v.owner_id = $1"}
	args := []any{opts.OwnerID}

	if opts.Filter.Search != "" {
		args = append(args, "%"+opts.Filter.Search+"%")
		where = append(where, fmt.Sprintf("(v.company_name ILIKE $%d OR v.contact_person ILIKE $%d OR v.email ILIKE $%d)", len(args), len(args), len(args)))
	}
	if opts.Filter.Category != nil {
		args = append(args, string(*opts.Filter.Category))
		where = append(where, fmt.Sprintf("$%d = ANY(v.categories)", len(args)))
	}
	if opts.Filter.Active != nil {
		args = append(args, *opts.Filter.Active)
		where = append(where, fmt.Sprintf("v.active = $%d", len(args)))
	}
	if opts.Filter.InsuranceStatus != nil {
		where = append(where, insuranceStatusWhere(*opts.Filter.InsuranceStatus))
	}
	whereClause := strings.Join(where, " AND ")

	sortExpr := "v.company_name ASC"
	if opts.Sort == domain.VendorSortRating {
		sortDir := "ASC"
		if opts.SortDesc {
			sortDir = "DESC"
		}
		sortExpr = fmt.Sprintf("average_rating %s NULLS LAST", sortDir)
	} else if opts.SortDesc {
		sortExpr = "v.company_name DESC"
	}

	args = append(args, opts.Limit, opts.Offset)
	q := fmt.Sprintf(
		`SELECT %s, %s
		 FROM vendors v
		 WHERE %s
		 ORDER BY %s, v.id ASC
		 LIMIT $%d OFFSET $%d`,
		qualifiedVendorColumns("v"), vendorStatsSelect, whereClause, sortExpr, len(args)-1, len(args),
	)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list vendors for owner %s: %w", opts.OwnerID, err)
	}
	defer rows.Close()

	vendors := make([]*domain.VendorWithStats, 0)
	for rows.Next() {
		v, err := scanVendorWithStats(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan vendor-with-stats row: %w", err)
		}
		vendors = append(vendors, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate vendor-with-stats rows: %w", err)
	}

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM vendors v WHERE %s`, whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count vendors for owner %s: %w", opts.OwnerID, err)
	}

	return vendors, total, nil
}

func (r *VendorRepository) ListForCategory(ctx context.Context, ownerID uuid.UUID, category domain.WorkOrderCategory) ([]*domain.VendorWithStats, error) {
	q := fmt.Sprintf(
		`SELECT %s, %s
		 FROM vendors v
		 WHERE v.owner_id = $1 AND v.active = true AND $2 = ANY(v.categories)
		 ORDER BY v.company_name ASC`,
		qualifiedVendorColumns("v"), vendorStatsSelect,
	)

	rows, err := r.pool.Query(ctx, q, ownerID, string(category))
	if err != nil {
		return nil, fmt.Errorf("list vendors for owner %s category %q: %w", ownerID, category, err)
	}
	defer rows.Close()

	vendors := make([]*domain.VendorWithStats, 0)
	for rows.Next() {
		v, err := scanVendorWithStats(rows)
		if err != nil {
			return nil, fmt.Errorf("scan vendor-with-stats row: %w", err)
		}
		vendors = append(vendors, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vendor-with-stats rows: %w", err)
	}
	return vendors, nil
}

func (r *VendorRepository) Update(ctx context.Context, v *domain.Vendor) error {
	const q = `
		UPDATE vendors
		SET company_name = $2, categories = $3, contact_person = $4, phone = $5, email = $6, address = $7,
			serves_all_properties = $8, insurance_expiry = $9, license_number = $10, license_expiry = $11,
			coi_link = $12, tax_doc_link = $13, rate_type = $14, rate_amount = $15, payment_terms = $16,
			internal_notes = $17, active = $18, updated_at = $19
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		v.ID, v.CompanyName, categoryArgs(v.Categories), v.ContactPerson, v.Phone, v.Email, v.Address,
		v.ServesAllProperties, v.InsuranceExpiry, v.LicenseNumber, v.LicenseExpiry,
		v.COILink, v.TaxDocLink, rateTypeArg(v.RateType), v.RateAmount, paymentTermsArg(v.PaymentTerms),
		v.InternalNotes, v.Active, v.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update vendor %s: %w", v.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update vendor %s: %w", v.ID, domain.ErrNotFound)
	}
	return nil
}

func (r *VendorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM vendors WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete vendor %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete vendor %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

func (r *VendorRepository) GetPropertiesServed(ctx context.Context, vendorID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT property_id FROM vendor_properties WHERE vendor_id = $1`, vendorID)
	if err != nil {
		return nil, fmt.Errorf("get properties served for vendor %s: %w", vendorID, err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan vendor_properties row: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vendor_properties rows: %w", err)
	}
	return ids, nil
}

// SetPropertiesServed replaces the full set — a simple delete-then-
// insert inside one transaction, matching how the frontend always
// resubmits the whole picked list rather than a diff.
func (r *VendorRepository) SetPropertiesServed(ctx context.Context, vendorID uuid.UUID, propertyIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin set properties served for vendor %s: %w", vendorID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM vendor_properties WHERE vendor_id = $1`, vendorID); err != nil {
		return fmt.Errorf("clear vendor_properties for vendor %s: %w", vendorID, err)
	}
	for _, propertyID := range propertyIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO vendor_properties (vendor_id, property_id) VALUES ($1, $2)`, vendorID, propertyID); err != nil {
			return fmt.Errorf("insert vendor_properties for vendor %s: %w", vendorID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit set properties served for vendor %s: %w", vendorID, err)
	}
	return nil
}

// GetSpendSummary sums actual_cost across the vendor's completed work
// orders — see VendorSpendSummary's doc comment for why this (not a
// fabricated invoice/paid-status history) is the Financials tab.
func (r *VendorRepository) GetSpendSummary(ctx context.Context, vendorID uuid.UUID) (*domain.VendorSpendSummary, error) {
	const q = `
		SELECT
			COALESCE(SUM(actual_cost) FILTER (WHERE status = 'completed' AND date_trunc('month', completed_at) = date_trunc('month', CURRENT_DATE)), 0) AS this_month,
			COALESCE(SUM(actual_cost) FILTER (WHERE status = 'completed' AND date_trunc('year', completed_at) = date_trunc('year', CURRENT_DATE)), 0) AS year_to_date
		FROM work_orders
		WHERE vendor_id = $1`

	s := &domain.VendorSpendSummary{}
	if err := r.pool.QueryRow(ctx, q, vendorID).Scan(&s.ThisMonth, &s.YearToDate); err != nil {
		return nil, fmt.Errorf("get spend summary for vendor %s: %w", vendorID, err)
	}
	return s, nil
}

// qualifiedVendorColumns is vendorColumns prefixed with alias — used by
// ListForOwner/ListForCategory, which select from vendors alongside
// correlated subqueries that also reference the vendors row (see
// vendorStatsSelect), so an unqualified column list would be ambiguous
// there even though there's no join partner table.
func qualifiedVendorColumns(alias string) string {
	return alias + `.id, ` + alias + `.owner_id, ` + alias + `.company_name, ` + alias + `.categories, ` + alias + `.contact_person, ` +
		alias + `.phone, ` + alias + `.email, ` + alias + `.address, ` + alias + `.serves_all_properties, ` + alias + `.insurance_expiry, ` +
		alias + `.license_number, ` + alias + `.license_expiry, ` + alias + `.coi_link, ` + alias + `.tax_doc_link, ` + alias + `.rate_type, ` +
		alias + `.rate_amount, ` + alias + `.payment_terms, ` + alias + `.internal_notes, ` + alias + `.active, ` + alias + `.created_at, ` + alias + `.updated_at`
}

func scanVendor(row rowScanner) (*domain.Vendor, error) {
	var v domain.Vendor
	var categories []string
	var insuranceExpiry, licenseExpiry sql.NullTime
	var rateType, paymentTerms sql.NullString
	var rateAmount sql.NullFloat64

	err := row.Scan(
		&v.ID, &v.OwnerID, &v.CompanyName, &categories, &v.ContactPerson, &v.Phone, &v.Email, &v.Address,
		&v.ServesAllProperties, &insuranceExpiry, &v.LicenseNumber, &licenseExpiry, &v.COILink, &v.TaxDocLink,
		&rateType, &rateAmount, &paymentTerms, &v.InternalNotes, &v.Active, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	v.Categories = toCategorySlice(categories)
	applyVendorNullables(&v, insuranceExpiry, licenseExpiry, rateType, paymentTerms, rateAmount)
	return &v, nil
}

func scanVendorWithStats(row rowScanner) (*domain.VendorWithStats, error) {
	var v domain.VendorWithStats
	var categories []string
	var insuranceExpiry, licenseExpiry sql.NullTime
	var rateType, paymentTerms sql.NullString
	var rateAmount sql.NullFloat64
	var averageRating sql.NullFloat64

	err := row.Scan(
		&v.ID, &v.OwnerID, &v.CompanyName, &categories, &v.ContactPerson, &v.Phone, &v.Email, &v.Address,
		&v.ServesAllProperties, &insuranceExpiry, &v.LicenseNumber, &licenseExpiry, &v.COILink, &v.TaxDocLink,
		&rateType, &rateAmount, &paymentTerms, &v.InternalNotes, &v.Active, &v.CreatedAt, &v.UpdatedAt,
		&v.OpenWorkOrders, &averageRating, &v.PropertiesServedCount,
	)
	if err != nil {
		return nil, err
	}
	v.Categories = toCategorySlice(categories)
	applyVendorNullables(&v.Vendor, insuranceExpiry, licenseExpiry, rateType, paymentTerms, rateAmount)
	if averageRating.Valid {
		r := averageRating.Float64
		v.AverageRating = &r
	}
	return &v, nil
}

func toCategorySlice(ss []string) []domain.WorkOrderCategory {
	out := make([]domain.WorkOrderCategory, len(ss))
	for i, s := range ss {
		out[i] = domain.WorkOrderCategory(s)
	}
	return out
}

// categoryArgs converts []domain.WorkOrderCategory to []string for the
// query argument — pgx's array-argument encoding has proven, well-
// tested support for []string (see CoResidents in lease_repository.go),
// whereas a slice of a named string type takes a less certain path
// through its codec.
func categoryArgs(categories []domain.WorkOrderCategory) []string {
	out := make([]string, len(categories))
	for i, c := range categories {
		out[i] = string(c)
	}
	return out
}

func applyVendorNullables(
	v *domain.Vendor,
	insuranceExpiry, licenseExpiry sql.NullTime,
	rateType, paymentTerms sql.NullString,
	rateAmount sql.NullFloat64,
) {
	if insuranceExpiry.Valid {
		v.InsuranceExpiry = &insuranceExpiry.Time
	}
	if licenseExpiry.Valid {
		v.LicenseExpiry = &licenseExpiry.Time
	}
	if rateType.Valid {
		t := domain.VendorRateType(rateType.String)
		v.RateType = &t
	}
	if rateAmount.Valid {
		r := rateAmount.Float64
		v.RateAmount = &r
	}
	if paymentTerms.Valid {
		t := domain.VendorPaymentTerms(paymentTerms.String)
		v.PaymentTerms = &t
	}
}

func rateTypeArg(t *domain.VendorRateType) *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}

func paymentTermsArg(t *domain.VendorPaymentTerms) *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}
