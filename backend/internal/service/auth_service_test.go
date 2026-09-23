package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/service"
)

// fakeUserRepository is an in-memory stand-in for domain.UserRepository.
type fakeUserRepository struct {
	byID    map[uuid.UUID]*domain.User
	byEmail map[string]*domain.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		byID:    make(map[uuid.UUID]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (f *fakeUserRepository) Create(_ context.Context, u *domain.User) error {
	if _, exists := f.byEmail[u.Email]; exists {
		return domain.ErrAlreadyExists
	}
	f.byID[u.ID] = u
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepository) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepository) TouchLastLogin(_ context.Context, id uuid.UUID, at time.Time) error {
	if u, ok := f.byID[id]; ok {
		u.LastLoginAt = &at
	}
	return nil
}

// fakeCache is an in-memory stand-in for domain.Cache.
type fakeCache struct {
	values map[string]string
}

func newFakeCache() *fakeCache {
	return &fakeCache{values: make(map[string]string)}
}

func (f *fakeCache) Get(_ context.Context, key string) (string, error) {
	return f.values[key], nil
}

func (f *fakeCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	f.values[key] = value
	return nil
}

func (f *fakeCache) Delete(_ context.Context, key string) error {
	delete(f.values, key)
	return nil
}

func newTestAuthService(users domain.UserRepository, cache domain.Cache) *service.AuthService {
	return service.NewAuthService(users, cache, "test-access-secret-min-32-characters!!", "test-refresh-secret-min-32-characters!!", time.Minute, time.Hour)
}

func TestAuthService_RegisterAndLogin(t *testing.T) {
	users := newFakeUserRepository()
	svc := newTestAuthService(users, nil)
	ctx := context.Background()

	u, err := svc.Register(ctx, "owner@example.com", "hunter22222")
	if err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}
	if u.Email != "owner@example.com" {
		t.Errorf("Register() email = %q, want owner@example.com", u.Email)
	}

	t.Run("duplicate email rejected", func(t *testing.T) {
		_, err := svc.Register(ctx, "owner@example.com", "hunter22222")
		if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Fatalf("Register() error = %v, want %v", err, domain.ErrAlreadyExists)
		}
	})

	t.Run("password too short rejected", func(t *testing.T) {
		_, err := svc.Register(ctx, "short@example.com", "short")
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("Register() error = %v, want %v", err, domain.ErrInvalidInput)
		}
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("Register() error = %v, want errors.As to find domain.ValidationErrors", err)
		}
		if len(verrs) != 1 || verrs[0].Field != "password" {
			t.Fatalf("Register() ValidationErrors = %v, want exactly one failure for field %q", verrs, "password")
		}
	})

	t.Run("correct credentials succeed", func(t *testing.T) {
		access, refresh, _, err := svc.Login(ctx, "owner@example.com", "hunter22222")
		if err != nil {
			t.Fatalf("Login() unexpected error = %v", err)
		}
		if access == "" || refresh == "" {
			t.Fatal("Login() returned empty token(s)")
		}
	})

	t.Run("wrong password rejected", func(t *testing.T) {
		_, _, _, err := svc.Login(ctx, "owner@example.com", "wrong-password")
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("Login() error = %v, want %v", err, domain.ErrUnauthorized)
		}
	})

	t.Run("unknown email rejected", func(t *testing.T) {
		_, _, _, err := svc.Login(ctx, "nobody@example.com", "hunter22222")
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("Login() error = %v, want %v", err, domain.ErrUnauthorized)
		}
	})
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	users := newFakeUserRepository()
	svc := newTestAuthService(users, nil)
	ctx := context.Background()

	_, err := svc.Register(ctx, "owner@example.com", "hunter22222")
	if err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}
	access, _, _, err := svc.Login(ctx, "owner@example.com", "hunter22222")
	if err != nil {
		t.Fatalf("Login() unexpected error = %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		claims, err := svc.ValidateAccessToken(ctx, access)
		if err != nil {
			t.Fatalf("ValidateAccessToken() unexpected error = %v", err)
		}
		if claims.Role != domain.UserRoleManager {
			t.Errorf("ValidateAccessToken() role = %q, want %q", claims.Role, domain.UserRoleManager)
		}
	})

	t.Run("garbage token rejected", func(t *testing.T) {
		_, err := svc.ValidateAccessToken(ctx, "not-a-real-token")
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("ValidateAccessToken() error = %v, want %v", err, domain.ErrUnauthorized)
		}
	})
}

func TestAuthService_RefreshToken_RotatesAndRevokesPrevious(t *testing.T) {
	users := newFakeUserRepository()
	cache := newFakeCache()
	svc := newTestAuthService(users, cache)
	ctx := context.Background()

	_, err := svc.Register(ctx, "owner@example.com", "hunter22222")
	if err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}
	_, refresh1, _, err := svc.Login(ctx, "owner@example.com", "hunter22222")
	if err != nil {
		t.Fatalf("Login() unexpected error = %v", err)
	}

	_, refresh2, _, err := svc.RefreshToken(ctx, refresh1)
	if err != nil {
		t.Fatalf("RefreshToken() unexpected error = %v", err)
	}
	if refresh2 == refresh1 {
		t.Error("RefreshToken() returned the same refresh token instead of rotating")
	}

	t.Run("reusing rotated-out token is rejected", func(t *testing.T) {
		_, _, _, err := svc.RefreshToken(ctx, refresh1)
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("RefreshToken() error = %v, want %v", err, domain.ErrUnauthorized)
		}
	})
}

func TestAuthService_Logout_RevokesRefreshToken(t *testing.T) {
	users := newFakeUserRepository()
	cache := newFakeCache()
	svc := newTestAuthService(users, cache)
	ctx := context.Background()

	_, err := svc.Register(ctx, "owner@example.com", "hunter22222")
	if err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}
	_, refresh, _, err := svc.Login(ctx, "owner@example.com", "hunter22222")
	if err != nil {
		t.Fatalf("Login() unexpected error = %v", err)
	}

	if err := svc.Logout(ctx, refresh); err != nil {
		t.Fatalf("Logout() unexpected error = %v", err)
	}

	_, _, _, err = svc.RefreshToken(ctx, refresh)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("RefreshToken() after logout error = %v, want %v", err, domain.ErrUnauthorized)
	}
}
