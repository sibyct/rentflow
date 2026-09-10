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
	repo  domain.PropertyRepository
	cache domain.Cache // may be nil; caching is a performance optimization, not a dependency
	log   *slog.Logger
}

func NewPropertyService(repo domain.PropertyRepository, cache domain.Cache, log *slog.Logger) *PropertyService {
	return &PropertyService{repo: repo, cache: cache, log: log}
}

var _ domain.PropertyService = (*PropertyService)(nil)

func (s *PropertyService) CreateProperty(ctx context.Context, input domain.CreatePropertyInput) (*domain.Property, error) {
	if input.Status == "" {
		input.Status = domain.PropertyStatusActive
	}
	if verrs := validateProperty(input.Address, input.UnitCount, input.Status); len(verrs) > 0 {
		return nil, fmt.Errorf("create property: %w", verrs)
	}

	// Business rule, not a structural one: whether this address is a
	// duplicate depends on what's already stored for this owner, so it
	// can't be a struct validation tag — it belongs here, not in the DTO.
	exists, err := s.repo.ExistsByOwnerAddress(ctx, input.OwnerID, input.Address)
	if err != nil {
		return nil, fmt.Errorf("create property: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("create property: %w", domain.ValidationErrors{
			{Field: "address", Message: "you already have a property at this address"},
		})
	}

	now := time.Now().UTC()
	p := &domain.Property{
		ID:        uuid.New(),
		Address:   input.Address,
		UnitCount: input.UnitCount,
		Status:    input.Status,
		OwnerID:   input.OwnerID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create property: %w", err)
	}

	return p, nil
}

func (s *PropertyService) GetProperty(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	key := propertyCacheKey(id)

	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, key); err == nil && cached != "" {
			var p domain.Property
			if err := json.Unmarshal([]byte(cached), &p); err == nil {
				return &p, nil
			}
		}
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get property %s: %w", id, err)
	}

	s.cacheProperty(ctx, p)

	return p, nil
}

func (s *PropertyService) ListProperties(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Property, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	properties, total, err := s.repo.List(ctx, ownerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list properties for owner %s: %w", ownerID, err)
	}
	return properties, total, nil
}

func (s *PropertyService) UpdateProperty(ctx context.Context, id uuid.UUID, input domain.UpdatePropertyInput) (*domain.Property, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update property %s: %w", id, err)
	}

	var verrs domain.ValidationErrors
	if input.Address != nil {
		if *input.Address == "" {
			verrs = append(verrs, &domain.ValidationError{Field: "address", Message: "cannot be empty"})
		} else {
			p.Address = *input.Address
		}
	}
	if input.UnitCount != nil {
		if *input.UnitCount < 1 {
			verrs = append(verrs, &domain.ValidationError{Field: "unit_count", Message: "must be at least 1"})
		} else {
			p.UnitCount = *input.UnitCount
		}
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

func (s *PropertyService) DeleteProperty(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete property %s: %w", id, err)
	}

	s.invalidatePropertyCache(ctx, id)

	return nil
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
func validateProperty(address string, unitCount int, status domain.PropertyStatus) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if address == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "address", Message: "is required"})
	}
	if unitCount < 1 {
		verrs = append(verrs, &domain.ValidationError{Field: "unit_count", Message: "must be at least 1"})
	}
	if !status.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", status)})
	}
	return verrs
}
