package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/service"
)

// fakePropertyRepository is an in-memory stand-in for domain.PropertyRepository.
type fakePropertyRepository struct {
	properties map[uuid.UUID]*domain.Property
	createErr  error
	getErr     error
}

func newFakePropertyRepository() *fakePropertyRepository {
	return &fakePropertyRepository{properties: make(map[uuid.UUID]*domain.Property)}
}

func (f *fakePropertyRepository) Create(_ context.Context, p *domain.Property) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.properties[p.ID] = p
	return nil
}

func (f *fakePropertyRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Property, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	p, ok := f.properties[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (f *fakePropertyRepository) List(_ context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Property, int, error) {
	var matched []*domain.Property
	for _, p := range f.properties {
		if p.OwnerID == ownerID {
			matched = append(matched, p)
		}
	}
	total := len(matched)
	if offset > len(matched) {
		return []*domain.Property{}, total, nil
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	return matched[offset:end], total, nil
}

func (f *fakePropertyRepository) Update(_ context.Context, p *domain.Property) error {
	if _, ok := f.properties[p.ID]; !ok {
		return domain.ErrNotFound
	}
	f.properties[p.ID] = p
	return nil
}

func (f *fakePropertyRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.properties[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.properties, id)
	return nil
}

func (f *fakePropertyRepository) ExistsByOwnerAddress(_ context.Context, ownerID uuid.UUID, address string) (bool, error) {
	for _, p := range f.properties {
		if p.OwnerID == ownerID && p.Address == address {
			return true, nil
		}
	}
	return false, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestPropertyService_CreateProperty(t *testing.T) {
	ownerID := uuid.New()

	tests := []struct {
		name      string
		input     domain.CreatePropertyInput
		wantErr   error
		wantField string // if set, asserts errors.As finds this field among the ValidationErrors
	}{
		{
			name: "valid input creates property with default status",
			input: domain.CreatePropertyInput{
				Address:   "123 Main St",
				UnitCount: 4,
				OwnerID:   ownerID,
			},
		},
		{
			name: "valid input with explicit status",
			input: domain.CreatePropertyInput{
				Address:   "456 Oak Ave",
				UnitCount: 1,
				Status:    domain.PropertyStatusMaintenance,
				OwnerID:   ownerID,
			},
		},
		{
			name: "empty address is rejected",
			input: domain.CreatePropertyInput{
				Address:   "",
				UnitCount: 1,
				OwnerID:   ownerID,
			},
			wantErr:   domain.ErrInvalidInput,
			wantField: "address",
		},
		{
			name: "zero unit count is rejected",
			input: domain.CreatePropertyInput{
				Address:   "789 Pine Rd",
				UnitCount: 0,
				OwnerID:   ownerID,
			},
			wantErr:   domain.ErrInvalidInput,
			wantField: "unit_count",
		},
		{
			name: "unknown status is rejected",
			input: domain.CreatePropertyInput{
				Address:   "789 Pine Rd",
				UnitCount: 2,
				Status:    "condemned",
				OwnerID:   ownerID,
			},
			wantErr:   domain.ErrInvalidInput,
			wantField: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakePropertyRepository()
			svc := service.NewPropertyService(repo, nil, noopLogger())

			got, err := svc.CreateProperty(context.Background(), tt.input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CreateProperty() error = %v, want %v", err, tt.wantErr)
				}
				if tt.wantField != "" {
					var verrs domain.ValidationErrors
					if !errors.As(err, &verrs) {
						t.Fatalf("CreateProperty() error = %v, want errors.As to find domain.ValidationErrors", err)
					}
					if !hasField(verrs, tt.wantField) {
						t.Fatalf("CreateProperty() ValidationErrors = %v, want a failure for field %q", verrs, tt.wantField)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateProperty() unexpected error = %v", err)
			}
			if got.ID == uuid.Nil {
				t.Error("CreateProperty() returned property with nil ID")
			}
			if got.Status == "" {
				t.Error("CreateProperty() returned property with empty status")
			}
			if got.Address != tt.input.Address {
				t.Errorf("CreateProperty() address = %q, want %q", got.Address, tt.input.Address)
			}
		})
	}
}

func TestPropertyService_GetProperty(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())

	existing := &domain.Property{
		ID:      uuid.New(),
		Address: "1 Existing Way",
		Status:  domain.PropertyStatusActive,
	}
	repo.properties[existing.ID] = existing

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetProperty(context.Background(), existing.ID)
		if err != nil {
			t.Fatalf("GetProperty() unexpected error = %v", err)
		}
		if got.Address != existing.Address {
			t.Errorf("GetProperty() address = %q, want %q", got.Address, existing.Address)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetProperty(context.Background(), uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetProperty() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}

func TestPropertyService_UpdateProperty(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())

	existing := &domain.Property{
		ID:        uuid.New(),
		Address:   "1 Existing Way",
		UnitCount: 2,
		Status:    domain.PropertyStatusActive,
		UpdatedAt: time.Now().UTC().Add(-time.Hour),
	}
	repo.properties[existing.ID] = existing

	newAddress := "2 Updated Way"
	got, err := svc.UpdateProperty(context.Background(), existing.ID, domain.UpdatePropertyInput{
		Address: &newAddress,
	})
	if err != nil {
		t.Fatalf("UpdateProperty() unexpected error = %v", err)
	}
	if got.Address != newAddress {
		t.Errorf("UpdateProperty() address = %q, want %q", got.Address, newAddress)
	}
	if got.UnitCount != existing.UnitCount {
		t.Errorf("UpdateProperty() unit count changed unexpectedly to %d", got.UnitCount)
	}

	t.Run("not found", func(t *testing.T) {
		addr := "nowhere"
		_, err := svc.UpdateProperty(context.Background(), uuid.New(), domain.UpdatePropertyInput{Address: &addr})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("UpdateProperty() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("empty address rejected", func(t *testing.T) {
		empty := ""
		_, err := svc.UpdateProperty(context.Background(), existing.ID, domain.UpdatePropertyInput{Address: &empty})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("UpdateProperty() error = %v, want %v", err, domain.ErrInvalidInput)
		}
	})
}

func TestPropertyService_DeleteProperty(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())

	existing := &domain.Property{ID: uuid.New(), Address: "1 Existing Way"}
	repo.properties[existing.ID] = existing

	if err := svc.DeleteProperty(context.Background(), existing.ID); err != nil {
		t.Fatalf("DeleteProperty() unexpected error = %v", err)
	}
	if _, ok := repo.properties[existing.ID]; ok {
		t.Error("DeleteProperty() property still present after delete")
	}

	if err := svc.DeleteProperty(context.Background(), existing.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("DeleteProperty() second delete error = %v, want %v", err, domain.ErrNotFound)
	}
}

func TestPropertyService_ListProperties_ClampsLimit(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())
	ownerID := uuid.New()

	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.properties[id] = &domain.Property{ID: id, OwnerID: ownerID, Address: "addr"}
	}

	got, total, err := svc.ListProperties(context.Background(), ownerID, -1, -5)
	if err != nil {
		t.Fatalf("ListProperties() unexpected error = %v", err)
	}
	if total != 3 {
		t.Errorf("ListProperties() total = %d, want 3", total)
	}
	if len(got) != 3 {
		t.Errorf("ListProperties() returned %d properties, want 3", len(got))
	}
}

func TestPropertyService_CreateProperty_AccumulatesAllValidationErrors(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())

	_, err := svc.CreateProperty(context.Background(), domain.CreatePropertyInput{
		Address:   "",
		UnitCount: 0,
		Status:    "condemned",
		OwnerID:   uuid.New(),
	})

	var verrs domain.ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("CreateProperty() error = %v, want errors.As to find domain.ValidationErrors", err)
	}
	for _, field := range []string{"address", "unit_count", "status"} {
		if !hasField(verrs, field) {
			t.Errorf("CreateProperty() ValidationErrors = %v, missing failure for field %q", verrs, field)
		}
	}
}

func hasField(verrs domain.ValidationErrors, field string) bool {
	for _, fe := range verrs {
		if fe.Field == field {
			return true
		}
	}
	return false
}

// TestPropertyService_CreateProperty_DuplicateAddress covers the one
// business rule in this service that needs state (an existing row),
// not just the shape of the input — see domain.PropertyRepository's
// ExistsByOwnerAddress.
func TestPropertyService_CreateProperty_DuplicateAddress(t *testing.T) {
	ownerID := uuid.New()

	t.Run("first property at an address succeeds", func(t *testing.T) {
		repo := newFakePropertyRepository()
		svc := service.NewPropertyService(repo, nil, noopLogger())

		_, err := svc.CreateProperty(context.Background(), domain.CreatePropertyInput{
			Address: "1 Duplicate Ave", UnitCount: 1, OwnerID: ownerID,
		})
		if err != nil {
			t.Fatalf("CreateProperty() unexpected error = %v", err)
		}
	})

	t.Run("second property at the same address for the same owner is rejected", func(t *testing.T) {
		repo := newFakePropertyRepository()
		svc := service.NewPropertyService(repo, nil, noopLogger())
		ctx := context.Background()

		input := domain.CreatePropertyInput{Address: "1 Duplicate Ave", UnitCount: 1, OwnerID: ownerID}
		if _, err := svc.CreateProperty(ctx, input); err != nil {
			t.Fatalf("CreateProperty() first call unexpected error = %v", err)
		}

		_, err := svc.CreateProperty(ctx, input)
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("CreateProperty() error = %v, want %v", err, domain.ErrInvalidInput)
		}
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "address") {
			t.Fatalf("CreateProperty() error = %v, want a ValidationErrors failure for field %q", err, "address")
		}
	})

	t.Run("same address for a different owner is allowed", func(t *testing.T) {
		repo := newFakePropertyRepository()
		svc := service.NewPropertyService(repo, nil, noopLogger())
		ctx := context.Background()

		if _, err := svc.CreateProperty(ctx, domain.CreatePropertyInput{
			Address: "1 Duplicate Ave", UnitCount: 1, OwnerID: ownerID,
		}); err != nil {
			t.Fatalf("CreateProperty() first call unexpected error = %v", err)
		}

		_, err := svc.CreateProperty(ctx, domain.CreatePropertyInput{
			Address: "1 Duplicate Ave", UnitCount: 2, OwnerID: uuid.New(),
		})
		if err != nil {
			t.Fatalf("CreateProperty() for a different owner unexpected error = %v", err)
		}
	})
}
