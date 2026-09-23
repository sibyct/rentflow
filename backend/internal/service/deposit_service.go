package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// DepositService tracks the security deposit behind each lease: what was
// collected and where it is held, itemized deductions with supporting
// documents, and the final refund or forfeiture. A deposit record is
// created automatically from the lease's security_deposit; this service
// then owns what happens to the money.
//
// Not modelled (yet): posting a deposit's deductions as income, or its
// refund as a payment out of the trust account, in the ledger.
type DepositService struct {
	repo domain.DepositRepository
	refs domain.LedgerRepository // ownership checks for account / attachment ids
	log  *slog.Logger
	now  clock
}

func NewDepositService(repo domain.DepositRepository, refs domain.LedgerRepository, log *slog.Logger) *DepositService {
	return &DepositService{repo: repo, refs: refs, log: log, now: systemClock}
}

var _ domain.DepositService = (*DepositService)(nil)

func (s *DepositService) EnsureForLease(ctx context.Context, ownerID uuid.UUID, lease *domain.Lease, propertyID uuid.UUID) error {
	if lease.SecurityDeposit == nil {
		return nil
	}
	cents := domain.DollarsToCents(*lease.SecurityDeposit)
	if cents <= 0 {
		return nil
	}
	now := s.now()
	d := &domain.SecurityDeposit{
		ID: uuid.New(), OwnerID: ownerID, LeaseID: lease.ID, PropertyID: propertyID, UnitID: lease.UnitID,
		AmountCents: cents, CollectedOn: dateOf(lease.StartDate), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.Ensure(ctx, d); err != nil {
		return fmt.Errorf("ensure deposit for lease %s: %w", lease.ID, err)
	}
	return nil
}

func (s *DepositService) owned(ctx context.Context, ownerID, id uuid.UUID) (*domain.DepositRow, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return d, nil
}

func (s *DepositService) Get(ctx context.Context, ownerID, id uuid.UUID) (*domain.DepositRow, error) {
	d, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("get deposit %s: %w", id, err)
	}
	return d, nil
}

func (s *DepositService) List(ctx context.Context, ownerID uuid.UUID, opts domain.DepositListOptions) ([]*domain.DepositRow, int, error) {
	// Lazy backfill: covers a lease whose create-time hook failed, or one
	// created before deposits were tracked. Idempotent and cheap.
	if _, err := s.repo.EnsureMissingForOwner(ctx, ownerID); err != nil {
		s.log.WarnContext(ctx, "deposit backfill failed", slog.String("owner_id", ownerID.String()), slog.Any("error", err))
	}
	opts.OwnerID = ownerID
	opts.Limit, opts.Offset = clampPage(opts.Limit, opts.Offset)
	if opts.Sort == "" {
		opts.Sort = domain.DepositSortCollectedOn
		opts.SortDesc = true
	}
	rows, total, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list deposits: %w", err)
	}
	return rows, total, nil
}

func (s *DepositService) SetHeldInAccount(ctx context.Context, ownerID, id uuid.UUID, accountID *uuid.UUID) (*domain.DepositRow, error) {
	d, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("set deposit %s account: %w", id, err)
	}
	if accountID != nil {
		ok, err := s.refs.BankAccountOwnedBy(ctx, *accountID, ownerID)
		if err != nil {
			return nil, fmt.Errorf("set deposit %s account: %w", id, err)
		}
		if !ok {
			return nil, fmt.Errorf("set deposit %s account: %w", id, validationError("held_in_account_id", "unknown bank account"))
		}
	}
	now := s.now()
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityDeposit, id, ownerID, "account_changed", map[string]domain.FieldChange{
		"held_in_account_id": {Old: fmt.Sprint(d.HeldInAccountID), New: fmt.Sprint(accountID)},
	}, now)
	if err := s.repo.SetHeldInAccount(ctx, id, accountID, now, audit); err != nil {
		return nil, fmt.Errorf("set deposit %s account: %w", id, err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *DepositService) AddDeduction(ctx context.Context, ownerID, id uuid.UUID, input domain.AddDeductionInput) (*domain.DepositRow, error) {
	if _, err := s.owned(ctx, ownerID, id); err != nil {
		return nil, fmt.Errorf("add deduction to deposit %s: %w", id, err)
	}
	desc := strings.TrimSpace(input.Description)
	var verrs domain.ValidationErrors
	if desc == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "description", Message: "is required"})
	}
	if len(desc) > 300 {
		verrs = append(verrs, &domain.ValidationError{Field: "description", Message: "is too long"})
	}
	if input.AmountCents <= 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "amount_cents", Message: "must be greater than zero"})
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("add deduction to deposit %s: %w", id, verrs)
	}
	if err := checkOwnedRefs(ctx, s.refs, ownerID, nil, input.AttachmentID); err != nil {
		return nil, fmt.Errorf("add deduction to deposit %s: %w", id, err)
	}

	now := s.now()
	ded := &domain.DepositDeduction{ID: uuid.New(), DepositID: id, Description: desc, AmountCents: input.AmountCents, AttachmentID: input.AttachmentID, CreatedAt: now}
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityDeposit, id, ownerID, "deduction_added", map[string]domain.FieldChange{
		"description": {New: desc}, "amount_cents": {New: input.AmountCents},
	}, now)
	if err := s.repo.AddDeduction(ctx, ded, audit); err != nil {
		return nil, fmt.Errorf("add deduction to deposit %s: %w", id, err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *DepositService) RemoveDeduction(ctx context.Context, ownerID, id, deductionID uuid.UUID) (*domain.DepositRow, error) {
	if _, err := s.owned(ctx, ownerID, id); err != nil {
		return nil, fmt.Errorf("remove deduction from deposit %s: %w", id, err)
	}
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityDeposit, id, ownerID, "deduction_removed", map[string]domain.FieldChange{
		"deduction_id": {Old: deductionID.String()},
	}, s.now())
	if err := s.repo.RemoveDeduction(ctx, id, deductionID, audit); err != nil {
		return nil, fmt.Errorf("remove deduction from deposit %s: %w", id, err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *DepositService) Settle(ctx context.Context, ownerID, id uuid.UUID, input domain.SettleDepositInput) (*domain.DepositRow, error) {
	if _, err := s.owned(ctx, ownerID, id); err != nil {
		return nil, fmt.Errorf("settle deposit %s: %w", id, err)
	}
	method := strings.TrimSpace(input.Method)
	var verrs domain.ValidationErrors
	if input.RefundCents < 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "refund_cents", Message: "cannot be negative"})
	}
	if input.RefundedOn.IsZero() {
		verrs = append(verrs, &domain.ValidationError{Field: "refunded_on", Message: "is required"})
	}
	if len(method) > 40 {
		verrs = append(verrs, &domain.ValidationError{Field: "refund_method", Message: "is too long"})
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("settle deposit %s: %w", id, verrs)
	}
	now := s.now()
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityDeposit, id, ownerID, "settled", map[string]domain.FieldChange{
		"refund_cents": {New: input.RefundCents}, "refund_method": {New: method}, "refunded_on": {New: dateOf(input.RefundedOn).Format("2006-01-02")},
	}, now)
	if err := s.repo.Settle(ctx, id, input.RefundCents, method, dateOf(input.RefundedOn), now, audit); err != nil {
		return nil, fmt.Errorf("settle deposit %s: %w", id, err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *DepositService) Forfeit(ctx context.Context, ownerID, id uuid.UUID) (*domain.DepositRow, error) {
	if _, err := s.owned(ctx, ownerID, id); err != nil {
		return nil, fmt.Errorf("forfeit deposit %s: %w", id, err)
	}
	now := s.now()
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityDeposit, id, ownerID, "forfeited", nil, now)
	if err := s.repo.Forfeit(ctx, id, now, audit); err != nil {
		return nil, fmt.Errorf("forfeit deposit %s: %w", id, err)
	}
	return s.repo.GetByID(ctx, id)
}
