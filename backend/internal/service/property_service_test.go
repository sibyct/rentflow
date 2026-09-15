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

func (f *fakePropertyRepository) List(_ context.Context, opts domain.PropertyListOptions) ([]*domain.Property, int, error) {
	var matched []*domain.Property
	for _, p := range f.properties {
		if p.OwnerID == opts.OwnerID {
			matched = append(matched, p)
		}
	}
	total := len(matched)
	if opts.Offset > len(matched) {
		return []*domain.Property{}, total, nil
	}
	end := opts.Offset + opts.Limit
	if end > len(matched) {
		end = len(matched)
	}
	return matched[opts.Offset:end], total, nil
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

func (f *fakePropertyRepository) BulkUpdateStatus(_ context.Context, ownerID uuid.UUID, ids []uuid.UUID, status domain.PropertyStatus) (int, error) {
	wanted := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	n := 0
	for id, p := range f.properties {
		if p.OwnerID == ownerID && wanted[id] {
			p.Status = status
			n++
		}
	}
	return n, nil
}

func (f *fakePropertyRepository) ExistsByOwnerAddress(_ context.Context, ownerID uuid.UUID, addressLine1 string) (bool, error) {
	for _, p := range f.properties {
		if p.OwnerID == ownerID && p.AddressLine1 == addressLine1 {
			return true, nil
		}
	}
	return false, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func validCreateInput(ownerID uuid.UUID) domain.CreatePropertyInput {
	return domain.CreatePropertyInput{
		Name:         "Willow Creek Apartments",
		Type:         domain.PropertyTypeResidentialMultiUnit,
		AddressLine1: "123 Main St",
		Units:        4,
		OwnerID:      ownerID,
	}
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
			name:  "valid input creates property with default status",
			input: validCreateInput(ownerID),
		},
		{
			name: "valid input with explicit status",
			input: func() domain.CreatePropertyInput {
				in := validCreateInput(ownerID)
				in.AddressLine1 = "456 Oak Ave"
				in.Status = domain.PropertyStatusActive
				return in
			}(),
		},
		{
			name: "empty name is rejected",
			input: func() domain.CreatePropertyInput {
				in := validCreateInput(ownerID)
				in.Name = ""
				return in
			}(),
			wantErr:   domain.ErrInvalidInput,
			wantField: "name",
		},
		{
			name: "empty address is rejected",
			input: func() domain.CreatePropertyInput {
				in := validCreateInput(ownerID)
				in.AddressLine1 = ""
				return in
			}(),
			wantErr:   domain.ErrInvalidInput,
			wantField: "address_line1",
		},
		{
			name: "zero units is rejected",
			input: func() domain.CreatePropertyInput {
				in := validCreateInput(ownerID)
				in.AddressLine1 = "789 Pine Rd"
				in.Units = 0
				return in
			}(),
			wantErr:   domain.ErrInvalidInput,
			wantField: "units",
		},
		{
			name: "unknown type is rejected",
			input: func() domain.CreatePropertyInput {
				in := validCreateInput(ownerID)
				in.AddressLine1 = "789 Pine Rd"
				in.Type = "condemned"
				return in
			}(),
			wantErr:   domain.ErrInvalidInput,
			wantField: "type",
		},
		{
			name: "unknown status is rejected",
			input: func() domain.CreatePropertyInput {
				in := validCreateInput(ownerID)
				in.AddressLine1 = "789 Pine Rd"
				in.Status = "condemned"
				return in
			}(),
			wantErr:   domain.ErrInvalidInput,
			wantField: "status",
		},
		{
			name: "unknown amenity is rejected",
			input: func() domain.CreatePropertyInput {
				in := validCreateInput(ownerID)
				in.AddressLine1 = "789 Pine Rd"
				in.Amenities = []string{"Parking", "Jacuzzi"}
				return in
			}(),
			wantErr:   domain.ErrInvalidInput,
			wantField: "amenities",
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
			if got.AddressLine1 != tt.input.AddressLine1 {
				t.Errorf("CreateProperty() address_line1 = %q, want %q", got.AddressLine1, tt.input.AddressLine1)
			}
		})
	}
}

func TestPropertyService_GetProperty(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())

	ownerID := uuid.New()
	existing := &domain.Property{
		ID:           uuid.New(),
		Name:         "Existing Place",
		AddressLine1: "1 Existing Way",
		Status:       domain.PropertyStatusActive,
		OwnerID:      ownerID,
	}
	repo.properties[existing.ID] = existing

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetProperty(context.Background(), existing.ID, ownerID)
		if err != nil {
			t.Fatalf("GetProperty() unexpected error = %v", err)
		}
		if got.AddressLine1 != existing.AddressLine1 {
			t.Errorf("GetProperty() address_line1 = %q, want %q", got.AddressLine1, existing.AddressLine1)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetProperty(context.Background(), uuid.New(), ownerID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetProperty() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	// A property that exists but belongs to someone else must read as
	// not-found, not forbidden — see domain.PropertyService's GetProperty
	// doc comment on why (no confirming a given ID belongs to anyone).
	t.Run("belongs to a different owner", func(t *testing.T) {
		_, err := svc.GetProperty(context.Background(), existing.ID, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetProperty() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}

func TestPropertyService_UpdateProperty(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())

	ownerID := uuid.New()
	existing := &domain.Property{
		ID:           uuid.New(),
		Name:         "Existing Place",
		AddressLine1: "1 Existing Way",
		Units:        2,
		Status:       domain.PropertyStatusActive,
		OwnerID:      ownerID,
		UpdatedAt:    time.Now().UTC().Add(-time.Hour),
	}
	repo.properties[existing.ID] = existing

	newAddress := "2 Updated Way"
	got, err := svc.UpdateProperty(context.Background(), existing.ID, ownerID, domain.UpdatePropertyInput{
		AddressLine1: &newAddress,
	})
	if err != nil {
		t.Fatalf("UpdateProperty() unexpected error = %v", err)
	}
	if got.AddressLine1 != newAddress {
		t.Errorf("UpdateProperty() address_line1 = %q, want %q", got.AddressLine1, newAddress)
	}
	if got.Units != existing.Units {
		t.Errorf("UpdateProperty() units changed unexpectedly to %d", got.Units)
	}

	t.Run("not found", func(t *testing.T) {
		addr := "nowhere"
		_, err := svc.UpdateProperty(context.Background(), uuid.New(), ownerID, domain.UpdatePropertyInput{AddressLine1: &addr})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("UpdateProperty() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("belongs to a different owner", func(t *testing.T) {
		addr := "nowhere"
		_, err := svc.UpdateProperty(context.Background(), existing.ID, uuid.New(), domain.UpdatePropertyInput{AddressLine1: &addr})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("UpdateProperty() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("empty address rejected", func(t *testing.T) {
		empty := ""
		_, err := svc.UpdateProperty(context.Background(), existing.ID, ownerID, domain.UpdatePropertyInput{AddressLine1: &empty})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("UpdateProperty() error = %v, want %v", err, domain.ErrInvalidInput)
		}
	})
}

func TestPropertyService_DeleteProperty(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())

	ownerID := uuid.New()
	existing := &domain.Property{ID: uuid.New(), Name: "Existing Place", AddressLine1: "1 Existing Way", OwnerID: ownerID}
	repo.properties[existing.ID] = existing

	t.Run("belongs to a different owner", func(t *testing.T) {
		if err := svc.DeleteProperty(context.Background(), existing.ID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("DeleteProperty() error = %v, want %v", err, domain.ErrNotFound)
		}
		if _, ok := repo.properties[existing.ID]; !ok {
			t.Error("DeleteProperty() removed a property belonging to a different owner")
		}
	})

	if err := svc.DeleteProperty(context.Background(), existing.ID, ownerID); err != nil {
		t.Fatalf("DeleteProperty() unexpected error = %v", err)
	}
	if _, ok := repo.properties[existing.ID]; ok {
		t.Error("DeleteProperty() property still present after delete")
	}

	if err := svc.DeleteProperty(context.Background(), existing.ID, ownerID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("DeleteProperty() second delete error = %v, want %v", err, domain.ErrNotFound)
	}
}

func TestPropertyService_ListProperties_ClampsLimit(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())
	ownerID := uuid.New()

	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.properties[id] = &domain.Property{ID: id, OwnerID: ownerID, Name: "P", AddressLine1: "addr"}
	}

	got, total, err := svc.ListProperties(context.Background(), domain.PropertyListOptions{OwnerID: ownerID, Limit: -1, Offset: -5})
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
		Name:         "",
		AddressLine1: "",
		Units:        0,
		Type:         "condemned",
		Status:       "condemned",
		OwnerID:      uuid.New(),
	})

	var verrs domain.ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("CreateProperty() error = %v, want errors.As to find domain.ValidationErrors", err)
	}
	for _, field := range []string{"name", "type", "address_line1", "units", "status"} {
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

		in := validCreateInput(ownerID)
		in.AddressLine1 = "1 Duplicate Ave"
		if _, err := svc.CreateProperty(context.Background(), in); err != nil {
			t.Fatalf("CreateProperty() unexpected error = %v", err)
		}
	})

	t.Run("second property at the same address for the same owner is rejected", func(t *testing.T) {
		repo := newFakePropertyRepository()
		svc := service.NewPropertyService(repo, nil, noopLogger())
		ctx := context.Background()

		input := validCreateInput(ownerID)
		input.AddressLine1 = "1 Duplicate Ave"
		if _, err := svc.CreateProperty(ctx, input); err != nil {
			t.Fatalf("CreateProperty() first call unexpected error = %v", err)
		}

		_, err := svc.CreateProperty(ctx, input)
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("CreateProperty() error = %v, want %v", err, domain.ErrInvalidInput)
		}
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "address_line1") {
			t.Fatalf("CreateProperty() error = %v, want a ValidationErrors failure for field %q", err, "address_line1")
		}
	})

	t.Run("same address for a different owner is allowed", func(t *testing.T) {
		repo := newFakePropertyRepository()
		svc := service.NewPropertyService(repo, nil, noopLogger())
		ctx := context.Background()

		first := validCreateInput(ownerID)
		first.AddressLine1 = "1 Duplicate Ave"
		if _, err := svc.CreateProperty(ctx, first); err != nil {
			t.Fatalf("CreateProperty() first call unexpected error = %v", err)
		}

		second := validCreateInput(uuid.New())
		second.AddressLine1 = "1 Duplicate Ave"
		if _, err := svc.CreateProperty(ctx, second); err != nil {
			t.Fatalf("CreateProperty() for a different owner unexpected error = %v", err)
		}
	})
}

func TestPropertyService_BulkUpdateStatus(t *testing.T) {
	repo := newFakePropertyRepository()
	svc := service.NewPropertyService(repo, nil, noopLogger())
	ownerID := uuid.New()

	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		id := uuid.New()
		ids = append(ids, id)
		repo.properties[id] = &domain.Property{ID: id, OwnerID: ownerID, Name: "P", AddressLine1: "addr", Status: domain.PropertyStatusActive}
	}
	otherOwnerID := uuid.New()
	otherID := uuid.New()
	repo.properties[otherID] = &domain.Property{ID: otherID, OwnerID: otherOwnerID, Name: "Other", AddressLine1: "addr", Status: domain.PropertyStatusActive}

	n, err := svc.BulkUpdateStatus(context.Background(), ownerID, append(ids, otherID), domain.PropertyStatusArchived)
	if err != nil {
		t.Fatalf("BulkUpdateStatus() unexpected error = %v", err)
	}
	if n != 3 {
		t.Errorf("BulkUpdateStatus() updated = %d, want 3 (not the other owner's property)", n)
	}
	if repo.properties[otherID].Status != domain.PropertyStatusActive {
		t.Error("BulkUpdateStatus() modified a property belonging to a different owner")
	}
	for _, id := range ids {
		if repo.properties[id].Status != domain.PropertyStatusArchived {
			t.Errorf("BulkUpdateStatus() property %s status = %q, want archived", id, repo.properties[id].Status)
		}
	}

	t.Run("empty ids rejected", func(t *testing.T) {
		if _, err := svc.BulkUpdateStatus(context.Background(), ownerID, nil, domain.PropertyStatusArchived); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("BulkUpdateStatus() error = %v, want %v", err, domain.ErrInvalidInput)
		}
	})

	t.Run("unknown status rejected", func(t *testing.T) {
		if _, err := svc.BulkUpdateStatus(context.Background(), ownerID, ids, "condemned"); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("BulkUpdateStatus() error = %v, want %v", err, domain.ErrInvalidInput)
		}
	})
}
