//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	repository "propertymanagement/internal/repository/postgres"
)

func TestUserRepository_CRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewUserRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	u := &domain.User{
		ID:           uuid.New(),
		Email:        "owner+" + uuid.NewString() + "@example.com",
		PasswordHash: "hashed",
		Role:         domain.UserRoleManager,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	t.Run("create then get by id and email", func(t *testing.T) {
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Create() unexpected error = %v", err)
		}

		byID, err := repo.GetByID(ctx, u.ID)
		if err != nil {
			t.Fatalf("GetByID() unexpected error = %v", err)
		}
		if byID.Email != u.Email {
			t.Errorf("GetByID() email = %q, want %q", byID.Email, u.Email)
		}

		byEmail, err := repo.GetByEmail(ctx, u.Email)
		if err != nil {
			t.Fatalf("GetByEmail() unexpected error = %v", err)
		}
		if byEmail.ID != u.ID {
			t.Errorf("GetByEmail() id = %v, want %v", byEmail.ID, u.ID)
		}
	})

	t.Run("create with duplicate email returns ErrAlreadyExists", func(t *testing.T) {
		dup := &domain.User{
			ID:           uuid.New(),
			Email:        u.Email,
			PasswordHash: "hashed",
			Role:         domain.UserRoleManager,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		err := repo.Create(ctx, dup)
		if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Fatalf("Create() error = %v, want errors.Is(err, %v)", err, domain.ErrAlreadyExists)
		}
	})

	t.Run("get by id missing returns ErrNotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetByID() error = %v, want errors.Is(err, %v)", err, domain.ErrNotFound)
		}
	})

	t.Run("get by email missing returns ErrNotFound", func(t *testing.T) {
		_, err := repo.GetByEmail(ctx, "nobody-"+uuid.NewString()+"@example.com")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetByEmail() error = %v, want errors.Is(err, %v)", err, domain.ErrNotFound)
		}
	})
}
