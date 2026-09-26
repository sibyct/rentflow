package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/service"
)

// fakeUnitDocumentRepository is an in-memory stand-in for
// domain.UnitDocumentRepository.
type fakeUnitDocumentRepository struct {
	docs map[uuid.UUID]*domain.UnitDocument
}

func newFakeUnitDocumentRepository() *fakeUnitDocumentRepository {
	return &fakeUnitDocumentRepository{docs: make(map[uuid.UUID]*domain.UnitDocument)}
}

func (f *fakeUnitDocumentRepository) Create(_ context.Context, d *domain.UnitDocument) error {
	f.docs[d.ID] = d
	return nil
}

func (f *fakeUnitDocumentRepository) ListByUnit(_ context.Context, unitID uuid.UUID) ([]*domain.UnitDocument, error) {
	var out []*domain.UnitDocument
	for _, d := range f.docs {
		if d.UnitID == unitID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeUnitDocumentRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.UnitDocument, error) {
	d, ok := f.docs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return d, nil
}

func (f *fakeUnitDocumentRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.docs[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.docs, id)
	return nil
}

func setupUnitDocumentTest(t *testing.T) (*service.UnitService, *fakeUnitDocumentRepository, *fakeAttachmentRepository, *fakeUnitRepository, uuid.UUID, *domain.Unit) {
	t.Helper()
	propertyRepo := newFakePropertyRepository()
	unitRepo := newFakeUnitRepository()
	documentRepo := newFakeUnitDocumentRepository()
	attachmentRepo := newFakeAttachmentRepository()
	svc := service.NewUnitService(unitRepo, propertyRepo, documentRepo, attachmentRepo, noopLogger())

	ownerID := uuid.New()
	property := &domain.Property{ID: uuid.New(), Name: "Willow Creek Apartments", Type: domain.PropertyTypeResidentialMultiUnit, AddressLine1: "123 Main St", OwnerID: ownerID}
	propertyRepo.properties[property.ID] = property

	unit := &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1A", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}
	unitRepo.units[unit.ID] = unit

	return svc, documentRepo, attachmentRepo, unitRepo, ownerID, unit
}

func TestUnitService_AddDocument(t *testing.T) {
	svc, _, attachmentRepo, _, ownerID, unit := setupUnitDocumentTest(t)
	photoID := newFakeReadyAttachment(attachmentRepo, ownerID)
	uploader := uuid.New()

	t.Run("valid input creates a document", func(t *testing.T) {
		got, err := svc.AddDocument(context.Background(), unit.ID, ownerID, uploader, photoID, domain.UnitDocumentCategoryInspection, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("AddDocument() unexpected error = %v", err)
		}
		if got.Category != domain.UnitDocumentCategoryInspection {
			t.Errorf("AddDocument() category = %q, want %q", got.Category, domain.UnitDocumentCategoryInspection)
		}
		if got.UploadedBy != uploader {
			t.Errorf("AddDocument() uploaded_by = %v, want %v", got.UploadedBy, uploader)
		}
	})

	t.Run("unknown category is rejected", func(t *testing.T) {
		_, err := svc.AddDocument(context.Background(), unit.ID, ownerID, uploader, photoID, "castle", domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "category") {
			t.Fatalf("AddDocument() error = %v, want a ValidationErrors failure for field %q", err, "category")
		}
	})

	t.Run("attachment owned by someone else is rejected", func(t *testing.T) {
		foreignID := newFakeReadyAttachment(attachmentRepo, uuid.New())
		_, err := svc.AddDocument(context.Background(), unit.ID, ownerID, uploader, foreignID, domain.UnitDocumentCategoryOther, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "attachment_id") {
			t.Fatalf("AddDocument() error = %v, want a ValidationErrors failure for field %q", err, "attachment_id")
		}
	})

	t.Run("unit belongs to a different owner reads as not found", func(t *testing.T) {
		_, err := svc.AddDocument(context.Background(), unit.ID, uuid.New(), uploader, photoID, domain.UnitDocumentCategoryOther, domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("AddDocument() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	// Property-scope enforcement (Piece 1): a staff member scoped away
	// from this unit's parent property must read it as not-found.
	t.Run("owned but outside scoped access", func(t *testing.T) {
		_, err := svc.AddDocument(context.Background(), unit.ID, ownerID, uploader, photoID, domain.UnitDocumentCategoryOther, domain.PropertyAccess{PropertyIDs: []uuid.UUID{uuid.New()}})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("AddDocument() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}

func TestUnitService_ListDocuments(t *testing.T) {
	svc, documentRepo, attachmentRepo, _, ownerID, unit := setupUnitDocumentTest(t)
	photoID := newFakeReadyAttachment(attachmentRepo, ownerID)
	doc := &domain.UnitDocument{ID: uuid.New(), UnitID: unit.ID, AttachmentID: photoID, Category: domain.UnitDocumentCategoryPhoto, UploadedBy: ownerID}
	documentRepo.docs[doc.ID] = doc

	t.Run("found", func(t *testing.T) {
		got, err := svc.ListDocuments(context.Background(), unit.ID, ownerID, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("ListDocuments() unexpected error = %v", err)
		}
		if len(got) != 1 || got[0].ID != doc.ID {
			t.Fatalf("ListDocuments() = %+v, want exactly one document %v", got, doc.ID)
		}
	})

	t.Run("belongs to a different owner", func(t *testing.T) {
		_, err := svc.ListDocuments(context.Background(), unit.ID, uuid.New(), domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("ListDocuments() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}

func TestUnitService_DeleteDocument(t *testing.T) {
	svc, documentRepo, attachmentRepo, unitRepo, ownerID, unit := setupUnitDocumentTest(t)
	photoID := newFakeReadyAttachment(attachmentRepo, ownerID)
	doc := &domain.UnitDocument{ID: uuid.New(), UnitID: unit.ID, AttachmentID: photoID, Category: domain.UnitDocumentCategoryPhoto, UploadedBy: ownerID}
	documentRepo.docs[doc.ID] = doc

	t.Run("belongs to a different owner", func(t *testing.T) {
		if err := svc.DeleteDocument(context.Background(), unit.ID, doc.ID, uuid.New(), domain.AllPropertyAccess()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("DeleteDocument() error = %v, want %v", err, domain.ErrNotFound)
		}
		if _, ok := documentRepo.docs[doc.ID]; !ok {
			t.Error("DeleteDocument() removed a document belonging to a different owner")
		}
	})

	t.Run("document belongs to a different unit", func(t *testing.T) {
		otherUnit := &domain.Unit{ID: uuid.New(), PropertyID: unit.PropertyID, UnitName: "1B", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}
		unitRepo.units[otherUnit.ID] = otherUnit
		if err := svc.DeleteDocument(context.Background(), otherUnit.ID, doc.ID, ownerID, domain.AllPropertyAccess()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("DeleteDocument() error = %v, want %v for a document/unit mismatch", err, domain.ErrNotFound)
		}
		if _, ok := documentRepo.docs[doc.ID]; !ok {
			t.Error("DeleteDocument() removed a document via the wrong unit's route")
		}
	})

	if err := svc.DeleteDocument(context.Background(), unit.ID, doc.ID, ownerID, domain.AllPropertyAccess()); err != nil {
		t.Fatalf("DeleteDocument() unexpected error = %v", err)
	}
	if _, ok := documentRepo.docs[doc.ID]; ok {
		t.Error("DeleteDocument() document still present after delete")
	}
}
