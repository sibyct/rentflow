package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// PropertyOwnerService manages the real-world owners statements are
// addressed to, and which of the account's properties belong to each.
type PropertyOwnerService struct {
	repo         domain.PropertyOwnerRepository
	propertyRepo domain.PropertyRepository
	log          *slog.Logger
	now          clock
}

func NewPropertyOwnerService(repo domain.PropertyOwnerRepository, propertyRepo domain.PropertyRepository, log *slog.Logger) *PropertyOwnerService {
	return &PropertyOwnerService{repo: repo, propertyRepo: propertyRepo, log: log, now: systemClock}
}

var _ domain.PropertyOwnerService = (*PropertyOwnerService)(nil)

func (s *PropertyOwnerService) validate(ctx context.Context, ownerID uuid.UUID, in *domain.PropertyOwnerInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)
	in.Phone = strings.TrimSpace(in.Phone)

	var verrs domain.ValidationErrors
	if in.Name == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "name", Message: "is required"})
	}
	if len(in.Name) > 200 || len(in.Phone) > 50 {
		verrs = append(verrs, &domain.ValidationError{Field: "name", Message: "is too long"})
	}
	if in.Email != "" {
		if _, err := mail.ParseAddress(in.Email); err != nil {
			verrs = append(verrs, &domain.ValidationError{Field: "email", Message: "is not a valid email address"})
		}
	}
	if in.ManagementFeeBps < 0 || in.ManagementFeeBps > 10000 {
		verrs = append(verrs, &domain.ValidationError{Field: "management_fee_bps", Message: "must be between 0% and 100%"})
	}
	if in.AutoStatements && in.Email == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "email", Message: "is required to email statements automatically"})
	}
	if len(verrs) > 0 {
		return verrs
	}
	for _, id := range in.PropertyIDs {
		if _, err := ownedProperty(ctx, s.propertyRepo, id, ownerID); err != nil {
			return validationError("property_ids", "contains an unknown property")
		}
	}
	return nil
}

func (s *PropertyOwnerService) owned(ctx context.Context, ownerID, id uuid.UUID) (*domain.PropertyOwnerRow, error) {
	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return o, nil
}

func (s *PropertyOwnerService) Create(ctx context.Context, ownerID uuid.UUID, input domain.PropertyOwnerInput) (*domain.PropertyOwnerRow, error) {
	if err := s.validate(ctx, ownerID, &input); err != nil {
		return nil, fmt.Errorf("create property owner: %w", err)
	}
	now := s.now()
	o := &domain.PropertyOwner{
		ID: uuid.New(), OwnerID: ownerID, Name: input.Name, Email: input.Email, Phone: input.Phone,
		ManagementFeeBps: input.ManagementFeeBps, AutoStatements: input.AutoStatements, CreatedAt: now, UpdatedAt: now,
	}
	audit := domain.NewAuditEntry(ownerID, "property_owner", o.ID, ownerID, "created", map[string]domain.FieldChange{
		"name": {New: o.Name}, "management_fee_bps": {New: o.ManagementFeeBps},
	}, now)
	if err := s.repo.Create(ctx, o, input.PropertyIDs, audit); err != nil {
		return nil, fmt.Errorf("create property owner: %w", err)
	}
	return s.repo.GetByID(ctx, o.ID)
}

func (s *PropertyOwnerService) Get(ctx context.Context, ownerID, id uuid.UUID) (*domain.PropertyOwnerRow, error) {
	o, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("get property owner %s: %w", id, err)
	}
	return o, nil
}

func (s *PropertyOwnerService) Update(ctx context.Context, ownerID, id uuid.UUID, input domain.PropertyOwnerInput) (*domain.PropertyOwnerRow, error) {
	existing, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("update property owner %s: %w", id, err)
	}
	if err := s.validate(ctx, ownerID, &input); err != nil {
		return nil, fmt.Errorf("update property owner %s: %w", id, err)
	}
	changes := map[string]domain.FieldChange{}
	if existing.ManagementFeeBps != input.ManagementFeeBps {
		changes["management_fee_bps"] = domain.FieldChange{Old: existing.ManagementFeeBps, New: input.ManagementFeeBps}
	}
	if existing.Name != input.Name {
		changes["name"] = domain.FieldChange{Old: existing.Name, New: input.Name}
	}
	if existing.Email != input.Email {
		changes["email"] = domain.FieldChange{Old: existing.Email, New: input.Email}
	}
	now := s.now()
	o := &existing.PropertyOwner
	o.Name, o.Email, o.Phone, o.ManagementFeeBps, o.AutoStatements, o.UpdatedAt = input.Name, input.Email, input.Phone, input.ManagementFeeBps, input.AutoStatements, now
	audit := domain.NewAuditEntry(ownerID, "property_owner", id, ownerID, "updated", changes, now)
	if err := s.repo.Update(ctx, o, input.PropertyIDs, audit); err != nil {
		return nil, fmt.Errorf("update property owner %s: %w", id, err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *PropertyOwnerService) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	if _, err := s.owned(ctx, ownerID, id); err != nil {
		return fmt.Errorf("delete property owner %s: %w", id, err)
	}
	audit := domain.NewAuditEntry(ownerID, "property_owner", id, ownerID, "deleted", nil, s.now())
	if err := s.repo.Delete(ctx, id, audit); err != nil {
		return fmt.Errorf("delete property owner %s: %w", id, err)
	}
	return nil
}

func (s *PropertyOwnerService) List(ctx context.Context, ownerID uuid.UUID) ([]*domain.PropertyOwnerRow, error) {
	rows, err := s.repo.ListForOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list property owners: %w", err)
	}
	return rows, nil
}
