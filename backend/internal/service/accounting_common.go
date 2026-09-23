package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// The accounting services all resolve ownership the same way the rest of
// the codebase does — through the property chain, returning ErrNotFound
// (never ErrForbidden) for someone else's row — so the helpers live here
// rather than being copied into each service.

// clock is injected so date-dependent behavior (late fees, statuses,
// generation) is testable; production code uses time.Now().UTC().
type clock func() time.Time

func systemClock() time.Time { return time.Now().UTC() }

// dateOf truncates t to its calendar date at UTC midnight — the form
// every DATE column scans into.
func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func ownedProperty(ctx context.Context, repo domain.PropertyRepository, propertyID, ownerID uuid.UUID) (*domain.Property, error) {
	p, err := repo.GetByID(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	if p.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func ownedUnit(ctx context.Context, unitRepo domain.UnitRepository, propertyRepo domain.PropertyRepository, unitID, ownerID uuid.UUID) (*domain.Unit, *domain.Property, error) {
	u, err := unitRepo.GetByID(ctx, unitID)
	if err != nil {
		return nil, nil, err
	}
	p, err := ownedProperty(ctx, propertyRepo, u.PropertyID, ownerID)
	if err != nil {
		return nil, nil, err
	}
	return u, p, nil
}

func ownedLease(ctx context.Context, leaseRepo domain.LeaseRepository, unitRepo domain.UnitRepository, propertyRepo domain.PropertyRepository, leaseID, ownerID uuid.UUID) (*domain.Lease, *domain.Unit, *domain.Property, error) {
	l, err := leaseRepo.GetByID(ctx, leaseID)
	if err != nil {
		return nil, nil, nil, err
	}
	u, p, err := ownedUnit(ctx, unitRepo, propertyRepo, l.UnitID, ownerID)
	if err != nil {
		return nil, nil, nil, err
	}
	return l, u, p, nil
}

// ownedTransaction loads a ledger row and hides it (ErrNotFound) unless
// it belongs to ownerID.
func ownedTransaction(ctx context.Context, repo domain.LedgerRepository, id, ownerID uuid.UUID) (*domain.Transaction, error) {
	t, err := repo.GetTransaction(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func isNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }

// requireOwnedAttachment validates that attachmentID, if set, is a
// ready attachment owned by ownerID — the same IDOR-safe check every
// other cross-reference in this codebase does, so a work order or
// vendor can never point at someone else's file, or one that never
// finished uploading.
func requireOwnedAttachment(ctx context.Context, repo domain.AttachmentRepository, ownerID uuid.UUID, attachmentID *uuid.UUID, field string) error {
	if attachmentID == nil {
		return nil
	}
	a, err := repo.GetByID(ctx, *attachmentID)
	if err != nil {
		if isNotFound(err) {
			return domain.ValidationErrors{{Field: field, Message: "unknown attachment"}}
		}
		return err
	}
	if a.OwnerID != ownerID || a.Status != domain.AttachmentStatusReady {
		return domain.ValidationErrors{{Field: field, Message: "unknown attachment"}}
	}
	return nil
}

func errorsIsAlreadyExists(err error) bool { return errors.Is(err, domain.ErrAlreadyExists) }

func validationError(field, message string) domain.ValidationErrors {
	return domain.ValidationErrors{{Field: field, Message: message}}
}

// checkOwnedRefs verifies the optional bank-account / attachment ids a
// client supplied inside a ledger write belong to ownerID, so one
// account can never link to another's records.
func checkOwnedRefs(ctx context.Context, repo domain.LedgerRepository, ownerID uuid.UUID, bankAccountID, attachmentID *uuid.UUID) error {
	if bankAccountID != nil {
		ok, err := repo.BankAccountOwnedBy(ctx, *bankAccountID, ownerID)
		if err != nil {
			return err
		}
		if !ok {
			return validationError("bank_account_id", "unknown bank account")
		}
	}
	if attachmentID != nil {
		ok, err := repo.AttachmentOwnedBy(ctx, *attachmentID, ownerID)
		if err != nil {
			return err
		}
		if !ok {
			return validationError("attachment_id", "unknown attachment")
		}
	}
	return nil
}

func clampPage(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func monthLabel(period time.Time) string {
	return fmt.Sprintf("%s %d", period.Month().String(), period.Year())
}
