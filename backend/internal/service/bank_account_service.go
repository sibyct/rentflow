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

const maxStatementLinesPerImport = 1000

// BankAccountService manages bank accounts (manual balance) and their
// reconciliation: statement lines entered or imported by hand, matched
// one-to-one against recorded payments. Nothing here is specific to a
// manual source — a bank-feed integration would just write more
// statement lines with source "plaid".
type BankAccountService struct {
	repo         domain.BankAccountRepository
	propertyRepo domain.PropertyRepository
	log          *slog.Logger
	now          clock
}

func NewBankAccountService(repo domain.BankAccountRepository, propertyRepo domain.PropertyRepository, log *slog.Logger) *BankAccountService {
	return &BankAccountService{repo: repo, propertyRepo: propertyRepo, log: log, now: systemClock}
}

var _ domain.BankAccountService = (*BankAccountService)(nil)

func (s *BankAccountService) validate(ctx context.Context, ownerID uuid.UUID, in *domain.CreateBankAccountInput) error {
	in.Nickname = strings.TrimSpace(in.Nickname)
	in.BankName = strings.TrimSpace(in.BankName)
	in.Last4 = strings.TrimSpace(in.Last4)

	var verrs domain.ValidationErrors
	if in.Nickname == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "nickname", Message: "is required"})
	}
	if len(in.Nickname) > 100 || len(in.BankName) > 100 {
		verrs = append(verrs, &domain.ValidationError{Field: "nickname", Message: "names are limited to 100 characters"})
	}
	if !in.Type.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "account_type", Message: "must be operating or security_deposit_trust"})
	}
	if in.Last4 != "" && (len(in.Last4) != 4 || strings.Trim(in.Last4, "0123456789") != "") {
		verrs = append(verrs, &domain.ValidationError{Field: "last4", Message: "must be exactly 4 digits"})
	}
	if len(verrs) > 0 {
		return verrs
	}
	if in.PropertyID != nil {
		if _, err := ownedProperty(ctx, s.propertyRepo, *in.PropertyID, ownerID); err != nil {
			return validationError("property_id", "unknown property")
		}
	}
	return nil
}

func (s *BankAccountService) owned(ctx context.Context, ownerID, id uuid.UUID) (*domain.BankAccount, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return a, nil
}

func (s *BankAccountService) Create(ctx context.Context, ownerID uuid.UUID, input domain.CreateBankAccountInput) (*domain.BankAccount, error) {
	if err := s.validate(ctx, ownerID, &input); err != nil {
		return nil, fmt.Errorf("create bank account: %w", err)
	}
	now := s.now()
	a := &domain.BankAccount{
		ID: uuid.New(), OwnerID: ownerID, Nickname: input.Nickname, BankName: input.BankName, Type: input.Type,
		PropertyID: input.PropertyID, BalanceCents: input.BalanceCents, BalanceAsOf: input.BalanceAsOf, Last4: input.Last4,
		Provider: domain.BankAccountProviderManual, CreatedAt: now, UpdatedAt: now,
	}
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityBankAccount, a.ID, ownerID, "created", map[string]domain.FieldChange{
		"nickname": {New: a.Nickname}, "account_type": {New: string(a.Type)}, "balance_cents": {New: a.BalanceCents},
	}, now)
	if err := s.repo.Create(ctx, a, audit); err != nil {
		return nil, fmt.Errorf("create bank account: %w", err)
	}
	return a, nil
}

func (s *BankAccountService) Get(ctx context.Context, ownerID, id uuid.UUID) (*domain.BankAccount, error) {
	a, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("get bank account %s: %w", id, err)
	}
	return a, nil
}

func (s *BankAccountService) Update(ctx context.Context, ownerID, id uuid.UUID, input domain.UpdateBankAccountInput) (*domain.BankAccount, error) {
	a, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("update bank account %s: %w", id, err)
	}
	if err := s.validate(ctx, ownerID, &input); err != nil {
		return nil, fmt.Errorf("update bank account %s: %w", id, err)
	}

	changes := map[string]domain.FieldChange{}
	track := func(field string, old, new any) {
		if fmt.Sprint(old) != fmt.Sprint(new) {
			changes[field] = domain.FieldChange{Old: old, New: new}
		}
	}
	track("nickname", a.Nickname, input.Nickname)
	track("bank_name", a.BankName, input.BankName)
	track("account_type", string(a.Type), string(input.Type))
	track("balance_cents", a.BalanceCents, input.BalanceCents)
	track("last4", a.Last4, input.Last4)
	track("property_id", fmt.Sprint(a.PropertyID), fmt.Sprint(input.PropertyID))

	now := s.now()
	a.Nickname, a.BankName, a.Type, a.PropertyID = input.Nickname, input.BankName, input.Type, input.PropertyID
	a.BalanceCents, a.BalanceAsOf, a.Last4, a.UpdatedAt = input.BalanceCents, input.BalanceAsOf, input.Last4, now
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityBankAccount, id, ownerID, "updated", changes, now)
	if err := s.repo.Update(ctx, a, audit); err != nil {
		return nil, fmt.Errorf("update bank account %s: %w", id, err)
	}
	return a, nil
}

func (s *BankAccountService) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	if _, err := s.owned(ctx, ownerID, id); err != nil {
		return fmt.Errorf("delete bank account %s: %w", id, err)
	}
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityBankAccount, id, ownerID, "deleted", nil, s.now())
	if err := s.repo.Delete(ctx, id, audit); err != nil {
		return fmt.Errorf("delete bank account %s: %w", id, err)
	}
	return nil
}

func (s *BankAccountService) List(ctx context.Context, ownerID uuid.UUID) ([]*domain.BankAccountRow, error) {
	rows, err := s.repo.ListForOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list bank accounts: %w", err)
	}
	return rows, nil
}

func (s *BankAccountService) AddStatementLines(ctx context.Context, ownerID, accountID uuid.UUID, source domain.StatementLineSource, inputs []domain.StatementLineInput) (int, error) {
	if _, err := s.owned(ctx, ownerID, accountID); err != nil {
		return 0, fmt.Errorf("add statement lines to %s: %w", accountID, err)
	}
	if len(inputs) == 0 {
		return 0, fmt.Errorf("add statement lines: %w", validationError("lines", "add at least one line"))
	}
	if len(inputs) > maxStatementLinesPerImport {
		return 0, fmt.Errorf("add statement lines: %w", validationError("lines", fmt.Sprintf("at most %d lines per import", maxStatementLinesPerImport)))
	}
	if source != domain.StatementLineSourceManual && source != domain.StatementLineSourceCSV {
		return 0, fmt.Errorf("add statement lines: %w", validationError("source", "must be manual or csv"))
	}

	now := s.now()
	lines := make([]*domain.StatementLine, 0, len(inputs))
	var verrs domain.ValidationErrors
	for i, in := range inputs {
		desc := strings.TrimSpace(in.Description)
		if in.PostedOn.IsZero() {
			verrs = append(verrs, &domain.ValidationError{Field: fmt.Sprintf("lines[%d].posted_on", i), Message: "is required"})
		}
		if in.AmountCents == 0 {
			verrs = append(verrs, &domain.ValidationError{Field: fmt.Sprintf("lines[%d].amount_cents", i), Message: "cannot be zero"})
		}
		if len(desc) > 300 {
			desc = desc[:300]
		}
		lines = append(lines, &domain.StatementLine{
			ID: uuid.New(), OwnerID: ownerID, BankAccountID: accountID, PostedOn: dateOf(in.PostedOn),
			Description: desc, AmountCents: in.AmountCents, Source: source, ExternalID: strings.TrimSpace(in.ExternalID), CreatedAt: now,
		})
	}
	if len(verrs) > 0 {
		return 0, fmt.Errorf("add statement lines: %w", verrs)
	}
	added, err := s.repo.InsertLines(ctx, lines)
	if err != nil {
		return 0, fmt.Errorf("add statement lines: %w", err)
	}
	return added, nil
}

func (s *BankAccountService) ListStatementLines(ctx context.Context, ownerID uuid.UUID, opts domain.StatementLineListOptions) ([]*domain.StatementLine, int, error) {
	if _, err := s.owned(ctx, ownerID, opts.BankAccountID); err != nil {
		return nil, 0, fmt.Errorf("list statement lines: %w", err)
	}
	opts.OwnerID = ownerID
	opts.Limit, opts.Offset = clampPage(opts.Limit, opts.Offset)
	lines, total, err := s.repo.ListLines(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list statement lines: %w", err)
	}
	return lines, total, nil
}

func (s *BankAccountService) ownedLine(ctx context.Context, ownerID, accountID, lineID uuid.UUID) (*domain.StatementLine, error) {
	if _, err := s.owned(ctx, ownerID, accountID); err != nil {
		return nil, err
	}
	l, err := s.repo.GetLine(ctx, lineID)
	if err != nil {
		return nil, err
	}
	if l.OwnerID != ownerID || l.BankAccountID != accountID {
		return nil, domain.ErrNotFound
	}
	return l, nil
}

func (s *BankAccountService) DeleteStatementLine(ctx context.Context, ownerID, accountID, lineID uuid.UUID) error {
	l, err := s.ownedLine(ctx, ownerID, accountID, lineID)
	if err != nil {
		return fmt.Errorf("delete statement line %s: %w", lineID, err)
	}
	if l.MatchedPaymentID != nil {
		return fmt.Errorf("delete statement line %s: %w", lineID, validationError("line", "unmatch it before deleting"))
	}
	if err := s.repo.DeleteLine(ctx, lineID); err != nil {
		return fmt.Errorf("delete statement line %s: %w", lineID, err)
	}
	return nil
}

// Match links a statement line to a payment. v1 is exact-amount only:
// the signed amounts must be equal (a $1,500 deposit matches a $1,500
// payment received), which keeps a mis-click from silently "reconciling"
// two unrelated things.
func (s *BankAccountService) Match(ctx context.Context, ownerID, accountID, lineID, paymentID uuid.UUID) error {
	l, err := s.ownedLine(ctx, ownerID, accountID, lineID)
	if err != nil {
		return fmt.Errorf("match line %s: %w", lineID, err)
	}
	if l.MatchedPaymentID != nil {
		return fmt.Errorf("match line %s: %w", lineID, domain.ErrConflict)
	}
	candidates, err := s.repo.ListUnmatchedPayments(ctx, ownerID, accountID)
	if err != nil {
		return fmt.Errorf("match line %s: %w", lineID, err)
	}
	var payment *domain.ReconcilablePayment
	for _, p := range candidates {
		if p.PaymentID == paymentID {
			payment = p
			break
		}
	}
	if payment == nil {
		return fmt.Errorf("match line %s: %w", lineID, domain.ErrNotFound)
	}
	if payment.AmountCents != l.AmountCents {
		return fmt.Errorf("match line %s: %w", lineID, validationError("payment_id", "the amounts must match exactly"))
	}
	now := s.now()
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityBankAccount, accountID, ownerID, "reconciled", map[string]domain.FieldChange{
		"statement_line_id": {New: lineID.String()}, "payment_id": {New: paymentID.String()}, "amount_cents": {New: l.AmountCents},
	}, now)
	if err := s.repo.Match(ctx, lineID, paymentID, now, audit); err != nil {
		return fmt.Errorf("match line %s: %w", lineID, err)
	}
	return nil
}

func (s *BankAccountService) Unmatch(ctx context.Context, ownerID, accountID, lineID uuid.UUID) error {
	l, err := s.ownedLine(ctx, ownerID, accountID, lineID)
	if err != nil {
		return fmt.Errorf("unmatch line %s: %w", lineID, err)
	}
	old := ""
	if l.MatchedPaymentID != nil {
		old = l.MatchedPaymentID.String()
	}
	audit := domain.NewAuditEntry(ownerID, domain.AuditEntityBankAccount, accountID, ownerID, "unreconciled", map[string]domain.FieldChange{
		"statement_line_id": {New: lineID.String()}, "payment_id": {Old: old},
	}, time.Now().UTC())
	if err := s.repo.Unmatch(ctx, lineID, audit); err != nil {
		return fmt.Errorf("unmatch line %s: %w", lineID, err)
	}
	return nil
}

func (s *BankAccountService) Reconciliation(ctx context.Context, ownerID, accountID uuid.UUID) (*domain.Reconciliation, error) {
	a, err := s.owned(ctx, ownerID, accountID)
	if err != nil {
		return nil, fmt.Errorf("reconciliation for %s: %w", accountID, err)
	}
	matched, unmatched, unmatchedCents, err := s.repo.LineCounts(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("reconciliation for %s: %w", accountID, err)
	}
	payments, err := s.repo.ListUnmatchedPayments(ctx, ownerID, accountID)
	if err != nil {
		return nil, fmt.Errorf("reconciliation for %s: %w", accountID, err)
	}
	var paymentsCents int64
	for _, p := range payments {
		paymentsCents += p.AmountCents
	}
	return &domain.Reconciliation{
		Account: a, MatchedLineCount: matched, UnmatchedLineCount: unmatched, UnmatchedLinesCents: unmatchedCents,
		UnmatchedPayments: payments, UnmatchedPaymentsCents: paymentsCents,
	}, nil
}
