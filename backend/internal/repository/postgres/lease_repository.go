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

type LeaseRepository struct {
	pool *pgxpool.Pool
}

func NewLeaseRepository(pool *pgxpool.Pool) *LeaseRepository {
	return &LeaseRepository{pool: pool}
}

var _ domain.LeaseRepository = (*LeaseRepository)(nil)

const leaseColumns = `
	id, unit_id, lease_type, status, start_date, end_date, move_in_date, move_out_date,
	monthly_rent, security_deposit, deposit_status, rent_due_day, late_fee_amount, late_fee_grace_days,
	primary_resident_name, co_residents, emergency_contact, renewal_status, termination_reason,
	termination_notice_date, signed, signed_date, notes, created_at, updated_at`

func (r *LeaseRepository) Create(ctx context.Context, l *domain.Lease) error {
	const q = `
		INSERT INTO leases (` + leaseColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)`

	_, err := r.pool.Exec(ctx, q,
		l.ID, l.UnitID, l.Type, l.Status, l.StartDate, l.EndDate, l.MoveInDate, l.MoveOutDate,
		l.MonthlyRent, l.SecurityDeposit, depositStatusArg(l.DepositStatus), l.RentDueDay, l.LateFeeAmount, l.LateFeeGraceDays,
		l.PrimaryResidentName, l.CoResidents, l.EmergencyContact, l.RenewalStatus, terminationReasonArg(l.TerminationReason),
		l.TerminationNoticeDate, l.Signed, l.SignedDate, l.Notes, l.CreatedAt, l.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert lease %s: %w", l.ID, err)
	}
	return nil
}

func (r *LeaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Lease, error) {
	const q = `SELECT ` + leaseColumns + ` FROM leases WHERE id = $1`

	l, err := scanLease(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get lease %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get lease %s: %w", id, err)
	}
	return l, nil
}

// qualifiedLeaseColumns is leaseColumns prefixed with "l." for
// ListForOwner's join — leases, units, and properties all have id,
// created_at, and updated_at, so an unqualified SELECT across the join
// would be ambiguous.
const qualifiedLeaseColumns = `
	l.id, l.unit_id, l.lease_type, l.status, l.start_date, l.end_date, l.move_in_date, l.move_out_date,
	l.monthly_rent, l.security_deposit, l.deposit_status, l.rent_due_day, l.late_fee_amount, l.late_fee_grace_days,
	l.primary_resident_name, l.co_residents, l.emergency_contact, l.renewal_status, l.termination_reason,
	l.termination_notice_date, l.signed, l.signed_date, l.notes, l.created_at, l.updated_at`

var leaseSortColumns = map[domain.LeaseSortKey]string{
	domain.LeaseSortStartDate:   "l.start_date",
	domain.LeaseSortEndDate:     "l.end_date",
	domain.LeaseSortMonthlyRent: "l.monthly_rent",
	domain.LeaseSortStatus:      "l.status",
}

// displayStatusWhere translates a domain.LeaseDisplayStatus filter into
// the SQL that reproduces domain.Lease.DisplayStatus's logic: the
// stored `status` column alone can't distinguish upcoming/
// expiring_soon/expired, all three of which are "status = 'active'"
// plus a date comparison against CURRENT_DATE.
func displayStatusWhere(s domain.LeaseDisplayStatus) string {
	switch s {
	case domain.LeaseDisplayDraft:
		return "l.status = 'draft'"
	case domain.LeaseDisplayTerminated:
		return "l.status = 'terminated'"
	case domain.LeaseDisplayUpcoming:
		return "l.status = 'active' AND l.start_date > CURRENT_DATE"
	case domain.LeaseDisplayExpiringSoon:
		return fmt.Sprintf(
			"l.status = 'active' AND l.start_date <= CURRENT_DATE AND l.end_date IS NOT NULL AND l.end_date >= CURRENT_DATE AND l.end_date <= CURRENT_DATE + INTERVAL '%d days'",
			domain.LeaseExpiringSoonDays,
		)
	case domain.LeaseDisplayExpired:
		return "l.status = 'active' AND l.end_date IS NOT NULL AND l.end_date < CURRENT_DATE"
	default: // LeaseDisplayActive: started, not draft/terminated, and not caught by the expiring/expired windows above
		return fmt.Sprintf(
			"l.status = 'active' AND l.start_date <= CURRENT_DATE AND (l.end_date IS NULL OR l.end_date > CURRENT_DATE + INTERVAL '%d days')",
			domain.LeaseExpiringSoonDays,
		)
	}
}

func (r *LeaseRepository) ListForOwner(ctx context.Context, opts domain.LeaseListOptions) ([]*domain.LeaseWithUnitProperty, int, error) {
	// p.owner_id = $1 is the actual row-security boundary — see
	// UnitRepository.ListForOwner's identical rationale.
	where := []string{"p.owner_id = $1"}
	args := []any{opts.OwnerID}

	if opts.Filter.Search != "" {
		args = append(args, "%"+opts.Filter.Search+"%")
		where = append(where, fmt.Sprintf("(u.unit_name ILIKE $%d OR p.name ILIKE $%d OR l.primary_resident_name ILIKE $%d)", len(args), len(args), len(args)))
	}
	if opts.Filter.PropertyID != nil {
		args = append(args, *opts.Filter.PropertyID)
		where = append(where, fmt.Sprintf("p.id = $%d", len(args)))
	}
	if opts.Filter.UnitID != nil {
		args = append(args, *opts.Filter.UnitID)
		where = append(where, fmt.Sprintf("l.unit_id = $%d", len(args)))
	}
	if opts.Filter.DisplayStatus != nil {
		where = append(where, displayStatusWhere(*opts.Filter.DisplayStatus))
	}
	whereClause := strings.Join(where, " AND ")

	sortColumn, ok := leaseSortColumns[opts.Sort]
	if !ok {
		sortColumn = leaseSortColumns[domain.LeaseSortStartDate]
	}
	sortDir := "ASC"
	if opts.SortDesc {
		sortDir = "DESC"
	}

	args = append(args, opts.Limit, opts.Offset)
	q := fmt.Sprintf(
		`SELECT %s, u.unit_name, p.id AS property_id, p.name AS property_name
		 FROM leases l
		 JOIN units u ON u.id = l.unit_id
		 JOIN properties p ON p.id = u.property_id
		 WHERE %s
		 ORDER BY %s %s, l.id ASC
		 LIMIT $%d OFFSET $%d`,
		qualifiedLeaseColumns, whereClause, sortColumn, sortDir, len(args)-1, len(args),
	)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list leases for owner %s: %w", opts.OwnerID, err)
	}
	defer rows.Close()

	leases := make([]*domain.LeaseWithUnitProperty, 0)
	for rows.Next() {
		l, err := scanLeaseWithUnitProperty(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan lease-with-unit-property row: %w", err)
		}
		leases = append(leases, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate lease-with-unit-property rows: %w", err)
	}

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM leases l JOIN units u ON u.id = l.unit_id JOIN properties p ON p.id = u.property_id WHERE %s`, whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count leases for owner %s: %w", opts.OwnerID, err)
	}

	return leases, total, nil
}

func (r *LeaseRepository) Update(ctx context.Context, l *domain.Lease) error {
	const q = `
		UPDATE leases
		SET lease_type = $2, status = $3, start_date = $4, end_date = $5, move_in_date = $6, move_out_date = $7,
			monthly_rent = $8, security_deposit = $9, deposit_status = $10, rent_due_day = $11, late_fee_amount = $12,
			late_fee_grace_days = $13, primary_resident_name = $14, co_residents = $15, emergency_contact = $16,
			renewal_status = $17, termination_reason = $18, termination_notice_date = $19, signed = $20,
			signed_date = $21, notes = $22, updated_at = $23
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		l.ID, l.Type, l.Status, l.StartDate, l.EndDate, l.MoveInDate, l.MoveOutDate,
		l.MonthlyRent, l.SecurityDeposit, depositStatusArg(l.DepositStatus), l.RentDueDay, l.LateFeeAmount,
		l.LateFeeGraceDays, l.PrimaryResidentName, l.CoResidents, l.EmergencyContact,
		l.RenewalStatus, terminationReasonArg(l.TerminationReason), l.TerminationNoticeDate, l.Signed,
		l.SignedDate, l.Notes, l.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update lease %s: %w", l.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update lease %s: %w", l.ID, domain.ErrNotFound)
	}
	return nil
}

func (r *LeaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM leases WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		// Ledger rows reference leases with ON DELETE RESTRICT: a lease
		// with billing history is terminated, not deleted.
		if isForeignKeyViolation(err) {
			return fmt.Errorf("delete lease %s: has accounting history: %w", id, domain.ErrConflict)
		}
		return fmt.Errorf("delete lease %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete lease %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

func (r *LeaseRepository) HasActiveLease(ctx context.Context, unitID uuid.UUID, excludeLeaseID *uuid.UUID) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM leases WHERE unit_id = $1 AND status = 'active'`
	args := []any{unitID}
	if excludeLeaseID != nil {
		args = append(args, *excludeLeaseID)
		q += fmt.Sprintf(" AND id != $%d", len(args))
	}
	q += ")"

	var exists bool
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check active lease for unit %s: %w", unitID, err)
	}
	return exists, nil
}

func scanLease(row rowScanner) (*domain.Lease, error) {
	var l domain.Lease
	var endDate, moveInDate, moveOutDate, terminationNoticeDate, signedDate sql.NullTime
	var securityDeposit, lateFeeAmount sql.NullFloat64
	var depositStatus, terminationReason sql.NullString
	var rentDueDay, lateFeeGraceDays sql.NullInt32

	err := row.Scan(
		&l.ID, &l.UnitID, &l.Type, &l.Status, &l.StartDate, &endDate, &moveInDate, &moveOutDate,
		&l.MonthlyRent, &securityDeposit, &depositStatus, &rentDueDay, &lateFeeAmount, &lateFeeGraceDays,
		&l.PrimaryResidentName, &l.CoResidents, &l.EmergencyContact, &l.RenewalStatus, &terminationReason,
		&terminationNoticeDate, &l.Signed, &signedDate, &l.Notes, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	applyLeaseNullables(&l, endDate, moveInDate, moveOutDate, terminationNoticeDate, signedDate, securityDeposit, lateFeeAmount, depositStatus, terminationReason, rentDueDay, lateFeeGraceDays)
	return &l, nil
}

func scanLeaseWithUnitProperty(row rowScanner) (*domain.LeaseWithUnitProperty, error) {
	var l domain.LeaseWithUnitProperty
	var endDate, moveInDate, moveOutDate, terminationNoticeDate, signedDate sql.NullTime
	var securityDeposit, lateFeeAmount sql.NullFloat64
	var depositStatus, terminationReason sql.NullString
	var rentDueDay, lateFeeGraceDays sql.NullInt32

	err := row.Scan(
		&l.ID, &l.UnitID, &l.Type, &l.Status, &l.StartDate, &endDate, &moveInDate, &moveOutDate,
		&l.MonthlyRent, &securityDeposit, &depositStatus, &rentDueDay, &lateFeeAmount, &lateFeeGraceDays,
		&l.PrimaryResidentName, &l.CoResidents, &l.EmergencyContact, &l.RenewalStatus, &terminationReason,
		&terminationNoticeDate, &l.Signed, &signedDate, &l.Notes, &l.CreatedAt, &l.UpdatedAt,
		&l.UnitName, &l.PropertyID, &l.PropertyName,
	)
	if err != nil {
		return nil, err
	}
	applyLeaseNullables(&l.Lease, endDate, moveInDate, moveOutDate, terminationNoticeDate, signedDate, securityDeposit, lateFeeAmount, depositStatus, terminationReason, rentDueDay, lateFeeGraceDays)
	return &l, nil
}

// applyLeaseNullables centralizes the nullable-column handling shared
// by scanLease and scanLeaseWithUnitProperty — see
// property_repository.go's scanProperty for why pgx needs database/
// sql's Null* wrapper types as the intermediate scan target.
func applyLeaseNullables(
	l *domain.Lease,
	endDate, moveInDate, moveOutDate, terminationNoticeDate, signedDate sql.NullTime,
	securityDeposit, lateFeeAmount sql.NullFloat64,
	depositStatus, terminationReason sql.NullString,
	rentDueDay, lateFeeGraceDays sql.NullInt32,
) {
	if endDate.Valid {
		l.EndDate = &endDate.Time
	}
	if moveInDate.Valid {
		l.MoveInDate = &moveInDate.Time
	}
	if moveOutDate.Valid {
		l.MoveOutDate = &moveOutDate.Time
	}
	if terminationNoticeDate.Valid {
		l.TerminationNoticeDate = &terminationNoticeDate.Time
	}
	if signedDate.Valid {
		l.SignedDate = &signedDate.Time
	}
	if securityDeposit.Valid {
		v := securityDeposit.Float64
		l.SecurityDeposit = &v
	}
	if lateFeeAmount.Valid {
		v := lateFeeAmount.Float64
		l.LateFeeAmount = &v
	}
	if depositStatus.Valid {
		v := domain.DepositStatus(depositStatus.String)
		l.DepositStatus = &v
	}
	if terminationReason.Valid {
		v := domain.TerminationReason(terminationReason.String)
		l.TerminationReason = &v
	}
	if rentDueDay.Valid {
		v := int(rentDueDay.Int32)
		l.RentDueDay = &v
	}
	if lateFeeGraceDays.Valid {
		v := int(lateFeeGraceDays.Int32)
		l.LateFeeGraceDays = &v
	}
}

func depositStatusArg(s *domain.DepositStatus) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

func terminationReasonArg(r *domain.TerminationReason) *string {
	if r == nil {
		return nil
	}
	v := string(*r)
	return &v
}
