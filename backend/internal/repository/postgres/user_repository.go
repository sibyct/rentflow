package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"propertymanagement/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

var _ domain.UserRepository = (*UserRepository)(nil)

const uniqueViolationCode = "23505"

const userColumns = `
	id, email, name, password_hash, role, account_owner_id, staff_role, status, all_properties,
	invited_by, invited_at, invite_expires_at, invite_token_hash, last_login_at, created_at, updated_at,
	password_reset_token_hash, password_reset_expires_at`

func scanUser(row rowScanner) (*domain.User, error) {
	var u domain.User
	var role, status string
	var staffRole sql.NullString
	var accountOwner, invitedBy uuid.NullUUID
	var invitedAt, inviteExpiresAt, lastLoginAt, passwordResetExpiresAt sql.NullTime
	var inviteTokenHash, passwordResetTokenHash sql.NullString

	if err := row.Scan(
		&u.ID, &u.Email, &u.Name, &u.PasswordHash, &role, &accountOwner, &staffRole, &status, &u.AllProperties,
		&invitedBy, &invitedAt, &inviteExpiresAt, &inviteTokenHash, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		&passwordResetTokenHash, &passwordResetExpiresAt,
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
		r := domain.StaffRole(staffRole.String)
		u.StaffRole = &r
	}
	if inviteTokenHash.Valid {
		u.InviteTokenHash = inviteTokenHash.String
	}
	if passwordResetTokenHash.Valid {
		u.PasswordResetTokenHash = passwordResetTokenHash.String
	}
	u.PasswordResetExpiresAt = nullTimePtr(passwordResetExpiresAt)
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (id, email, name, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	status := u.Status
	if status == "" {
		status = domain.StaffStatusActive
	}
	_, err := r.pool.Exec(ctx, q, u.ID, u.Email, u.Name, u.PasswordHash, u.Role, status, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("insert user %s: %w", u.Email, domain.ErrAlreadyExists)
		}
		return fmt.Errorf("insert user %s: %w", u.Email, err)
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by email %s: %w", email, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get user by email %s: %w", email, err)
	}
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by id %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get user by id %s: %w", id, err)
	}
	return u, nil
}

func (r *UserRepository) TouchLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	if _, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = $2 WHERE id = $1`, id, at); err != nil {
		return fmt.Errorf("touch last login for %s: %w", id, err)
	}
	return nil
}
