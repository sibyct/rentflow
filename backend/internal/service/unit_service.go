package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// UnitService resolves every operation against the unit's parent
// property before touching a unit row: units have no owner_id of their
// own (see domain.Unit), so ownership is reached by loading the
// property and comparing its OwnerID — the same IDOR-safe pattern
// PropertyService uses, just one hop further.
type UnitService struct {
	repo           domain.UnitRepository
	propertyRepo   domain.PropertyRepository
	documentRepo   domain.UnitDocumentRepository
	attachmentRepo domain.AttachmentRepository
	// leaseRepo backs DeleteUnit's active-lease guard (R13) and
	// AddDocument's related-lease validation (R15) — it is the same
	// LeaseRepository LeaseService already holds, just a second
	// consumer of the same port.
	leaseRepo domain.LeaseRepository
	log       *slog.Logger
}

func NewUnitService(repo domain.UnitRepository, propertyRepo domain.PropertyRepository, documentRepo domain.UnitDocumentRepository, attachmentRepo domain.AttachmentRepository, leaseRepo domain.LeaseRepository, log *slog.Logger) *UnitService {
	return &UnitService{repo: repo, propertyRepo: propertyRepo, documentRepo: documentRepo, attachmentRepo: attachmentRepo, leaseRepo: leaseRepo, log: log}
}

var _ domain.UnitService = (*UnitService)(nil)

func (s *UnitService) CreateUnit(ctx context.Context, ownerID uuid.UUID, input domain.CreateUnitInput, access domain.PropertyAccess) (*domain.Unit, error) {
	if _, err := s.requireOwnedProperty(ctx, input.PropertyID, ownerID, access); err != nil {
		return nil, fmt.Errorf("create unit: %w", err)
	}

	if input.Status == "" {
		input.Status = domain.UnitStatusVacant
	}
	if verrs := validateUnit(input.UnitName, input.Type, input.Furnished, input.Status, input.Bedrooms, input.Bathrooms, input.Sqft, input.RentDueDay); len(verrs) > 0 {
		return nil, fmt.Errorf("create unit: %w", verrs)
	}

	exists, err := s.repo.ExistsByPropertyUnitName(ctx, input.PropertyID, input.UnitName)
	if err != nil {
		return nil, fmt.Errorf("create unit: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("create unit: %w", domain.ValidationErrors{
			{Field: "unit_name", Message: "a unit with this name already exists at this property"},
		})
	}

	u := newUnitFromInput(input)
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("create unit: %w", err)
	}
	return u, nil
}

func (s *UnitService) CreateUnitsBulk(ctx context.Context, ownerID, propertyID uuid.UUID, inputs []domain.CreateUnitInput, access domain.PropertyAccess) ([]*domain.Unit, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("bulk create units: %w", domain.ValidationErrors{
			{Field: "units", Message: "must include at least one unit"},
		})
	}

	if _, err := s.requireOwnedProperty(ctx, propertyID, ownerID, access); err != nil {
		return nil, fmt.Errorf("bulk create units: %w", err)
	}

	existing, err := s.repo.ListByProperty(ctx, propertyID)
	if err != nil {
		return nil, fmt.Errorf("bulk create units: %w", err)
	}
	takenNames := make(map[string]bool, len(existing))
	for _, u := range existing {
		takenNames[u.UnitName] = true
	}

	var verrs domain.ValidationErrors
	units := make([]*domain.Unit, 0, len(inputs))
	for i, input := range inputs {
		input.PropertyID = propertyID
		if input.Status == "" {
			input.Status = domain.UnitStatusVacant
		}

		for _, fe := range validateUnit(input.UnitName, input.Type, input.Furnished, input.Status, input.Bedrooms, input.Bathrooms, input.Sqft, input.RentDueDay) {
			verrs = append(verrs, &domain.ValidationError{Field: fmt.Sprintf("units[%d].%s", i, fe.Field), Message: fe.Message})
		}
		if input.UnitName != "" {
			if takenNames[input.UnitName] {
				verrs = append(verrs, &domain.ValidationError{Field: fmt.Sprintf("units[%d].unit_name", i), Message: "a unit with this name already exists at this property"})
			}
			takenNames[input.UnitName] = true
		}

		units = append(units, newUnitFromInput(input))
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("bulk create units: %w", verrs)
	}

	if err := s.repo.CreateMany(ctx, units); err != nil {
		return nil, fmt.Errorf("bulk create units: %w", err)
	}
	return units, nil
}

func (s *UnitService) GetUnit(ctx context.Context, id, ownerID uuid.UUID, access domain.PropertyAccess) (*domain.Unit, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get unit %s: %w", id, err)
	}
	if _, err := s.requireOwnedProperty(ctx, u.PropertyID, ownerID, access); err != nil {
		return nil, fmt.Errorf("get unit %s: %w", id, err)
	}
	return u, nil
}

func (s *UnitService) ListUnitsByProperty(ctx context.Context, propertyID, ownerID uuid.UUID, access domain.PropertyAccess) ([]*domain.Unit, error) {
	if _, err := s.requireOwnedProperty(ctx, propertyID, ownerID, access); err != nil {
		return nil, fmt.Errorf("list units for property %s: %w", propertyID, err)
	}

	units, err := s.repo.ListByProperty(ctx, propertyID)
	if err != nil {
		return nil, fmt.Errorf("list units for property %s: %w", propertyID, err)
	}
	return units, nil
}

// ListUnitsForOwner needs no per-row ownership check: the row-security
// boundary is enforced inside UnitRepository.ListForOwner's SQL itself
// (p.owner_id = opts.OwnerID), the same pattern PropertyService.
// ListProperties relies on for PropertyRepository.List.
func (s *UnitService) ListUnitsForOwner(ctx context.Context, ownerID uuid.UUID, opts domain.UnitListOptions) ([]*domain.UnitWithProperty, int, error) {
	opts.OwnerID = ownerID
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	if opts.Sort == "" {
		opts.Sort = domain.UnitSortPropertyName
	}

	units, total, err := s.repo.ListForOwner(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list units for owner %s: %w", ownerID, err)
	}
	return units, total, nil
}

func (s *UnitService) UpdateUnit(ctx context.Context, id, ownerID uuid.UUID, input domain.UpdateUnitInput, access domain.PropertyAccess) (*domain.Unit, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update unit %s: %w", id, err)
	}
	if _, err := s.requireOwnedProperty(ctx, u.PropertyID, ownerID, access); err != nil {
		return nil, fmt.Errorf("update unit %s: %w", id, err)
	}

	var verrs domain.ValidationErrors

	if input.UnitName != nil {
		if *input.UnitName == "" {
			verrs = append(verrs, &domain.ValidationError{Field: "unit_name", Message: "cannot be empty"})
		} else if *input.UnitName != u.UnitName {
			exists, err := s.repo.ExistsByPropertyUnitName(ctx, u.PropertyID, *input.UnitName)
			if err != nil {
				return nil, fmt.Errorf("update unit %s: %w", id, err)
			}
			if exists {
				verrs = append(verrs, &domain.ValidationError{Field: "unit_name", Message: "a unit with this name already exists at this property"})
			} else {
				u.UnitName = *input.UnitName
			}
		}
	}
	if input.Floor != nil {
		u.Floor = *input.Floor
	}
	if input.Type != nil {
		if !input.Type.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "unit_type", Message: fmt.Sprintf("unknown type %q", *input.Type)})
		} else {
			u.Type = *input.Type
		}
	}
	if input.Bedrooms != nil {
		if *input.Bedrooms < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "bedrooms", Message: "cannot be negative"})
		} else {
			u.Bedrooms = input.Bedrooms
		}
	}
	if input.Bathrooms != nil {
		if *input.Bathrooms < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "bathrooms", Message: "cannot be negative"})
		} else {
			u.Bathrooms = input.Bathrooms
		}
	}
	if input.Sqft != nil {
		if *input.Sqft <= 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "sqft", Message: "must be positive"})
		} else {
			u.Sqft = input.Sqft
		}
	}
	if input.Furnished != nil {
		if !input.Furnished.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "furnished", Message: fmt.Sprintf("unknown furnished value %q", *input.Furnished)})
		} else {
			u.Furnished = input.Furnished
		}
	}
	if input.Status != nil {
		if !input.Status.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", *input.Status)})
		} else {
			applyUnitStatusTransition(u, *input.Status, time.Now().UTC())
		}
	}
	if input.MarketRent != nil {
		if *input.MarketRent < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "market_rent", Message: "cannot be negative"})
		} else {
			u.MarketRent = input.MarketRent
		}
	}
	if input.CurrentRent != nil {
		// Current Rent is system-derived (the active lease's rent, falling
		// back to this raw column only while vacant) and must never be set
		// directly through a standalone edit — see R1/R20. A brand-new
		// unit's CreateUnitInput.CurrentRent (the vacant fallback) is
		// unaffected; this only blocks UpdateUnit.
		verrs = append(verrs, &domain.ValidationError{Field: "current_rent", Message: "is system-derived and cannot be set directly; it follows the unit's active lease"})
	}
	if input.SecurityDeposit != nil {
		if *input.SecurityDeposit < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "security_deposit", Message: "cannot be negative"})
		} else {
			u.SecurityDeposit = input.SecurityDeposit
		}
	}
	if input.RentDueDay != nil {
		if *input.RentDueDay < 1 || *input.RentDueDay > 31 {
			verrs = append(verrs, &domain.ValidationError{Field: "rent_due_day", Message: "must be between 1 and 31"})
		} else {
			u.RentDueDay = input.RentDueDay
		}
	}
	if input.TenantName != nil {
		u.TenantName = *input.TenantName
	}
	if input.Notes != nil {
		u.Notes = *input.Notes
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("update unit %s: %w", id, verrs)
	}
	u.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update unit %s: %w", id, err)
	}
	return u, nil
}

func (s *UnitService) DeleteUnit(ctx context.Context, id, ownerID uuid.UUID, access domain.PropertyAccess) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete unit %s: %w", id, err)
	}
	if _, err := s.requireOwnedProperty(ctx, u.PropertyID, ownerID, access); err != nil {
		return fmt.Errorf("delete unit %s: %w", id, err)
	}

	// A unit with an active lease must have that lease terminated or
	// reassigned first (R13) — without this check, leases.unit_id's
	// ON DELETE CASCADE would silently delete the active lease along
	// with the unit.
	active, err := s.leaseRepo.HasActiveLease(ctx, id, nil)
	if err != nil {
		return fmt.Errorf("delete unit %s: %w", id, err)
	}
	if active {
		return fmt.Errorf("delete unit %s: %w", id, domain.ValidationErrors{
			{Field: "id", Message: "this unit has an active lease — terminate or reassign it before deleting the unit"},
		})
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete unit %s: %w", id, err)
	}
	return nil
}

func (s *UnitService) GetPropertyUnitStats(ctx context.Context, propertyID, ownerID uuid.UUID, access domain.PropertyAccess) (*domain.PropertyUnitStats, error) {
	if _, err := s.requireOwnedProperty(ctx, propertyID, ownerID, access); err != nil {
		return nil, fmt.Errorf("get unit stats for property %s: %w", propertyID, err)
	}

	stats, err := s.repo.GetPropertyUnitStats(ctx, propertyID)
	if err != nil {
		return nil, fmt.Errorf("get unit stats for property %s: %w", propertyID, err)
	}
	return stats, nil
}

func (s *UnitService) GetPropertyUnitStatsBulk(ctx context.Context, propertyIDs []uuid.UUID) (map[uuid.UUID]*domain.PropertyUnitStats, error) {
	stats, err := s.repo.GetPropertyUnitStatsBulk(ctx, propertyIDs)
	if err != nil {
		return nil, fmt.Errorf("bulk get unit stats for %d properties: %w", len(propertyIDs), err)
	}
	return stats, nil
}

func (s *UnitService) requireOwnedUnit(ctx context.Context, unitID, ownerID uuid.UUID, access domain.PropertyAccess) (*domain.Unit, error) {
	u, err := s.repo.GetByID(ctx, unitID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireOwnedProperty(ctx, u.PropertyID, ownerID, access); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UnitService) ListDocuments(ctx context.Context, unitID, ownerID uuid.UUID, access domain.PropertyAccess) ([]*domain.UnitDocument, error) {
	if _, err := s.requireOwnedUnit(ctx, unitID, ownerID, access); err != nil {
		return nil, fmt.Errorf("list documents for unit %s: %w", unitID, err)
	}

	docs, err := s.documentRepo.ListByUnit(ctx, unitID)
	if err != nil {
		return nil, fmt.Errorf("list documents for unit %s: %w", unitID, err)
	}
	return docs, nil
}

func (s *UnitService) AddDocument(ctx context.Context, unitID, ownerID, uploadedBy, attachmentID uuid.UUID, category domain.UnitDocumentCategory, relatedLeaseID *uuid.UUID, access domain.PropertyAccess) (*domain.UnitDocument, error) {
	if _, err := s.requireOwnedUnit(ctx, unitID, ownerID, access); err != nil {
		return nil, fmt.Errorf("add document to unit %s: %w", unitID, err)
	}
	if err := requireOwnedAttachment(ctx, s.attachmentRepo, ownerID, &attachmentID, "attachment_id"); err != nil {
		return nil, fmt.Errorf("add document to unit %s: %w", unitID, err)
	}
	if category == "" {
		category = domain.UnitDocumentCategoryOther
	}
	if !category.Valid() {
		return nil, fmt.Errorf("add document to unit %s: %w", unitID, domain.ValidationErrors{
			{Field: "category", Message: fmt.Sprintf("unknown category %q", category)},
		})
	}
	if relatedLeaseID != nil {
		// The related lease must belong to this same unit (R15) — not
		// just any lease this owner/access can see, the same
		// "must-belong-to-the-named-parent" check DeleteDocument already
		// applies below for unitID itself.
		l, err := s.leaseRepo.GetByID(ctx, *relatedLeaseID)
		if err != nil {
			return nil, fmt.Errorf("add document to unit %s: %w", unitID, err)
		}
		if l.UnitID != unitID {
			return nil, fmt.Errorf("add document to unit %s: %w", unitID, domain.ValidationErrors{
				{Field: "related_lease_id", Message: "must be a lease on this unit"},
			})
		}
	}

	d := &domain.UnitDocument{
		ID:             uuid.New(),
		UnitID:         unitID,
		AttachmentID:   attachmentID,
		Category:       category,
		UploadedBy:     uploadedBy,
		RelatedLeaseID: relatedLeaseID,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.documentRepo.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("add document to unit %s: %w", unitID, err)
	}

	// Create doesn't return the joined display fields (filename,
	// uploaded-by name, …) — re-fetch so the caller gets the same shape
	// ListDocuments would.
	saved, err := s.documentRepo.GetByID(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("add document to unit %s: %w", unitID, err)
	}
	return saved, nil
}

func (s *UnitService) DeleteDocument(ctx context.Context, unitID, documentID, ownerID uuid.UUID, access domain.PropertyAccess) error {
	if _, err := s.requireOwnedUnit(ctx, unitID, ownerID, access); err != nil {
		return fmt.Errorf("delete document %s: %w", documentID, err)
	}

	d, err := s.documentRepo.GetByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("delete document %s: %w", documentID, err)
	}
	// The document must belong to the unit named in the URL, not just
	// any unit this owner/access can see — otherwise a valid document id
	// from a different (even in-scope) unit could be used to delete it
	// via this unit's route.
	if d.UnitID != unitID {
		return fmt.Errorf("delete document %s: %w", documentID, domain.ErrNotFound)
	}

	if err := s.documentRepo.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("delete document %s: %w", documentID, err)
	}
	return nil
}

// requireOwnedProperty loads propertyID and returns domain.ErrNotFound
// (not ErrForbidden) if it belongs to someone else — an authenticated
// user should not be able to distinguish "not yours" from "doesn't
// exist" by probing IDs, matching PropertyService's own contract.
func (s *UnitService) requireOwnedProperty(ctx context.Context, propertyID, ownerID uuid.UUID, access domain.PropertyAccess) (*domain.Property, error) {
	p, err := s.propertyRepo.GetByID(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	if p.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	if !access.All && !slices.Contains(access.PropertyIDs, p.ID) {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

// applyUnitStatusTransition sets u.Status and stamps/clears VacatedAt on
// the actual transition, not every call that happens to repeat the
// current status — a no-op "still vacant" update shouldn't reset the
// days-vacant clock back to zero. Shared by UnitService.UpdateUnit and
// LeaseService's lease-lifecycle wiring (CreateLease/UpdateLease/
// GenerateRenewalLease setting a unit Occupied per R11, TerminateLease
// setting it Vacant per R10) so the stamping rule lives in exactly one
// place.
func applyUnitStatusTransition(u *domain.Unit, newStatus domain.UnitStatus, now time.Time) {
	if newStatus == domain.UnitStatusVacant && u.Status != domain.UnitStatusVacant {
		u.VacatedAt = &now
	} else if newStatus != domain.UnitStatusVacant {
		u.VacatedAt = nil
	}
	u.Status = newStatus
}

func newUnitFromInput(input domain.CreateUnitInput) *domain.Unit {
	now := time.Now().UTC()
	u := &domain.Unit{
		ID:              uuid.New(),
		PropertyID:      input.PropertyID,
		UnitName:        input.UnitName,
		Floor:           input.Floor,
		Type:            input.Type,
		Bedrooms:        input.Bedrooms,
		Bathrooms:       input.Bathrooms,
		Sqft:            input.Sqft,
		Furnished:       input.Furnished,
		Status:          input.Status,
		MarketRent:      input.MarketRent,
		CurrentRent:     input.CurrentRent,
		SecurityDeposit: input.SecurityDeposit,
		RentDueDay:      input.RentDueDay,
		TenantName:      input.TenantName,
		Notes:           input.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if u.Status == domain.UnitStatusVacant {
		u.VacatedAt = &now
	}
	return u
}

func validateUnit(
	unitName string,
	typ domain.UnitType,
	furnished *domain.UnitFurnished,
	status domain.UnitStatus,
	bedrooms *int,
	bathrooms *float64,
	sqft *int,
	rentDueDay *int,
) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if unitName == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "unit_name", Message: "is required"})
	}
	if !typ.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "unit_type", Message: fmt.Sprintf("unknown type %q", typ)})
	}
	if furnished != nil && !furnished.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "furnished", Message: fmt.Sprintf("unknown furnished value %q", *furnished)})
	}
	if !status.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", status)})
	}
	if bedrooms != nil && *bedrooms < 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "bedrooms", Message: "cannot be negative"})
	}
	if bathrooms != nil && *bathrooms < 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "bathrooms", Message: "cannot be negative"})
	}
	if sqft != nil && *sqft <= 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "sqft", Message: "must be positive"})
	}
	if rentDueDay != nil && (*rentDueDay < 1 || *rentDueDay > 31) {
		verrs = append(verrs, &domain.ValidationError{Field: "rent_due_day", Message: "must be between 1 and 31"})
	}
	return verrs
}
