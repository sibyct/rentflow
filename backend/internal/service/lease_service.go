package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// LeaseService resolves every operation against the lease's unit, then
// that unit's property, before touching a lease row: leases have no
// owner_id of their own — see UnitService's identical two-hop pattern,
// just one hop further still.
type LeaseService struct {
	repo           domain.LeaseRepository
	unitRepo       domain.UnitRepository
	propertyRepo   domain.PropertyRepository
	documentRepo   domain.LeaseDocumentRepository
	attachmentRepo domain.AttachmentRepository
	log            *slog.Logger
	// deposits is optional (nil in tests): when set, a lease's security
	// deposit gets a tracked record. See SetDepositEnsurer.
	deposits domain.DepositEnsurer
}

func NewLeaseService(repo domain.LeaseRepository, unitRepo domain.UnitRepository, propertyRepo domain.PropertyRepository, documentRepo domain.LeaseDocumentRepository, attachmentRepo domain.AttachmentRepository, log *slog.Logger) *LeaseService {
	return &LeaseService{repo: repo, unitRepo: unitRepo, propertyRepo: propertyRepo, documentRepo: documentRepo, attachmentRepo: attachmentRepo, log: log}
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
	if l.Status == domain.LeaseStatusActive {
		s.setUnitOccupied(ctx, unit)
	}
	s.ensureDeposit(ctx, ownerID, l, unit.PropertyID)
	return l, nil
}

// setUnitOccupied backs R11: creating/activating a lease on a unit
// automatically sets that unit's status to Occupied. Best-effort, like
// ensureDeposit — the lease write has already succeeded, so a failure
// here is logged rather than rolled back; it would otherwise require a
// cross-repository transaction this codebase's pgxpool.Pool-per-repo
// wiring doesn't support.
func (s *LeaseService) setUnitOccupied(ctx context.Context, unit *domain.Unit) {
	if unit.Status == domain.UnitStatusOccupied {
		return
	}
	applyUnitStatusTransition(unit, domain.UnitStatusOccupied, time.Now().UTC())
	unit.UpdatedAt = time.Now().UTC()
	if err := s.unitRepo.Update(ctx, unit); err != nil {
		s.log.WarnContext(ctx, "failed to mark unit occupied after lease activation", slog.String("unit_id", unit.ID.String()), slog.Any("error", err))
	}
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
	origStatus := l.Status
	origRenewalStatus := l.RenewalStatus

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
		} else if *input.Status == domain.LeaseStatusTerminated && origStatus != domain.LeaseStatusTerminated {
			// Ending a lease always goes through TerminateLease (R10) so
			// the unit-vacancy wiring and audit entry always happen with
			// it — never a silent Status flip through this generic PATCH.
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: "use Terminate Lease to end a lease, not a direct status update"})
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
		// Monthly Rent is only directly editable on a draft lease — once
		// it has gone active, a rent change must go through ChangeRent so
		// it's recorded as a dated, reasoned amendment rather than a
		// silent overwrite (R3/R4). A still-draft lease has no real rent
		// history yet, so direct edits stay allowed there.
		if origStatus != domain.LeaseStatusDraft {
			verrs = append(verrs, &domain.ValidationError{Field: "monthly_rent", Message: "cannot be edited directly on a lease that has gone active; use Change Rent instead"})
		} else if *input.MonthlyRent < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "monthly_rent", Message: "cannot be negative"})
		} else {
			l.MonthlyRent = *input.MonthlyRent
		}
	}
	if input.SecurityDeposit != nil {
		if *input.SecurityDeposit < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "security_deposit", Message: "cannot be negative"})
		} else {
			l.SecurityDeposit = input.SecurityDeposit
		}
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
		if *input.LateFeeAmount < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "late_fee_amount", Message: "cannot be negative"})
		} else {
			l.LateFeeAmount = input.LateFeeAmount
		}
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
	if input.ProposedRent != nil {
		if *input.ProposedRent < 0 {
			verrs = append(verrs, &domain.ValidationError{Field: "proposed_rent", Message: "cannot be negative"})
		} else {
			l.ProposedRent = input.ProposedRent
		}
	}
	if input.ProposedEndDate != nil {
		l.ProposedEndDate = input.ProposedEndDate
	}
	if input.OfferSentDate != nil {
		l.OfferSentDate = input.OfferSentDate
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
	// RenewalStatus's conditional field requirements (R8): entering
	// Offered needs a proposed rent to show; entering Declined needs a
	// termination reason/notice date, same as an actual termination.
	// Checked against the lease's final state for this call (whatever
	// combination of existing values and this request's fields land),
	// not just what this one request happened to include, since a
	// client might set RenewalStatus in one call and the proposed terms
	// in an earlier one.
	if l.RenewalStatus != origRenewalStatus {
		if l.RenewalStatus == domain.RenewalStatusOffered && l.ProposedRent == nil {
			verrs = append(verrs, &domain.ValidationError{Field: "proposed_rent", Message: "is required when renewal status is offered"})
		}
		if l.RenewalStatus == domain.RenewalStatusDeclined && (l.TerminationReason == nil || l.TerminationNoticeDate == nil) {
			verrs = append(verrs, &domain.ValidationError{Field: "termination_reason", Message: "termination reason and notice date are required when a renewal is declined"})
		}
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
	// Update never writes monthly_rent (see leaseColumnsForRead's doc
	// comment) — a draft lease's direct rent edit, allowed above, needs
	// this separate narrow write.
	if input.MonthlyRent != nil && origStatus == domain.LeaseStatusDraft {
		if err := s.repo.UpdateMonthlyRent(ctx, id, l.MonthlyRent); err != nil {
			return nil, fmt.Errorf("update lease %s: %w", id, err)
		}
	}
	if l.Status == domain.LeaseStatusActive && origStatus != domain.LeaseStatusActive {
		s.setUnitOccupied(ctx, unit)
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

// ListRentHistory returns a lease's dated rent amendments (R4), most
// recent effective date first.
func (s *LeaseService) ListRentHistory(ctx context.Context, id, ownerID uuid.UUID, access domain.PropertyAccess) ([]*domain.LeaseRentHistoryEntry, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list rent history for lease %s: %w", id, err)
	}
	if _, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access); err != nil {
		return nil, fmt.Errorf("list rent history for lease %s: %w", id, err)
	}

	history, err := s.repo.ListRentHistory(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list rent history for lease %s: %w", id, err)
	}
	return history, nil
}

// ChangeRent is the only way to alter an active lease's effective rent
// (R3) — it always appends a new lease_rent_history row, never
// overwrites leases.monthly_rent (R4). actorID is the signed-in staff
// user (claims.ActorID), recorded on both the history row and the
// audit entry (R18).
func (s *LeaseService) ChangeRent(ctx context.Context, id, ownerID, actorID uuid.UUID, input domain.ChangeRentInput, access domain.PropertyAccess) (*domain.Lease, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("change rent for lease %s: %w", id, err)
	}
	if _, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access); err != nil {
		return nil, fmt.Errorf("change rent for lease %s: %w", id, err)
	}
	if l.Status != domain.LeaseStatusActive {
		return nil, fmt.Errorf("change rent for lease %s: %w", id, domain.ValidationErrors{
			{Field: "status", Message: "rent can only be changed on an active lease"},
		})
	}
	if input.Amount < 0 {
		return nil, fmt.Errorf("change rent for lease %s: %w", id, domain.ValidationErrors{
			{Field: "amount", Message: "cannot be negative"},
		})
	}
	now := time.Now().UTC()
	// A rent change is never backdated (R6) except as an explicitly
	// flagged data-entry correction.
	if !input.IsCorrection && truncateToDate(input.EffectiveDate).Before(truncateToDate(now)) {
		return nil, fmt.Errorf("change rent for lease %s: %w", id, domain.ValidationErrors{
			{Field: "effective_date", Message: "cannot be in the past unless explicitly flagged as a correction"},
		})
	}

	entry := &domain.LeaseRentHistoryEntry{
		ID:            uuid.New(),
		LeaseID:       id,
		Amount:        input.Amount,
		EffectiveDate: input.EffectiveDate,
		Reason:        input.Reason,
		IsCorrection:  input.IsCorrection,
		CreatedBy:     actorID,
		CreatedAt:     now,
	}
	changes := map[string]domain.FieldChange{
		"monthly_rent":   {Old: l.MonthlyRent, New: input.Amount},
		"effective_date": {Old: nil, New: input.EffectiveDate.Format("2006-01-02")},
	}
	audit := domain.NewLeaseAuditEntry(ownerID, id, actorID, domain.LeaseAuditActionRentChanged, changes, now)

	if err := s.repo.AppendRentChange(ctx, entry, audit); err != nil {
		return nil, fmt.Errorf("change rent for lease %s: %w", id, err)
	}

	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("change rent for lease %s: %w", id, err)
	}
	return updated, nil
}

// GenerateRenewalLease turns an Accepted renewal into a new, real Lease
// record (R7) — the source lease's own row is never rewritten into the
// new terms; it is retired (Status -> Terminated, RenewedIntoLeaseID
// set) while a fresh Lease row carries the renewed terms forward.
func (s *LeaseService) GenerateRenewalLease(ctx context.Context, id, ownerID, actorID uuid.UUID, input domain.GenerateRenewalLeaseInput, access domain.PropertyAccess) (*domain.Lease, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("generate renewal lease for %s: %w", id, err)
	}
	unit, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access)
	if err != nil {
		return nil, fmt.Errorf("generate renewal lease for %s: %w", id, err)
	}
	if l.RenewalStatus != domain.RenewalStatusAccepted {
		return nil, fmt.Errorf("generate renewal lease for %s: %w", id, domain.ValidationErrors{
			{Field: "renewal_status", Message: "a renewal lease can only be generated once the renewal is accepted"},
		})
	}
	if l.RenewedIntoLeaseID != nil {
		return nil, fmt.Errorf("generate renewal lease for %s: %w", id, domain.ValidationErrors{
			{Field: "id", Message: "this lease has already been renewed"},
		})
	}
	if input.StartDate.IsZero() {
		return nil, fmt.Errorf("generate renewal lease for %s: %w", id, domain.ValidationErrors{
			{Field: "start_date", Message: "is required"},
		})
	}

	// Every field defaults from the source lease's stored terms —
	// ProposedRent/ProposedEndDate when the caller doesn't override them
	// (R9) — rather than requiring the caller to re-supply everything.
	monthlyRent := l.MonthlyRent
	if l.ProposedRent != nil {
		monthlyRent = *l.ProposedRent
	}
	if input.MonthlyRent != nil {
		monthlyRent = *input.MonthlyRent
	}
	if monthlyRent < 0 {
		return nil, fmt.Errorf("generate renewal lease for %s: %w", id, domain.ValidationErrors{
			{Field: "monthly_rent", Message: "cannot be negative"},
		})
	}

	endDate := l.ProposedEndDate
	if input.EndDate != nil {
		endDate = input.EndDate
	}
	if endDate != nil && endDate.Before(input.StartDate) {
		return nil, fmt.Errorf("generate renewal lease for %s: %w", id, domain.ValidationErrors{
			{Field: "end_date", Message: "cannot be before the start date"},
		})
	}
	if l.Type == domain.LeaseTypeFixed && endDate == nil {
		return nil, fmt.Errorf("generate renewal lease for %s: %w", id, domain.ValidationErrors{
			{Field: "end_date", Message: "is required for a fixed-term lease"},
		})
	}

	securityDeposit := l.SecurityDeposit
	if input.SecurityDeposit != nil {
		securityDeposit = input.SecurityDeposit
	}
	rentDueDay := l.RentDueDay
	if input.RentDueDay != nil {
		rentDueDay = input.RentDueDay
	}

	now := time.Now().UTC()
	newLease := &domain.Lease{
		ID:                   uuid.New(),
		UnitID:               l.UnitID,
		Type:                 l.Type,
		Status:               domain.LeaseStatusActive,
		StartDate:            input.StartDate,
		EndDate:              endDate,
		MonthlyRent:          monthlyRent,
		SecurityDeposit:      securityDeposit,
		RentDueDay:           rentDueDay,
		LateFeeAmount:        l.LateFeeAmount,
		LateFeeGraceDays:     l.LateFeeGraceDays,
		PrimaryResidentName:  l.PrimaryResidentName,
		PrimaryResidentPhone: l.PrimaryResidentPhone,
		PrimaryResidentEmail: l.PrimaryResidentEmail,
		CoResidents:          append([]string{}, l.CoResidents...),
		EmergencyContact:     l.EmergencyContact,
		RenewalStatus:        domain.RenewalStatusNotStarted,
		RenewedFromLeaseID:   &l.ID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	changes := map[string]domain.FieldChange{
		"renewed_into_lease_id": {Old: nil, New: newLease.ID.String()},
		"monthly_rent":          {Old: l.MonthlyRent, New: newLease.MonthlyRent},
	}
	audit := domain.NewLeaseAuditEntry(ownerID, id, actorID, domain.LeaseAuditActionRenewed, changes, now)

	if err := s.repo.CreateRenewal(ctx, newLease, l.ID, audit); err != nil {
		return nil, fmt.Errorf("generate renewal lease for %s: %w", id, err)
	}
	// The unit stays occupied throughout a renewal — this call is
	// belt-and-suspenders in case it was ever anything else (R11).
	s.setUnitOccupied(ctx, unit)
	s.ensureDeposit(ctx, ownerID, newLease, unit.PropertyID)
	return newLease, nil
}

// TerminateLease is the dedicated action for ending a lease — distinct
// from a generic UpdateLease(Status: terminated) call so the unit
// vacancy (R10) and audit-log (R18) side effects always happen
// together, and so an optional move-out inspection document gets
// auto-linked to this lease (R15).
func (s *LeaseService) TerminateLease(ctx context.Context, id, ownerID, actorID uuid.UUID, input domain.TerminateLeaseInput, access domain.PropertyAccess) (*domain.Lease, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("terminate lease %s: %w", id, err)
	}
	unit, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access)
	if err != nil {
		return nil, fmt.Errorf("terminate lease %s: %w", id, err)
	}
	if l.Status == domain.LeaseStatusTerminated {
		return nil, fmt.Errorf("terminate lease %s: %w", id, domain.ValidationErrors{
			{Field: "status", Message: "this lease is already terminated"},
		})
	}
	if !input.TerminationReason.Valid() {
		return nil, fmt.Errorf("terminate lease %s: %w", id, domain.ValidationErrors{
			{Field: "termination_reason", Message: fmt.Sprintf("unknown termination reason %q", input.TerminationReason)},
		})
	}
	if input.TerminationNoticeDate.IsZero() {
		return nil, fmt.Errorf("terminate lease %s: %w", id, domain.ValidationErrors{
			{Field: "termination_notice_date", Message: "is required"},
		})
	}
	if input.MoveOutInspectionAttachmentID != nil {
		if err := requireOwnedAttachment(ctx, s.attachmentRepo, ownerID, input.MoveOutInspectionAttachmentID, "move_out_inspection_attachment_id"); err != nil {
			return nil, fmt.Errorf("terminate lease %s: %w", id, err)
		}
	}

	oldStatus := string(l.Status)
	now := time.Now().UTC()
	l.Status = domain.LeaseStatusTerminated
	reason := input.TerminationReason
	l.TerminationReason = &reason
	noticeDate := input.TerminationNoticeDate
	l.TerminationNoticeDate = &noticeDate
	if input.MoveOutDate != nil {
		l.MoveOutDate = input.MoveOutDate
	}
	l.UpdatedAt = now

	applyUnitStatusTransition(unit, domain.UnitStatusVacant, now)
	unit.UpdatedAt = now

	changes := map[string]domain.FieldChange{
		"status": {Old: oldStatus, New: string(l.Status)},
	}
	audit := domain.NewLeaseAuditEntry(ownerID, id, actorID, domain.LeaseAuditActionTerminated, changes, now)

	var autoDoc *domain.UnitDocument
	if input.MoveOutInspectionAttachmentID != nil {
		autoDoc = &domain.UnitDocument{
			ID:             uuid.New(),
			UnitID:         unit.ID,
			AttachmentID:   *input.MoveOutInspectionAttachmentID,
			Category:       domain.UnitDocumentCategoryInspection,
			UploadedBy:     actorID,
			RelatedLeaseID: &l.ID,
			IsAutomated:    true,
			CreatedAt:      now,
		}
	}

	if err := s.repo.Terminate(ctx, l, unit, audit, autoDoc); err != nil {
		return nil, fmt.Errorf("terminate lease %s: %w", id, err)
	}
	return l, nil
}

// CorrectTermination fixes a data-entry mistake in an already-
// terminated lease's termination details. Deliberately separate from
// TerminateLease: it never touches unit status (the unit's vacancy was
// already settled by the original termination) and requires a
// mandatory Reason so every correction is self-explaining in the
// audit trail — never a silent inline edit to what the form otherwise
// renders as a lease's historical record.
func (s *LeaseService) CorrectTermination(ctx context.Context, id, ownerID, actorID uuid.UUID, input domain.CorrectTerminationInput, access domain.PropertyAccess) (*domain.Lease, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("correct termination for lease %s: %w", id, err)
	}
	if _, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access); err != nil {
		return nil, fmt.Errorf("correct termination for lease %s: %w", id, err)
	}
	if l.Status != domain.LeaseStatusTerminated {
		return nil, fmt.Errorf("correct termination for lease %s: %w", id, domain.ValidationErrors{
			{Field: "status", Message: "only an already-terminated lease's termination details can be corrected"},
		})
	}
	if !input.TerminationReason.Valid() {
		return nil, fmt.Errorf("correct termination for lease %s: %w", id, domain.ValidationErrors{
			{Field: "termination_reason", Message: fmt.Sprintf("unknown termination reason %q", input.TerminationReason)},
		})
	}
	if input.TerminationNoticeDate.IsZero() {
		return nil, fmt.Errorf("correct termination for lease %s: %w", id, domain.ValidationErrors{
			{Field: "termination_notice_date", Message: "is required"},
		})
	}
	if strings.TrimSpace(input.Reason) == "" {
		return nil, fmt.Errorf("correct termination for lease %s: %w", id, domain.ValidationErrors{
			{Field: "reason", Message: "is required — explain why these termination details are being corrected"},
		})
	}

	now := time.Now().UTC()
	changes := map[string]domain.FieldChange{}
	add := func(field, oldV, newV string) {
		if oldV != newV {
			changes[field] = domain.FieldChange{Old: oldV, New: newV}
		}
	}
	oldReason := ""
	if l.TerminationReason != nil {
		oldReason = string(*l.TerminationReason)
	}
	oldNoticeDate := ""
	if l.TerminationNoticeDate != nil {
		oldNoticeDate = l.TerminationNoticeDate.Format("2006-01-02")
	}
	oldMoveOutDate := ""
	if l.MoveOutDate != nil {
		oldMoveOutDate = l.MoveOutDate.Format("2006-01-02")
	}
	newMoveOutDate := ""
	if input.MoveOutDate != nil {
		newMoveOutDate = input.MoveOutDate.Format("2006-01-02")
	}
	add("termination_reason", oldReason, string(input.TerminationReason))
	add("termination_notice_date", oldNoticeDate, input.TerminationNoticeDate.Format("2006-01-02"))
	add("move_out_date", oldMoveOutDate, newMoveOutDate)
	changes["correction_reason"] = domain.FieldChange{Old: nil, New: input.Reason}

	reason := input.TerminationReason
	l.TerminationReason = &reason
	noticeDate := input.TerminationNoticeDate
	l.TerminationNoticeDate = &noticeDate
	l.MoveOutDate = input.MoveOutDate
	l.UpdatedAt = now

	audit := domain.NewLeaseAuditEntry(ownerID, id, actorID, domain.LeaseAuditActionTerminationCorrected, changes, now)

	if err := s.repo.CorrectTermination(ctx, l, audit); err != nil {
		return nil, fmt.Errorf("correct termination for lease %s: %w", id, err)
	}
	return l, nil
}

// ListAudit returns a lease's audit trail (R18), most recent first.
func (s *LeaseService) ListAudit(ctx context.Context, id, ownerID uuid.UUID, access domain.PropertyAccess) ([]*domain.LeaseAuditEntry, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list audit for lease %s: %w", id, err)
	}
	if _, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access); err != nil {
		return nil, fmt.Errorf("list audit for lease %s: %w", id, err)
	}

	entries, err := s.repo.ListAudit(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list audit for lease %s: %w", id, err)
	}
	return entries, nil
}

// ListDocuments, AddDocument, and DeleteDocument back a lease's own
// Documents tab (R14) — a lease-specific document lives only here,
// never duplicated onto its unit's UnitDocument list.
func (s *LeaseService) ListDocuments(ctx context.Context, leaseID, ownerID uuid.UUID, access domain.PropertyAccess) ([]*domain.LeaseDocument, error) {
	l, err := s.repo.GetByID(ctx, leaseID)
	if err != nil {
		return nil, fmt.Errorf("list documents for lease %s: %w", leaseID, err)
	}
	if _, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access); err != nil {
		return nil, fmt.Errorf("list documents for lease %s: %w", leaseID, err)
	}

	docs, err := s.documentRepo.ListByLease(ctx, leaseID)
	if err != nil {
		return nil, fmt.Errorf("list documents for lease %s: %w", leaseID, err)
	}
	return docs, nil
}

func (s *LeaseService) AddDocument(ctx context.Context, leaseID, ownerID, uploadedBy, attachmentID uuid.UUID, category domain.LeaseDocumentCategory, access domain.PropertyAccess) (*domain.LeaseDocument, error) {
	l, err := s.repo.GetByID(ctx, leaseID)
	if err != nil {
		return nil, fmt.Errorf("add document to lease %s: %w", leaseID, err)
	}
	if _, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access); err != nil {
		return nil, fmt.Errorf("add document to lease %s: %w", leaseID, err)
	}
	if err := requireOwnedAttachment(ctx, s.attachmentRepo, ownerID, &attachmentID, "attachment_id"); err != nil {
		return nil, fmt.Errorf("add document to lease %s: %w", leaseID, err)
	}
	if category == "" {
		category = domain.LeaseDocumentCategoryOther
	}
	if !category.Valid() {
		return nil, fmt.Errorf("add document to lease %s: %w", leaseID, domain.ValidationErrors{
			{Field: "category", Message: fmt.Sprintf("unknown category %q", category)},
		})
	}

	d := &domain.LeaseDocument{
		ID:           uuid.New(),
		LeaseID:      leaseID,
		AttachmentID: attachmentID,
		Category:     category,
		UploadedBy:   uploadedBy,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.documentRepo.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("add document to lease %s: %w", leaseID, err)
	}

	saved, err := s.documentRepo.GetByID(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("add document to lease %s: %w", leaseID, err)
	}
	return saved, nil
}

func (s *LeaseService) DeleteDocument(ctx context.Context, leaseID, documentID, ownerID uuid.UUID, access domain.PropertyAccess) error {
	l, err := s.repo.GetByID(ctx, leaseID)
	if err != nil {
		return fmt.Errorf("delete document %s: %w", documentID, err)
	}
	if _, err := s.requireOwnedUnit(ctx, l.UnitID, ownerID, access); err != nil {
		return fmt.Errorf("delete document %s: %w", documentID, err)
	}

	d, err := s.documentRepo.GetByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("delete document %s: %w", documentID, err)
	}
	// The document must belong to the lease named in the URL, not just
	// any lease this owner/access can see — same rationale as
	// UnitService.DeleteDocument's identical check.
	if d.LeaseID != leaseID {
		return fmt.Errorf("delete document %s: %w", documentID, domain.ErrNotFound)
	}

	if err := s.documentRepo.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("delete document %s: %w", documentID, err)
	}
	return nil
}

// truncateToDate strips the time-of-day component for date-only
// comparisons (e.g. ChangeRent's past-date check) — mirrors
// domain.Lease's unexported helper of the same name and purpose.
func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
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
