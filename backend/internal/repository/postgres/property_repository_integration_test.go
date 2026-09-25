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

func newTestProperty(ownerID uuid.UUID, addressLine1 string) *domain.Property {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &domain.Property{
		ID:           uuid.New(),
		Name:         "Integration Test Property",
		Type:         domain.PropertyTypeResidentialMultiUnit,
		AddressLine1: addressLine1,
		City:         "Austin",
		Country:      "United States",
		Units:        6,
		Status:       domain.PropertyStatusActive,
		Amenities:    []string{"Parking", "Pool"},
		OwnerID:      ownerID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestPropertyRepository_CRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewPropertyRepository(pool)
	ownerID := seedTestUser(t, pool)
	ctx := context.Background()

	p := newTestProperty(ownerID, "1 Integration Way")

	t.Run("create then get", func(t *testing.T) {
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create() unexpected error = %v", err)
		}

		got, err := repo.GetByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetByID() unexpected error = %v", err)
		}
		if got.AddressLine1 != p.AddressLine1 || got.Units != p.Units || got.Status != p.Status {
			t.Errorf("GetByID() = %+v, want fields matching %+v", got, p)
		}
		if len(got.Amenities) != 2 || got.Amenities[0] != "Parking" {
			t.Errorf("GetByID() amenities = %v, want [Parking Pool]", got.Amenities)
		}
		if got.Ownership != nil {
			t.Errorf("GetByID() ownership = %v, want nil (never set)", *got.Ownership)
		}
		if got.YearBuilt != nil {
			t.Errorf("GetByID() year_built = %v, want nil (never set)", *got.YearBuilt)
		}
	})

	t.Run("get missing returns ErrNotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetByID() error = %v, want errors.Is(err, %v)", err, domain.ErrNotFound)
		}
	})

	t.Run("create and get roundtrips nullable fields", func(t *testing.T) {
		owned := domain.PropertyOwnershipOwned
		yearBuilt := 1998
		onboardDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		withNullables := newTestProperty(ownerID, "9 Nullable Fields Way")
		withNullables.Ownership = &owned
		withNullables.YearBuilt = &yearBuilt
		withNullables.OnboardDate = &onboardDate

		if err := repo.Create(ctx, withNullables); err != nil {
			t.Fatalf("Create() unexpected error = %v", err)
		}
		got, err := repo.GetByID(ctx, withNullables.ID)
		if err != nil {
			t.Fatalf("GetByID() unexpected error = %v", err)
		}
		if got.Ownership == nil || *got.Ownership != owned {
			t.Errorf("GetByID() ownership = %v, want %v", got.Ownership, owned)
		}
		if got.YearBuilt == nil || *got.YearBuilt != yearBuilt {
			t.Errorf("GetByID() year_built = %v, want %v", got.YearBuilt, yearBuilt)
		}
		if got.OnboardDate == nil || !got.OnboardDate.Equal(onboardDate) {
			t.Errorf("GetByID() onboard_date = %v, want %v", got.OnboardDate, onboardDate)
		}
	})

	t.Run("list filters by owner, status, and paginates", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			extra := newTestProperty(ownerID, "extra "+uuid.NewString())
			if err := repo.Create(ctx, extra); err != nil {
				t.Fatalf("Create() unexpected error = %v", err)
			}
		}
		archived := newTestProperty(ownerID, "archived "+uuid.NewString())
		archived.Status = domain.PropertyStatusArchived
		if err := repo.Create(ctx, archived); err != nil {
			t.Fatalf("Create() unexpected error = %v", err)
		}

		got, total, err := repo.List(ctx, domain.PropertyListOptions{OwnerID: ownerID, PropertyAccess: domain.AllPropertyAccess(), Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("List() unexpected error = %v", err)
		}
		if total != 5 { // p + nullables + 2 extra + archived
			t.Errorf("List() total = %d, want 5", total)
		}
		if len(got) != 2 {
			t.Errorf("List() returned %d rows, want 2 (page size)", len(got))
		}

		activeStatus := domain.PropertyStatusActive
		_, activeTotal, err := repo.List(ctx, domain.PropertyListOptions{
			OwnerID: ownerID, PropertyAccess: domain.AllPropertyAccess(), Limit: 10, Filter: domain.PropertyListFilter{Status: &activeStatus},
		})
		if err != nil {
			t.Fatalf("List() with status filter unexpected error = %v", err)
		}
		if activeTotal != 4 {
			t.Errorf("List() active total = %d, want 4", activeTotal)
		}
	})

	t.Run("list searches name and address", func(t *testing.T) {
		named := newTestProperty(ownerID, "42 Searchable Ln")
		named.Name = "Very Findable Towers"
		if err := repo.Create(ctx, named); err != nil {
			t.Fatalf("Create() unexpected error = %v", err)
		}

		got, total, err := repo.List(ctx, domain.PropertyListOptions{
			OwnerID: ownerID, PropertyAccess: domain.AllPropertyAccess(), Limit: 10, Filter: domain.PropertyListFilter{Search: "Findable"},
		})
		if err != nil {
			t.Fatalf("List() with search unexpected error = %v", err)
		}
		if total != 1 || len(got) != 1 || got[0].ID != named.ID {
			t.Errorf("List() search results = %+v (total %d), want just %s", got, total, named.ID)
		}
	})

	t.Run("update", func(t *testing.T) {
		p.AddressLine1 = "2 Integration Way"
		p.UpdatedAt = time.Now().UTC()
		if err := repo.Update(ctx, p); err != nil {
			t.Fatalf("Update() unexpected error = %v", err)
		}
		got, err := repo.GetByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetByID() unexpected error = %v", err)
		}
		if got.AddressLine1 != "2 Integration Way" {
			t.Errorf("Update() address_line1 = %q, want %q", got.AddressLine1, "2 Integration Way")
		}
	})

	t.Run("update missing returns ErrNotFound", func(t *testing.T) {
		ghost := newTestProperty(ownerID, "nowhere")
		if err := repo.Update(ctx, ghost); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("Update() error = %v, want errors.Is(err, %v)", err, domain.ErrNotFound)
		}
	})

	t.Run("bulk update status only affects the given owner's rows", func(t *testing.T) {
		otherOwnerID := seedTestUser(t, pool)
		other := newTestProperty(otherOwnerID, "1 Someone Else's Way")
		if err := repo.Create(ctx, other); err != nil {
			t.Fatalf("Create() unexpected error = %v", err)
		}

		n, err := repo.BulkUpdateStatus(ctx, ownerID, []uuid.UUID{p.ID, other.ID}, domain.PropertyStatusArchived, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("BulkUpdateStatus() unexpected error = %v", err)
		}
		if n != 1 {
			t.Errorf("BulkUpdateStatus() updated = %d, want 1", n)
		}

		got, err := repo.GetByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetByID() unexpected error = %v", err)
		}
		if got.Status != domain.PropertyStatusArchived {
			t.Errorf("BulkUpdateStatus() status = %q, want archived", got.Status)
		}

		gotOther, err := repo.GetByID(ctx, other.ID)
		if err != nil {
			t.Fatalf("GetByID() unexpected error = %v", err)
		}
		if gotOther.Status == domain.PropertyStatusArchived {
			t.Error("BulkUpdateStatus() modified a property belonging to a different owner")
		}
	})

	t.Run("delete missing returns ErrNotFound", func(t *testing.T) {
		if err := repo.Delete(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("Delete() error = %v, want errors.Is(err, %v)", err, domain.ErrNotFound)
		}
	})

	t.Run("ExistsByOwnerAddress", func(t *testing.T) {
		exists, err := repo.ExistsByOwnerAddress(ctx, ownerID, p.AddressLine1)
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
