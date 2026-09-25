package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type StaffRepository struct {
	pool *pgxpool.Pool
}

func NewStaffRepository(pool *pgxpool.Pool) *StaffRepository {
	return &StaffRepository{pool: pool}
}

var _ domain.StaffRepository = (*StaffRepository)(nil)

func insertAccountAudit(ctx context.Context, q execer, a domain.StaffAuditEntry) error {
	changes, err := json.Marshal(a.Changes)
	if err != nil {
		return fmt.Errorf("marshal account audit changes: %w", err)
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO account_audit_log (id, account_owner_id, actor_id, action, subject_user_id, subject_name, changes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		a.ID, a.AccountOwnerID, a.ActorID, a.Action, a.SubjectUserID, a.SubjectName, changes, a.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert account audit entry: %w", err)
	}
	return nil
}

func (r *StaffRepository) CreateInvite(ctx context.Context, u *domain.User, propertyIDs []uuid.UUID, audit domain.StaffAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create invite: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (
			id, email, name, password_hash, role, account_owner_id, staff_role, status, all_properties,
			invited_by, invited_at, invite_expires_at, invite_token_hash, created_at, updated_at
		) VALUES ($1, $2, $3, '', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)`,
		u.ID, u.Email, u.Name, domain.UserRoleManager, u.AccountOwnerID, string(*u.StaffRole), string(u.Status), u.AllProperties,
		u.InvitedBy, u.InvitedAt, u.InviteExpiresAt, u.InviteTokenHash, u.CreatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("insert invite %s: %w", u.Email, domain.ErrAlreadyExists)
		}
		return fmt.Errorf("insert invite %s: %w", u.Email, err)
	}
	if err := insertPropertyAccess(ctx, tx, u.ID, propertyIDs); err != nil {
		return err
	}
	if err := insertAccountAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create invite: %w", err)
	}
	return nil
}

func insertPropertyAccess(ctx context.Context, q execer, userID uuid.UUID, propertyIDs []uuid.UUID) error {
	for _, propertyID := range propertyIDs {
		if _, err := q.Exec(ctx, `INSERT INTO staff_property_access (user_id, property_id) VALUES ($1, $2)`, userID, propertyID); err != nil {
			return fmt.Errorf("insert staff property access for %s: %w", userID, err)
		}
	}
	return nil
}

func (r *StaffRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get user %s: %w", id, err)
	}
	return u, nil
}

func (r *StaffRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by email %s: %w", email, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get user by email %s: %w", email, err)
	}
	return u, nil
}

func (r *StaffRepository) GetByInviteTokenHash(ctx context.Context, hash string) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE invite_token_hash = $1`, hash))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by invite token: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get user by invite token: %w", err)
	}
	return u, nil
}

func (r *StaffRepository) GetByPasswordResetTokenHash(ctx context.Context, hash string) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE password_reset_token_hash = $1`, hash))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by password reset token: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get user by password reset token: %w", err)
	}
	return u, nil
}

const staffListSelect = `
	SELECT u.id, u.email, u.name, u.password_hash, u.role, u.account_owner_id, u.staff_role, u.status, u.all_properties,
	       u.invited_by, u.invited_at, u.invite_expires_at, u.invite_token_hash, u.last_login_at, u.created_at, u.updated_at,
	       COALESCE(NULLIF(inviter.name, ''), inviter.email, '')
	FROM users u
	LEFT JOIN users inviter ON inviter.id = u.invited_by`

func scanStaffMember(row rowScanner) (*domain.StaffMember, error) {
	var m domain.StaffMember
	u := &m.User
	var role, status string
	var staffRole sql.NullString
	var accountOwner, invitedBy uuid.NullUUID
	var invitedAt, inviteExpiresAt, lastLoginAt sql.NullTime
	var inviteTokenHash sql.NullString

	if err := row.Scan(
		&u.ID, &u.Email, &u.Name, &u.PasswordHash, &role, &accountOwner, &staffRole, &status, &u.AllProperties,
		&invitedBy, &invitedAt, &inviteExpiresAt, &inviteTokenHash, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt, &m.InvitedByName,
	); err != nil {
		return nil, err
	}
	u.Role = domain.UserRole(role)
	u.Status = domain.StaffStatus(status)
	u.AccountOwnerID = nullUUIDPtr(accountOwner)
	u.InvitedBy = nullUUIDPtr(invitedBy)
	u.InvitedAt = nullTimePtr(invitedAt)
	u.InviteExpiresAt = nullTimePtr(inviteExpiresAt)
	u.LastLoginAt = nullTimePtr(lastLoginAt)
	if staffRole.Valid {
		sr := domain.StaffRole(staffRole.String)
		u.StaffRole = &sr
	}
	if inviteTokenHash.Valid {
		u.InviteTokenHash = inviteTokenHash.String
	}
	m.IsAccountOwner = u.AccountOwnerID == nil
	m.PropertyAccess = domain.PropertyAccess{All: u.IsAccountOwner() || u.AllProperties}
	return &m, nil
}

// attachPropertyAccess fills in PropertyIDs/PropertyNames for every
// member whose access isn't "all" — one query for the whole list rather
// than one per row.
func (r *StaffRepository) attachPropertyAccess(ctx context.Context, members []*domain.StaffMember) error {
	var scoped []uuid.UUID
	byID := map[uuid.UUID]*domain.StaffMember{}
	for _, m := range members {
		if !m.PropertyAccess.All {
			scoped = append(scoped, m.ID)
			byID[m.ID] = m
		}
	}
	if len(scoped) == 0 {
		return nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT spa.user_id, p.id, p.name FROM staff_property_access spa
		JOIN properties p ON p.id = spa.property_id
		WHERE spa.user_id = ANY($1) ORDER BY p.name`, scoped)
	if err != nil {
		return fmt.Errorf("list staff property access: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID, propertyID uuid.UUID
		var propertyName string
		if err := rows.Scan(&userID, &propertyID, &propertyName); err != nil {
			return fmt.Errorf("scan staff property access: %w", err)
		}
		if m, ok := byID[userID]; ok {
			m.PropertyAccess.PropertyIDs = append(m.PropertyAccess.PropertyIDs, propertyID)
			m.PropertyNames = append(m.PropertyNames, propertyName)
		}
	}
	return rows.Err()
}

// GetPropertyAccess reads one user's current property scope — the
// single-user counterpart to attachPropertyAccess, used per-request by
// the ResolvePropertyAccess middleware rather than the Users & Roles
// list screen. A root account row (account_owner_id NULL) or an
// Admin-role staff row always resolves to all-access, same as
// scanStaffMember/the invite-and-update-time forcing in StaffService —
// this is a read of that same stored state, not a separate rule.
func (r *StaffRepository) GetPropertyAccess(ctx context.Context, userID uuid.UUID) (domain.PropertyAccess, error) {
	var accountOwnerID uuid.NullUUID
	var staffRole sql.NullString
	var allProperties bool
	if err := r.pool.QueryRow(ctx, `SELECT account_owner_id, staff_role, all_properties FROM users WHERE id = $1`, userID).
		Scan(&accountOwnerID, &staffRole, &allProperties); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PropertyAccess{}, domain.ErrNotFound
		}
		return domain.PropertyAccess{}, fmt.Errorf("get user %s property access: %w", userID, err)
	}
	if !accountOwnerID.Valid || (staffRole.Valid && domain.StaffRole(staffRole.String) == domain.StaffRoleAdmin) || allProperties {
		return domain.AllPropertyAccess(), nil
	}

	rows, err := r.pool.Query(ctx, `SELECT property_id FROM staff_property_access WHERE user_id = $1`, userID)
	if err != nil {
		return domain.PropertyAccess{}, fmt.Errorf("list property access for user %s: %w", userID, err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return domain.PropertyAccess{}, fmt.Errorf("scan property access row: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return domain.PropertyAccess{}, err
	}
	return domain.PropertyAccess{PropertyIDs: ids}, nil
}

func (r *StaffRepository) ListForAccount(ctx context.Context, accountOwnerID uuid.UUID) ([]*domain.StaffMember, error) {
	rows, err := r.pool.Query(ctx, staffListSelect+`
		WHERE u.id = $1 OR u.account_owner_id = $1
		ORDER BY (u.account_owner_id IS NULL) DESC, u.created_at`, accountOwnerID)
	if err != nil {
		return nil, fmt.Errorf("list staff for account %s: %w", accountOwnerID, err)
	}
	defer rows.Close()

	var out []*domain.StaffMember
	for rows.Next() {
		m, err := scanStaffMember(rows)
		if err != nil {
			return nil, fmt.Errorf("scan staff member: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.attachPropertyAccess(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StaffRepository) GetMember(ctx context.Context, accountOwnerID, userID uuid.UUID) (*domain.StaffMember, error) {
	m, err := scanStaffMember(r.pool.QueryRow(ctx, staffListSelect+` WHERE u.id = $1 AND (u.id = $2 OR u.account_owner_id = $2)`, userID, accountOwnerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get staff member %s: %w", userID, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get staff member %s: %w", userID, err)
	}
	if err := r.attachPropertyAccess(ctx, []*domain.StaffMember{m}); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *StaffRepository) UpdateRoleAndAccess(ctx context.Context, userID uuid.UUID, role domain.StaffRole, access domain.PropertyAccess, updatedAt time.Time, audit domain.StaffAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update staff %s: %w", userID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE users SET staff_role = $2, all_properties = $3, updated_at = $4 WHERE id = $1 AND account_owner_id IS NOT NULL`,
		userID, string(role), access.All, updatedAt)
	if err != nil {
		return fmt.Errorf("update staff %s: %w", userID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update staff %s: %w", userID, domain.ErrNotFound)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM staff_property_access WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clear staff property access for %s: %w", userID, err)
	}
	if !access.All {
		if err := insertPropertyAccess(ctx, tx, userID, access.PropertyIDs); err != nil {
			return err
		}
	}
	if err := insertAccountAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update staff %s: %w", userID, err)
	}
	return nil
}

func (r *StaffRepository) SetStatus(ctx context.Context, userID uuid.UUID, status domain.StaffStatus, updatedAt time.Time, audit domain.StaffAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin set status for %s: %w", userID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE users SET status = $2, updated_at = $3 WHERE id = $1 AND account_owner_id IS NOT NULL`, userID, string(status), updatedAt)
	if err != nil {
		return fmt.Errorf("set status for %s: %w", userID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set status for %s: %w", userID, domain.ErrNotFound)
	}
	if err := insertAccountAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit set status for %s: %w", userID, err)
	}
	return nil
}

func (r *StaffRepository) ResendInvite(ctx context.Context, userID uuid.UUID, tokenHash string, invitedAt, expiresAt time.Time, audit domain.StaffAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin resend invite for %s: %w", userID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE users SET status = 'invited', invite_token_hash = $2, invited_at = $3, invite_expires_at = $4, updated_at = $3
		WHERE id = $1 AND account_owner_id IS NOT NULL`, userID, tokenHash, invitedAt, expiresAt)
	if err != nil {
		return fmt.Errorf("resend invite for %s: %w", userID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("resend invite for %s: %w", userID, domain.ErrNotFound)
	}
	if err := insertAccountAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit resend invite for %s: %w", userID, err)
	}
	return nil
}

// SetPasswordResetToken is admin-triggered (see the account_owner_id
// guard below, matching ResendInvite/SetStatus) — it never touches
// status or role, only stashes a token an already-active user can
// exchange for a new password.
func (r *StaffRepository) SetPasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time, audit domain.StaffAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin set password reset token for %s: %w", userID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE users SET password_reset_token_hash = $2, password_reset_expires_at = $3
		WHERE id = $1 AND account_owner_id IS NOT NULL`, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("set password reset token for %s: %w", userID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set password reset token for %s: %w", userID, domain.ErrNotFound)
	}
	if err := insertAccountAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit set password reset token for %s: %w", userID, err)
	}
	return nil
}

// ConfirmPasswordReset has no status/account_owner_id guard, unlike
// SetPasswordResetToken above — the token itself (verified by the
// service before this is called) is the only authorization needed,
// same as AcceptInvite below.
func (r *StaffRepository) ConfirmPasswordReset(ctx context.Context, userID uuid.UUID, passwordHash string, resetAt time.Time, audit domain.StaffAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin confirm password reset for %s: %w", userID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $2, password_reset_token_hash = NULL, password_reset_expires_at = NULL, updated_at = $3
		WHERE id = $1`, userID, passwordHash, resetAt)
	if err != nil {
		return fmt.Errorf("confirm password reset for %s: %w", userID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("confirm password reset for %s: %w", userID, domain.ErrNotFound)
	}
	if err := insertAccountAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit confirm password reset for %s: %w", userID, err)
	}
	return nil
}

func (r *StaffRepository) AcceptInvite(ctx context.Context, userID uuid.UUID, passwordHash string, acceptedAt time.Time, audit domain.StaffAuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin accept invite for %s: %w", userID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $2, status = 'active', invite_token_hash = NULL, invite_expires_at = NULL, updated_at = $3
		WHERE id = $1 AND status = 'invited'`, userID, passwordHash, acceptedAt)
	if err != nil {
		return fmt.Errorf("accept invite for %s: %w", userID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("accept invite for %s: %w", userID, domain.ErrConflict)
	}
	if err := insertAccountAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit accept invite for %s: %w", userID, err)
	}
	return nil
}

func (r *StaffRepository) CountActiveAdmins(ctx context.Context, accountOwnerID uuid.UUID) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE account_owner_id = $1 AND staff_role = 'admin' AND status = 'active'`, accountOwnerID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count active admins for %s: %w", accountOwnerID, err)
	}
	return n, nil
}

func (r *StaffRepository) ListAudit(ctx context.Context, accountOwnerID uuid.UUID, limit, offset int) ([]*domain.StaffAuditEntry, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM account_audit_log WHERE account_owner_id = $1`, accountOwnerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count account audit log: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.account_owner_id, a.actor_id, COALESCE(NULLIF(actor.name, ''), actor.email, ''), a.action, a.subject_user_id, a.subject_name, a.changes, a.created_at
		FROM account_audit_log a
		JOIN users actor ON actor.id = a.actor_id
		WHERE a.account_owner_id = $1
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $2 OFFSET $3`, accountOwnerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list account audit log: %w", err)
	}
	defer rows.Close()

	var out []*domain.StaffAuditEntry
	for rows.Next() {
		var e domain.StaffAuditEntry
		var subject uuid.NullUUID
		var raw []byte
		if err := rows.Scan(&e.ID, &e.AccountOwnerID, &e.ActorID, &e.ActorName, &e.Action, &subject, &e.SubjectName, &raw, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan account audit entry: %w", err)
		}
		e.SubjectUserID = nullUUIDPtr(subject)
		if err := json.Unmarshal(raw, &e.Changes); err != nil {
			return nil, 0, fmt.Errorf("unmarshal account audit changes: %w", err)
		}
		out = append(out, &e)
	}
	return out, total, rows.Err()
}
