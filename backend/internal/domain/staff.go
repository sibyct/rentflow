package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// StaffRole is who a staff member is within the account — never set on
// the root account row itself (it's implicitly full admin; see
// User.AccountID). Distinct from the older, coarser UserRole, which
// keeps meaning whatever it always has for a root account.
type StaffRole string

const (
	StaffRoleAdmin                  StaffRole = "admin"
	StaffRolePropertyManager        StaffRole = "property_manager"
	StaffRoleMaintenanceCoordinator StaffRole = "maintenance_coordinator"
	StaffRoleAccountant             StaffRole = "accountant"
)

func (r StaffRole) Valid() bool {
	switch r {
	case StaffRoleAdmin, StaffRolePropertyManager, StaffRoleMaintenanceCoordinator, StaffRoleAccountant:
		return true
	default:
		return false
	}
}

// StaffStatus is the state someone actually sets by taking an action.
// See User.DisplayStatus for the richer, derived "invite expired" state
// this deliberately excludes.
type StaffStatus string

const (
	StaffStatusActive      StaffStatus = "active"
	StaffStatusInvited     StaffStatus = "invited"
	StaffStatusDeactivated StaffStatus = "deactivated"
)

type StaffDisplayStatus string

const (
	StaffDisplayActive        StaffDisplayStatus = "active"
	StaffDisplayInvited       StaffDisplayStatus = "invited"
	StaffDisplayInviteExpired StaffDisplayStatus = "invite_expired"
	StaffDisplayDeactivated   StaffDisplayStatus = "deactivated"
)

// InviteTokenTTL is how long an invite link stays usable.
const InviteTokenTTL = 7 * 24 * time.Hour

// PasswordResetTokenTTL is how long an admin-triggered reset link stays
// usable — shorter than InviteTokenTTL since it grants access to an
// already-active account rather than just starting one.
const PasswordResetTokenTTL = 24 * time.Hour

// PropertyAccess is which of an account's properties a staff member can
// see. All true means every property, including ones added later —
// PropertyIDs is only meaningful when All is false. A root account (and
// an Admin-role staff member) always effectively has All access; this
// type only ever describes a non-admin staff member's actual scope.
type PropertyAccess struct {
	All         bool
	PropertyIDs []uuid.UUID
}

func AllPropertyAccess() PropertyAccess { return PropertyAccess{All: true} }

// StaffMember is the read model the Users & roles screen lists — a User
// row plus its resolved property access and display context.
type StaffMember struct {
	User
	IsAccountOwner bool
	PropertyAccess PropertyAccess
	InvitedByName  string
	PropertyNames  []string // display names for PropertyAccess.PropertyIDs, same order
}

func (m *StaffMember) DisplayStatus(now time.Time) StaffDisplayStatus {
	if m.IsAccountOwner {
		return StaffDisplayActive
	}
	return m.User.DisplayStatus(now)
}

// EffectiveRole is what a staff list shows in the Role column — the
// account owner is always "Admin" even though StaffRole is never stored
// on that row.
func (m *StaffMember) EffectiveRole() StaffRole {
	if m.IsAccountOwner || m.StaffRole == nil {
		return StaffRoleAdmin
	}
	return *m.StaffRole
}

type InviteStaffInput struct {
	Name           string
	Email          string
	Role           StaffRole
	PropertyAccess PropertyAccess
}

// UpdateStaffInput is a full replace of a staff member's role and
// property scope — mirrors UpdateVendorInput's PropertiesServed
// nil-vs-empty precedent, except here both fields are always sent
// together since the edit drawer always submits both.
type UpdateStaffInput struct {
	Role           StaffRole
	PropertyAccess PropertyAccess
}

type AcceptInviteInput struct {
	Token    string
	Password string
}

// InviteLookup is what the (unauthenticated) accept-invite page reads
// before showing its form.
type InviteLookup struct {
	Name           string
	Email          string
	Role           StaffRole
	PropertyAccess PropertyAccess
	PropertyNames  []string
	InvitedByName  string
	Expired        bool
}

type ConfirmPasswordResetInput struct {
	Token    string
	Password string
}

// PasswordResetLookup is what the (unauthenticated) reset-password page
// reads before showing its form — mirrors InviteLookup, minus the
// role/access fields an invite needs to preview and a reset doesn't.
type PasswordResetLookup struct {
	Email   string
	Expired bool
}

// StaffAuditEntry mirrors accounting's AuditEntry shape but for the
// separate account_audit_log table — who can log in and act, not money.
type StaffAuditEntry struct {
	ID             uuid.UUID
	AccountOwnerID uuid.UUID
	ActorID        uuid.UUID
	ActorName      string
	Action         string
	SubjectUserID  *uuid.UUID
	SubjectName    string
	Changes        map[string]FieldChange
	CreatedAt      time.Time
}

func NewStaffAuditEntry(accountOwnerID, actorID uuid.UUID, action string, subjectUserID *uuid.UUID, subjectName string, changes map[string]FieldChange, now time.Time) StaffAuditEntry {
	if changes == nil {
		changes = map[string]FieldChange{}
	}
	return StaffAuditEntry{
		ID: uuid.New(), AccountOwnerID: accountOwnerID, ActorID: actorID, Action: action,
		SubjectUserID: subjectUserID, SubjectName: subjectName, Changes: changes, CreatedAt: now,
	}
}

// StaffRepository is the port implemented by internal/repository/postgres.
// Every method is scoped by accountOwnerID (the root account's id) —
// never by an individual staff member's own row id — so one account's
// staff can never see or touch another's.
type StaffRepository interface {
	CreateInvite(ctx context.Context, u *User, propertyIDs []uuid.UUID, audit StaffAuditEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	// GetByEmail looks up any user by email regardless of which account
	// they belong to — used only to detect a duplicate invite; callers
	// must check AccountID() before treating the result as belonging to
	// the caller's own account.
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByInviteTokenHash(ctx context.Context, hash string) (*User, error)
	GetByPasswordResetTokenHash(ctx context.Context, hash string) (*User, error)
	ListForAccount(ctx context.Context, accountOwnerID uuid.UUID) ([]*StaffMember, error)
	// GetMember returns one member of accountOwnerID's staff list (the
	// root row included), or ErrNotFound if userID belongs to a
	// different account entirely — the ownership check IS the query.
	GetMember(ctx context.Context, accountOwnerID, userID uuid.UUID) (*StaffMember, error)

	UpdateRoleAndAccess(ctx context.Context, userID uuid.UUID, role StaffRole, access PropertyAccess, updatedAt time.Time, audit StaffAuditEntry) error
	SetStatus(ctx context.Context, userID uuid.UUID, status StaffStatus, updatedAt time.Time, audit StaffAuditEntry) error
	ResendInvite(ctx context.Context, userID uuid.UUID, tokenHash string, invitedAt, expiresAt time.Time, audit StaffAuditEntry) error
	AcceptInvite(ctx context.Context, userID uuid.UUID, passwordHash string, acceptedAt time.Time, audit StaffAuditEntry) error
	SetPasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time, audit StaffAuditEntry) error
	ConfirmPasswordReset(ctx context.Context, userID uuid.UUID, passwordHash string, resetAt time.Time, audit StaffAuditEntry) error

	// CountActiveAdmins counts STAFF rows only (staff_role = admin,
	// status = active) — the root account is never included, since it
	// can never be deactivated through this API and so never needs to be
	// "protected" by the count. See StaffService's last-active-admin note.
	CountActiveAdmins(ctx context.Context, accountOwnerID uuid.UUID) (int, error)

	ListAudit(ctx context.Context, accountOwnerID uuid.UUID, limit, offset int) ([]*StaffAuditEntry, int, error)

	// GetPropertyAccess is a lightweight version of attachPropertyAccess
	// for exactly one user — used per-request by ResolvePropertyAccess
	// middleware, so it reads only users.all_properties plus
	// staff_property_access, no StaffMember/property-name join.
	GetPropertyAccess(ctx context.Context, userID uuid.UUID) (PropertyAccess, error)
}

// StaffInviteResult is exactly one of Created or Conflict — never both —
// so the handler can tell a fresh invite from a duplicate-email match
// without the service reaching for the generic error channel for what
// is, from the caller's point of view, a normal outcome to handle
// (offer Resend/Reactivate), not a failure.
type StaffInviteResult struct {
	Created   *StaffMember
	Conflict  *StaffMember
	InviteURL string // only set when Created != nil
}

type StaffService interface {
	Invite(ctx context.Context, accountOwnerID, actorID uuid.UUID, input InviteStaffInput) (*StaffInviteResult, error)
	List(ctx context.Context, accountOwnerID uuid.UUID) ([]*StaffMember, error)
	Update(ctx context.Context, accountOwnerID, actorID, userID uuid.UUID, input UpdateStaffInput) (*StaffMember, error)
	Deactivate(ctx context.Context, accountOwnerID, actorID, userID uuid.UUID) error
	Reactivate(ctx context.Context, accountOwnerID, actorID, userID uuid.UUID) error
	ResendInvite(ctx context.Context, accountOwnerID, actorID, userID uuid.UUID) (inviteURL string, err error)
	AuditLog(ctx context.Context, accountOwnerID uuid.UUID, limit, offset int) ([]*StaffAuditEntry, int, error)
	// ResetPassword is admin-triggered, for a staff member who's locked
	// themselves out — same manageable-staff/root-owner guard as
	// Deactivate/ResendInvite, never the account owner's own row (see
	// AcceptInvitePage's note that self-service reset isn't available
	// yet — this only ever resets someone ELSE's password).
	ResetPassword(ctx context.Context, accountOwnerID, actorID, userID uuid.UUID) (resetURL string, err error)

	// LookupInvite / AcceptInvite and LookupPasswordReset /
	// ConfirmPasswordReset are unauthenticated — reached from the link
	// in the invite/reset email, before the recipient has signed in.
	LookupInvite(ctx context.Context, token string) (*InviteLookup, error)
	AcceptInvite(ctx context.Context, input AcceptInviteInput) error
	LookupPasswordReset(ctx context.Context, token string) (*PasswordResetLookup, error)
	ConfirmPasswordReset(ctx context.Context, input ConfirmPasswordResetInput) error

	// ResolveAccess computes actorID's current property scope within
	// accountOwnerID's account — the account owner and Admin-role staff
	// always get AllPropertyAccess(); anyone else gets whatever
	// GetPropertyAccess reads back for their row. Resolved fresh per
	// request (never cached in the JWT) since access can change anytime.
	ResolveAccess(ctx context.Context, accountOwnerID, actorID uuid.UUID) (PropertyAccess, error)
}
