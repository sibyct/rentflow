package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/service"
)

func setupVendorTest(t *testing.T) (*service.VendorService, *fakeVendorRepository, *fakePropertyRepository, uuid.UUID) {
	t.Helper()
	svc, vendorRepo, propertyRepo, _, ownerID := setupVendorTestWithAttachments(t)
	return svc, vendorRepo, propertyRepo, ownerID
}

// setupVendorTestWithAttachments is setupVendorTest plus the fake
// domain.AttachmentRepository, for tests that need to seed an
// attachment to link a COI/tax document to.
func setupVendorTestWithAttachments(t *testing.T) (*service.VendorService, *fakeVendorRepository, *fakePropertyRepository, *fakeAttachmentRepository, uuid.UUID) {
	t.Helper()
	vendorRepo := newFakeVendorRepository()
	propertyRepo := newFakePropertyRepository()
	attachmentRepo := newFakeAttachmentRepository()
	svc := service.NewVendorService(vendorRepo, propertyRepo, attachmentRepo, noopLogger())

	ownerID := uuid.New()
	return svc, vendorRepo, propertyRepo, attachmentRepo, ownerID
}

// fakeAttachmentRepository is an in-memory stand-in for
// domain.AttachmentRepository, shared by the work order and vendor
// tests that validate a *_attachment_id field.
type fakeAttachmentRepository struct {
	attachments map[uuid.UUID]*domain.Attachment
}

func newFakeAttachmentRepository() *fakeAttachmentRepository {
	return &fakeAttachmentRepository{attachments: make(map[uuid.UUID]*domain.Attachment)}
}

func (f *fakeAttachmentRepository) Create(_ context.Context, a *domain.Attachment) error {
	f.attachments[a.ID] = a
	return nil
}

func (f *fakeAttachmentRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Attachment, error) {
	a, ok := f.attachments[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return a, nil
}

func (f *fakeAttachmentRepository) MarkReady(_ context.Context, id uuid.UUID, sizeBytes int64) error {
	a, ok := f.attachments[id]
	if !ok {
		return domain.ErrNotFound
	}
	a.Status = domain.AttachmentStatusReady
	a.SizeBytes = sizeBytes
	return nil
}

// newFakeReadyAttachment seeds an attachment owned by ownerID and
// already 'ready' — the state a real upload is in by the time its id
// reaches a create/update request.
func newFakeReadyAttachment(repo *fakeAttachmentRepository, ownerID uuid.UUID) uuid.UUID {
	id := uuid.New()
	repo.attachments[id] = &domain.Attachment{ID: id, OwnerID: ownerID, Filename: "file.jpg", ContentType: "image/jpeg", Status: domain.AttachmentStatusReady}
	return id
}

func validCreateVendorInput() domain.CreateVendorInput {
	return domain.CreateVendorInput{
		CompanyName:         "Ace Plumbing Co.",
		Categories:          []domain.WorkOrderCategory{domain.WorkOrderCategoryPlumbing},
		ServesAllProperties: true,
	}
}

func TestVendorService_CreateVendor(t *testing.T) {
	t.Run("valid input creates an active vendor", func(t *testing.T) {
		svc, _, _, ownerID := setupVendorTest(t)

		got, err := svc.CreateVendor(context.Background(), ownerID, validCreateVendorInput())
		if err != nil {
			t.Fatalf("CreateVendor() unexpected error = %v", err)
		}
		if !got.Active {
			t.Errorf("CreateVendor() active = false, want true")
		}
		if got.OwnerID != ownerID {
			t.Errorf("CreateVendor() owner_id = %v, want %v", got.OwnerID, ownerID)
		}
	})

	t.Run("missing company name is rejected", func(t *testing.T) {
		svc, _, _, ownerID := setupVendorTest(t)
		input := validCreateVendorInput()
		input.CompanyName = ""

		_, err := svc.CreateVendor(context.Background(), ownerID, input)

		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("CreateVendor() error = %v, want domain.ValidationErrors", err)
		}
	})

	t.Run("no categories is rejected", func(t *testing.T) {
		svc, _, _, ownerID := setupVendorTest(t)
		input := validCreateVendorInput()
		input.Categories = nil

		_, err := svc.CreateVendor(context.Background(), ownerID, input)

		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("CreateVendor() error = %v, want domain.ValidationErrors", err)
		}
	})

	t.Run("unknown category is rejected", func(t *testing.T) {
		svc, _, _, ownerID := setupVendorTest(t)
		input := validCreateVendorInput()
		input.Categories = []domain.WorkOrderCategory{"not-a-real-category"}

		_, err := svc.CreateVendor(context.Background(), ownerID, input)

		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("CreateVendor() error = %v, want domain.ValidationErrors", err)
		}
	})

	t.Run("properties served must belong to the owner", func(t *testing.T) {
		svc, _, propertyRepo, ownerID := setupVendorTest(t)
		other := &domain.Property{ID: uuid.New(), Name: "Someone Else's Place", OwnerID: uuid.New()}
		propertyRepo.properties[other.ID] = other

		input := validCreateVendorInput()
		input.ServesAllProperties = false
		input.PropertiesServed = []uuid.UUID{other.ID}

		_, err := svc.CreateVendor(context.Background(), ownerID, input)

		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("CreateVendor() error = %v, want domain.ValidationErrors", err)
		}
	})

	t.Run("serves all properties ignores any properties served list", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		input := validCreateVendorInput()
		input.ServesAllProperties = true
		input.PropertiesServed = []uuid.UUID{uuid.New()}

		got, err := svc.CreateVendor(context.Background(), ownerID, input)
		if err != nil {
			t.Fatalf("CreateVendor() unexpected error = %v", err)
		}
		if stored := vendorRepo.propertiesServed[got.ID]; stored != nil {
			t.Errorf("CreateVendor() propertiesServed = %v, want nil (serves all)", stored)
		}
	})

	t.Run("scoped properties served are persisted", func(t *testing.T) {
		svc, vendorRepo, propertyRepo, ownerID := setupVendorTest(t)
		mine := &domain.Property{ID: uuid.New(), Name: "Willow Creek", OwnerID: ownerID}
		propertyRepo.properties[mine.ID] = mine

		input := validCreateVendorInput()
		input.ServesAllProperties = false
		input.PropertiesServed = []uuid.UUID{mine.ID}

		got, err := svc.CreateVendor(context.Background(), ownerID, input)
		if err != nil {
			t.Fatalf("CreateVendor() unexpected error = %v", err)
		}
		stored := vendorRepo.propertiesServed[got.ID]
		if len(stored) != 1 || stored[0] != mine.ID {
			t.Errorf("CreateVendor() propertiesServed = %v, want [%v]", stored, mine.ID)
		}
	})
}

func TestVendorService_GetVendor(t *testing.T) {
	t.Run("returns the vendor with stats", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: ownerID, CompanyName: "Ace Plumbing"}
		vendorRepo.vendors[v.ID] = v

		got, err := svc.GetVendor(context.Background(), v.ID, ownerID)
		if err != nil {
			t.Fatalf("GetVendor() unexpected error = %v", err)
		}
		if got.CompanyName != "Ace Plumbing" {
			t.Errorf("GetVendor() company_name = %q, want %q", got.CompanyName, "Ace Plumbing")
		}
	})

	t.Run("vendor belonging to a different owner reads as not found", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: uuid.New(), CompanyName: "Ace Plumbing"}
		vendorRepo.vendors[v.ID] = v

		_, err := svc.GetVendor(context.Background(), v.ID, ownerID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetVendor() error = %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("unknown vendor is not found", func(t *testing.T) {
		svc, _, _, ownerID := setupVendorTest(t)

		_, err := svc.GetVendor(context.Background(), uuid.New(), ownerID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetVendor() error = %v, want domain.ErrNotFound", err)
		}
	})
}

func TestVendorService_UpdateVendor(t *testing.T) {
	t.Run("updates fields on the vendor", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: ownerID, CompanyName: "Ace Plumbing", Categories: []domain.WorkOrderCategory{domain.WorkOrderCategoryPlumbing}, Active: true}
		vendorRepo.vendors[v.ID] = v

		newName := "Ace Plumbing & Heating"
		got, err := svc.UpdateVendor(context.Background(), v.ID, ownerID, domain.UpdateVendorInput{CompanyName: &newName})
		if err != nil {
			t.Fatalf("UpdateVendor() unexpected error = %v", err)
		}
		if got.CompanyName != newName {
			t.Errorf("UpdateVendor() company_name = %q, want %q", got.CompanyName, newName)
		}
	})

	t.Run("vendor belonging to a different owner reads as not found", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: uuid.New(), CompanyName: "Ace Plumbing"}
		vendorRepo.vendors[v.ID] = v

		newName := "Hijacked"
		_, err := svc.UpdateVendor(context.Background(), v.ID, ownerID, domain.UpdateVendorInput{CompanyName: &newName})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("UpdateVendor() error = %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("clearing company name is rejected", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: ownerID, CompanyName: "Ace Plumbing", Categories: []domain.WorkOrderCategory{domain.WorkOrderCategoryPlumbing}}
		vendorRepo.vendors[v.ID] = v

		empty := ""
		_, err := svc.UpdateVendor(context.Background(), v.ID, ownerID, domain.UpdateVendorInput{CompanyName: &empty})

		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("UpdateVendor() error = %v, want domain.ValidationErrors", err)
		}
	})

	t.Run("setting properties served replaces the scope", func(t *testing.T) {
		svc, vendorRepo, propertyRepo, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: ownerID, CompanyName: "Ace Plumbing", Categories: []domain.WorkOrderCategory{domain.WorkOrderCategoryPlumbing}, ServesAllProperties: false}
		vendorRepo.vendors[v.ID] = v
		mine := &domain.Property{ID: uuid.New(), Name: "Willow Creek", OwnerID: ownerID}
		propertyRepo.properties[mine.ID] = mine

		_, err := svc.UpdateVendor(context.Background(), v.ID, ownerID, domain.UpdateVendorInput{
			PropertiesServed:    []uuid.UUID{mine.ID},
			PropertiesServedSet: true,
		})
		if err != nil {
			t.Fatalf("UpdateVendor() unexpected error = %v", err)
		}
		stored := vendorRepo.propertiesServed[v.ID]
		if len(stored) != 1 || stored[0] != mine.ID {
			t.Errorf("UpdateVendor() propertiesServed = %v, want [%v]", stored, mine.ID)
		}
	})
}

func TestVendorService_DeleteVendor(t *testing.T) {
	t.Run("deletes an owned vendor", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: ownerID, CompanyName: "Ace Plumbing"}
		vendorRepo.vendors[v.ID] = v

		if err := svc.DeleteVendor(context.Background(), v.ID, ownerID); err != nil {
			t.Fatalf("DeleteVendor() unexpected error = %v", err)
		}
		if _, ok := vendorRepo.vendors[v.ID]; ok {
			t.Errorf("DeleteVendor() vendor still present after delete")
		}
	})

	t.Run("vendor belonging to a different owner reads as not found", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: uuid.New(), CompanyName: "Ace Plumbing"}
		vendorRepo.vendors[v.ID] = v

		err := svc.DeleteVendor(context.Background(), v.ID, ownerID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("DeleteVendor() error = %v, want domain.ErrNotFound", err)
		}
		if _, ok := vendorRepo.vendors[v.ID]; !ok {
			t.Errorf("DeleteVendor() vendor was deleted despite ownership mismatch")
		}
	})
}

func TestVendorService_GetPropertiesServed(t *testing.T) {
	t.Run("returns the current selection", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: ownerID, CompanyName: "Ace Plumbing"}
		vendorRepo.vendors[v.ID] = v
		want := []uuid.UUID{uuid.New(), uuid.New()}
		vendorRepo.propertiesServed[v.ID] = want

		got, err := svc.GetPropertiesServed(context.Background(), v.ID, ownerID)
		if err != nil {
			t.Fatalf("GetPropertiesServed() unexpected error = %v", err)
		}
		if len(got) != len(want) {
			t.Errorf("GetPropertiesServed() = %v, want %v", got, want)
		}
	})

	t.Run("vendor belonging to a different owner reads as not found", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: uuid.New(), CompanyName: "Ace Plumbing"}
		vendorRepo.vendors[v.ID] = v

		_, err := svc.GetPropertiesServed(context.Background(), v.ID, ownerID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetPropertiesServed() error = %v, want domain.ErrNotFound", err)
		}
	})
}

func TestVendorService_AttachmentFields(t *testing.T) {
	t.Run("create accepts an owned, ready attachment for the COI", func(t *testing.T) {
		svc, _, _, attachmentRepo, ownerID := setupVendorTestWithAttachments(t)
		coiID := newFakeReadyAttachment(attachmentRepo, ownerID)

		input := validCreateVendorInput()
		input.COIAttachmentID = &coiID
		got, err := svc.CreateVendor(context.Background(), ownerID, input)
		if err != nil {
			t.Fatalf("CreateVendor() unexpected error = %v", err)
		}
		if got.COIAttachmentID == nil || *got.COIAttachmentID != coiID {
			t.Errorf("CreateVendor() coi_attachment_id = %v, want %v", got.COIAttachmentID, coiID)
		}
	})

	t.Run("create rejects an attachment owned by someone else", func(t *testing.T) {
		svc, _, _, attachmentRepo, ownerID := setupVendorTestWithAttachments(t)
		coiID := newFakeReadyAttachment(attachmentRepo, uuid.New())

		input := validCreateVendorInput()
		input.COIAttachmentID = &coiID
		_, err := svc.CreateVendor(context.Background(), ownerID, input)

		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("CreateVendor() error = %v, want domain.ValidationErrors", err)
		}
	})

	t.Run("update can clear a previously set tax document", func(t *testing.T) {
		svc, vendorRepo, _, attachmentRepo, ownerID := setupVendorTestWithAttachments(t)
		taxDocID := newFakeReadyAttachment(attachmentRepo, ownerID)
		v := &domain.Vendor{
			ID: uuid.New(), OwnerID: ownerID, CompanyName: "Ace Plumbing",
			Categories: []domain.WorkOrderCategory{domain.WorkOrderCategoryPlumbing}, TaxDocAttachmentID: &taxDocID,
		}
		vendorRepo.vendors[v.ID] = v

		got, err := svc.UpdateVendor(context.Background(), v.ID, ownerID, domain.UpdateVendorInput{TaxDocAttachmentIDSet: true})
		if err != nil {
			t.Fatalf("UpdateVendor() unexpected error = %v", err)
		}
		if got.TaxDocAttachmentID != nil {
			t.Errorf("UpdateVendor() tax_doc_attachment_id = %v, want nil", got.TaxDocAttachmentID)
		}
	})
}

func TestVendorService_GetSpendSummary(t *testing.T) {
	t.Run("vendor belonging to a different owner reads as not found", func(t *testing.T) {
		svc, vendorRepo, _, ownerID := setupVendorTest(t)
		v := &domain.Vendor{ID: uuid.New(), OwnerID: uuid.New(), CompanyName: "Ace Plumbing"}
		vendorRepo.vendors[v.ID] = v

		_, err := svc.GetSpendSummary(context.Background(), v.ID, ownerID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetSpendSummary() error = %v, want domain.ErrNotFound", err)
		}
	})
}
