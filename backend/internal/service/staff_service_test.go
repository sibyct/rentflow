package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// fakeStaffRepo is a minimal in-memory domain.StaffRepository for
// exercising StaffService's business rules without a database.
type fakeStaffRepo struct {
	users            map[uuid.UUID]*domain.User
	access           map[uuid.UUID][]uuid.UUID // userID -> property ids, when not all
	audit            []*domain.StaffAuditEntry
	invitedTokenHash map[uuid.UUID]string
}

func newFakeStaffRepo(root *domain.User) *fakeStaffRepo {
	return &fakeStaffRepo{
		users:            map[uuid.UUID]*domain.User{root.ID: root},
		access:           map[uuid.UUID][]uuid.UUID{},
		invitedTokenHash: map[uuid.UUID]string{},
	}
}

func (f *fakeStaffRepo) CreateInvite(_ context.Context, u *domain.User, propertyIDs []uuid.UUID, audit domain.StaffAuditEntry) error {
	for _, existing := range f.users {
		if existing.Email == u.Email {
			return domain.ErrAlreadyExists
		}
	}
	cp := *u
	f.users[u.ID] = &cp
	f.access[u.ID] = propertyIDs
	f.invitedTokenHash[u.ID] = u.InviteTokenHash
	f.audit = append(f.audit, &audit)
	return nil
}

func (f *fakeStaffRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeStaffRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeStaffRepo) GetByInviteTokenHash(_ context.Context, hash string) (*domain.User, error) {
	for id, h := range f.invitedTokenHash {
		if h == hash && h != "" {
			cp := *f.users[id]
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeStaffRepo) ListForAccount(_ context.Context, accountOwnerID uuid.UUID) ([]*domain.StaffMember, error) {
	var out []*domain.StaffMember
	for _, u := range f.users {
		if u.ID == accountOwnerID || (u.AccountOwnerID != nil && *u.AccountOwnerID == accountOwnerID) {
			out = append(out, f.toMember(u))
		}
	}
	return out, nil
}

func (f *fakeStaffRepo) GetMember(_ context.Context, accountOwnerID, userID uuid.UUID) (*domain.StaffMember, error) {
	u, ok := f.users[userID]
	if !ok || (u.ID != accountOwnerID && (u.AccountOwnerID == nil || *u.AccountOwnerID != accountOwnerID)) {
		return nil, domain.ErrNotFound
	}
	return f.toMember(u), nil
}

func (f *fakeStaffRepo) toMember(u *domain.User) *domain.StaffMember {
	m := &domain.StaffMember{User: *u, IsAccountOwner: u.AccountOwnerID == nil}
	if m.IsAccountOwner || u.AllProperties {
		m.PropertyAccess = domain.AllPropertyAccess()
	} else {
		m.PropertyAccess = domain.PropertyAccess{All: false, PropertyIDs: f.access[u.ID]}
	}
	return m
}

func (f *fakeStaffRepo) UpdateRoleAndAccess(_ context.Context, userID uuid.UUID, role domain.StaffRole, access domain.PropertyAccess, updatedAt time.Time, audit domain.StaffAuditEntry) error {
	u, ok := f.users[userID]
	if !ok || u.AccountOwnerID == nil {
		return domain.ErrNotFound
	}
	u.StaffRole = &role
	u.AllProperties = access.All
	u.UpdatedAt = updatedAt
	f.access[userID] = access.PropertyIDs
	f.audit = append(f.audit, &audit)
	return nil
}

func (f *fakeStaffRepo) SetStatus(_ context.Context, userID uuid.UUID, status domain.StaffStatus, updatedAt time.Time, audit domain.StaffAuditEntry) error {
	u, ok := f.users[userID]
	if !ok || u.AccountOwnerID == nil {
		return domain.ErrNotFound
	}
	u.Status = status
	u.UpdatedAt = updatedAt
	f.audit = append(f.audit, &audit)
	return nil
}

func (f *fakeStaffRepo) ResendInvite(_ context.Context, userID uuid.UUID, tokenHash string, invitedAt, expiresAt time.Time, audit domain.StaffAuditEntry) error {
	u, ok := f.users[userID]
	if !ok || u.AccountOwnerID == nil {
		return domain.ErrNotFound
	}
	u.Status = domain.StaffStatusInvited
	u.InvitedAt = &invitedAt
	u.InviteExpiresAt = &expiresAt
	f.invitedTokenHash[userID] = tokenHash
	f.audit = append(f.audit, &audit)
	return nil
}

func (f *fakeStaffRepo) AcceptInvite(_ context.Context, userID uuid.UUID, passwordHash string, acceptedAt time.Time, audit domain.StaffAuditEntry) error {
	u, ok := f.users[userID]
	if !ok || u.Status != domain.StaffStatusInvited {
		return domain.ErrConflict
	}
	u.PasswordHash = passwordHash
	u.Status = domain.StaffStatusActive
	u.InviteTokenHash = ""
	u.InviteExpiresAt = nil
	u.UpdatedAt = acceptedAt
	delete(f.invitedTokenHash, userID)
	f.audit = append(f.audit, &audit)
	return nil
}

func (f *fakeStaffRepo) CountActiveAdmins(_ context.Context, accountOwnerID uuid.UUID) (int, error) {
	n := 0
	for _, u := range f.users {
		if u.AccountOwnerID != nil && *u.AccountOwnerID == accountOwnerID && u.StaffRole != nil && *u.StaffRole == domain.StaffRoleAdmin && u.Status == domain.StaffStatusActive {
			n++
		}
	}
	return n, nil
}

func (f *fakeStaffRepo) ListAudit(_ context.Context, accountOwnerID uuid.UUID, limit, offset int) ([]*domain.StaffAuditEntry, int, error) {
	var out []*domain.StaffAuditEntry
	for _, a := range f.audit {
		if a.AccountOwnerID == accountOwnerID {
			out = append(out, a)
		}
	}
	return out, len(out), nil
}

// fakeStaffPropertyRepo backs StaffService's property-validation lookups.
type fakeStaffPropertyRepo struct {
	domain.PropertyRepository
	properties []*domain.Property
}

func (f *fakeStaffPropertyRepo) List(_ context.Context, _ domain.PropertyListOptions) ([]*domain.Property, int, error) {
	return f.properties, len(f.properties), nil
}

// fakeOutbox is a no-op domain.EmailOutboxRepository — StaffService
// treats a failed enqueue as best-effort, so tests don't need it to
// succeed, only to exist.
type fakeOutbox struct{ enqueued int }

func (f *fakeOutbox) Enqueue(_ context.Context, _ *domain.OutboxEmail) error {
	f.enqueued++
	return nil
}
func (f *fakeOutbox) ClaimPending(_ context.Context, _ int) ([]*domain.OutboxEmail, error) {
	return nil, nil
}
func (f *fakeOutbox) MarkSent(_ context.Context, _ uuid.UUID, _ time.Time) error        { return nil }
func (f *fakeOutbox) MarkFailed(_ context.Context, _ uuid.UUID, _ string, _ bool) error { return nil }

func newStaffTestFixture(t *testing.T) (*StaffService, *fakeStaffRepo, uuid.UUID) {
	t.Helper()
	root := &domain.User{ID: uuid.New(), Email: "root@example.com", Name: "Root Owner", Status: domain.StaffStatusActive, CreatedAt: day(2026, time.January, 1)}
	repo := newFakeStaffRepo(root)
	properties := &fakeStaffPropertyRepo{properties: []*domain.Property{
		{ID: uuid.New(), Name: "Willow Creek"}, {ID: uuid.New(), Name: "Oak Terrace"},
	}}
	svc := NewStaffService(repo, properties, &fakeOutbox{}, false, "https://app.example.com", discardLog())
	svc.now = fixedClock(day(2026, time.September, 22))
	return svc, repo, root.ID
}

func inviteAdmin(t *testing.T, svc *StaffService, repo *fakeStaffRepo, accountID uuid.UUID, name, email string) uuid.UUID {
	t.Helper()
	result, err := svc.Invite(context.Background(), accountID, accountID, domain.InviteStaffInput{
		Name: name, Email: email, Role: domain.StaffRoleAdmin, PropertyAccess: domain.AllPropertyAccess(),
	})
	if err != nil || result.Created == nil {
		t.Fatalf("invite %s: result=%+v err=%v", name, result, err)
	}
	// Accept immediately so the new admin counts as active.
	u := repo.users[result.Created.ID]
	if err := repo.AcceptInvite(context.Background(), u.ID, "hash", svc.now(), domain.NewStaffAuditEntry(accountID, accountID, "invite_accepted", &u.ID, name, nil, svc.now())); err != nil {
		t.Fatalf("accept invite for %s: %v", name, err)
	}
	return u.ID
}

func TestStaffService_Invite(t *testing.T) {
	svc, _, accountID := newStaffTestFixture(t)

	t.Run("creates a new invited staff member", func(t *testing.T) {
		result, err := svc.Invite(context.Background(), accountID, accountID, domain.InviteStaffInput{
			Name: "Priya Shah", Email: "priya@example.com", Role: domain.StaffRoleAdmin, PropertyAccess: domain.AllPropertyAccess(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Created == nil || result.Conflict != nil {
			t.Fatalf("expected a created member, got %+v", result)
		}
		if result.Created.Status != domain.StaffStatusInvited {
			t.Errorf("status = %s, want invited", result.Created.Status)
		}
		if result.InviteURL == "" {
			t.Error("expected a non-empty invite URL")
		}
	})

	t.Run("a second invite to the same email in the same account is a conflict, not a duplicate", func(t *testing.T) {
		_, err := svc.Invite(context.Background(), accountID, accountID, domain.InviteStaffInput{
			Name: "Priya S.", Email: "priya@example.com", Role: domain.StaffRoleAccountant, PropertyAccess: domain.AllPropertyAccess(),
		})
		if err != nil {
			t.Fatal(err)
		}
		result, err := svc.Invite(context.Background(), accountID, accountID, domain.InviteStaffInput{
			Name: "Priya S.", Email: "priya@example.com", Role: domain.StaffRoleAccountant, PropertyAccess: domain.AllPropertyAccess(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Conflict == nil || result.Created != nil {
			t.Fatalf("expected a conflict, got %+v", result)
		}
	})

	t.Run("an unknown property id is rejected", func(t *testing.T) {
		_, err := svc.Invite(context.Background(), accountID, accountID, domain.InviteStaffInput{
			Name: "Dana", Email: "dana@example.com", Role: domain.StaffRolePropertyManager,
			PropertyAccess: domain.PropertyAccess{PropertyIDs: []uuid.UUID{uuid.New()}},
		})
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("err = %v, want ValidationErrors", err)
		}
	})

	t.Run("an admin invite always gets full property access, regardless of what was requested", func(t *testing.T) {
		result, err := svc.Invite(context.Background(), accountID, accountID, domain.InviteStaffInput{
			Name: "Grace", Email: "grace@example.com", Role: domain.StaffRoleAdmin,
			PropertyAccess: domain.PropertyAccess{PropertyIDs: []uuid.UUID{uuid.New()}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Created.PropertyAccess.All {
			t.Errorf("admin property access = %+v, want All", result.Created.PropertyAccess)
		}
	})
}

func TestStaffService_LastActiveAdminSafeguard(t *testing.T) {
	svc, repo, accountID := newStaffTestFixture(t)
	admin1 := inviteAdmin(t, svc, repo, accountID, "Alex", "alex@example.com")
	admin2 := inviteAdmin(t, svc, repo, accountID, "Priya", "priya@example.com")

	t.Run("demoting one of two active admins is fine", func(t *testing.T) {
		_, err := svc.Update(context.Background(), accountID, accountID, admin2, domain.UpdateStaffInput{
			Role: domain.StaffRoleAccountant, PropertyAccess: domain.AllPropertyAccess(),
		})
		if err != nil {
			t.Fatalf("unexpected error demoting the second admin: %v", err)
		}
	})

	t.Run("demoting the only remaining active admin is blocked", func(t *testing.T) {
		_, err := svc.Update(context.Background(), accountID, accountID, admin1, domain.UpdateStaffInput{
			Role: domain.StaffRoleAccountant, PropertyAccess: domain.AllPropertyAccess(),
		})
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("err = %v, want ValidationErrors (last active admin)", err)
		}
	})

	t.Run("deactivating the only remaining active admin is blocked", func(t *testing.T) {
		err := svc.Deactivate(context.Background(), accountID, accountID, admin1)
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("err = %v, want ValidationErrors (last active admin)", err)
		}
	})

	t.Run("the root account owner is never a manageable target", func(t *testing.T) {
		err := svc.Deactivate(context.Background(), accountID, accountID, accountID)
		if !errors.Is(err, domain.ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})
}

func TestStaffService_DeactivateReactivate(t *testing.T) {
	svc, repo, accountID := newStaffTestFixture(t)
	inviteAdmin(t, svc, repo, accountID, "Alex", "alex@example.com") // a second admin, so deactivating pm below doesn't trip the last-admin safeguard
	pm := inviteAdmin(t, svc, repo, accountID, "Marcus", "marcus@example.com")

	if err := svc.Deactivate(context.Background(), accountID, accountID, pm); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if repo.users[pm].Status != domain.StaffStatusDeactivated {
		t.Errorf("status = %s, want deactivated", repo.users[pm].Status)
	}

	t.Run("deactivating again is a conflict", func(t *testing.T) {
		if err := svc.Deactivate(context.Background(), accountID, accountID, pm); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})

	if err := svc.Reactivate(context.Background(), accountID, accountID, pm); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if repo.users[pm].Status != domain.StaffStatusActive {
		t.Errorf("status after reactivate = %s, want active", repo.users[pm].Status)
	}
}

func TestStaffService_AcceptInvite(t *testing.T) {
	svc, repo, accountID := newStaffTestFixture(t)
	result, err := svc.Invite(context.Background(), accountID, accountID, domain.InviteStaffInput{
		Name: "Tom Reyes", Email: "tom@example.com", Role: domain.StaffRoleAccountant, PropertyAccess: domain.AllPropertyAccess(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Recover the raw token the same way a real client would — from the
	// invite URL the service just returned.
	rawToken := result.InviteURL[len(result.InviteURL)-64:]

	t.Run("lookup succeeds for a fresh invite", func(t *testing.T) {
		lookup, err := svc.LookupInvite(context.Background(), rawToken)
		if err != nil {
			t.Fatal(err)
		}
		if lookup.Email != "tom@example.com" || lookup.Expired {
			t.Errorf("lookup = %+v", lookup)
		}
	})

	t.Run("a short password is rejected", func(t *testing.T) {
		err := svc.AcceptInvite(context.Background(), domain.AcceptInviteInput{Token: rawToken, Password: "short"})
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("err = %v, want ValidationErrors", err)
		}
	})

	t.Run("accepting sets an active status and a real password hash", func(t *testing.T) {
		if err := svc.AcceptInvite(context.Background(), domain.AcceptInviteInput{Token: rawToken, Password: "a-strong-password"}); err != nil {
			t.Fatal(err)
		}
		u := repo.users[result.Created.ID]
		if u.Status != domain.StaffStatusActive || u.PasswordHash == "" {
			t.Errorf("user after accept = %+v", u)
		}
	})

	t.Run("accepting the same invite twice fails", func(t *testing.T) {
		err := svc.AcceptInvite(context.Background(), domain.AcceptInviteInput{Token: rawToken, Password: "a-strong-password"})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound (token already cleared)", err)
		}
	})
}

func TestStaffService_LookupInvite_Expired(t *testing.T) {
	svc, _, accountID := newStaffTestFixture(t)
	result, err := svc.Invite(context.Background(), accountID, accountID, domain.InviteStaffInput{
		Name: "Owen Park", Email: "owen@example.com", Role: domain.StaffRolePropertyManager, PropertyAccess: domain.AllPropertyAccess(),
	})
	if err != nil {
		t.Fatal(err)
	}
	rawToken := result.InviteURL[len(result.InviteURL)-64:]

	// Jump the clock past the 7-day expiry.
	svc.now = fixedClock(day(2026, time.October, 15))

	lookup, err := svc.LookupInvite(context.Background(), rawToken)
	if err != nil {
		t.Fatal(err)
	}
	if !lookup.Expired {
		t.Error("expected the invite to show as expired")
	}

	err = svc.AcceptInvite(context.Background(), domain.AcceptInviteInput{Token: rawToken, Password: "a-strong-password"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("accepting an expired invite: err = %v, want ErrConflict", err)
	}
}
