package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// maxRecurrenceCatchUp bounds how many past occurrences one
// GenerateRecurring call will materialize per template.
const maxRecurrenceCatchUp = 36

// ExpenseService manages expenses: manual entry, edits, recurring
// templates and the expenses auto-created from completed work orders.
type ExpenseService struct {
	repo         domain.LedgerRepository
	propertyRepo domain.PropertyRepository
	unitRepo     domain.UnitRepository
	vendorRepo   domain.VendorRepository
	log          *slog.Logger
	now          clock
}

func NewExpenseService(repo domain.LedgerRepository, propertyRepo domain.PropertyRepository, unitRepo domain.UnitRepository, vendorRepo domain.VendorRepository, log *slog.Logger) *ExpenseService {
	return &ExpenseService{repo: repo, propertyRepo: propertyRepo, unitRepo: unitRepo, vendorRepo: vendorRepo, log: log, now: systemClock}
}

var _ domain.ExpenseService = (*ExpenseService)(nil)

// expenseFields is the editable subset shared by create and update.
type expenseFields struct {
	PropertyID          uuid.UUID
	UnitID              *uuid.UUID
	Category            domain.ExpenseCategory
	VendorID            *uuid.UUID
	VendorName          string
	AmountCents         int64
	IncurredOn          time.Time
	DueOn               *time.Time
	Description         string
	IsRecurring         bool
	RecurrenceFrequency *domain.RecurrenceFrequency
	AttachmentID        *uuid.UUID
}

func validateExpense(f *expenseFields) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if !f.Category.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "category", Message: "unknown category"})
	}
	if f.AmountCents <= 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "amount_cents", Message: "must be greater than zero"})
	}
	if f.IncurredOn.IsZero() {
		verrs = append(verrs, &domain.ValidationError{Field: "incurred_on", Message: "is required"})
	}
	if f.DueOn != nil && !f.IncurredOn.IsZero() && dateOf(*f.DueOn).Before(dateOf(f.IncurredOn)) {
		verrs = append(verrs, &domain.ValidationError{Field: "due_on", Message: "cannot be before the date incurred"})
	}
	switch {
	case f.IsRecurring && (f.RecurrenceFrequency == nil || !f.RecurrenceFrequency.Valid()):
		verrs = append(verrs, &domain.ValidationError{Field: "recurrence_frequency", Message: "is required for a recurring expense"})
	case !f.IsRecurring && f.RecurrenceFrequency != nil:
		verrs = append(verrs, &domain.ValidationError{Field: "recurrence_frequency", Message: "only applies to a recurring expense"})
	}
	if len(f.VendorName) > 200 {
		verrs = append(verrs, &domain.ValidationError{Field: "vendor_name", Message: "is too long"})
	}
	if len(f.Description) > 2000 {
		verrs = append(verrs, &domain.ValidationError{Field: "description", Message: "is too long"})
	}
	return verrs
}

// resolveExpenseRefs checks every id the client supplied belongs to
// ownerID and that the unit lives in the property.
func (s *ExpenseService) resolveExpenseRefs(ctx context.Context, ownerID uuid.UUID, f *expenseFields, bankAccountID *uuid.UUID) error {
	if _, err := ownedProperty(ctx, s.propertyRepo, f.PropertyID, ownerID); err != nil {
		return err
	}
	if f.UnitID != nil {
		u, err := s.unitRepo.GetByID(ctx, *f.UnitID)
		if err != nil || u.PropertyID != f.PropertyID {
			return validationError("unit_id", "unit does not belong to that property")
		}
	}
	if f.VendorID != nil {
		v, err := s.vendorRepo.GetByID(ctx, *f.VendorID)
		if err != nil || v.OwnerID != ownerID {
			return validationError("vendor_id", "unknown vendor")
		}
	}
	return checkOwnedRefs(ctx, s.repo, ownerID, bankAccountID, f.AttachmentID)
}

func (s *ExpenseService) Create(ctx context.Context, ownerID uuid.UUID, input domain.CreateExpenseInput) (*domain.TransactionRow, error) {
	f := &expenseFields{
		PropertyID: input.PropertyID, UnitID: input.UnitID, Category: input.Category,
		VendorID: input.VendorID, VendorName: strings.TrimSpace(input.VendorName),
		AmountCents: input.AmountCents, IncurredOn: input.IncurredOn, DueOn: input.DueOn,
		Description: strings.TrimSpace(input.Description), IsRecurring: input.IsRecurring,
		RecurrenceFrequency: input.RecurrenceFrequency, AttachmentID: input.AttachmentID,
	}
	verrs := validateExpense(f)
	if input.PaidOn != nil && input.PaidOn.IsZero() {
		verrs = append(verrs, &domain.ValidationError{Field: "paid_on", Message: "is not a valid date"})
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("create expense: %w", verrs)
	}
	if err := s.resolveExpenseRefs(ctx, ownerID, f, input.BankAccountID); err != nil {
		return nil, fmt.Errorf("create expense: %w", err)
	}

	now := s.now()
	category := f.Category
	incurred := dateOf(f.IncurredOn)
	t := &domain.Transaction{
		ID:                  uuid.New(),
		OwnerID:             ownerID,
		PropertyID:          f.PropertyID,
		UnitID:              f.UnitID,
		Kind:                domain.TransactionKindExpense,
		Type:                domain.TransactionTypeExpense,
		Category:            &category,
		AmountCents:         f.AmountCents,
		IncurredOn:          incurred,
		VendorID:            f.VendorID,
		VendorName:          f.VendorName,
		Description:         f.Description,
		TaxDeductible:       input.TaxDeductible,
		IsRecurring:         f.IsRecurring,
		RecurrenceFrequency: f.RecurrenceFrequency,
		AttachmentID:        f.AttachmentID,
		Source:              domain.TransactionSourceManual,
		CreatedBy:           ownerID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if f.DueOn != nil {
		d := dateOf(*f.DueOn)
		t.DueOn = &d
	}

	var payment *domain.TransactionPayment
	audits := []domain.AuditEntry{domain.NewAuditEntry(ownerID, domain.AuditEntityTransaction, t.ID, ownerID, "created", map[string]domain.FieldChange{
		"category":     {New: string(category)},
		"amount_cents": {New: t.AmountCents},
		"incurred_on":  {New: incurred.Format("2006-01-02")},
	}, now)}
	if input.PaidOn != nil {
		payment = newPayment(t.ID, ownerID, domain.RecordPaymentInput{
			AmountCents: t.AmountCents, PaidOn: dateOf(*input.PaidOn), Method: strings.TrimSpace(input.PaymentMethod), BankAccountID: input.BankAccountID,
		}, now)
		audits = append(audits, paymentAudit(ownerID, ownerID, payment, "payment_recorded", now))
	}

	if err := s.repo.CreateTransaction(ctx, t, payment, audits); err != nil {
		return nil, fmt.Errorf("create expense: %w", err)
	}
	return s.repo.GetTransactionRow(ctx, t.ID)
}

func (s *ExpenseService) getOwnedExpense(ctx context.Context, ownerID, id uuid.UUID) (*domain.TransactionRow, error) {
	row, err := s.repo.GetTransactionRow(ctx, id)
	if err != nil {
		return nil, err
	}
	if row.OwnerID != ownerID || row.Kind != domain.TransactionKindExpense {
		return nil, domain.ErrNotFound
	}
	return row, nil
}

func (s *ExpenseService) Get(ctx context.Context, ownerID, id uuid.UUID) (*domain.TransactionRow, error) {
	row, err := s.getOwnedExpense(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("get expense %s: %w", id, err)
	}
	return row, nil
}

func (s *ExpenseService) List(ctx context.Context, ownerID uuid.UUID, opts domain.ExpenseListOptions) ([]*domain.TransactionRow, int, error) {
	opts.OwnerID = ownerID
	opts.Limit, opts.Offset = clampPage(opts.Limit, opts.Offset)
	if opts.Sort == "" {
		opts.Sort = domain.ExpenseSortIncurredOn
		opts.SortDesc = true
	}
	rows, total, err := s.repo.ListExpenses(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", err)
	}
	return rows, total, nil
}

func (s *ExpenseService) Update(ctx context.Context, ownerID, id uuid.UUID, input domain.UpdateExpenseInput) (*domain.TransactionRow, error) {
	row, err := s.getOwnedExpense(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("update expense %s: %w", id, err)
	}
	if row.Voided() {
		return nil, fmt.Errorf("update expense %s: %w", id, domain.ErrNotFound)
	}
	// A work-order expense mirrors the work order's actual cost; editing
	// it here would silently diverge from the source of truth.
	if row.Source == domain.TransactionSourceWorkOrder {
		return nil, fmt.Errorf("update expense %s: %w", id, domain.ErrConflict)
	}

	f := &expenseFields{
		PropertyID: input.PropertyID, UnitID: input.UnitID, Category: input.Category,
		VendorID: input.VendorID, VendorName: strings.TrimSpace(input.VendorName),
		AmountCents: input.AmountCents, IncurredOn: input.IncurredOn, DueOn: input.DueOn,
		Description: strings.TrimSpace(input.Description), IsRecurring: input.IsRecurring,
		RecurrenceFrequency: input.RecurrenceFrequency, AttachmentID: input.AttachmentID,
	}
	if verrs := validateExpense(f); len(verrs) > 0 {
		return nil, fmt.Errorf("update expense %s: %w", id, verrs)
	}
	if input.AmountCents < row.PaidCents {
		return nil, fmt.Errorf("update expense %s: %w", id,
			validationError("amount_cents", "cannot be less than the amount already paid"))
	}
	if err := s.resolveExpenseRefs(ctx, ownerID, f, nil); err != nil {
		return nil, fmt.Errorf("update expense %s: %w", id, err)
	}

	before := row.Transaction
	after := before
	category := f.Category
	after.PropertyID = f.PropertyID
	after.UnitID = f.UnitID
	after.Category = &category
	after.VendorID = f.VendorID
	after.VendorName = f.VendorName
	after.AmountCents = f.AmountCents
	after.IncurredOn = dateOf(f.IncurredOn)
	after.DueOn = nil
	if f.DueOn != nil {
		d := dateOf(*f.DueOn)
		after.DueOn = &d
	}
	after.Description = f.Description
	after.TaxDeductible = input.TaxDeductible
	after.IsRecurring = f.IsRecurring
	after.RecurrenceFrequency = f.RecurrenceFrequency
	after.AttachmentID = f.AttachmentID
	now := s.now()
	after.UpdatedBy = &ownerID
	after.UpdatedAt = now

	changes := before.Diff(&after)
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityTransaction, id, ownerID, "updated", changes, now)
	if err := s.repo.UpdateTransaction(ctx, &after, audit); err != nil {
		return nil, fmt.Errorf("update expense %s: %w", id, err)
	}
	return s.repo.GetTransactionRow(ctx, id)
}

// SyncFromWorkOrder keeps the work order's expense in step with the work
// order: completed with a positive actual cost → an expense exists at
// that amount; anything else → it is voided. Rules that keep it safe to
// call on every write:
//   - a voided expense is a tombstone and is never recreated or revived;
//   - an expense that already has payments is never re-amounted or voided
//     (that would rewrite money that has moved) — it is logged instead.
func (s *ExpenseService) SyncFromWorkOrder(ctx context.Context, ownerID uuid.UUID, w *domain.WorkOrder) error {
	var cents int64
	if w.ActualCost != nil {
		cents = domain.DollarsToCents(*w.ActualCost)
	}
	shouldExist := w.Status == domain.WorkOrderStatusCompleted && cents > 0

	existing, err := s.repo.GetTransactionRowByWorkOrder(ctx, w.ID)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("sync work order %s expense: %w", w.ID, err)
	}
	found := err == nil
	now := s.now()

	switch {
	case !found && shouldExist:
		return s.createWorkOrderExpense(ctx, ownerID, w, cents, now)
	case !found:
		return nil
	case existing.Voided():
		return nil
	case shouldExist && existing.AmountCents != cents:
		if existing.PaidCents > 0 {
			s.log.WarnContext(ctx, "work order cost changed after its expense was paid; expense left unchanged",
				slog.String("work_order_id", w.ID.String()), slog.String("transaction_id", existing.ID.String()))
			return nil
		}
		after := existing.Transaction
		after.AmountCents = cents
		after.UpdatedBy = &ownerID
		after.UpdatedAt = now
		audit := domain.NewAuditEntry(ownerID, domain.AuditEntityTransaction, existing.ID, ownerID, "updated",
			existing.Transaction.Diff(&after), now)
		if err := s.repo.UpdateTransaction(ctx, &after, audit); err != nil {
			return fmt.Errorf("sync work order %s expense: %w", w.ID, err)
		}
	case !shouldExist:
		if existing.PaidCents > 0 {
			s.log.WarnContext(ctx, "work order reopened after its expense was paid; expense left in place",
				slog.String("work_order_id", w.ID.String()), slog.String("transaction_id", existing.ID.String()))
			return nil
		}
		audit := domain.NewAuditEntry(ownerID, domain.AuditEntityTransaction, existing.ID, ownerID, "voided", nil, now)
		if err := s.repo.VoidTransaction(ctx, existing.ID, now, audit); err != nil {
			return fmt.Errorf("sync work order %s expense: %w", w.ID, err)
		}
	}
	return nil
}

func (s *ExpenseService) createWorkOrderExpense(ctx context.Context, ownerID uuid.UUID, w *domain.WorkOrder, cents int64, now time.Time) error {
	incurred := dateOf(now)
	if w.CompletedAt != nil {
		incurred = dateOf(*w.CompletedAt)
	}
	category := domain.ExpenseCategoryRepairs
	woID := w.ID
	t := &domain.Transaction{
		ID:          uuid.New(),
		OwnerID:     ownerID,
		PropertyID:  w.PropertyID,
		UnitID:      w.UnitID,
		Kind:        domain.TransactionKindExpense,
		Type:        domain.TransactionTypeExpense,
		Category:    &category,
		AmountCents: cents,
		IncurredOn:  incurred,
		WorkOrderID: &woID,
		VendorID:    w.VendorID,
		VendorName:  w.AssignedTo, // freeform fallback when no Vendor record is linked
		Description: fmt.Sprintf("Work order: %s", w.Title),
		Source:      domain.TransactionSourceWorkOrder,
		CreatedBy:   ownerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if w.VendorID != nil {
		t.VendorName = ""
		if v, err := s.vendorRepo.GetByID(ctx, *w.VendorID); err == nil && v.PaymentTerms != nil {
			days := map[domain.VendorPaymentTerms]int{
				domain.VendorPaymentTermsNet15: 15, domain.VendorPaymentTermsNet30: 30, domain.VendorPaymentTermsNet45: 45,
			}[*v.PaymentTerms]
			due := incurred.AddDate(0, 0, days)
			t.DueOn = &due
		}
	}
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityTransaction, t.ID, ownerID, "created", map[string]domain.FieldChange{
		"source":        {New: string(t.Source)},
		"work_order_id": {New: w.ID.String()},
		"amount_cents":  {New: cents},
	}, now)
	if err := s.repo.CreateTransaction(ctx, t, nil, []domain.AuditEntry{audit}); err != nil {
		if errorsIsAlreadyExists(err) {
			return nil // raced with another sync; the unique index kept it to one
		}
		return fmt.Errorf("sync work order %s expense: %w", w.ID, err)
	}
	return nil
}

// GenerateRecurring materializes every due occurrence of each recurring
// template (monthly / quarterly / yearly) up to asOf. Occurrence k is
// computed from the template's own date (start + k periods, clamped), so
// month-end dates don't drift, and the unique (parent, date) index makes
// re-running a no-op.
func (s *ExpenseService) GenerateRecurring(ctx context.Context, ownerID uuid.UUID, asOf time.Time) (int, error) {
	templates, err := s.repo.ListRecurringTemplates(ctx, ownerID)
	if err != nil {
		return 0, fmt.Errorf("generate recurring expenses: %w", err)
	}
	now := s.now()
	asOf = dateOf(asOf)

	var rows []*domain.Transaction
	for _, tpl := range templates {
		if tpl.RecurrenceFrequency == nil {
			continue
		}
		step := map[domain.RecurrenceFrequency]int{
			domain.RecurrenceMonthly: 1, domain.RecurrenceQuarterly: 3, domain.RecurrenceYearly: 12,
		}[*tpl.RecurrenceFrequency]
		var dueOffset *int
		if tpl.DueOn != nil {
			d := int(dateOf(*tpl.DueOn).Sub(dateOf(tpl.IncurredOn)).Hours() / 24)
			dueOffset = &d
		}
		parent := tpl.ID
		for k := 1; k <= maxRecurrenceCatchUp; k++ {
			occ := domain.AddMonthsClamped(dateOf(tpl.IncurredOn), k*step)
			if occ.After(asOf) {
				break
			}
			c := *tpl
			c.ID = uuid.New()
			c.IncurredOn = occ
			c.DueOn = nil
			if dueOffset != nil {
				d := occ.AddDate(0, 0, *dueOffset)
				c.DueOn = &d
			}
			c.IsRecurring = false
			c.RecurrenceFrequency = nil
			c.RecurrenceParentID = &parent
			c.Source = domain.TransactionSourceRecurring
			c.AttachmentID = nil
			c.CreatedBy = ownerID
			c.CreatedAt = now
			c.UpdatedBy = nil
			c.UpdatedAt = now
			rows = append(rows, &c)
		}
	}
	created, err := s.repo.InsertGenerated(ctx, rows, ownerID)
	if err != nil {
		return 0, fmt.Errorf("generate recurring expenses: %w", err)
	}
	return len(created), nil
}
