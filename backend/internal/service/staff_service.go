package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"propertymanagement/internal/domain"
)

const minStaffPasswordLength = 8

// StaffService manages the people who can sign in to an account besides
// its root owner: inviting them, changing their role and property
// scope, and deactivating/reactivating them. Every method is scoped by
// accountOwnerID — the root account's id, from AuthClaims.UserID, which
// already means exactly this for every other service in the codebase
// (see domain.AuthClaims's doc comment) — so this needs no new
// ownership-checking idiom of its own.
type StaffService struct {
	repo         domain.StaffRepository
	propertyRepo domain.PropertyRepository
	outbox       domain.EmailOutboxRepository
	mailEnabled  bool
	publicAppURL string
	log          *slog.Logger
	now          clock
}

func NewStaffService(repo domain.StaffRepository, propertyRepo domain.PropertyRepository, outbox domain.EmailOutboxRepository, mailEnabled bool, publicAppURL string, log *slog.Logger) *StaffService {
	return &StaffService{repo: repo, propertyRepo: propertyRepo, outbox: outbox, mailEnabled: mailEnabled, publicAppURL: publicAppURL, log: log, now: systemClock}
}

var _ domain.StaffService = (*StaffService)(nil)

// generateInviteToken returns the raw token (goes in the email link,
// never stored) and its SHA-256 hex hash (stored, compared against on
// lookup) — the same "never store the secret itself" split a password
// hash already uses.
func generateInviteToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate invite token: %w", err)
	}
	raw = hex.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

func hashInviteToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *StaffService) inviteURL(rawToken string) string {
	return fmt.Sprintf("%s/accept-invite?token=%s", strings.TrimRight(s.publicAppURL, "/"), rawToken)
}

func validatePropertyAccess(role domain.StaffRole, access *domain.PropertyAccess, propertyNames map[uuid.UUID]string) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if role == domain.StaffRoleAdmin {
		// Admins always have full access — enforced here, not just
		// trusted from the client, same as ExpenseService normalizing
		// its own derived fields before a write.
		*access = domain.AllPropertyAccess()
		return nil
	}
	if access.All {
		return nil
	}
	if len(access.PropertyIDs) == 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "property_ids", Message: "choose at least one property, or grant access to all"})
		return verrs
	}
	for _, id := range access.PropertyIDs {
		if _, ok := propertyNames[id]; !ok {
			verrs = append(verrs, &domain.ValidationError{Field: "property_ids", Message: "contains an unknown property"})
			return verrs
		}
	}
	return verrs
}

// ownedPropertyNames loads every property id→name pair for accountOwnerID,
// used both to validate a submitted property list and to reject one
// containing a property from someone else's portfolio.
func (s *StaffService) ownedPropertyNames(ctx context.Context, accountOwnerID uuid.UUID) (map[uuid.UUID]string, error) {
	properties, _, err := s.propertyRepo.List(ctx, domain.PropertyListOptions{OwnerID: accountOwnerID, Limit: 500})
	if err != nil {
		return nil, fmt.Errorf("list properties for %s: %w", accountOwnerID, err)
	}
	names := make(map[uuid.UUID]string, len(properties))
	for _, p := range properties {
		names[p.ID] = p.Name
	}
	return names, nil
}

func (s *StaffService) Invite(ctx context.Context, accountOwnerID, actorID uuid.UUID, input domain.InviteStaffInput) (*domain.StaffInviteResult, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	var verrs domain.ValidationErrors
	if input.Name == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "name", Message: "is required"})
	}
	if len(input.Name) > 120 {
		verrs = append(verrs, &domain.ValidationError{Field: "name", Message: "is too long"})
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		verrs = append(verrs, &domain.ValidationError{Field: "email", Message: "must be a valid email address"})
	}
	if !input.Role.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "role", Message: "unknown role"})
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("invite staff: %w", verrs)
	}

	names, err := s.ownedPropertyNames(ctx, accountOwnerID)
	if err != nil {
		return nil, fmt.Errorf("invite staff: %w", err)
	}
	if verrs := validatePropertyAccess(input.Role, &input.PropertyAccess, names); len(verrs) > 0 {
		return nil, fmt.Errorf("invite staff: %w", verrs)
	}

	// Duplicate-email handling: a match in the SAME account comes back
	// as a normal outcome the caller offers Resend/Reactivate for. A
	// match anywhere else is reported generically — never leaking
	// whether a given email belongs to another account.
	existingByEmail, lookupErr := s.findByEmail(ctx, input.Email)
	if lookupErr != nil {
		return nil, fmt.Errorf("invite staff: %w", lookupErr)
	}
	if existingByEmail != nil {
		if existingByEmail.AccountID() == accountOwnerID {
			member, err := s.repo.GetMember(ctx, accountOwnerID, existingByEmail.ID)
			if err != nil {
				return nil, fmt.Errorf("invite staff: %w", err)
			}
			return &domain.StaffInviteResult{Conflict: member}, nil
		}
		return nil, fmt.Errorf("invite staff: %w", domain.ValidationErrors{{Field: "email", Message: "this email is already in use"}})
	}

	now := s.now()
	rawToken, hash, err := generateInviteToken()
	if err != nil {
		return nil, fmt.Errorf("invite staff: %w", err)
	}
	role := input.Role
	expiresAt := now.Add(domain.InviteTokenTTL)
	u := &domain.User{
		ID: uuid.New(), Email: input.Email, Name: input.Name, AccountOwnerID: &accountOwnerID, StaffRole: &role,
		Status: domain.StaffStatusInvited, AllProperties: input.PropertyAccess.All,
		InvitedBy: &actorID, InvitedAt: &now, InviteExpiresAt: &expiresAt, InviteTokenHash: hash, CreatedAt: now, UpdatedAt: now,
	}
	audit := domain.NewStaffAuditEntry(accountOwnerID, actorID, "invite_sent", &u.ID, u.Name, nil, now)
	if err := s.repo.CreateInvite(ctx, u, input.PropertyAccess.PropertyIDs, audit); err != nil {
		return nil, fmt.Errorf("invite staff: %w", err)
	}

	url := s.inviteURL(rawToken)
	s.enqueueInviteEmail(ctx, accountOwnerID, u, url)

	member, err := s.repo.GetMember(ctx, accountOwnerID, u.ID)
	if err != nil {
		return nil, fmt.Errorf("invite staff: %w", err)
	}
	return &domain.StaffInviteResult{Created: member, InviteURL: url}, nil
}

// findByEmail is a thin GetByEmail wrapper that turns ErrNotFound into a
// nil result, since "no match" is the common case here, not an error.
func (s *StaffService) findByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (s *StaffService) enqueueInviteEmail(ctx context.Context, accountOwnerID uuid.UUID, u *domain.User, url string) {
	if !s.mailEnabled {
		return
	}
	body := fmt.Sprintf(`<p>Hello %s,</p><p>You've been invited to join RentFlow as <strong>%s</strong>.</p><p><a href="%s">Accept your invite</a></p><p>This link expires in 7 days.</p>`,
		htmlEscape(u.Name), roleLabel(*u.StaffRole), url)
	if err := s.outbox.Enqueue(ctx, &domain.OutboxEmail{
		ID: uuid.New(), OwnerID: accountOwnerID, To: u.Email, Subject: "You're invited to RentFlow", HTML: body, CreatedAt: s.now(),
	}); err != nil {
		s.log.WarnContext(ctx, "failed to queue invite email", slog.String("user_id", u.ID.String()), slog.Any("error", err))
	}
}

func roleLabel(r domain.StaffRole) string {
	switch r {
	case domain.StaffRoleAdmin:
		return "Admin"
	case domain.StaffRolePropertyManager:
		return "Property Manager"
	case domain.StaffRoleMaintenanceCoordinator:
		return "Maintenance Coordinator"
	case domain.StaffRoleAccountant:
		return "Accountant"
	default:
		return string(r)
	}
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func (s *StaffService) List(ctx context.Context, accountOwnerID uuid.UUID) ([]*domain.StaffMember, error) {
	members, err := s.repo.ListForAccount(ctx, accountOwnerID)
	if err != nil {
		return nil, fmt.Errorf("list staff: %w", err)
	}
	return members, nil
}

// requireManageableStaff loads userID as a member of accountOwnerID's
// staff, rejecting the account's own root row — the root can't be
// edited, deactivated or reactivated through this API at all, so every
// mutating method checks this first, before the last-admin math even
// applies.
func (s *StaffService) requireManageableStaff(ctx context.Context, accountOwnerID, userID uuid.UUID) (*domain.StaffMember, error) {
	member, err := s.repo.GetMember(ctx, accountOwnerID, userID)
	if err != nil {
		return nil, err
	}
	if member.IsAccountOwner {
		return nil, fmt.Errorf("the account owner can't be managed here: %w", domain.ErrConflict)
	}
	return member, nil
}

// wouldRemoveLastAdmin reports whether changing member away from an
// active Admin (by demotion or deactivation) would leave the account
// with no OTHER staff admin. The root owner is deliberately not counted
// — it can never be deactivated through this API, so it never needs
// protecting, but a second real admin is exactly the kind of delegated
// coverage this safeguard exists to preserve.
func (s *StaffService) wouldRemoveLastAdmin(ctx context.Context, accountOwnerID uuid.UUID, member *domain.StaffMember) (bool, error) {
	if member.EffectiveRole() != domain.StaffRoleAdmin || member.Status != domain.StaffStatusActive {
		return false, nil
	}
	n, err := s.repo.CountActiveAdmins(ctx, accountOwnerID)
	if err != nil {
		return false, err
	}
	return n <= 1, nil
}

func (s *StaffService) Update(ctx context.Context, accountOwnerID, actorID, userID uuid.UUID, input domain.UpdateStaffInput) (*domain.StaffMember, error) {
	member, err := s.requireManageableStaff(ctx, accountOwnerID, userID)
	if err != nil {
		return nil, fmt.Errorf("update staff %s: %w", userID, err)
	}
	if !input.Role.Valid() {
		return nil, fmt.Errorf("update staff %s: %w", userID, domain.ValidationErrors{{Field: "role", Message: "unknown role"}})
	}

	if input.Role != domain.StaffRoleAdmin {
		if removes, err := s.wouldRemoveLastAdmin(ctx, accountOwnerID, member); err != nil {
			return nil, fmt.Errorf("update staff %s: %w", userID, err)
		} else if removes {
			return nil, fmt.Errorf("update staff %s: %w", userID, domain.ValidationErrors{
				{Field: "role", Message: "promote another user to Admin before changing the only active Admin's role"},
			})
		}
	}

	names, err := s.ownedPropertyNames(ctx, accountOwnerID)
	if err != nil {
		return nil, fmt.Errorf("update staff %s: %w", userID, err)
	}
	if verrs := validatePropertyAccess(input.Role, &input.PropertyAccess, names); len(verrs) > 0 {
		return nil, fmt.Errorf("update staff %s: %w", userID, verrs)
	}

	changes := map[string]domain.FieldChange{}
	if member.EffectiveRole() != input.Role {
		changes["role"] = domain.FieldChange{Old: roleLabel(member.EffectiveRole()), New: roleLabel(input.Role)}
	}
	oldAccess := formatAccessForAudit(member.PropertyAccess, member.PropertyNames)
	newNames := namesFor(input.PropertyAccess, names)
	newAccess := formatAccessForAudit(input.PropertyAccess, newNames)
	if oldAccess != newAccess {
		changes["property_access"] = domain.FieldChange{Old: oldAccess, New: newAccess}
	}

	now := s.now()
	audit := domain.NewStaffAuditEntry(accountOwnerID, actorID, "role_or_access_changed", &userID, member.Name, changes, now)
	if err := s.repo.UpdateRoleAndAccess(ctx, userID, input.Role, input.PropertyAccess, now, audit); err != nil {
		return nil, fmt.Errorf("update staff %s: %w", userID, err)
	}
	return s.repo.GetMember(ctx, accountOwnerID, userID)
}

func namesFor(access domain.PropertyAccess, all map[uuid.UUID]string) []string {
	if access.All {
		return nil
	}
	out := make([]string, 0, len(access.PropertyIDs))
	for _, id := range access.PropertyIDs {
		out = append(out, all[id])
	}
	return out
}

func formatAccessForAudit(access domain.PropertyAccess, names []string) string {
	if access.All {
		return "All properties"
	}
	if len(names) == 1 {
		return "1 property"
	}
	return fmt.Sprintf("%d properties", len(names))
}

func (s *StaffService) Deactivate(ctx context.Context, accountOwnerID, actorID, userID uuid.UUID) error {
	member, err := s.requireManageableStaff(ctx, accountOwnerID, userID)
	if err != nil {
		return fmt.Errorf("deactivate staff %s: %w", userID, err)
	}
	if member.Status != domain.StaffStatusActive {
		return fmt.Errorf("deactivate staff %s: %w", userID, domain.ErrConflict)
	}
	if removes, err := s.wouldRemoveLastAdmin(ctx, accountOwnerID, member); err != nil {
		return fmt.Errorf("deactivate staff %s: %w", userID, err)
	} else if removes {
		return fmt.Errorf("deactivate staff %s: %w", userID, domain.ValidationErrors{
			{Field: "status", Message: "promote another user to Admin before deactivating the only active Admin"},
		})
	}

	now := s.now()
	audit := domain.NewStaffAuditEntry(accountOwnerID, actorID, "deactivated", &userID, member.Name, nil, now)
	if err := s.repo.SetStatus(ctx, userID, domain.StaffStatusDeactivated, now, audit); err != nil {
		return fmt.Errorf("deactivate staff %s: %w", userID, err)
	}
	return nil
}

func (s *StaffService) Reactivate(ctx context.Context, accountOwnerID, actorID, userID uuid.UUID) error {
	member, err := s.requireManageableStaff(ctx, accountOwnerID, userID)
	if err != nil {
		return fmt.Errorf("reactivate staff %s: %w", userID, err)
	}
	if member.Status != domain.StaffStatusDeactivated {
		return fmt.Errorf("reactivate staff %s: %w", userID, domain.ErrConflict)
	}

	now := s.now()
	audit := domain.NewStaffAuditEntry(accountOwnerID, actorID, "reactivated", &userID, member.Name, map[string]domain.FieldChange{
		"status": {Old: "Deactivated", New: "Active"},
	}, now)
	if err := s.repo.SetStatus(ctx, userID, domain.StaffStatusActive, now, audit); err != nil {
		return fmt.Errorf("reactivate staff %s: %w", userID, err)
	}
	return nil
}

func (s *StaffService) ResendInvite(ctx context.Context, accountOwnerID, actorID, userID uuid.UUID) (string, error) {
	member, err := s.requireManageableStaff(ctx, accountOwnerID, userID)
	if err != nil {
		return "", fmt.Errorf("resend invite for %s: %w", userID, err)
	}
	if member.Status != domain.StaffStatusInvited {
		return "", fmt.Errorf("resend invite for %s: %w", userID, domain.ErrConflict)
	}

	now := s.now()
	rawToken, hash, err := generateInviteToken()
	if err != nil {
		return "", fmt.Errorf("resend invite for %s: %w", userID, err)
	}
	expiresAt := now.Add(domain.InviteTokenTTL)
	audit := domain.NewStaffAuditEntry(accountOwnerID, actorID, "invite_resent", &userID, member.Name, nil, now)
	if err := s.repo.ResendInvite(ctx, userID, hash, now, expiresAt, audit); err != nil {
		return "", fmt.Errorf("resend invite for %s: %w", userID, err)
	}

	url := s.inviteURL(rawToken)
	role := member.EffectiveRole()
	s.enqueueInviteEmail(ctx, accountOwnerID, &domain.User{ID: userID, Email: member.Email, Name: member.Name, StaffRole: &role}, url)
	return url, nil
}

func (s *StaffService) AuditLog(ctx context.Context, accountOwnerID uuid.UUID, limit, offset int) ([]*domain.StaffAuditEntry, int, error) {
	limit, offset = clampPage(limit, offset)
	entries, total, err := s.repo.ListAudit(ctx, accountOwnerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list account audit log: %w", err)
	}
	return entries, total, nil
}

func (s *StaffService) LookupInvite(ctx context.Context, token string) (*domain.InviteLookup, error) {
	if token == "" {
		return nil, fmt.Errorf("lookup invite: %w", domain.ErrNotFound)
	}
	u, err := s.repo.GetByInviteTokenHash(ctx, hashInviteToken(token))
	if err != nil {
		return nil, fmt.Errorf("lookup invite: %w", err)
	}
	if u.Status != domain.StaffStatusInvited || u.StaffRole == nil {
		return nil, fmt.Errorf("lookup invite: %w", domain.ErrNotFound)
	}
	member, err := s.repo.GetMember(ctx, u.AccountID(), u.ID)
	if err != nil {
		return nil, fmt.Errorf("lookup invite: %w", err)
	}

	now := s.now()
	return &domain.InviteLookup{
		Name: u.Name, Email: u.Email, Role: *u.StaffRole, PropertyAccess: member.PropertyAccess,
		PropertyNames: member.PropertyNames, InvitedByName: member.InvitedByName,
		Expired: u.InviteExpiresAt != nil && u.InviteExpiresAt.Before(now),
	}, nil
}

func (s *StaffService) AcceptInvite(ctx context.Context, input domain.AcceptInviteInput) error {
	if len(input.Password) < minStaffPasswordLength {
		return fmt.Errorf("accept invite: %w", domain.ValidationErrors{
			{Field: "password", Message: fmt.Sprintf("must be at least %d characters", minStaffPasswordLength)},
		})
	}
	u, err := s.repo.GetByInviteTokenHash(ctx, hashInviteToken(input.Token))
	if err != nil {
		return fmt.Errorf("accept invite: %w", err)
	}
	if u.Status != domain.StaffStatusInvited {
		return fmt.Errorf("accept invite: %w", domain.ErrConflict)
	}
	now := s.now()
	if u.InviteExpiresAt != nil && u.InviteExpiresAt.Before(now) {
		return fmt.Errorf("accept invite: this invite has expired: %w", domain.ErrConflict)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("accept invite: hash password: %w", err)
	}
	audit := domain.NewStaffAuditEntry(u.AccountID(), u.ID, "invite_accepted", &u.ID, u.Name, nil, now)
	if err := s.repo.AcceptInvite(ctx, u.ID, string(hash), now, audit); err != nil {
		return fmt.Errorf("accept invite: %w", err)
	}
	return nil
}
