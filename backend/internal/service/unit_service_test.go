package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/service"
)

// fakeUnitRepository is an in-memory stand-in for domain.UnitRepository.
type fakeUnitRepository struct {
	units map[uuid.UUID]*domain.Unit
}

func newFakeUnitRepository() *fakeUnitRepository {
	return &fakeUnitRepository{units: make(map[uuid.UUID]*domain.Unit)}
}

func (f *fakeUnitRepository) Create(_ context.Context, u *domain.Unit) error {
	f.units[u.ID] = u
	return nil
}

func (f *fakeUnitRepository) CreateMany(_ context.Context, units []*domain.Unit) error {
	for _, u := range units {
		f.units[u.ID] = u
	}
	return nil
}

func (f *fakeUnitRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Unit, error) {
	u, ok := f.units[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUnitRepository) ListByProperty(_ context.Context, propertyID uuid.UUID) ([]*domain.Unit, error) {
	out := make([]*domain.Unit, 0)
	for _, u := range f.units {
		if u.PropertyID == propertyID {
			out = append(out, u)
		}
	}
	return out, nil
}

func (f *fakeUnitRepository) Update(_ context.Context, u *domain.Unit) error {
	if _, ok := f.units[u.ID]; !ok {
		return domain.ErrNotFound
	}
	f.units[u.ID] = u
	return nil
}

func (f *fakeUnitRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.units[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.units, id)
	return nil
}

func (f *fakeUnitRepository) ExistsByPropertyUnitName(_ context.Context, propertyID uuid.UUID, unitName string) (bool, error) {
	for _, u := range f.units {
		if u.PropertyID == propertyID && u.UnitName == unitName {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeUnitRepository) GetPropertyUnitStats(_ context.Context, propertyID uuid.UUID) (*domain.PropertyUnitStats, error) {
	stats := &domain.PropertyUnitStats{PropertyID: propertyID}
	for _, u := range f.units {
		if u.PropertyID != propertyID {
			continue
		}
		stats.UnitCount++
		if u.Status == domain.UnitStatusOccupied {
			stats.OccupiedCount++
			if u.CurrentRent != nil {
				stats.TotalCollected += *u.CurrentRent
			}
		}
	}
	if stats.UnitCount > 0 {
		stats.OccupancyPct = stats.OccupiedCount * 100 / stats.UnitCount
	}
	return stats, nil
}

func (f *fakeUnitRepository) ListForOwner(_ context.Context, opts domain.UnitListOptions) ([]*domain.UnitWithProperty, int, error) {
	out := make([]*domain.UnitWithProperty, 0)
	for _, u := range f.units {
		if opts.Filter.PropertyID != nil && u.PropertyID != *opts.Filter.PropertyID {
			continue
		}
		if opts.Filter.Status != nil && u.Status != *opts.Filter.Status {
			continue
		}
		if opts.Filter.Type != nil && u.Type != *opts.Filter.Type {
			continue
		}
		out = append(out, &domain.UnitWithProperty{Unit: *u})
	}
	return out, len(out), nil
}

func (f *fakeUnitRepository) GetPropertyUnitStatsBulk(ctx context.Context, propertyIDs []uuid.UUID) (map[uuid.UUID]*domain.PropertyUnitStats, error) {
	out := make(map[uuid.UUID]*domain.PropertyUnitStats, len(propertyIDs))
	for _, id := range propertyIDs {
		stats, _ := f.GetPropertyUnitStats(ctx, id)
		out[id] = stats
	}
	return out, nil
}

func setupUnitTest(t *testing.T) (*service.UnitService, *fakePropertyRepository, *fakeUnitRepository, uuid.UUID, *domain.Property) {
	t.Helper()
	propertyRepo := newFakePropertyRepository()
	unitRepo := newFakeUnitRepository()
	svc := service.NewUnitService(unitRepo, propertyRepo, newFakeUnitDocumentRepository(), newFakeAttachmentRepository(), noopLogger())

	ownerID := uuid.New()
	property := &domain.Property{
		ID:           uuid.New(),
		Name:         "Willow Creek Apartments",
		Type:         domain.PropertyTypeResidentialMultiUnit,
		AddressLine1: "123 Main St",
		OwnerID:      ownerID,
	}
	propertyRepo.properties[property.ID] = property

	return svc, propertyRepo, unitRepo, ownerID, property
}

func validCreateUnitInput(propertyID uuid.UUID) domain.CreateUnitInput {
	return domain.CreateUnitInput{
		PropertyID: propertyID,
		UnitName:   "1A",
		Type:       domain.UnitTypeOneBed,
	}
}

func TestUnitService_CreateUnit(t *testing.T) {
	t.Run("valid input creates unit with default status", func(t *testing.T) {
		svc, _, _, ownerID, property := setupUnitTest(t)

		got, err := svc.CreateUnit(context.Background(), ownerID, validCreateUnitInput(property.ID), domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("CreateUnit() unexpected error = %v", err)
		}
		if got.Status != domain.UnitStatusVacant {
			t.Errorf("CreateUnit() status = %q, want %q", got.Status, domain.UnitStatusVacant)
		}
	})

	t.Run("property belongs to a different owner reads as not found", func(t *testing.T) {
		svc, _, _, _, property := setupUnitTest(t)

		_, err := svc.CreateUnit(context.Background(), uuid.New(), validCreateUnitInput(property.ID), domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("CreateUnit() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("unknown type is rejected", func(t *testing.T) {
		svc, _, _, ownerID, property := setupUnitTest(t)

		input := validCreateUnitInput(property.ID)
		input.Type = "castle"
		_, err := svc.CreateUnit(context.Background(), ownerID, input, domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("CreateUnit() error = %v, want %v", err, domain.ErrInvalidInput)
		}
	})

	t.Run("duplicate unit name within the same property is rejected", func(t *testing.T) {
		svc, _, _, ownerID, property := setupUnitTest(t)

		input := validCreateUnitInput(property.ID)
		if _, err := svc.CreateUnit(context.Background(), ownerID, input, domain.AllPropertyAccess()); err != nil {
			t.Fatalf("CreateUnit() first call unexpected error = %v", err)
		}

		_, err := svc.CreateUnit(context.Background(), ownerID, input, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "unit_name") {
			t.Fatalf("CreateUnit() error = %v, want a ValidationErrors failure for field %q", err, "unit_name")
		}
	})
}

func TestUnitService_GetUnit(t *testing.T) {
	svc, _, unitRepo, ownerID, property := setupUnitTest(t)

	existing := &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1A", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}
	unitRepo.units[existing.ID] = existing

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetUnit(context.Background(), existing.ID, ownerID, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("GetUnit() unexpected error = %v", err)
		}
		if got.UnitName != "1A" {
			t.Errorf("GetUnit() unit_name = %q, want %q", got.UnitName, "1A")
		}
	})

	t.Run("belongs to a property owned by someone else", func(t *testing.T) {
		_, err := svc.GetUnit(context.Background(), existing.ID, uuid.New(), domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetUnit() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	// Property-scope enforcement (Piece 1): a staff member scoped away
	// from this unit's parent property must read it as not-found.
	t.Run("owned but outside scoped access", func(t *testing.T) {
		_, err := svc.GetUnit(context.Background(), existing.ID, ownerID, domain.PropertyAccess{PropertyIDs: []uuid.UUID{uuid.New()}})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetUnit() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("owned and within scoped access", func(t *testing.T) {
		got, err := svc.GetUnit(context.Background(), existing.ID, ownerID, domain.PropertyAccess{PropertyIDs: []uuid.UUID{property.ID}})
		if err != nil {
			t.Fatalf("GetUnit() unexpected error = %v", err)
		}
		if got.ID != existing.ID {
			t.Errorf("GetUnit() id = %v, want %v", got.ID, existing.ID)
		}
	})
}

func TestUnitService_UpdateUnit(t *testing.T) {
	svc, _, unitRepo, ownerID, property := setupUnitTest(t)

	existing := &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1A", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}
	unitRepo.units[existing.ID] = existing
	other := &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1B", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}
	unitRepo.units[other.ID] = other

	t.Run("rename succeeds", func(t *testing.T) {
		newName := "2A"
		got, err := svc.UpdateUnit(context.Background(), existing.ID, ownerID, domain.UpdateUnitInput{UnitName: &newName}, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("UpdateUnit() unexpected error = %v", err)
		}
		if got.UnitName != "2A" {
			t.Errorf("UpdateUnit() unit_name = %q, want %q", got.UnitName, "2A")
		}
	})

	t.Run("rename to an already-taken name is rejected", func(t *testing.T) {
		clash := "1B"
		_, err := svc.UpdateUnit(context.Background(), existing.ID, ownerID, domain.UpdateUnitInput{UnitName: &clash}, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "unit_name") {
			t.Fatalf("UpdateUnit() error = %v, want a ValidationErrors failure for field %q", err, "unit_name")
		}
	})

	t.Run("belongs to a different owner", func(t *testing.T) {
		newName := "3A"
		_, err := svc.UpdateUnit(context.Background(), existing.ID, uuid.New(), domain.UpdateUnitInput{UnitName: &newName}, domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("UpdateUnit() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}

func TestUnitService_DeleteUnit(t *testing.T) {
	svc, _, unitRepo, ownerID, property := setupUnitTest(t)

	existing := &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1A", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}
	unitRepo.units[existing.ID] = existing

	t.Run("belongs to a different owner", func(t *testing.T) {
		if err := svc.DeleteUnit(context.Background(), existing.ID, uuid.New(), domain.AllPropertyAccess()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("DeleteUnit() error = %v, want %v", err, domain.ErrNotFound)
		}
		if _, ok := unitRepo.units[existing.ID]; !ok {
			t.Error("DeleteUnit() removed a unit belonging to a different owner's property")
		}
	})

	if err := svc.DeleteUnit(context.Background(), existing.ID, ownerID, domain.AllPropertyAccess()); err != nil {
		t.Fatalf("DeleteUnit() unexpected error = %v", err)
	}
	if _, ok := unitRepo.units[existing.ID]; ok {
		t.Error("DeleteUnit() unit still present after delete")
	}
}

func TestUnitService_CreateUnitsBulk(t *testing.T) {
	t.Run("valid batch creates every unit", func(t *testing.T) {
		svc, _, _, ownerID, property := setupUnitTest(t)

		inputs := []domain.CreateUnitInput{
			{UnitName: "1A", Type: domain.UnitTypeOneBed},
			{UnitName: "1B", Type: domain.UnitTypeStudio},
			{UnitName: "1C", Type: domain.UnitTypeTwoBed},
		}
		got, err := svc.CreateUnitsBulk(context.Background(), ownerID, property.ID, inputs, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("CreateUnitsBulk() unexpected error = %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("CreateUnitsBulk() returned %d units, want 3", len(got))
		}
	})

	t.Run("duplicate name within the batch rejects the whole batch", func(t *testing.T) {
		svc, _, unitRepo, ownerID, property := setupUnitTest(t)

		inputs := []domain.CreateUnitInput{
			{UnitName: "1A", Type: domain.UnitTypeOneBed},
			{UnitName: "1A", Type: domain.UnitTypeStudio},
		}
		_, err := svc.CreateUnitsBulk(context.Background(), ownerID, property.ID, inputs, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("CreateUnitsBulk() error = %v, want errors.As to find domain.ValidationErrors", err)
		}
		if len(unitRepo.units) != 0 {
			t.Error("CreateUnitsBulk() partially committed a rejected batch")
		}
	})

	t.Run("property belongs to a different owner", func(t *testing.T) {
		svc, _, _, _, property := setupUnitTest(t)

		_, err := svc.CreateUnitsBulk(context.Background(), uuid.New(), property.ID, []domain.CreateUnitInput{{UnitName: "1A", Type: domain.UnitTypeOneBed}}, domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("CreateUnitsBulk() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}

func TestUnitService_ListUnitsForOwner_AppliesDefaults(t *testing.T) {
	svc, _, unitRepo, ownerID, property := setupUnitTest(t)

	unitRepo.units[uuid.New()] = &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1A", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}

	units, total, err := svc.ListUnitsForOwner(context.Background(), ownerID, domain.UnitListOptions{PropertyAccess: domain.AllPropertyAccess(), Limit: -1, Offset: -5})
	if err != nil {
		t.Fatalf("ListUnitsForOwner() unexpected error = %v", err)
	}
	if total != 1 || len(units) != 1 {
		t.Fatalf("ListUnitsForOwner() = %d results (total %d), want 1", len(units), total)
	}
}

func TestUnitService_GetPropertyUnitStats(t *testing.T) {
	svc, _, unitRepo, ownerID, property := setupUnitTest(t)

	occupiedRent := 1500.0
	unitRepo.units[uuid.New()] = &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1A", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusOccupied, CurrentRent: &occupiedRent}
	unitRepo.units[uuid.New()] = &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1B", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}

	stats, err := svc.GetPropertyUnitStats(context.Background(), property.ID, ownerID, domain.AllPropertyAccess())
	if err != nil {
		t.Fatalf("GetPropertyUnitStats() unexpected error = %v", err)
	}
	if stats.UnitCount != 2 || stats.OccupiedCount != 1 {
		t.Errorf("GetPropertyUnitStats() = %+v, want UnitCount=2 OccupiedCount=1", stats)
	}
	if stats.TotalCollected != occupiedRent {
		t.Errorf("GetPropertyUnitStats() TotalCollected = %v, want %v", stats.TotalCollected, occupiedRent)
	}
	if stats.OccupancyPct != 50 {
		t.Errorf("GetPropertyUnitStats() OccupancyPct = %d, want 50", stats.OccupancyPct)
	}

	t.Run("belongs to a different owner", func(t *testing.T) {
		_, err := svc.GetPropertyUnitStats(context.Background(), property.ID, uuid.New(), domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetPropertyUnitStats() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}
