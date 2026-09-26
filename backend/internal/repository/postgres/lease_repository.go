package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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
	primary_resident_name, primary_resident_phone, primary_resident_email, co_residents, emergency_contact,
	renewal_status, proposed_rent, proposed_end_date, offer_sent_date, termination_reason, termination_notice_date,
	renewed_into_lease_id, renewed_from_lease_id, signed, signed_date, notes, created_at, updated_at`

// leaseColumnsForRead is leaseColumns with monthly_rent replaced by the
// *effective* rent: the latest lease_rent_history row whose
// effective_date has arrived, falling back to the raw stored
// leases.monthly_rent otherwise (see Lease.MonthlyRent's doc comment
// and units' identical unitColumnsForRead trick). Every read query in
// this file uses this rather than leaseColumns; writes (Create/Update)
// still target the raw leases.monthly_rent column directly, which is
// frozen at its creation-time value once a lease has ever gone active
// (see LeaseService.ChangeRent/UpdateLease).
const leaseColumnsForRead = `
	id, unit_id, lease_type, status, start_date, end_date, move_in_date, move_out_date,
	COALESCE(
		(SELECT h.amount FROM lease_rent_history h WHERE h.lease_id = leases.id AND h.effective_date <= CURRENT_DATE ORDER BY h.effective_date DESC, h.created_at DESC LIMIT 1),
		monthly_rent
	) AS monthly_rent,
	security_deposit, deposit_status, rent_due_day, late_fee_amount, late_fee_grace_days,
	primary_resident_name, primary_resident_phone, primary_resident_email, co_residents, emergency_contact,
	renewal_status, proposed_rent, proposed_end_date, offer_sent_date, termination_reason, termination_notice_date,
	renewed_into_lease_id, renewed_from_lease_id, signed, signed_date, notes, created_at, updated_at`

func (r *LeaseRepository) Create(ctx context.Context, l *domain.Lease) error {
	const q = `
		INSERT INTO leases (` + leaseColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32)`

	_, err := r.pool.Exec(ctx, q,
		l.ID, l.UnitID, l.Type, l.Status, l.StartDate, l.EndDate, l.MoveInDate, l.MoveOutDate,
		l.MonthlyRent, l.SecurityDeposit, depositStatusArg(l.DepositStatus), l.RentDueDay, l.LateFeeAmount, l.LateFeeGraceDays,
		l.PrimaryResidentName, l.PrimaryResidentPhone, l.PrimaryResidentEmail, l.CoResidents, l.EmergencyContact,
		l.RenewalStatus, l.ProposedRent, l.ProposedEndDate, l.OfferSentDate, terminationReasonArg(l.TerminationReason), l.TerminationNoticeDate,
		l.RenewedIntoLeaseID, l.RenewedFromLeaseID, l.Signed, l.SignedDate, l.Notes, l.CreatedAt, l.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert lease %s: %w", l.ID, err)
	}
	return nil
}

func (r *LeaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Lease, error) {
	const q = `SELECT ` + leaseColumnsForRead + ` FROM leases WHERE id = $1`

	l, err := scanLease(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get lease %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get lease %s: %w", id, err)
	}
	return l, nil
}

// qualifiedLeaseColumns is leaseColumnsForRead prefixed with "l." for
// ListForOwner's join — leases, units, and properties all have id,
// created_at, and updated_at, so an unqualified SELECT across the join
// would be ambiguous.
const qualifiedLeaseColumns = `
	l.id, l.unit_id, l.lease_type, l.status, l.start_date, l.end_date, l.move_in_date, l.move_out_date,
	COALESCE(
		(SELECT h.amount FROM lease_rent_history h WHERE h.lease_id = l.id AND h.effective_date <= CURRENT_DATE ORDER BY h.effective_date DESC, h.created_at DESC LIMIT 1),
		l.monthly_rent
	) AS monthly_rent,
	l.security_deposit, l.deposit_status, l.rent_due_day, l.late_fee_amount, l.late_fee_grace_days,
	l.primary_resident_name, l.primary_resident_phone, l.primary_resident_email, l.co_residents, l.emergency_contact,
	l.renewal_status, l.proposed_rent, l.proposed_end_date, l.offer_sent_date, l.termination_reason, l.termination_notice_date,
	l.renewed_into_lease_id, l.renewed_from_lease_id, l.signed, l.signed_date, l.notes, l.created_at, l.updated_at`

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
	if !opts.PropertyAccess.All {
		args = append(args, opts.PropertyAccess.PropertyIDs)
		where = append(where, fmt.Sprintf("p.id = ANY($%d)", len(args)))
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
	// monthly_rent is intentionally NOT written here (see
	// leaseColumnsForRead's doc comment) — once a lease has ever gone
	// active, LeaseService.UpdateLease rejects a direct MonthlyRent
	// change, so this raw column stays frozen at its creation-time
	// value and every later change flows through AppendRentChange
	// instead.
	const q = `
		UPDATE leases
		SET lease_type = $2, status = $3, start_date = $4, end_date = $5, move_in_date = $6, move_out_date = $7,
			security_deposit = $8, deposit_status = $9, rent_due_day = $10, late_fee_amount = $11,
			late_fee_grace_days = $12, primary_resident_name = $13, primary_resident_phone = $14, primary_resident_email = $15,
			co_residents = $16, emergency_contact = $17, renewal_status = $18, proposed_rent = $19, proposed_end_date = $20,
			offer_sent_date = $21, termination_reason = $22, termination_notice_date = $23, signed = $24, signed_date = $25,
			notes = $26, updated_at = $27
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		l.ID, l.Type, l.Status, l.StartDate, l.EndDate, l.MoveInDate, l.MoveOutDate,
		l.SecurityDeposit, depositStatusArg(l.DepositStatus), l.RentDueDay, l.LateFeeAmount,
		l.LateFeeGraceDays, l.PrimaryResidentName, l.PrimaryResidentPhone, l.PrimaryResidentEmail, l.CoResidents, l.EmergencyContact,
		l.RenewalStatus, l.ProposedRent, l.ProposedEndDate, l.OfferSentDate, terminationReasonArg(l.TerminationReason), l.TerminationNoticeDate,
		l.Signed, l.SignedDate, l.Notes, l.UpdatedAt,
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

func (r *LeaseRepository) UpdateMonthlyRent(ctx context.Context, leaseID uuid.UUID, amount float64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE leases SET monthly_rent = $2, updated_at = now() WHERE id = $1`, leaseID, amount)
	if err != nil {
		return fmt.Errorf("update monthly rent for lease %s: %w", leaseID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update monthly rent for lease %s: %w", leaseID, domain.ErrNotFound)
	}
	return nil
}

func (r *LeaseRepository) ListRentHistory(ctx context.Context, leaseID uuid.UUID) ([]*domain.LeaseRentHistoryEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, lease_id, amount, effective_date, reason, is_correction, created_by, created_at
		FROM lease_rent_history WHERE lease_id = $1 ORDER BY effective_date DESC, created_at DESC`, leaseID)
	if err != nil {
		return nil, fmt.Errorf("list rent history for lease %s: %w", leaseID, err)
	}
	defer rows.Close()

	entries := make([]*domain.LeaseRentHistoryEntry, 0)
	for rows.Next() {
		var e domain.LeaseRentHistoryEntry
		if err := rows.Scan(&e.ID, &e.LeaseID, &e.Amount, &e.EffectiveDate, &e.Reason, &e.IsCorrection, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan lease rent history row: %w", err)
		}
		entries = append(entries, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lease rent history rows: %w", err)
	}
	return entries, nil
}

// AppendRentChange never touches leases.monthly_rent — it only ever
// inserts a new lease_rent_history row plus its audit entry, in one
// transaction, matching LedgerRepository.CreateTransaction's
// "repository method takes the audit entry, writes atomically"
// convention.
func (r *LeaseRepository) AppendRentChange(ctx context.Context, entry *domain.LeaseRentHistoryEntry, audit domain.LeaseAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin append rent change for lease %s: %w", entry.LeaseID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO lease_rent_history (id, lease_id, amount, effective_date, reason, is_correction, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		entry.ID, entry.LeaseID, entry.Amount, entry.EffectiveDate, entry.Reason, entry.IsCorrection, entry.CreatedBy, entry.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert lease rent history for lease %s: %w", entry.LeaseID, err)
	}
	if err := insertLeaseAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit append rent change for lease %s: %w", entry.LeaseID, err)
	}
	return nil
}

// CreateRenewal retires sourceLeaseID (status -> terminated,
// renewed_into_lease_id set) before inserting newLease, in that order,
// within one transaction — the source must stop being 'active' before
// the new lease can become 'active', or the partial unique index
// idx_leases_one_active_per_unit would reject the insert.
func (r *LeaseRepository) CreateRenewal(ctx context.Context, newLease *domain.Lease, sourceLeaseID uuid.UUID, audit domain.LeaseAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create renewal for lease %s: %w", sourceLeaseID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE leases SET status = 'terminated', renewed_into_lease_id = $2, updated_at = $3 WHERE id = $1`,
		sourceLeaseID, newLease.ID, audit.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("retire source lease %s: %w", sourceLeaseID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("retire source lease %s: %w", sourceLeaseID, domain.ErrNotFound)
	}

	const q = `
		INSERT INTO leases (` + leaseColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32)`
	if _, err := tx.Exec(ctx, q,
		newLease.ID, newLease.UnitID, newLease.Type, newLease.Status, newLease.StartDate, newLease.EndDate, newLease.MoveInDate, newLease.MoveOutDate,
		newLease.MonthlyRent, newLease.SecurityDeposit, depositStatusArg(newLease.DepositStatus), newLease.RentDueDay, newLease.LateFeeAmount, newLease.LateFeeGraceDays,
		newLease.PrimaryResidentName, newLease.PrimaryResidentPhone, newLease.PrimaryResidentEmail, newLease.CoResidents, newLease.EmergencyContact,
		newLease.RenewalStatus, newLease.ProposedRent, newLease.ProposedEndDate, newLease.OfferSentDate, terminationReasonArg(newLease.TerminationReason), newLease.TerminationNoticeDate,
		newLease.RenewedIntoLeaseID, newLease.RenewedFromLeaseID, newLease.Signed, newLease.SignedDate, newLease.Notes, newLease.CreatedAt, newLease.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert renewal lease %s: %w", newLease.ID, err)
	}

	if err := insertLeaseAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create renewal for lease %s: %w", sourceLeaseID, err)
	}
	return nil
}

// Terminate atomically sets l's status/termination fields, sets unit's
// status/vacated_at, writes audit, and — when autoDocument is non-nil —
// inserts the auto-linked unit_documents row (R15), all in one
// transaction so a lease is never left terminated with its unit still
// showing occupied.
func (r *LeaseRepository) Terminate(ctx context.Context, l *domain.Lease, unit *domain.Unit, audit domain.LeaseAuditEntry, autoDocument *domain.UnitDocument) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin terminate lease %s: %w", l.ID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE leases SET status = $2, termination_reason = $3, termination_notice_date = $4, move_out_date = $5, updated_at = $6
		WHERE id = $1`,
		l.ID, l.Status, terminationReasonArg(l.TerminationReason), l.TerminationNoticeDate, l.MoveOutDate, l.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("terminate lease %s: %w", l.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("terminate lease %s: %w", l.ID, domain.ErrNotFound)
	}

	if _, err := tx.Exec(ctx, `UPDATE units SET status = $2, vacated_at = $3, updated_at = $4 WHERE id = $1`,
		unit.ID, unit.Status, unit.VacatedAt, unit.UpdatedAt,
	); err != nil {
		return fmt.Errorf("vacate unit %s: %w", unit.ID, err)
	}

	if err := insertLeaseAudit(ctx, tx, audit); err != nil {
		return err
	}

	if autoDocument != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO unit_documents (id, unit_id, attachment_id, category, uploaded_by, related_lease_id, is_automated, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			autoDocument.ID, autoDocument.UnitID, autoDocument.AttachmentID, autoDocument.Category, autoDocument.UploadedBy,
			autoDocument.RelatedLeaseID, autoDocument.IsAutomated, autoDocument.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert auto-generated unit document for lease %s: %w", l.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit terminate lease %s: %w", l.ID, err)
	}
	return nil
}

// CorrectTermination writes l's (already-corrected in memory)
// termination fields and audit in one transaction — no unit update,
// unlike Terminate: the unit's vacancy was already settled by the
// original termination and a correction never changes it.
func (r *LeaseRepository) CorrectTermination(ctx context.Context, l *domain.Lease, audit domain.LeaseAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin correct termination for lease %s: %w", l.ID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE leases SET termination_reason = $2, termination_notice_date = $3, move_out_date = $4, updated_at = $5
		WHERE id = $1`,
		l.ID, terminationReasonArg(l.TerminationReason), l.TerminationNoticeDate, l.MoveOutDate, l.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("correct termination for lease %s: %w", l.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("correct termination for lease %s: %w", l.ID, domain.ErrNotFound)
	}

	if err := insertLeaseAudit(ctx, tx, audit); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit correct termination for lease %s: %w", l.ID, err)
	}
	return nil
}

func (r *LeaseRepository) ListAudit(ctx context.Context, leaseID uuid.UUID) ([]*domain.LeaseAuditEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_id, lease_id, actor_id, action, changes, created_at
		FROM lease_audit_log WHERE lease_id = $1 ORDER BY created_at DESC`, leaseID)
	if err != nil {
		return nil, fmt.Errorf("list audit for lease %s: %w", leaseID, err)
	}
	defer rows.Close()

	entries := make([]*domain.LeaseAuditEntry, 0)
	for rows.Next() {
		var e domain.LeaseAuditEntry
		var changes []byte
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.LeaseID, &e.ActorID, &e.Action, &changes, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan lease audit row: %w", err)
		}
		if err := json.Unmarshal(changes, &e.Changes); err != nil {
			return nil, fmt.Errorf("unmarshal lease audit changes: %w", err)
		}
		entries = append(entries, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lease audit rows: %w", err)
	}
	return entries, nil
}

func insertLeaseAudit(ctx context.Context, q execer, a domain.LeaseAuditEntry) error {
	changes, err := json.Marshal(a.Changes)
	if err != nil {
		return fmt.Errorf("marshal lease audit changes: %w", err)
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO lease_audit_log (id, owner_id, lease_id, actor_id, action, changes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		a.ID, a.OwnerID, a.LeaseID, a.ActorID, a.Action, changes, a.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert lease audit entry: %w", err)
	}
	return nil
}

func scanLease(row rowScanner) (*domain.Lease, error) {
	var l domain.Lease
	var endDate, moveInDate, moveOutDate, proposedEndDate, offerSentDate, terminationNoticeDate, signedDate sql.NullTime
	var securityDeposit, lateFeeAmount, proposedRent sql.NullFloat64
	var depositStatus, terminationReason sql.NullString
	var rentDueDay, lateFeeGraceDays sql.NullInt32
	var renewedIntoLeaseID, renewedFromLeaseID uuid.NullUUID

	err := row.Scan(
		&l.ID, &l.UnitID, &l.Type, &l.Status, &l.StartDate, &endDate, &moveInDate, &moveOutDate,
		&l.MonthlyRent, &securityDeposit, &depositStatus, &rentDueDay, &lateFeeAmount, &lateFeeGraceDays,
		&l.PrimaryResidentName, &l.PrimaryResidentPhone, &l.PrimaryResidentEmail, &l.CoResidents, &l.EmergencyContact,
		&l.RenewalStatus, &proposedRent, &proposedEndDate, &offerSentDate, &terminationReason, &terminationNoticeDate,
		&renewedIntoLeaseID, &renewedFromLeaseID, &l.Signed, &signedDate, &l.Notes, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	applyLeaseNullables(&l, endDate, moveInDate, moveOutDate, proposedEndDate, offerSentDate, terminationNoticeDate, signedDate,
		securityDeposit, lateFeeAmount, proposedRent, depositStatus, terminationReason, rentDueDay, lateFeeGraceDays,
		renewedIntoLeaseID, renewedFromLeaseID)
	return &l, nil
}

func scanLeaseWithUnitProperty(row rowScanner) (*domain.LeaseWithUnitProperty, error) {
	var l domain.LeaseWithUnitProperty
	var endDate, moveInDate, moveOutDate, proposedEndDate, offerSentDate, terminationNoticeDate, signedDate sql.NullTime
	var securityDeposit, lateFeeAmount, proposedRent sql.NullFloat64
	var depositStatus, terminationReason sql.NullString
	var rentDueDay, lateFeeGraceDays sql.NullInt32
	var renewedIntoLeaseID, renewedFromLeaseID uuid.NullUUID

	err := row.Scan(
		&l.ID, &l.UnitID, &l.Type, &l.Status, &l.StartDate, &endDate, &moveInDate, &moveOutDate,
		&l.MonthlyRent, &securityDeposit, &depositStatus, &rentDueDay, &lateFeeAmount, &lateFeeGraceDays,
		&l.PrimaryResidentName, &l.PrimaryResidentPhone, &l.PrimaryResidentEmail, &l.CoResidents, &l.EmergencyContact,
		&l.RenewalStatus, &proposedRent, &proposedEndDate, &offerSentDate, &terminationReason, &terminationNoticeDate,
		&renewedIntoLeaseID, &renewedFromLeaseID, &l.Signed, &signedDate, &l.Notes, &l.CreatedAt, &l.UpdatedAt,
		&l.UnitName, &l.PropertyID, &l.PropertyName,
	)
	if err != nil {
		return nil, err
	}
	applyLeaseNullables(&l.Lease, endDate, moveInDate, moveOutDate, proposedEndDate, offerSentDate, terminationNoticeDate, signedDate,
		securityDeposit, lateFeeAmount, proposedRent, depositStatus, terminationReason, rentDueDay, lateFeeGraceDays,
		renewedIntoLeaseID, renewedFromLeaseID)
	return &l, nil
}

// applyLeaseNullables centralizes the nullable-column handling shared
// by scanLease and scanLeaseWithUnitProperty — see
// property_repository.go's scanProperty for why pgx needs database/
// sql's Null* wrapper types as the intermediate scan target.
func applyLeaseNullables(
	l *domain.Lease,
	endDate, moveInDate, moveOutDate, proposedEndDate, offerSentDate, terminationNoticeDate, signedDate sql.NullTime,
	securityDeposit, lateFeeAmount, proposedRent sql.NullFloat64,
	depositStatus, terminationReason sql.NullString,
	rentDueDay, lateFeeGraceDays sql.NullInt32,
	renewedIntoLeaseID, renewedFromLeaseID uuid.NullUUID,
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
	if proposedEndDate.Valid {
		l.ProposedEndDate = &proposedEndDate.Time
	}
	if offerSentDate.Valid {
		l.OfferSentDate = &offerSentDate.Time
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
	if proposedRent.Valid {
		v := proposedRent.Float64
		l.ProposedRent = &v
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
	if renewedIntoLeaseID.Valid {
		v := renewedIntoLeaseID.UUID
		l.RenewedIntoLeaseID = &v
	}
	if renewedFromLeaseID.Valid {
		v := renewedFromLeaseID.UUID
		l.RenewedFromLeaseID = &v
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
