//go:build integration

// Integration tests run against a real Postgres via testcontainers-go
// (requires a running Docker daemon). They are excluded from `make test`
// and run separately via `make test-integration` because they are slow
// and need Docker — unlike the service-layer unit tests, which use
// hand-written fakes and run everywhere.
package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"propertymanagement/internal/domain"
	repository "propertymanagement/internal/repository/postgres"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("propertymanagement_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pool, err := repository.NewPool(connectCtx, dsn, 5)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	applyMigrations(t, pool)

	return pool
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	dir := filepath.Join("..", "..", "..", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
}

func seedTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password_hash, role) VALUES ($1, $2, 'x', 'manager')`,
		id, id.String()+"@example.com",
	)
	if err != nil {
		t.Fatalf("seed test user: %v", err)
	}
	return id
}

func TestPropertyRepository_CRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewPropertyRepository(pool)
	ownerID := seedTestUser(t, pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	p := &domain.Property{
		ID:        uuid.New(),
		Address:   "1 Integration Way",
		UnitCount: 6,
		Status:    domain.PropertyStatusActive,
		OwnerID:   ownerID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("create then get", func(t *testing.T) {
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create() unexpected error = %v", err)
		}

		got, err := repo.GetByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetByID() unexpected error = %v", err)
		}
		if got.Address != p.Address || got.UnitCount != p.UnitCount || got.Status != p.Status {
			t.Errorf("GetByID() = %+v, want fields matching %+v", got, p)
		}
	})

	t.Run("get missing returns ErrNotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetByID() error = %v, want errors.Is(err, %v)", err, domain.ErrNotFound)
		}
	})

	t.Run("list filters by owner and paginates", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			extra := &domain.Property{
				ID:        uuid.New(),
				Address:   "extra",
				UnitCount: 1,
				Status:    domain.PropertyStatusActive,
				OwnerID:   ownerID,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := repo.Create(ctx, extra); err != nil {
				t.Fatalf("Create() unexpected error = %v", err)
			}
		}

		got, total, err := repo.List(ctx, ownerID, 2, 0)
		if err != nil {
			t.Fatalf("List() unexpected error = %v", err)
		}
		if total != 3 {
			t.Errorf("List() total = %d, want 3", total)
		}
		if len(got) != 2 {
			t.Errorf("List() returned %d rows, want 2 (page size)", len(got))
		}
	})

	t.Run("update", func(t *testing.T) {
		p.Address = "2 Integration Way"
		p.UpdatedAt = time.Now().UTC()
		if err := repo.Update(ctx, p); err != nil {
			t.Fatalf("Update() unexpected error = %v", err)
		}
		got, err := repo.GetByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetByID() unexpected error = %v", err)
		}
		if got.Address != "2 Integration Way" {
			t.Errorf("Update() address = %q, want %q", got.Address, "2 Integration Way")
		}
	})

	t.Run("update missing returns ErrNotFound", func(t *testing.T) {
		ghost := &domain.Property{ID: uuid.New(), Address: "nowhere", UnitCount: 1, Status: domain.PropertyStatusActive, UpdatedAt: now}
		if err := repo.Update(ctx, ghost); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("Update() error = %v, want errors.Is(err, %v)", err, domain.ErrNotFound)
		}
	})

	t.Run("delete missing returns ErrNotFound", func(t *testing.T) {
		if err := repo.Delete(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("Delete() error = %v, want errors.Is(err, %v)", err, domain.ErrNotFound)
		}
	})

	t.Run("ExistsByOwnerAddress", func(t *testing.T) {
		exists, err := repo.ExistsByOwnerAddress(ctx, ownerID, p.Address)
		if err != nil {
			t.Fatalf("ExistsByOwnerAddress() unexpected error = %v", err)
		}
		if !exists {
			t.Error("ExistsByOwnerAddress() = false, want true for an address that was just created")
		}

		exists, err = repo.ExistsByOwnerAddress(ctx, ownerID, "an address that was never created")
		if err != nil {
			t.Fatalf("ExistsByOwnerAddress() unexpected error = %v", err)
		}
		if exists {
			t.Error("ExistsByOwnerAddress() = true, want false for an address that was never created")
		}
	})

	t.Run("delete", func(t *testing.T) {
		if err := repo.Delete(ctx, p.ID); err != nil {
			t.Fatalf("Delete() unexpected error = %v", err)
		}
		if _, err := repo.GetByID(ctx, p.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetByID() after delete error = %v, want errors.Is(err, %v)", err, domain.ErrNotFound)
		}
	})
}
