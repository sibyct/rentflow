package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// MaintenanceRuleService resolves ownership via the rule's own
// PropertyID, same one-hop pattern as WorkOrderService.
type MaintenanceRuleService struct {
	repo          domain.RecurringRuleRepository
	workOrderRepo domain.WorkOrderRepository
	unitRepo      domain.UnitRepository
	propertyRepo  domain.PropertyRepository
	log           *slog.Logger
}

func NewMaintenanceRuleService(
	repo domain.RecurringRuleRepository,
	workOrderRepo domain.WorkOrderRepository,
	unitRepo domain.UnitRepository,
	propertyRepo domain.PropertyRepository,
	log *slog.Logger,
) *MaintenanceRuleService {
	return &MaintenanceRuleService{repo: repo, workOrderRepo: workOrderRepo, unitRepo: unitRepo, propertyRepo: propertyRepo, log: log}
}

var _ domain.RecurringRuleService = (*MaintenanceRuleService)(nil)

func (s *MaintenanceRuleService) CreateRule(ctx context.Context, ownerID uuid.UUID, input domain.CreateRecurringRuleInput) (*domain.RecurringRule, error) {
	if err := s.requireOwnedProperty(ctx, input.PropertyID, ownerID); err != nil {
		return nil, fmt.Errorf("create recurring rule: %w", err)
	}
	if err := s.validateUnitBelongsToProperty(ctx, input.UnitID, input.PropertyID); err != nil {
		return nil, fmt.Errorf("create recurring rule: %w", err)
	}
	if verrs := validateRecurringRule(input.Title, input.Category, input.FrequencyInterval, input.FrequencyUnit); len(verrs) > 0 {
		return nil, fmt.Errorf("create recurring rule: %w", verrs)
	}

	now := time.Now().UTC()
	rule := &domain.RecurringRule{
		ID:                uuid.New(),
		PropertyID:        input.PropertyID,
		UnitID:            input.UnitID,
		Title:             input.Title,
		Description:       input.Description,
		Category:          input.Category,
		FrequencyInterval: input.FrequencyInterval,
		FrequencyUnit:     input.FrequencyUnit,
		NextDueDate:       input.NextDueDate,
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create recurring rule: %w", err)
	}
	return rule, nil
}

func (s *MaintenanceRuleService) ListRulesForOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.RecurringRuleWithProperty, error) {
	rules, err := s.repo.ListForOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list recurring rules for owner %s: %w", ownerID, err)
	}
	return rules, nil
}

func (s *MaintenanceRuleService) UpdateRule(ctx context.Context, id, ownerID uuid.UUID, input domain.UpdateRecurringRuleInput) (*domain.RecurringRule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update recurring rule %s: %w", id, err)
	}
	if err := s.requireOwnedProperty(ctx, rule.PropertyID, ownerID); err != nil {
		return nil, fmt.Errorf("update recurring rule %s: %w", id, err)
	}

	var verrs domain.ValidationErrors

	if input.UnitIDSet {
		if err := s.validateUnitBelongsToProperty(ctx, input.UnitID, rule.PropertyID); err != nil {
			return nil, fmt.Errorf("update recurring rule %s: %w", id, err)
		}
		rule.UnitID = input.UnitID
	}
	if input.Title != nil {
		if *input.Title == "" {
			verrs = append(verrs, &domain.ValidationError{Field: "title", Message: "cannot be empty"})
		} else {
			rule.Title = *input.Title
		}
	}
	if input.Description != nil {
		rule.Description = *input.Description
	}
	if input.Category != nil {
		if !input.Category.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "category", Message: fmt.Sprintf("unknown category %q", *input.Category)})
		} else {
			rule.Category = *input.Category
		}
	}
	if input.FrequencyInterval != nil {
		if *input.FrequencyInterval <= 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "frequency_interval", Message: "must be positive"})
		} else {
			rule.FrequencyInterval = *input.FrequencyInterval
		}
	}
	if input.FrequencyUnit != nil {
		if !input.FrequencyUnit.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "frequency_unit", Message: fmt.Sprintf("unknown frequency unit %q", *input.FrequencyUnit)})
		} else {
			rule.FrequencyUnit = *input.FrequencyUnit
		}
	}
	if input.NextDueDate != nil {
		rule.NextDueDate = *input.NextDueDate
	}
	if input.Active != nil {
		rule.Active = *input.Active
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("update recurring rule %s: %w", id, verrs)
	}

	rule.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("update recurring rule %s: %w", id, err)
	}
	return rule, nil
}

func (s *MaintenanceRuleService) DeleteRule(ctx context.Context, id, ownerID uuid.UUID) error {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete recurring rule %s: %w", id, err)
	}
	if err := s.requireOwnedProperty(ctx, rule.PropertyID, ownerID); err != nil {
		return fmt.Errorf("delete recurring rule %s: %w", id, err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete recurring rule %s: %w", id, err)
	}
	return nil
}

// GenerateNow creates a real WorkOrder from rule's template and
// advances NextDueDate by one frequency step — the explicit,
// manager-triggered stand-in for a background scheduler this codebase
// doesn't have; see RecurringRule's doc comment.
func (s *MaintenanceRuleService) GenerateNow(ctx context.Context, id, ownerID uuid.UUID) (*domain.WorkOrder, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("generate work order from rule %s: %w", id, err)
	}
	if err := s.requireOwnedProperty(ctx, rule.PropertyID, ownerID); err != nil {
		return nil, fmt.Errorf("generate work order from rule %s: %w", id, err)
	}

	now := time.Now().UTC()
	w := &domain.WorkOrder{
		ID:              uuid.New(),
		PropertyID:      rule.PropertyID,
		UnitID:          rule.UnitID,
		Title:           rule.Title,
		Description:     rule.Description,
		Category:        rule.Category,
		Priority:        domain.WorkOrderPriorityMedium,
		Status:          domain.WorkOrderStatusNew,
		DueDate:         &rule.NextDueDate,
		RecurringRuleID: &rule.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.workOrderRepo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("generate work order from rule %s: %w", id, err)
	}

	rule.NextDueDate = rule.NextOccurrence()
	rule.UpdatedAt = now
	if err := s.repo.Update(ctx, rule); err != nil {
		// The work order was already created successfully; failing to
		// advance the schedule is logged, not fatal to the request —
		// worst case the manager sees the same rule "due" again and can
		// generate it a second time, which is recoverable, unlike
		// silently losing the work order they just asked for.
		s.log.WarnContext(ctx, "failed to advance recurring rule after generating a work order", "error", err, "rule_id", id)
	}

	return w, nil
}

func (s *MaintenanceRuleService) requireOwnedProperty(ctx context.Context, propertyID, ownerID uuid.UUID) error {
	p, err := s.propertyRepo.GetByID(ctx, propertyID)
	if err != nil {
		return err
	}
	if p.OwnerID != ownerID {
		return domain.ErrNotFound
	}
	return nil
}

func (s *MaintenanceRuleService) validateUnitBelongsToProperty(ctx context.Context, unitID *uuid.UUID, propertyID uuid.UUID) error {
	if unitID == nil {
		return nil
	}
	u, err := s.unitRepo.GetByID(ctx, *unitID)
	if err != nil {
		return err
	}
	if u.PropertyID != propertyID {
		return domain.ValidationErrors{{Field: "unit_id", Message: "does not belong to this property"}}
	}
	return nil
}

func validateRecurringRule(title string, category domain.WorkOrderCategory, frequencyInterval int, frequencyUnit domain.RecurringRuleFrequencyUnit) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if title == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "title", Message: "is required"})
	}
	if !category.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "category", Message: fmt.Sprintf("unknown category %q", category)})
	}
	if frequencyInterval <= 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "frequency_interval", Message: "must be positive"})
	}
	if !frequencyUnit.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "frequency_unit", Message: fmt.Sprintf("unknown frequency unit %q", frequencyUnit)})
	}
	return verrs
}
