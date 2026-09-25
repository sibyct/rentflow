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

// LeaseService resolves every operation against the lease's unit, then
// that unit's property, before touching a lease row: leases have no
// owner_id of their own — see UnitService's identical two-hop pattern,
// just one hop further still.
type LeaseService struct {
	repo         domain.LeaseRepository
	unitRepo     domain.UnitRepository
	propertyRepo domain.PropertyRepository
	log          *slog.Logger
	// deposits is optional (nil in tests): when set, a lease's security
	// deposit gets a tracked record. See SetDepositEnsurer.
	deposits domain.DepositEnsurer
}

func NewLeaseService(repo domain.LeaseRepository, unitRepo domain.UnitRepository, propertyRepo domain.PropertyRepository, log *slog.Logger) *LeaseService {
	return &LeaseService{repo: repo, unitRepo: unitRepo, propertyRepo: propertyRepo, log: log}
}

var _ domain.LeaseService = (*LeaseService)(nil)

// SetDepositEnsurer wires the deposit-tracking hook after construction (a
// setter, like WorkOrderService.SetExpenseSyncer, because the accounting
// services are built after and share these repositories).
func (s *LeaseService) SetDepositEnsurer(e domain.DepositEnsurer) { s.deposits = e }

// ensureDeposit is best-effort: the lease write has already succeeded, so
// a failure is logged and the deposits list's lazy backfill covers it.
func (s *LeaseService) ensureDeposit(ctx context.Context, ownerID uuid.UUID, l *domain.Lease, propertyID uuid.UUID) {
	if s.deposits == nil {
		return
	}
	if err := s.deposits.EnsureForLease(ctx, ownerID, l, propertyID); err != nil {
		s.log.WarnContext(ctx, "failed to ensure lease deposit", slog.String("lease_id", l.ID.String()), slog.Any("error", err))
	}
}

func (s *LeaseService) CreateLease(ctx context.Context, ownerID uuid.UUID, input domain.CreateLeaseInput, access domain.PropertyAccess) (*domain.Lease, error) {
	unit, err := s.requireOwnedUnit(ctx, input.UnitID, ownerID, access)
	if err != nil {
		return nil, fmt.Errorf("create lease: %w", err)
	}

	if input.Status == "" {
		input.Status = domain.LeaseStatusActive
	}
	if verrs := validateLease(input.Type, input.Status, input.StartDate, input.EndDate, input.MonthlyRent, input.RentDueDay, input.DepositStatus, input.PrimaryResidentName); len(verrs) > 0 {
		return nil, fmt.Errorf("create lease: %w", verrs)
	}

	if input.Status == domain.LeaseStatusActive {
		active, err := s.repo.HasActiveLease(ctx, input.UnitID, nil)
		if err != nil {
			return nil, fmt.Errorf("create lease: %w", err)
		}
		if active {
			return nil, fmt.Errorf("create lease: %w", domain.ValidationErrors{
				{Field: "status", Message: "this unit already has an active lease"},
			})
		}
	}

	now := time.Now().UTC()
	l := &domain.Lease{
		ID:                  uuid.New(),
		UnitID:              input.UnitID,
		Type:                input.Type,
		Status:              input.Status,
		StartDate:           input.StartDate,
		EndDate:             input.EndDate,
		MoveInDate:          input.MoveInDate,
		MoveOutDate:         input.MoveOutDate,
		MonthlyRent:         input.MonthlyRent,
		SecurityDeposit:     input.SecurityDeposit,
		DepositStatus:       input.DepositStatus,
		RentDueDay:          input.RentDueDay,
		LateFeeAmount:       input.LateFeeAmount,
		LateFeeGraceDays:    input.LateFeeGraceDays,
		PrimaryResidentName: input.PrimaryResidentName,
		CoResidents:         input.CoResidents,
		EmergencyContact:    input.EmergencyContact,
		RenewalStatus:       domain.RenewalStatusNotStarted,
		Notes:               input.Notes,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if l.CoResidents == nil {
		l.CoResidents = []string{}
	}

	if err := s.repo.Create(ctx, l); err != nil {
		return nil, fmt.Errorf("create lease: %w", err)
	}
	s.ensureDeposit(ctx, ownerID, l, unit.PropertyID)
	return l, nil
}

func (s *LeaseService) GetLease(ctx context.Context, id, ownerID uuid.UUID, access domain.PropertyAccess) (*domain.LeaseWithUnitProperty, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get lease %s: %w", id, err)
	}
	u, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access)
	if err != nil {
		return nil, fmt.Errorf("get lease %s: %w", id, err)
	}
	p, err := s.propertyRepo.GetByID(ctx, u.PropertyID)
	if err != nil {
		return nil, fmt.Errorf("get lease %s: %w", id, err)
	}
	return &domain.LeaseWithUnitProperty{Lease: *l, UnitName: u.UnitName, PropertyID: p.ID, PropertyName: p.Name}, nil
}

func (s *LeaseService) ListLeasesForOwner(ctx context.Context, ownerID uuid.UUID, opts domain.LeaseListOptions) ([]*domain.LeaseWithUnitProperty, int, error) {
	opts.OwnerID = ownerID
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	if opts.Sort == "" {
		opts.Sort = domain.LeaseSortStartDate
		opts.SortDesc = true
	}

	leases, total, err := s.repo.ListForOwner(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list leases for owner %s: %w", ownerID, err)
	}
	return leases, total, nil
}

func (s *LeaseService) UpdateLease(ctx context.Context, id, ownerID uuid.UUID, input domain.UpdateLeaseInput, access domain.PropertyAccess) (*domain.Lease, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update lease %s: %w", id, err)
	}
	unit, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access)
	if err != nil {
		return nil, fmt.Errorf("update lease %s: %w", id, err)
	}

	var verrs domain.ValidationErrors

	if input.Type != nil {
		if !input.Type.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "lease_type", Message: fmt.Sprintf("unknown type %q", *input.Type)})
		} else {
			l.Type = *input.Type
		}
	}
	if input.Status != nil {
		if !input.Status.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", *input.Status)})
		} else {
			l.Status = *input.Status
		}
	}
	if input.StartDate != nil {
		l.StartDate = *input.StartDate
	}
	if input.EndDate != nil {
		l.EndDate = input.EndDate
	}
	if l.Type == domain.LeaseTypeFixed && l.EndDate == nil {
		verrs = append(verrs, &domain.ValidationError{Field: "end_date", Message: "is required for a fixed-term lease"})
	}
	if l.EndDate != nil && l.EndDate.Before(l.StartDate) {
		verrs = append(verrs, &domain.ValidationError{Field: "end_date", Message: "cannot be before the start date"})
	}
	if input.MoveInDate != nil {
		l.MoveInDate = input.MoveInDate
	}
	if input.MoveOutDate != nil {
		l.MoveOutDate = input.MoveOutDate
	}
	if input.MonthlyRent != nil {
		if *input.MonthlyRent < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "monthly_rent", Message: "cannot be negative"})
		} else {
			l.MonthlyRent = *input.MonthlyRent
		}
	}
	if input.SecurityDeposit != nil {
		l.SecurityDeposit = input.SecurityDeposit
	}
	if input.DepositStatus != nil {
		if !input.DepositStatus.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "deposit_status", Message: fmt.Sprintf("unknown deposit status %q", *input.DepositStatus)})
		} else {
			l.DepositStatus = input.DepositStatus
		}
	}
	if input.RentDueDay != nil {
		if *input.RentDueDay < 1 || *input.RentDueDay > 31 {
			verrs = append(verrs, &domain.ValidationError{Field: "rent_due_day", Message: "must be between 1 and 31"})
		} else {
			l.RentDueDay = input.RentDueDay
		}
	}
	if input.LateFeeAmount != nil {
		l.LateFeeAmount = input.LateFeeAmount
	}
	if input.LateFeeGraceDays != nil {
		l.LateFeeGraceDays = input.LateFeeGraceDays
	}
	if input.PrimaryResidentName != nil {
		if *input.PrimaryResidentName == "" {
			verrs = append(verrs, &domain.ValidationError{Field: "primary_resident_name", Message: "cannot be empty"})
		} else {
			l.PrimaryResidentName = *input.PrimaryResidentName
		}
	}
	if input.CoResidents != nil {
		l.CoResidents = input.CoResidents
	}
	if input.EmergencyContact != nil {
		l.EmergencyContact = *input.EmergencyContact
	}
	if input.RenewalStatus != nil {
		if !input.RenewalStatus.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "renewal_status", Message: fmt.Sprintf("unknown renewal status %q", *input.RenewalStatus)})
		} else {
			l.RenewalStatus = *input.RenewalStatus
		}
	}
	if input.TerminationReason != nil {
		if !input.TerminationReason.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "termination_reason", Message: fmt.Sprintf("unknown termination reason %q", *input.TerminationReason)})
		} else {
			l.TerminationReason = input.TerminationReason
		}
	}
	if input.TerminationNoticeDate != nil {
		l.TerminationNoticeDate = input.TerminationNoticeDate
	}
	if input.Signed != nil {
		l.Signed = *input.Signed
	}
	if input.SignedDate != nil {
		l.SignedDate = input.SignedDate
	}
	if input.Notes != nil {
		l.Notes = *input.Notes
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("update lease %s: %w", id, verrs)
	}

	if l.Status == domain.LeaseStatusActive {
		active, err := s.repo.HasActiveLease(ctx, l.UnitID, &l.ID)
		if err != nil {
			return nil, fmt.Errorf("update lease %s: %w", id, err)
		}
		if active {
			return nil, fmt.Errorf("update lease %s: %w", id, domain.ValidationErrors{
				{Field: "status", Message: "this unit already has a different active lease"},
			})
		}
	}

	l.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, l); err != nil {
		return nil, fmt.Errorf("update lease %s: %w", id, err)
	}
	// A changed security_deposit amount flows to its tracked record while
	// that deposit is still open.
	s.ensureDeposit(ctx, ownerID, l, unit.PropertyID)
	return l, nil
}

func (s *LeaseService) DeleteLease(ctx context.Context, id, ownerID uuid.UUID, access domain.PropertyAccess) error {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete lease %s: %w", id, err)
	}
	if _, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access); err != nil {
		return fmt.Errorf("delete lease %s: %w", id, err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete lease %s: %w", id, err)
	}
	return nil
}

// requireOwnedUnit loads unitID and its parent property, returning
// domain.ErrNotFound (not ErrForbidden) if the property belongs to
// someone else — an authenticated user should not be able to
// distinguish "not yours" from "doesn't exist" by probing IDs.
func (s *LeaseService) requireOwnedUnit(ctx context.Context, unitID, ownerID uuid.UUID, access domain.PropertyAccess) (*domain.Unit, error) {
	u, err := s.unitRepo.GetByID(ctx, unitID)
	if err != nil {
		return nil, err
	}
	p, err := s.propertyRepo.GetByID(ctx, u.PropertyID)
	if err != nil {
		return nil, err
	}
	if p.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	if !access.All && !slices.Contains(access.PropertyIDs, p.ID) {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func validateLease(
	typ domain.LeaseType,
	status domain.LeaseStatus,
	startDate time.Time,
	endDate *time.Time,
	monthlyRent float64,
	rentDueDay *int,
	depositStatus *domain.DepositStatus,
	primaryResidentName string,
) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if !typ.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "lease_type", Message: fmt.Sprintf("unknown type %q", typ)})
	}
	if !status.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", status)})
	}
	if startDate.IsZero() {
		verrs = append(verrs, &domain.ValidationError{Field: "start_date", Message: "is required"})
	}
	if typ == domain.LeaseTypeFixed && endDate == nil {
		verrs = append(verrs, &domain.ValidationError{Field: "end_date", Message: "is required for a fixed-term lease"})
	}
	if endDate != nil && !startDate.IsZero() && endDate.Before(startDate) {
		verrs = append(verrs, &domain.ValidationError{Field: "end_date", Message: "cannot be before the start date"})
	}
	if monthlyRent < 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "monthly_rent", Message: "cannot be negative"})
	}
	if rentDueDay != nil && (*rentDueDay < 1 || *rentDueDay > 31) {
		verrs = append(verrs, &domain.ValidationError{Field: "rent_due_day", Message: "must be between 1 and 31"})
	}
	if depositStatus != nil && !depositStatus.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "deposit_status", Message: fmt.Sprintf("unknown deposit status %q", *depositStatus)})
	}
	if primaryResidentName == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "primary_resident_name", Message: "is required"})
	}
	return verrs
}
