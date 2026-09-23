package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// ChargeService creates one-off charges (utility rebill, damage, amenity
// fee, other). A charge is a ledger row of type "charge" attached to the
// unit's active lease, which is what makes it show up in the Rent Roll
// balance for its due month. Late fees are produced by RentRollService,
// never entered here.
type ChargeService struct {
	repo         domain.LedgerRepository
	unitRepo     domain.UnitRepository
	propertyRepo domain.PropertyRepository
	log          *slog.Logger
	now          clock
}

func NewChargeService(repo domain.LedgerRepository, unitRepo domain.UnitRepository, propertyRepo domain.PropertyRepository, log *slog.Logger) *ChargeService {
	return &ChargeService{repo: repo, unitRepo: unitRepo, propertyRepo: propertyRepo, log: log, now: systemClock}
}

var _ domain.ChargeService = (*ChargeService)(nil)

func (s *ChargeService) Create(ctx context.Context, ownerID uuid.UUID, input domain.CreateChargeInput) (*domain.TransactionRow, error) {
	input.Description = strings.TrimSpace(input.Description)

	var verrs domain.ValidationErrors
	if !input.ChargeType.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "charge_type", Message: "unknown charge type"})
	}
	if input.AmountCents <= 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "amount_cents", Message: "must be greater than zero"})
	}
	if input.DueOn.IsZero() {
		verrs = append(verrs, &domain.ValidationError{Field: "due_on", Message: "is required"})
	}
	if input.Description == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "description", Message: "is required"})
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("create charge: %w", verrs)
	}

	u, p, err := ownedUnit(ctx, s.unitRepo, s.propertyRepo, input.UnitID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("create charge: %w", err)
	}
	leaseID, err := s.repo.GetActiveLeaseIDForUnit(ctx, u.ID)
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("create charge: %w", validationError("unit_id", "this unit has no active lease to charge"))
		}
		return nil, fmt.Errorf("create charge: %w", err)
	}

	now := s.now()
	due := dateOf(input.DueOn)
	period := domain.FirstOfMonth(due)
	chargeType := input.ChargeType
	unitID := u.ID
	t := &domain.Transaction{
		ID:          uuid.New(),
		OwnerID:     ownerID,
		PropertyID:  p.ID,
		UnitID:      &unitID,
		Kind:        domain.TransactionKindIncome,
		Type:        domain.TransactionTypeCharge,
		ChargeType:  &chargeType,
		AmountCents: input.AmountCents,
		IncurredOn:  due,
		DueOn:       &due,
		Period:      &period,
		LeaseID:     &leaseID,
		Description: input.Description,
		Source:      domain.TransactionSourceManual,
		CreatedBy:   ownerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityTransaction, t.ID, ownerID, "created", map[string]domain.FieldChange{
		"type":         {New: string(t.Type)},
		"charge_type":  {New: string(chargeType)},
		"amount_cents": {New: t.AmountCents},
		"due_on":       {New: due.Format("2006-01-02")},
	}, now)
	if err := s.repo.CreateTransaction(ctx, t, nil, []domain.AuditEntry{audit}); err != nil {
		return nil, fmt.Errorf("create charge: %w", err)
	}
	return s.repo.GetTransactionRow(ctx, t.ID)
}

func (s *ChargeService) List(ctx context.Context, ownerID uuid.UUID, opts domain.ChargeListOptions) ([]*domain.TransactionRow, int, error) {
	opts.OwnerID = ownerID
	opts.Limit, opts.Offset = clampPage(opts.Limit, opts.Offset)
	rows, total, err := s.repo.ListCharges(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list charges: %w", err)
	}
	return rows, total, nil
}
