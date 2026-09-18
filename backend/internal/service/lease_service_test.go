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

// fakeLeaseRepository is an in-memory stand-in for domain.LeaseRepository.
type fakeLeaseRepository struct {
	leases map[uuid.UUID]*domain.Lease
}

func newFakeLeaseRepository() *fakeLeaseRepository {
	return &fakeLeaseRepository{leases: make(map[uuid.UUID]*domain.Lease)}
}

func (f *fakeLeaseRepository) Create(_ context.Context, l *domain.Lease) error {
	f.leases[l.ID] = l
	return nil
}

func (f *fakeLeaseRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Lease, error) {
	l, ok := f.leases[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return l, nil
}

func (f *fakeLeaseRepository) ListForOwner(_ context.Context, _ domain.LeaseListOptions) ([]*domain.LeaseWithUnitProperty, int, error) {
	out := make([]*domain.LeaseWithUnitProperty, 0, len(f.leases))
	for _, l := range f.leases {
		out = append(out, &domain.LeaseWithUnitProperty{Lease: *l})
	}
	return out, len(out), nil
}

func (f *fakeLeaseRepository) Update(_ context.Context, l *domain.Lease) error {
	if _, ok := f.leases[l.ID]; !ok {
		return domain.ErrNotFound
	}
	f.leases[l.ID] = l
	return nil
}

func (f *fakeLeaseRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.leases[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.leases, id)
	return nil
}

func (f *fakeLeaseRepository) HasActiveLease(_ context.Context, unitID uuid.UUID, excludeLeaseID *uuid.UUID) (bool, error) {
	for _, l := range f.leases {
		if l.UnitID != unitID || l.Status != domain.LeaseStatusActive {
			continue
		}
		if excludeLeaseID != nil && l.ID == *excludeLeaseID {
			continue
		}
		return true, nil
	}
	return false, nil
}

func setupLeaseTest(t *testing.T) (*service.LeaseService, *fakeLeaseRepository, uuid.UUID, *domain.Unit) {
	t.Helper()
	propertyRepo := newFakePropertyRepository()
	unitRepo := newFakeUnitRepository()
	leaseRepo := newFakeLeaseRepository()
	svc := service.NewLeaseService(leaseRepo, unitRepo, propertyRepo, noopLogger())

	ownerID := uuid.New()
	property := &domain.Property{ID: uuid.New(), Name: "Willow Creek Apartments", Type: domain.PropertyTypeResidentialMultiUnit, AddressLine1: "123 Main St", OwnerID: ownerID}
	propertyRepo.properties[property.ID] = property

	unit := &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1A", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}
	unitRepo.units[unit.ID] = unit

	return svc, leaseRepo, ownerID, unit
}

func validCreateLeaseInput(unitID uuid.UUID) domain.CreateLeaseInput {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	return domain.CreateLeaseInput{
		UnitID:              unitID,
		Type:                domain.LeaseTypeFixed,
		StartDate:           start,
		EndDate:             &end,
		MonthlyRent:         1500,
		PrimaryResidentName: "Jordan Rivera",
	}
}

func TestLeaseService_CreateLease(t *testing.T) {
	t.Run("valid input creates an active lease by default", func(t *testing.T) {
		svc, _, ownerID, unit := setupLeaseTest(t)

		got, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID))
		if err != nil {
			t.Fatalf("CreateLease() unexpected error = %v", err)
		}
		if got.Status != domain.LeaseStatusActive {
			t.Errorf("CreateLease() status = %q, want %q", got.Status, domain.LeaseStatusActive)
		}
		if got.RenewalStatus != domain.RenewalStatusNotStarted {
			t.Errorf("CreateLease() renewal_status = %q, want %q", got.RenewalStatus, domain.RenewalStatusNotStarted)
		}
	})

	t.Run("unit belongs to a different owner reads as not found", func(t *testing.T) {
		svc, _, _, unit := setupLeaseTest(t)

		_, err := svc.CreateLease(context.Background(), uuid.New(), validCreateLeaseInput(unit.ID))
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("CreateLease() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("fixed-term lease without an end date is rejected", func(t *testing.T) {
		svc, _, ownerID, unit := setupLeaseTest(t)

		input := validCreateLeaseInput(unit.ID)
		input.EndDate = nil
		_, err := svc.CreateLease(context.Background(), ownerID, input)
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "end_date") {
			t.Fatalf("CreateLease() error = %v, want a ValidationErrors failure for field %q", err, "end_date")
		}
	})

	t.Run("month-to-month lease needs no end date", func(t *testing.T) {
		svc, _, ownerID, unit := setupLeaseTest(t)

		input := validCreateLeaseInput(unit.ID)
		input.Type = domain.LeaseTypeMonthToMonth
		input.EndDate = nil
		if _, err := svc.CreateLease(context.Background(), ownerID, input); err != nil {
			t.Fatalf("CreateLease() unexpected error = %v", err)
		}
	})

	t.Run("a second active lease on the same unit is rejected", func(t *testing.T) {
		svc, _, ownerID, unit := setupLeaseTest(t)

		if _, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID)); err != nil {
			t.Fatalf("CreateLease() first call unexpected error = %v", err)
		}

		_, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID))
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "status") {
			t.Fatalf("CreateLease() error = %v, want a ValidationErrors failure for field %q", err, "status")
		}
	})

	t.Run("a draft lease does not conflict with an active one", func(t *testing.T) {
		svc, _, ownerID, unit := setupLeaseTest(t)

		if _, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID)); err != nil {
			t.Fatalf("CreateLease() first call unexpected error = %v", err)
		}

		draft := validCreateLeaseInput(unit.ID)
		draft.Status = domain.LeaseStatusDraft
		if _, err := svc.CreateLease(context.Background(), ownerID, draft); err != nil {
			t.Fatalf("CreateLease() draft lease unexpected error = %v", err)
		}
	})
}

func TestLeaseService_GetLease(t *testing.T) {
	svc, _, ownerID, unit := setupLeaseTest(t)

	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID))
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}

	t.Run("found includes unit and property context", func(t *testing.T) {
		got, err := svc.GetLease(context.Background(), created.ID, ownerID)
		if err != nil {
			t.Fatalf("GetLease() unexpected error = %v", err)
		}
		if got.UnitName != unit.UnitName {
			t.Errorf("GetLease() unit_name = %q, want %q", got.UnitName, unit.UnitName)
		}
	})

	t.Run("belongs to a different owner", func(t *testing.T) {
		_, err := svc.GetLease(context.Background(), created.ID, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetLease() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}

func TestLeaseService_UpdateLease(t *testing.T) {
	svc, _, ownerID, unit := setupLeaseTest(t)

	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID))
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}

	t.Run("terminate an active lease", func(t *testing.T) {
		terminated := domain.LeaseStatusTerminated
		got, err := svc.UpdateLease(context.Background(), created.ID, ownerID, domain.UpdateLeaseInput{Status: &terminated})
		if err != nil {
			t.Fatalf("UpdateLease() unexpected error = %v", err)
		}
		if got.Status != domain.LeaseStatusTerminated {
			t.Errorf("UpdateLease() status = %q, want %q", got.Status, domain.LeaseStatusTerminated)
		}
	})

	t.Run("belongs to a different owner", func(t *testing.T) {
		rent := 1600.0
		_, err := svc.UpdateLease(context.Background(), created.ID, uuid.New(), domain.UpdateLeaseInput{MonthlyRent: &rent})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("UpdateLease() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("empty primary resident name is rejected", func(t *testing.T) {
		empty := ""
		_, err := svc.UpdateLease(context.Background(), created.ID, ownerID, domain.UpdateLeaseInput{PrimaryResidentName: &empty})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("UpdateLease() error = %v, want %v", err, domain.ErrInvalidInput)
		}
	})
}

func TestLeaseService_DeleteLease(t *testing.T) {
	svc, leaseRepo, ownerID, unit := setupLeaseTest(t)

	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID))
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}

	t.Run("belongs to a different owner", func(t *testing.T) {
		if err := svc.DeleteLease(context.Background(), created.ID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("DeleteLease() error = %v, want %v", err, domain.ErrNotFound)
		}
		if _, ok := leaseRepo.leases[created.ID]; !ok {
			t.Error("DeleteLease() removed a lease belonging to a different owner's property")
		}
	})

	if err := svc.DeleteLease(context.Background(), created.ID, ownerID); err != nil {
		t.Fatalf("DeleteLease() unexpected error = %v", err)
	}
	if _, ok := leaseRepo.leases[created.ID]; ok {
		t.Error("DeleteLease() lease still present after delete")
	}
}
