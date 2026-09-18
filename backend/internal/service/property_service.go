package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

const propertyCacheTTL = 5 * time.Minute

type PropertyService struct {
	repo     domain.PropertyRepository
	unitRepo domain.UnitRepository
	cache    domain.Cache // may be nil; caching is a performance optimization, not a dependency
	log      *slog.Logger
}

func NewPropertyService(repo domain.PropertyRepository, unitRepo domain.UnitRepository, cache domain.Cache, log *slog.Logger) *PropertyService {
	return &PropertyService{repo: repo, unitRepo: unitRepo, cache: cache, log: log}
}

var _ domain.PropertyService = (*PropertyService)(nil)

func (s *PropertyService) CreateProperty(ctx context.Context, input domain.CreatePropertyInput) (*domain.Property, error) {
	if input.Status == "" {
		input.Status = domain.PropertyStatusOnboarding
	}
	if input.Amenities == nil {
		input.Amenities = []string{}
	}
	if input.Country == "" {
		input.Country = "United States"
	}

	if verrs := validateProperty(input.Name, input.Type, input.AddressLine1, input.Units, input.Ownership, input.YearBuilt, input.Amenities, input.Status); len(verrs) > 0 {
		return nil, fmt.Errorf("create property: %w", verrs)
	}

	// Business rule, not a structural one: whether this address is a
	// duplicate depends on what's already stored for this owner, so it
	// can't be a struct validation tag — it belongs here, not in the DTO.
	exists, err := s.repo.ExistsByOwnerAddress(ctx, input.OwnerID, input.AddressLine1)
	if err != nil {
		return nil, fmt.Errorf("create property: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("create property: %w", domain.ValidationErrors{
			{Field: "address_line1", Message: "you already have a property at this address"},
		})
	}

	now := time.Now().UTC()
	p := &domain.Property{
		ID:            uuid.New(),
		Name:          input.Name,
		Type:          input.Type,
		AddressLine1:  input.AddressLine1,
		AddressLine2:  input.AddressLine2,
		City:          input.City,
		StateProvince: input.StateProvince,
		PostalCode:    input.PostalCode,
		Country:       input.Country,
		Units:         input.Units,
		Ownership:     input.Ownership,
		OwnerName:     input.OwnerName,
		YearBuilt:     input.YearBuilt,
		OnboardDate:   input.OnboardDate,
		Amenities:     input.Amenities,
		Notes:         input.Notes,
		Status:        input.Status,
		OwnerID:       input.OwnerID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create property: %w", err)
	}

	// A residential_single_unit property IS the unit it contains — there
	// is exactly one, and it's not separately managed in the UI — so an
	// implicit unit row is created here rather than requiring a second
	// "add unit" step for every single-unit property. This keeps
	// GetPropertyUnitStats uniform across property types (see
	// UnitRepository): a single-unit property's stats come from this one
	// real row instead of a special case. The property form doesn't
	// collect bedrooms/bathrooms/rent today, so there's nothing to copy
	// onto it yet — those can be filled in later via the (hidden-from-
	// list) unit record if that data entry point is added.
	//
	// This insert isn't wrapped in the same transaction as the property
	// insert above (nothing else in this codebase uses transactions
	// across repositories yet); a failure here leaves the property
	// committed without its implicit unit, which is surfaced to the
	// caller as an error rather than silently swallowed.
	if p.Type == domain.PropertyTypeResidentialSingleUnit {
		unit := &domain.Unit{
			ID:         uuid.New(),
			PropertyID: p.ID,
			UnitName:   "Unit 1",
			Type:       domain.UnitTypeOther,
			Status:     domain.UnitStatusVacant,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := s.unitRepo.Create(ctx, unit); err != nil {
			return nil, fmt.Errorf("create property: implicit unit: %w", err)
		}
	}

	return p, nil
}

func (s *PropertyService) GetProperty(ctx context.Context, id, ownerID uuid.UUID) (*domain.Property, error) {
	key := propertyCacheKey(id)

	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, key); err == nil && cached != "" {
			var p domain.Property
			if err := json.Unmarshal([]byte(cached), &p); err == nil && p.OwnerID == ownerID {
				return &p, nil
			}
		}
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get property %s: %w", id, err)
	}
	if p.OwnerID != ownerID {
		return nil, fmt.Errorf("get property %s: %w", id, domain.ErrNotFound)
	}

	s.cacheProperty(ctx, p)

	return p, nil
}

func (s *PropertyService) ListProperties(ctx context.Context, opts domain.PropertyListOptions) ([]*domain.Property, int, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	if opts.Sort == "" {
		opts.Sort = domain.PropertySortCreatedAt
		opts.SortDesc = true
	}

	properties, total, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list properties for owner %s: %w", opts.OwnerID, err)
	}
	return properties, total, nil
}

func (s *PropertyService) UpdateProperty(ctx context.Context, id, ownerID uuid.UUID, input domain.UpdatePropertyInput) (*domain.Property, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update property %s: %w", id, err)
	}
	if p.OwnerID != ownerID {
		return nil, fmt.Errorf("update property %s: %w", id, domain.ErrNotFound)
	}

	var verrs domain.ValidationErrors

	if input.Name != nil {
		if *input.Name == "" {
			verrs = append(verrs, &domain.ValidationError{Field: "name", Message: "cannot be empty"})
		} else {
			p.Name = *input.Name
		}
	}
	if input.Type != nil {
		if !input.Type.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "type", Message: fmt.Sprintf("unknown type %q", *input.Type)})
		} else {
			p.Type = *input.Type
		}
	}
	if input.AddressLine1 != nil {
		if *input.AddressLine1 == "" {
			verrs = append(verrs, &domain.ValidationError{Field: "address_line1", Message: "cannot be empty"})
		} else {
			p.AddressLine1 = *input.AddressLine1
		}
	}
	if input.AddressLine2 != nil {
		p.AddressLine2 = *input.AddressLine2
	}
	if input.City != nil {
		p.City = *input.City
	}
	if input.StateProvince != nil {
		p.StateProvince = *input.StateProvince
	}
	if input.PostalCode != nil {
		p.PostalCode = *input.PostalCode
	}
	if input.Country != nil {
		p.Country = *input.Country
	}
	if input.Units != nil {
		if *input.Units < 1 {
			verrs = append(verrs, &domain.ValidationError{Field: "units", Message: "must be at least 1"})
		} else {
			p.Units = *input.Units
		}
	}
	if input.Ownership != nil {
		if !input.Ownership.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "ownership", Message: fmt.Sprintf("unknown ownership %q", *input.Ownership)})
		} else {
			p.Ownership = input.Ownership
		}
	}
	if input.OwnerName != nil {
		p.OwnerName = *input.OwnerName
	}
	if input.YearBuilt != nil {
		if !validYearBuilt(*input.YearBuilt) {
			verrs = append(verrs, &domain.ValidationError{Field: "year_built", Message: "must be a plausible year"})
		} else {
			p.YearBuilt = input.YearBuilt
		}
	}
	if input.OnboardDate != nil {
		p.OnboardDate = input.OnboardDate
	}
	if input.Amenities != nil {
		if invalid := invalidAmenities(input.Amenities); len(invalid) > 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "amenities", Message: fmt.Sprintf("unknown amenities: %v", invalid)})
		} else {
			p.Amenities = input.Amenities
		}
	}
	if input.Notes != nil {
		p.Notes = *input.Notes
	}
	if input.Status != nil {
		if !input.Status.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", *input.Status)})
		} else {
			p.Status = *input.Status
		}
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("update property %s: %w", id, verrs)
	}
	p.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update property %s: %w", id, err)
	}

	s.invalidatePropertyCache(ctx, id)

	return p, nil
}

func (s *PropertyService) DeleteProperty(ctx context.Context, id, ownerID uuid.UUID) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete property %s: %w", id, err)
	}
	if p.OwnerID != ownerID {
		return fmt.Errorf("delete property %s: %w", id, domain.ErrNotFound)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete property %s: %w", id, err)
	}

	s.invalidatePropertyCache(ctx, id)

	return nil
}

func (s *PropertyService) BulkUpdateStatus(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, status domain.PropertyStatus) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("bulk update property status: %w", domain.ValidationErrors{
			{Field: "ids", Message: "must include at least one property id"},
		})
	}
	if !status.Valid() {
		return 0, fmt.Errorf("bulk update property status: %w", domain.ValidationErrors{
			{Field: "status", Message: fmt.Sprintf("unknown status %q", status)},
		})
	}

	n, err := s.repo.BulkUpdateStatus(ctx, ownerID, ids, status)
	if err != nil {
		return 0, fmt.Errorf("bulk update property status: %w", err)
	}

	for _, id := range ids {
		s.invalidatePropertyCache(ctx, id)
	}

	return n, nil
}

func (s *PropertyService) cacheProperty(ctx context.Context, p *domain.Property) {
	if s.cache == nil {
		return
	}
	data, err := json.Marshal(p)
	if err != nil {
		return
	}
	if err := s.cache.Set(ctx, propertyCacheKey(p.ID), string(data), propertyCacheTTL); err != nil {
		s.log.WarnContext(ctx, "failed to cache property", "error", err, "property_id", p.ID)
	}
}

func (s *PropertyService) invalidatePropertyCache(ctx context.Context, id uuid.UUID) {
	if s.cache == nil {
		return
	}
	if err := s.cache.Delete(ctx, propertyCacheKey(id)); err != nil {
		s.log.WarnContext(ctx, "failed to invalidate property cache", "error", err, "property_id", id)
	}
}

func propertyCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("property:%s", id)
}

// validateProperty collects every field-level validation failure at once
// (rather than returning on the first one) so the caller can report all
// of them in a single round trip.
func validateProperty(
	name string,
	typ domain.PropertyType,
	addressLine1 string,
	units int,
	ownership *domain.PropertyOwnership,
	yearBuilt *int,
	amenities []string,
	status domain.PropertyStatus,
) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if name == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "name", Message: "is required"})
	}
	if !typ.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "type", Message: fmt.Sprintf("unknown type %q", typ)})
	}
	if addressLine1 == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "address_line1", Message: "is required"})
	}
	if units < 1 {
		verrs = append(verrs, &domain.ValidationError{Field: "units", Message: "must be at least 1"})
	}
	if ownership != nil && !ownership.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "ownership", Message: fmt.Sprintf("unknown ownership %q", *ownership)})
	}
	if yearBuilt != nil && !validYearBuilt(*yearBuilt) {
		verrs = append(verrs, &domain.ValidationError{Field: "year_built", Message: "must be a plausible year"})
	}
	if invalid := invalidAmenities(amenities); len(invalid) > 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "amenities", Message: fmt.Sprintf("unknown amenities: %v", invalid)})
	}
	if !status.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", status)})
	}
	return verrs
}

func validYearBuilt(year int) bool {
	return year >= 1800 && year <= 2100
}

func invalidAmenities(amenities []string) []string {
	var invalid []string
	for _, a := range amenities {
		if !domain.ValidAmenities[a] {
			invalid = append(invalid, a)
		}
	}
	return invalid
}
