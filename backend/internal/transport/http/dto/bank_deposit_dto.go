package dto

import (
	"fmt"

	"propertymanagement/internal/domain"
)

// ---- Bank accounts ----

type BankAccountRequest struct {
	Nickname     string `json:"nickname" validate:"required,min=1,max=100,noctrl"`
	BankName     string `json:"bank_name" validate:"omitempty,max=100,noctrl"`
	AccountType  string `json:"account_type" validate:"required,oneof=operating security_deposit_trust"`
	PropertyID   string `json:"property_id" validate:"omitempty,uuid4"`
	BalanceCents int64  `json:"balance_cents" validate:"gte=-100000000000,lte=100000000000"`
	BalanceAsOf  string `json:"balance_as_of" validate:"omitempty,datetime=2006-01-02"`
	Last4        string `json:"last4" validate:"omitempty,len=4,numeric"`
}

func (r *BankAccountRequest) Sanitize() {
	r.Nickname = sanitizeString(r.Nickname)
	r.BankName = sanitizeString(r.BankName)
}

func (r *BankAccountRequest) ToDomain() (domain.CreateBankAccountInput, error) {
	property, err := parseOptionalUUID("property_id", r.PropertyID)
	if err != nil {
		return domain.CreateBankAccountInput{}, err
	}
	asOf, err := parseOptionalDate(r.BalanceAsOf, accountingDateLayout)
	if err != nil {
		return domain.CreateBankAccountInput{}, fmt.Errorf("balance_as_of: %w", err)
	}
	return domain.CreateBankAccountInput{
		Nickname: r.Nickname, BankName: r.BankName, Type: domain.BankAccountType(r.AccountType),
		PropertyID: property, BalanceCents: r.BalanceCents, BalanceAsOf: asOf, Last4: r.Last4,
	}, nil
}

type BankAccountResponse struct {
	ID                    string `json:"id"`
	Nickname              string `json:"nickname"`
	BankName              string `json:"bank_name,omitempty"`
	AccountType           string `json:"account_type"`
	PropertyID            string `json:"property_id,omitempty"`
	PropertyName          string `json:"property_name,omitempty"`
	BalanceCents          int64  `json:"balance_cents"`
	BalanceAsOf           string `json:"balance_as_of,omitempty"`
	Last4                 string `json:"last4,omitempty"`
	Provider              string `json:"provider"`
	UnmatchedLineCount    int    `json:"unmatched_line_count"`
	UnmatchedPaymentCount int    `json:"unmatched_payment_count"`
}

func newBankAccountResponse(a *domain.BankAccount) BankAccountResponse {
	return BankAccountResponse{
		ID: a.ID.String(), Nickname: a.Nickname, BankName: a.BankName, AccountType: string(a.Type),
		PropertyID: optionalUUIDString(a.PropertyID), BalanceCents: a.BalanceCents, BalanceAsOf: formatOptionalDate(a.BalanceAsOf),
		Last4: a.Last4, Provider: a.Provider,
	}
}

func NewBankAccountResponse(a *domain.BankAccount) BankAccountResponse {
	return newBankAccountResponse(a)
}

func NewBankAccountRowListResponse(rows []*domain.BankAccountRow) []BankAccountResponse {
	out := make([]BankAccountResponse, len(rows))
	for i, r := range rows {
		resp := newBankAccountResponse(&r.BankAccount)
		resp.PropertyName = r.PropertyName
		resp.UnmatchedLineCount = r.UnmatchedLineCount
		resp.UnmatchedPaymentCount = r.UnmatchedPaymentCount
		out[i] = resp
	}
	return out
}

type StatementLineRequest struct {
	PostedOn    string `json:"posted_on" validate:"required,datetime=2006-01-02"`
	Description string `json:"description" validate:"omitempty,max=300,noctrl"`
	AmountCents int64  `json:"amount_cents" validate:"required,gte=-100000000000,lte=100000000000"`
	ExternalID  string `json:"external_id" validate:"omitempty,max=100,noctrl"`
}

type AddStatementLinesRequest struct {
	Source string                 `json:"source" validate:"required,oneof=manual csv"`
	Lines  []StatementLineRequest `json:"lines" validate:"required,min=1,max=1000,dive"`
}

func (r *AddStatementLinesRequest) Sanitize() {
	for i := range r.Lines {
		r.Lines[i].Description = sanitizeString(r.Lines[i].Description)
		r.Lines[i].ExternalID = sanitizeString(r.Lines[i].ExternalID)
	}
}

func (r *AddStatementLinesRequest) ToDomain() ([]domain.StatementLineInput, error) {
	out := make([]domain.StatementLineInput, len(r.Lines))
	for i, l := range r.Lines {
		posted, err := parseRequiredDate(fmt.Sprintf("lines[%d].posted_on", i), l.PostedOn)
		if err != nil {
			return nil, err
		}
		out[i] = domain.StatementLineInput{PostedOn: posted, Description: l.Description, AmountCents: l.AmountCents, ExternalID: l.ExternalID}
	}
	return out, nil
}

type StatementLineResponse struct {
	ID               string `json:"id"`
	PostedOn         string `json:"posted_on"`
	Description      string `json:"description"`
	AmountCents      int64  `json:"amount_cents"`
	Source           string `json:"source"`
	MatchedPaymentID string `json:"matched_payment_id,omitempty"`
}

func NewStatementLineListResponse(lines []*domain.StatementLine) []StatementLineResponse {
	out := make([]StatementLineResponse, len(lines))
	for i, l := range lines {
		out[i] = StatementLineResponse{
			ID: l.ID.String(), PostedOn: formatDate(l.PostedOn), Description: l.Description, AmountCents: l.AmountCents,
			Source: string(l.Source), MatchedPaymentID: optionalUUIDString(l.MatchedPaymentID),
		}
	}
	return out
}

type MatchRequest struct {
	PaymentID string `json:"payment_id" validate:"required,uuid4"`
}

type ReconcilablePaymentResponse struct {
	PaymentID     string `json:"payment_id"`
	TransactionID string `json:"transaction_id"`
	AmountCents   int64  `json:"amount_cents"`
	PaidOn        string `json:"paid_on"`
	Method        string `json:"method,omitempty"`
	Reference     string `json:"reference,omitempty"`
	Description   string `json:"description"`
}

type ReconciliationResponse struct {
	Account                BankAccountResponse           `json:"account"`
	MatchedLineCount       int                           `json:"matched_line_count"`
	UnmatchedLineCount     int                           `json:"unmatched_line_count"`
	UnmatchedLinesCents    int64                         `json:"unmatched_lines_cents"`
	UnmatchedPayments      []ReconcilablePaymentResponse `json:"unmatched_payments"`
	UnmatchedPaymentsCents int64                         `json:"unmatched_payments_cents"`
}

func NewReconciliationResponse(r *domain.Reconciliation) ReconciliationResponse {
	payments := make([]ReconcilablePaymentResponse, len(r.UnmatchedPayments))
	for i, p := range r.UnmatchedPayments {
		payments[i] = ReconcilablePaymentResponse{
			PaymentID: p.PaymentID.String(), TransactionID: p.TransactionID.String(), AmountCents: p.AmountCents,
			PaidOn: formatDate(p.PaidOn), Method: p.Method, Reference: p.Reference, Description: p.Description,
		}
	}
	return ReconciliationResponse{
		Account: newBankAccountResponse(r.Account), MatchedLineCount: r.MatchedLineCount, UnmatchedLineCount: r.UnmatchedLineCount,
		UnmatchedLinesCents: r.UnmatchedLinesCents, UnmatchedPayments: payments, UnmatchedPaymentsCents: r.UnmatchedPaymentsCents,
	}
}

// ---- Security deposits ----

type DeductionResponse struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	AmountCents  int64  `json:"amount_cents"`
	AttachmentID string `json:"attachment_id,omitempty"`
}

type DepositResponse struct {
	ID              string              `json:"id"`
	LeaseID         string              `json:"lease_id"`
	PropertyID      string              `json:"property_id"`
	PropertyName    string              `json:"property_name"`
	UnitID          string              `json:"unit_id"`
	UnitName        string              `json:"unit_name"`
	TenantName      string              `json:"tenant_name"`
	AmountCents     int64               `json:"amount_cents"`
	CollectedOn     string              `json:"collected_on"`
	HeldInAccountID string              `json:"held_in_account_id,omitempty"`
	HeldInAccount   string              `json:"held_in_account,omitempty"`
	Status          string              `json:"status"`
	DeductionsCents int64               `json:"deductions_cents"`
	RefundCents     int64               `json:"refund_cents"`
	RefundMethod    string              `json:"refund_method,omitempty"`
	RefundedOn      string              `json:"refunded_on,omitempty"`
	Deductions      []DeductionResponse `json:"deductions"`
}

func NewDepositResponse(d *domain.DepositRow) DepositResponse {
	deductions := make([]DeductionResponse, len(d.Deductions))
	for i, x := range d.Deductions {
		deductions[i] = DeductionResponse{ID: x.ID.String(), Description: x.Description, AmountCents: x.AmountCents, AttachmentID: optionalUUIDString(x.AttachmentID)}
	}
	return DepositResponse{
		ID: d.ID.String(), LeaseID: d.LeaseID.String(), PropertyID: d.PropertyID.String(), PropertyName: d.PropertyName,
		UnitID: d.UnitID.String(), UnitName: d.UnitName, TenantName: d.TenantName, AmountCents: d.AmountCents,
		CollectedOn: formatDate(d.CollectedOn), HeldInAccountID: optionalUUIDString(d.HeldInAccountID), HeldInAccount: d.HeldInAccount,
		Status: string(d.Status()), DeductionsCents: d.DeductionsCents, RefundCents: d.RefundCents, RefundMethod: d.RefundMethod,
		RefundedOn: formatOptionalDate(d.RefundedOn), Deductions: deductions,
	}
}

func NewDepositListResponse(rows []*domain.DepositRow) []DepositResponse {
	out := make([]DepositResponse, len(rows))
	for i, r := range rows {
		out[i] = NewDepositResponse(r)
	}
	return out
}

type SetHeldInAccountRequest struct {
	AccountID string `json:"account_id" validate:"omitempty,uuid4"`
}

type AddDeductionRequest struct {
	Description  string `json:"description" validate:"required,min=1,max=300,noctrl"`
	AmountCents  int64  `json:"amount_cents" validate:"required,gt=0,lte=100000000000"`
	AttachmentID string `json:"attachment_id" validate:"omitempty,uuid4"`
}

func (r *AddDeductionRequest) Sanitize() { r.Description = sanitizeString(r.Description) }

func (r *AddDeductionRequest) ToDomain() (domain.AddDeductionInput, error) {
	att, err := parseOptionalUUID("attachment_id", r.AttachmentID)
	if err != nil {
		return domain.AddDeductionInput{}, err
	}
	return domain.AddDeductionInput{Description: r.Description, AmountCents: r.AmountCents, AttachmentID: att}, nil
}

type SettleDepositRequest struct {
	RefundCents  int64  `json:"refund_cents" validate:"gte=0,lte=100000000000"`
	RefundMethod string `json:"refund_method" validate:"omitempty,max=40,noctrl"`
	RefundedOn   string `json:"refunded_on" validate:"required,datetime=2006-01-02"`
}

func (r *SettleDepositRequest) Sanitize() { r.RefundMethod = sanitizeString(r.RefundMethod) }

func (r *SettleDepositRequest) ToDomain() (domain.SettleDepositInput, error) {
	on, err := parseRequiredDate("refunded_on", r.RefundedOn)
	if err != nil {
		return domain.SettleDepositInput{}, err
	}
	return domain.SettleDepositInput{RefundCents: r.RefundCents, Method: r.RefundMethod, RefundedOn: on}, nil
}
