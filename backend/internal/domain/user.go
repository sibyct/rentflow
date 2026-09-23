package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRoleAdmin   UserRole = "admin"
	UserRoleManager UserRole = "manager"
	UserRoleTenant  UserRole = "tenant"
)

// User is the core business entity for an account holder. PasswordHash is
// never serialized to a transport DTO.
//
// AccountOwnerID nil means this row IS an account — the original
// one-user-per-account model, unchanged for anyone who never invites
// staff. Set, it means this row is staff invited by that account (see
// AccountID and domain/staff.go): every other table's owner_id keeps
// pointing at the root row, never at a staff row, so nothing about
// existing ownership checks changes.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	Role         UserRole
	CreatedAt    time.Time
	UpdatedAt    time.Time

	AccountOwnerID  *uuid.UUID
	StaffRole       *StaffRole
	Status          StaffStatus
	AllProperties   bool
	InvitedBy       *uuid.UUID
	InvitedAt       *time.Time
	InviteExpiresAt *time.Time
	InviteTokenHash string
	LastLoginAt     *time.Time
}

// AccountID is the portfolio this user acts within — its own id for a
// root account, or the root's id for staff. Every handler that scopes a
// resource by "owner" uses this, never u.ID directly, once staff exist.
func (u *User) AccountID() uuid.UUID {
	if u.AccountOwnerID != nil {
		return *u.AccountOwnerID
	}
	return u.ID
}

func (u *User) IsAccountOwner() bool { return u.AccountOwnerID == nil }

// DisplayStatus computes the richer status a staff list shows —
// "invite expired" — from InviteExpiresAt at read time, the same
// "derived, never stored" rationale as Lease.DisplayStatus.
func (u *User) DisplayStatus(now time.Time) StaffDisplayStatus {
	switch u.Status {
	case StaffStatusDeactivated:
		return StaffDisplayDeactivated
	case StaffStatusInvited:
		if u.InviteExpiresAt != nil && u.InviteExpiresAt.Before(now) {
			return StaffDisplayInviteExpired
		}
		return StaffDisplayInvited
	default:
		return StaffDisplayActive
	}
}

// UserRepository is the port implemented by internal/repository/postgres.
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	// TouchLastLogin best-effort-records when id last successfully
	// authenticated — never fatal to a login/refresh if it fails.
	TouchLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error
}

// AuthClaims is the framework-free result of validating an access token.
// The service layer translates JWT-library claims into this type so the
// domain and transport layers never import a JWT package directly.
//
// UserID is deliberately the ACCOUNT id — for a root account it's their
// own id; for staff it's the root's id — so every existing handler that
// scopes a resource by claims.UserID keeps working completely unchanged
// now that staff exist. ActorID is who is actually signed in (equal to
// UserID for a root account); StaffRole is nil for a root account
// (implicit full access) and set for staff. See domain/staff.go.
type AuthClaims struct {
	UserID    uuid.UUID
	Role      UserRole
	ActorID   uuid.UUID
	StaffRole *StaffRole
}

// IsAccountOwner reports whether the signed-in user is the account's
// root owner rather than invited staff.
func (c *AuthClaims) IsAccountOwner() bool { return c.ActorID == c.UserID }

// CanManageStaff reports whether the signed-in user may invite, edit,
// deactivate or reactivate other staff — the root account always can;
// staff can only if they were themselves given the Admin role.
func (c *AuthClaims) CanManageStaff() bool {
	return c.IsAccountOwner() || (c.StaffRole != nil && *c.StaffRole == StaffRoleAdmin)
}

// AuthService is the port implemented by internal/service and consumed by
// the HTTP transport layer (auth handler + auth middleware).
type AuthService interface {
	Register(ctx context.Context, email, password string) (*User, error)
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, user *User, err error)
	RefreshToken(ctx context.Context, oldRefreshToken string) (accessToken, refreshToken string, user *User, err error)
	Logout(ctx context.Context, refreshToken string) error
	ValidateAccessToken(ctx context.Context, tokenString string) (*AuthClaims, error)
}
