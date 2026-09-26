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
	leases  map[uuid.UUID]*domain.Lease
	history map[uuid.UUID][]*domain.LeaseRentHistoryEntry
	audit   map[uuid.UUID][]*domain.LeaseAuditEntry
}

func newFakeLeaseRepository() *fakeLeaseRepository {
	return &fakeLeaseRepository{
		leases:  make(map[uuid.UUID]*domain.Lease),
		history: make(map[uuid.UUID][]*domain.LeaseRentHistoryEntry),
		audit:   make(map[uuid.UUID][]*domain.LeaseAuditEntry),
	}
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

func (f *fakeLeaseRepository) UpdateMonthlyRent(_ context.Context, leaseID uuid.UUID, amount float64) error {
	l, ok := f.leases[leaseID]
	if !ok {
		return domain.ErrNotFound
	}
	l.MonthlyRent = amount
	return nil
}

func (f *fakeLeaseRepository) ListRentHistory(_ context.Context, leaseID uuid.UUID) ([]*domain.LeaseRentHistoryEntry, error) {
	return f.history[leaseID], nil
}

func (f *fakeLeaseRepository) AppendRentChange(_ context.Context, entry *domain.LeaseRentHistoryEntry, audit domain.LeaseAuditEntry) error {
	if _, ok := f.leases[entry.LeaseID]; !ok {
		return domain.ErrNotFound
	}
	f.history[entry.LeaseID] = append(f.history[entry.LeaseID], entry)
	f.audit[audit.LeaseID] = append(f.audit[audit.LeaseID], &audit)
	return nil
}

func (f *fakeLeaseRepository) CreateRenewal(_ context.Context, newLease *domain.Lease, sourceLeaseID uuid.UUID, audit domain.LeaseAuditEntry) error {
	source, ok := f.leases[sourceLeaseID]
	if !ok {
		return domain.ErrNotFound
	}
	source.Status = domain.LeaseStatusTerminated
	source.RenewedIntoLeaseID = &newLease.ID
	f.leases[newLease.ID] = newLease
	f.audit[audit.LeaseID] = append(f.audit[audit.LeaseID], &audit)
	return nil
}

func (f *fakeLeaseRepository) Terminate(_ context.Context, l *domain.Lease, _ *domain.Unit, audit domain.LeaseAuditEntry, _ *domain.UnitDocument) error {
	if _, ok := f.leases[l.ID]; !ok {
		return domain.ErrNotFound
	}
	f.leases[l.ID] = l
	f.audit[audit.LeaseID] = append(f.audit[audit.LeaseID], &audit)
	return nil
}

func (f *fakeLeaseRepository) CorrectTermination(_ context.Context, l *domain.Lease, audit domain.LeaseAuditEntry) error {
	if _, ok := f.leases[l.ID]; !ok {
		return domain.ErrNotFound
	}
	f.leases[l.ID] = l
	f.audit[audit.LeaseID] = append(f.audit[audit.LeaseID], &audit)
	return nil
}

func (f *fakeLeaseRepository) ListAudit(_ context.Context, leaseID uuid.UUID) ([]*domain.LeaseAuditEntry, error) {
	return f.audit[leaseID], nil
}

// fakeLeaseDocumentRepository is an in-memory stand-in for
// domain.LeaseDocumentRepository, mirroring fakeUnitDocumentRepository.
type fakeLeaseDocumentRepository struct {
	docs map[uuid.UUID]*domain.LeaseDocument
}

func newFakeLeaseDocumentRepository() *fakeLeaseDocumentRepository {
	return &fakeLeaseDocumentRepository{docs: make(map[uuid.UUID]*domain.LeaseDocument)}
}

func (f *fakeLeaseDocumentRepository) Create(_ context.Context, d *domain.LeaseDocument) error {
	f.docs[d.ID] = d
	return nil
}

func (f *fakeLeaseDocumentRepository) ListByLease(_ context.Context, leaseID uuid.UUID) ([]*domain.LeaseDocument, error) {
	var out []*domain.LeaseDocument
	for _, d := range f.docs {
		if d.LeaseID == leaseID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeLeaseDocumentRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.LeaseDocument, error) {
	d, ok := f.docs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return d, nil
}

func (f *fakeLeaseDocumentRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.docs[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.docs, id)
	return nil
}

func setupLeaseTest(t *testing.T) (*service.LeaseService, *fakeLeaseRepository, uuid.UUID, *domain.Unit) {
	t.Helper()
	propertyRepo := newFakePropertyRepository()
	unitRepo := newFakeUnitRepository()
	leaseRepo := newFakeLeaseRepository()
	svc := service.NewLeaseService(leaseRepo, unitRepo, propertyRepo, newFakeLeaseDocumentRepository(), newFakeAttachmentRepository(), noopLogger())

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

		got, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
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

		_, err := svc.CreateLease(context.Background(), uuid.New(), validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("CreateLease() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("fixed-term lease without an end date is rejected", func(t *testing.T) {
		svc, _, ownerID, unit := setupLeaseTest(t)

		input := validCreateLeaseInput(unit.ID)
		input.EndDate = nil
		_, err := svc.CreateLease(context.Background(), ownerID, input, domain.AllPropertyAccess())
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
		if _, err := svc.CreateLease(context.Background(), ownerID, input, domain.AllPropertyAccess()); err != nil {
			t.Fatalf("CreateLease() unexpected error = %v", err)
		}
	})

	t.Run("a second active lease on the same unit is rejected", func(t *testing.T) {
		svc, _, ownerID, unit := setupLeaseTest(t)

		if _, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess()); err != nil {
			t.Fatalf("CreateLease() first call unexpected error = %v", err)
		}

		_, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "status") {
			t.Fatalf("CreateLease() error = %v, want a ValidationErrors failure for field %q", err, "status")
		}
	})

	t.Run("a draft lease does not conflict with an active one", func(t *testing.T) {
		svc, _, ownerID, unit := setupLeaseTest(t)

		if _, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess()); err != nil {
			t.Fatalf("CreateLease() first call unexpected error = %v", err)
		}

		draft := validCreateLeaseInput(unit.ID)
		draft.Status = domain.LeaseStatusDraft
		if _, err := svc.CreateLease(context.Background(), ownerID, draft, domain.AllPropertyAccess()); err != nil {
			t.Fatalf("CreateLease() draft lease unexpected error = %v", err)
		}
	})
}

func TestLeaseService_GetLease(t *testing.T) {
	svc, _, ownerID, unit := setupLeaseTest(t)

	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}

	t.Run("found includes unit and property context", func(t *testing.T) {
		got, err := svc.GetLease(context.Background(), created.ID, ownerID, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("GetLease() unexpected error = %v", err)
		}
		if got.UnitName != unit.UnitName {
			t.Errorf("GetLease() unit_name = %q, want %q", got.UnitName, unit.UnitName)
		}
	})

	t.Run("belongs to a different owner", func(t *testing.T) {
		_, err := svc.GetLease(context.Background(), created.ID, uuid.New(), domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetLease() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	// Property-scope enforcement (Piece 1): a staff member scoped away
	// from this lease's unit's parent property must read it as not-found.
	t.Run("owned but outside scoped access", func(t *testing.T) {
		_, err := svc.GetLease(context.Background(), created.ID, ownerID, domain.PropertyAccess{PropertyIDs: []uuid.UUID{uuid.New()}})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetLease() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("owned and within scoped access", func(t *testing.T) {
		got, err := svc.GetLease(context.Background(), created.ID, ownerID, domain.PropertyAccess{PropertyIDs: []uuid.UUID{unit.PropertyID}})
		if err != nil {
			t.Fatalf("GetLease() unexpected error = %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("GetLease() id = %v, want %v", got.ID, created.ID)
		}
	})
}

func TestLeaseService_UpdateLease(t *testing.T) {
	svc, _, ownerID, unit := setupLeaseTest(t)

	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}

	t.Run("a direct status update to terminated is rejected", func(t *testing.T) {
		// Ending a lease must go through TerminateLease (R10) so the
		// unit-vacancy wiring and audit entry always happen — see
		// TestLeaseService_TerminateLease for the real path.
		terminated := domain.LeaseStatusTerminated
		_, err := svc.UpdateLease(context.Background(), created.ID, ownerID, domain.UpdateLeaseInput{Status: &terminated}, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "status") {
			t.Fatalf("UpdateLease() error = %v, want a ValidationErrors failure for field %q", err, "status")
		}
	})

	t.Run("belongs to a different owner", func(t *testing.T) {
		rent := 1600.0
		_, err := svc.UpdateLease(context.Background(), created.ID, uuid.New(), domain.UpdateLeaseInput{MonthlyRent: &rent}, domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("UpdateLease() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("empty primary resident name is rejected", func(t *testing.T) {
		empty := ""
		_, err := svc.UpdateLease(context.Background(), created.ID, ownerID, domain.UpdateLeaseInput{PrimaryResidentName: &empty}, domain.AllPropertyAccess())
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("UpdateLease() error = %v, want %v", err, domain.ErrInvalidInput)
		}
	})
}

func TestLeaseService_DeleteLease(t *testing.T) {
	svc, leaseRepo, ownerID, unit := setupLeaseTest(t)

	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}

	t.Run("belongs to a different owner", func(t *testing.T) {
		if err := svc.DeleteLease(context.Background(), created.ID, uuid.New(), domain.AllPropertyAccess()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("DeleteLease() error = %v, want %v", err, domain.ErrNotFound)
		}
		if _, ok := leaseRepo.leases[created.ID]; !ok {
			t.Error("DeleteLease() removed a lease belonging to a different owner's property")
		}
	})

	if err := svc.DeleteLease(context.Background(), created.ID, ownerID, domain.AllPropertyAccess()); err != nil {
		t.Fatalf("DeleteLease() unexpected error = %v", err)
	}
	if _, ok := leaseRepo.leases[created.ID]; ok {
		t.Error("DeleteLease() lease still present after delete")
	}
}

func TestLeaseService_ChangeRent(t *testing.T) {
	svc, leaseRepo, ownerID, unit := setupLeaseTest(t)
	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}
	actorID := uuid.New()

	t.Run("negative amount is rejected", func(t *testing.T) {
		_, err := svc.ChangeRent(context.Background(), created.ID, ownerID, actorID, domain.ChangeRentInput{Amount: -1, EffectiveDate: time.Now().Add(24 * time.Hour)}, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "amount") {
			t.Fatalf("ChangeRent() error = %v, want a ValidationErrors failure for field %q", err, "amount")
		}
	})

	t.Run("a past effective date is rejected unless flagged as a correction", func(t *testing.T) {
		past := time.Now().Add(-48 * time.Hour)
		_, err := svc.ChangeRent(context.Background(), created.ID, ownerID, actorID, domain.ChangeRentInput{Amount: 1600, EffectiveDate: past}, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "effective_date") {
			t.Fatalf("ChangeRent() error = %v, want a ValidationErrors failure for field %q", err, "effective_date")
		}

		if _, err := svc.ChangeRent(context.Background(), created.ID, ownerID, actorID, domain.ChangeRentInput{Amount: 1600, EffectiveDate: past, IsCorrection: true}, domain.AllPropertyAccess()); err != nil {
			t.Fatalf("ChangeRent() with IsCorrection unexpected error = %v", err)
		}
	})

	t.Run("appends a history row and audit entry rather than overwriting", func(t *testing.T) {
		before := len(leaseRepo.history[created.ID])
		future := time.Now().Add(48 * time.Hour)
		if _, err := svc.ChangeRent(context.Background(), created.ID, ownerID, actorID, domain.ChangeRentInput{Amount: 1750, EffectiveDate: future, Reason: "annual increase"}, domain.AllPropertyAccess()); err != nil {
			t.Fatalf("ChangeRent() unexpected error = %v", err)
		}
		if got := len(leaseRepo.history[created.ID]); got != before+1 {
			t.Errorf("ChangeRent() history rows = %d, want %d", got, before+1)
		}
		// The raw leases.monthly_rent column is never touched by
		// ChangeRent — see AppendRentChange's doc comment.
		if leaseRepo.leases[created.ID].MonthlyRent != validCreateLeaseInput(unit.ID).MonthlyRent {
			t.Errorf("ChangeRent() mutated the raw monthly_rent column, want it frozen at creation-time value")
		}
		if got := len(leaseRepo.audit[created.ID]); got == 0 {
			t.Error("ChangeRent() did not write an audit entry")
		}
	})

	t.Run("only an active lease's rent can be changed", func(t *testing.T) {
		draftInput := validCreateLeaseInput(unit.ID)
		draftInput.Status = domain.LeaseStatusDraft
		draft, err := svc.CreateLease(context.Background(), ownerID, draftInput, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("CreateLease() draft unexpected error = %v", err)
		}
		_, err = svc.ChangeRent(context.Background(), draft.ID, ownerID, actorID, domain.ChangeRentInput{Amount: 1600, EffectiveDate: time.Now()}, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "status") {
			t.Fatalf("ChangeRent() on a draft lease error = %v, want a ValidationErrors failure for field %q", err, "status")
		}
	})
}

func TestLeaseService_TerminateLease(t *testing.T) {
	svc, _, ownerID, unit := setupLeaseTest(t)
	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}
	actorID := uuid.New()

	t.Run("requires a termination reason and notice date", func(t *testing.T) {
		_, err := svc.TerminateLease(context.Background(), created.ID, ownerID, actorID, domain.TerminateLeaseInput{}, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("TerminateLease() error = %v, want a ValidationErrors failure", err)
		}
	})

	t.Run("terminates the lease and vacates the unit", func(t *testing.T) {
		got, err := svc.TerminateLease(context.Background(), created.ID, ownerID, actorID, domain.TerminateLeaseInput{
			TerminationReason:     domain.TerminationReasonNonRenewal,
			TerminationNoticeDate: time.Now(),
		}, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("TerminateLease() unexpected error = %v", err)
		}
		if got.Status != domain.LeaseStatusTerminated {
			t.Errorf("TerminateLease() status = %q, want %q", got.Status, domain.LeaseStatusTerminated)
		}
		if unit.Status != domain.UnitStatusVacant {
			t.Errorf("TerminateLease() unit status = %q, want %q", unit.Status, domain.UnitStatusVacant)
		}
		if unit.VacatedAt == nil {
			t.Error("TerminateLease() did not stamp the unit's VacatedAt")
		}
	})

	t.Run("an already-terminated lease cannot be terminated again", func(t *testing.T) {
		_, err := svc.TerminateLease(context.Background(), created.ID, ownerID, actorID, domain.TerminateLeaseInput{
			TerminationReason:     domain.TerminationReasonNonRenewal,
			TerminationNoticeDate: time.Now(),
		}, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "status") {
			t.Fatalf("TerminateLease() error = %v, want a ValidationErrors failure for field %q", err, "status")
		}
	})
}

func TestLeaseService_CorrectTermination(t *testing.T) {
	svc, leaseRepo, ownerID, unit := setupLeaseTest(t)
	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}
	actorID := uuid.New()

	t.Run("rejected on a lease that isn't terminated", func(t *testing.T) {
		_, err := svc.CorrectTermination(context.Background(), created.ID, ownerID, actorID, domain.CorrectTerminationInput{
			TerminationReason:     domain.TerminationReasonOther,
			TerminationNoticeDate: time.Now(),
			Reason:                "fixing a typo",
		}, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "status") {
			t.Fatalf("CorrectTermination() error = %v, want a ValidationErrors failure for field %q", err, "status")
		}
	})

	if _, err := svc.TerminateLease(context.Background(), created.ID, ownerID, actorID, domain.TerminateLeaseInput{
		TerminationReason:     domain.TerminationReasonNonRenewal,
		TerminationNoticeDate: time.Now(),
	}, domain.AllPropertyAccess()); err != nil {
		t.Fatalf("TerminateLease() unexpected error = %v", err)
	}

	t.Run("requires a reason explaining the correction", func(t *testing.T) {
		_, err := svc.CorrectTermination(context.Background(), created.ID, ownerID, actorID, domain.CorrectTerminationInput{
			TerminationReason:     domain.TerminationReasonEviction,
			TerminationNoticeDate: time.Now(),
		}, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "reason") {
			t.Fatalf("CorrectTermination() error = %v, want a ValidationErrors failure for field %q", err, "reason")
		}
	})

	t.Run("corrects the termination details and logs an audit entry", func(t *testing.T) {
		noticeDate := time.Now()
		got, err := svc.CorrectTermination(context.Background(), created.ID, ownerID, actorID, domain.CorrectTerminationInput{
			TerminationReason:     domain.TerminationReasonEviction,
			TerminationNoticeDate: noticeDate,
			Reason:                "original entry used the wrong reason code",
		}, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("CorrectTermination() unexpected error = %v", err)
		}
		if got.TerminationReason == nil || *got.TerminationReason != domain.TerminationReasonEviction {
			t.Errorf("CorrectTermination() termination reason = %v, want %q", got.TerminationReason, domain.TerminationReasonEviction)
		}
		if got.Status != domain.LeaseStatusTerminated {
			t.Errorf("CorrectTermination() status = %q, want it to stay %q", got.Status, domain.LeaseStatusTerminated)
		}
		entries := leaseRepo.audit[created.ID]
		last := entries[len(entries)-1]
		if last.Action != domain.LeaseAuditActionTerminationCorrected {
			t.Errorf("CorrectTermination() audit action = %q, want %q", last.Action, domain.LeaseAuditActionTerminationCorrected)
		}
		if _, ok := last.Changes["correction_reason"]; !ok {
			t.Error("CorrectTermination() audit entry missing correction_reason")
		}
	})
}

func TestLeaseService_GenerateRenewalLease(t *testing.T) {
	svc, leaseRepo, ownerID, unit := setupLeaseTest(t)
	created, err := svc.CreateLease(context.Background(), ownerID, validCreateLeaseInput(unit.ID), domain.AllPropertyAccess())
	if err != nil {
		t.Fatalf("CreateLease() unexpected error = %v", err)
	}
	actorID := uuid.New()
	renewalEnd := time.Now().Add(365 * 24 * time.Hour)
	renewalInput := domain.GenerateRenewalLeaseInput{StartDate: time.Now().Add(24 * time.Hour), EndDate: &renewalEnd}

	t.Run("rejected unless the renewal is accepted", func(t *testing.T) {
		_, err := svc.GenerateRenewalLease(context.Background(), created.ID, ownerID, actorID, renewalInput, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "renewal_status") {
			t.Fatalf("GenerateRenewalLease() error = %v, want a ValidationErrors failure for field %q", err, "renewal_status")
		}
	})

	t.Run("creates a new lease and retires the source lease", func(t *testing.T) {
		accepted := domain.RenewalStatusAccepted
		proposedRent := 1800.0
		if _, err := svc.UpdateLease(context.Background(), created.ID, ownerID, domain.UpdateLeaseInput{RenewalStatus: &accepted, ProposedRent: &proposedRent}, domain.AllPropertyAccess()); err != nil {
			t.Fatalf("UpdateLease() to accepted unexpected error = %v", err)
		}

		newLease, err := svc.GenerateRenewalLease(context.Background(), created.ID, ownerID, actorID, renewalInput, domain.AllPropertyAccess())
		if err != nil {
			t.Fatalf("GenerateRenewalLease() unexpected error = %v", err)
		}
		if newLease.Status != domain.LeaseStatusActive {
			t.Errorf("GenerateRenewalLease() new lease status = %q, want %q", newLease.Status, domain.LeaseStatusActive)
		}
		if newLease.MonthlyRent != proposedRent {
			t.Errorf("GenerateRenewalLease() new lease rent = %v, want %v (the proposed rent)", newLease.MonthlyRent, proposedRent)
		}
		if newLease.RenewedFromLeaseID == nil || *newLease.RenewedFromLeaseID != created.ID {
			t.Errorf("GenerateRenewalLease() new lease RenewedFromLeaseID = %v, want %v", newLease.RenewedFromLeaseID, created.ID)
		}

		source := leaseRepo.leases[created.ID]
		if source.Status != domain.LeaseStatusTerminated {
			t.Errorf("GenerateRenewalLease() source lease status = %q, want %q", source.Status, domain.LeaseStatusTerminated)
		}
		if source.RenewedIntoLeaseID == nil || *source.RenewedIntoLeaseID != newLease.ID {
			t.Errorf("GenerateRenewalLease() source lease RenewedIntoLeaseID = %v, want %v", source.RenewedIntoLeaseID, newLease.ID)
		}
		if unit.Status != domain.UnitStatusOccupied {
			t.Errorf("GenerateRenewalLease() unit status = %q, want %q (stays occupied)", unit.Status, domain.UnitStatusOccupied)
		}
	})

	t.Run("a lease that was already renewed cannot be renewed again", func(t *testing.T) {
		_, err := svc.GenerateRenewalLease(context.Background(), created.ID, ownerID, actorID, renewalInput, domain.AllPropertyAccess())
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) {
			t.Fatalf("GenerateRenewalLease() error = %v, want a ValidationErrors failure", err)
		}
	})
}
