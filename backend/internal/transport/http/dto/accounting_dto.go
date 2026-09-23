package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// Money on the wire is always integer cents (`*_cents`), never a
// float — see domain/accounting.go. Dates are "YYYY-MM-DD"; timestamps
// are RFC3339.

const accountingDateLayout = "2006-01-02"

type AccountingListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func formatDate(t time.Time) string { return t.Format(accountingDateLayout) }

func formatOptionalDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatDate(*t)
}

func optionalUUIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func parseRequiredDate(field, s string) (time.Time, error) {
	t, err := time.Parse(accountingDateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", field, err)
	}
	return t, nil
}

func parseOptionalUUID(field, s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	return &id, nil
}

// ---- Requests ----

type RecordPaymentRequest struct {
	AmountCents   int64  `json:"amount_cents" validate:"required,gt=0,lte=100000000000"`
	PaidOn        string `json:"paid_on" validate:"required,datetime=2006-01-02"`
	Method        string `json:"method" validate:"omitempty,max=40,noctrl"`
	Reference     string `json:"reference" validate:"omitempty,max=120,noctrl"`
	BankAccountID string `json:"bank_account_id" validate:"omitempty,uuid4"`
}

func (r *RecordPaymentRequest) Sanitize() {
	r.Method = sanitizeString(r.Method)
	r.Reference = sanitizeString(r.Reference)
}

func (r *RecordPaymentRequest) ToDomain() (domain.RecordPaymentInput, error) {
	paidOn, err := parseRequiredDate("paid_on", r.PaidOn)
	if err != nil {
		return domain.RecordPaymentInput{}, err
	}
	bank, err := parseOptionalUUID("bank_account_id", r.BankAccountID)
	if err != nil {
		return domain.RecordPaymentInput{}, err
	}
	return domain.RecordPaymentInput{
		AmountCents:   r.AmountCents,
		PaidOn:        paidOn,
		Method:        r.Method,
		Reference:     r.Reference,
		BankAccountID: bank,
	}, nil
}

type GenerateRentRequest struct {
	Period string `json:"period" validate:"required,datetime=2006-01-02"`
}

type ExpenseRequest struct {
	PropertyID          string `json:"property_id" validate:"required,uuid4"`
	UnitID              string `json:"unit_id" validate:"omitempty,uuid4"`
	Category            string `json:"category" validate:"required,oneof=repairs utilities insurance property_tax management_fee landscaping supplies legal mortgage other"`
	VendorID            string `json:"vendor_id" validate:"omitempty,uuid4"`
	VendorName          string `json:"vendor_name" validate:"omitempty,max=200,noctrl"`
	AmountCents         int64  `json:"amount_cents" validate:"required,gt=0,lte=100000000000"`
	IncurredOn          string `json:"incurred_on" validate:"required,datetime=2006-01-02"`
	DueOn               string `json:"due_on" validate:"omitempty,datetime=2006-01-02"`
	Description         string `json:"description" validate:"omitempty,max=2000,noctrl"`
	TaxDeductible       bool   `json:"tax_deductible"`
	IsRecurring         bool   `json:"is_recurring"`
	RecurrenceFrequency string `json:"recurrence_frequency" validate:"omitempty,oneof=monthly quarterly yearly"`
	AttachmentID        string `json:"attachment_id" validate:"omitempty,uuid4"`
	// Create-only: recording a full payment at creation ("Date paid").
	PaidOn        string `json:"paid_on" validate:"omitempty,datetime=2006-01-02"`
	PaymentMethod string `json:"payment_method" validate:"omitempty,max=40,noctrl"`
	BankAccountID string `json:"bank_account_id" validate:"omitempty,uuid4"`
}

func (r *ExpenseRequest) Sanitize() {
	r.VendorName = sanitizeString(r.VendorName)
	r.Description = sanitizeString(r.Description)
	r.PaymentMethod = sanitizeString(r.PaymentMethod)
}

type expenseParsed struct {
	propertyID, unitID, vendorID, attachmentID, bankID *uuid.UUID
	incurredOn                                         time.Time
	dueOn, paidOn                                      *time.Time
	frequency                                          *domain.RecurrenceFrequency
}

func (r *ExpenseRequest) parse() (*expenseParsed, error) {
	var p expenseParsed
	var err error
	if p.propertyID, err = parseOptionalUUID("property_id", r.PropertyID); err != nil {
		return nil, err
	}
	if p.unitID, err = parseOptionalUUID("unit_id", r.UnitID); err != nil {
		return nil, err
	}
	if p.vendorID, err = parseOptionalUUID("vendor_id", r.VendorID); err != nil {
		return nil, err
	}
	if p.attachmentID, err = parseOptionalUUID("attachment_id", r.AttachmentID); err != nil {
		return nil, err
	}
	if p.bankID, err = parseOptionalUUID("bank_account_id", r.BankAccountID); err != nil {
		return nil, err
	}
	if p.incurredOn, err = parseRequiredDate("incurred_on", r.IncurredOn); err != nil {
		return nil, err
	}
	if p.dueOn, err = parseOptionalDate(r.DueOn, accountingDateLayout); err != nil {
		return nil, fmt.Errorf("due_on: %w", err)
	}
	if p.paidOn, err = parseOptionalDate(r.PaidOn, accountingDateLayout); err != nil {
		return nil, fmt.Errorf("paid_on: %w", err)
	}
	if r.RecurrenceFrequency != "" {
		f := domain.RecurrenceFrequency(r.RecurrenceFrequency)
		p.frequency = &f
	}
	return &p, nil
}

func (r *ExpenseRequest) ToCreateDomain() (domain.CreateExpenseInput, error) {
	p, err := r.parse()
	if err != nil {
		return domain.CreateExpenseInput{}, err
	}
	return domain.CreateExpenseInput{
		PropertyID:          *p.propertyID,
		UnitID:              p.unitID,
		Category:            domain.ExpenseCategory(r.Category),
		VendorID:            p.vendorID,
		VendorName:          r.VendorName,
		AmountCents:         r.AmountCents,
		IncurredOn:          p.incurredOn,
		DueOn:               p.dueOn,
		Description:         r.Description,
		TaxDeductible:       r.TaxDeductible,
		IsRecurring:         r.IsRecurring,
		RecurrenceFrequency: p.frequency,
		AttachmentID:        p.attachmentID,
		PaidOn:              p.paidOn,
		PaymentMethod:       r.PaymentMethod,
		BankAccountID:       p.bankID,
	}, nil
}

func (r *ExpenseRequest) ToUpdateDomain() (domain.UpdateExpenseInput, error) {
	p, err := r.parse()
	if err != nil {
		return domain.UpdateExpenseInput{}, err
	}
	return domain.UpdateExpenseInput{
		PropertyID:          *p.propertyID,
		UnitID:              p.unitID,
		Category:            domain.ExpenseCategory(r.Category),
		VendorID:            p.vendorID,
		VendorName:          r.VendorName,
		AmountCents:         r.AmountCents,
		IncurredOn:          p.incurredOn,
		DueOn:               p.dueOn,
		Description:         r.Description,
		TaxDeductible:       r.TaxDeductible,
		IsRecurring:         r.IsRecurring,
		RecurrenceFrequency: p.frequency,
		AttachmentID:        p.attachmentID,
	}, nil
}

type CreateChargeRequest struct {
	UnitID      string `json:"unit_id" validate:"required,uuid4"`
	ChargeType  string `json:"charge_type" validate:"required,oneof=utility_rebill damage amenity other"`
	AmountCents int64  `json:"amount_cents" validate:"required,gt=0,lte=100000000000"`
	Description string `json:"description" validate:"required,min=1,max=500,noctrl"`
	DueOn       string `json:"due_on" validate:"required,datetime=2006-01-02"`
}

func (r *CreateChargeRequest) Sanitize() { r.Description = sanitizeString(r.Description) }

func (r *CreateChargeRequest) ToDomain() (domain.CreateChargeInput, error) {
	unitID, err := uuid.Parse(r.UnitID)
	if err != nil {
		return domain.CreateChargeInput{}, fmt.Errorf("unit_id: %w", err)
	}
	dueOn, err := parseRequiredDate("due_on", r.DueOn)
	if err != nil {
		return domain.CreateChargeInput{}, err
	}
	return domain.CreateChargeInput{
		UnitID:      unitID,
		ChargeType:  domain.ChargeType(r.ChargeType),
		AmountCents: r.AmountCents,
		Description: r.Description,
		DueOn:       dueOn,
	}, nil
}

type UpdateAccountingSettingsRequest struct {
	LateFeeKind       string `json:"late_fee_kind" validate:"required,oneof=flat percent"`
	LateFeeValue      int64  `json:"late_fee_value" validate:"gte=0,lte=100000000000"`
	GraceDays         int    `json:"grace_days" validate:"gte=0,lte=60"`
	DefaultRentDueDay int    `json:"default_rent_due_day" validate:"gte=1,lte=31"`
}

func (r *UpdateAccountingSettingsRequest) ToDomain() domain.UpdateAccountingSettingsInput {
	return domain.UpdateAccountingSettingsInput{
		LateFeeKind:       domain.LateFeeKind(r.LateFeeKind),
		LateFeeValue:      r.LateFeeValue,
		GraceDays:         r.GraceDays,
		DefaultRentDueDay: r.DefaultRentDueDay,
	}
}

// ---- Responses ----

// TransactionResponse is the shape of an expense or a charge (both are
// ledger rows). status is the computed one: unpaid/paid/overdue for an
// expense, paid/partial/late/unpaid for a charge.
type TransactionResponse struct {
	ID                  string `json:"id"`
	PropertyID          string `json:"property_id"`
	PropertyName        string `json:"property_name"`
	UnitID              string `json:"unit_id,omitempty"`
	UnitName            string `json:"unit_name,omitempty"`
	TenantName          string `json:"tenant_name,omitempty"`
	LeaseID             string `json:"lease_id,omitempty"`
	Kind                string `json:"kind"`
	Type                string `json:"type"`
	ChargeType          string `json:"charge_type,omitempty"`
	Category            string `json:"category,omitempty"`
	AmountCents         int64  `json:"amount_cents"`
	PaidCents           int64  `json:"paid_cents"`
	OutstandingCents    int64  `json:"outstanding_cents"`
	Status              string `json:"status"`
	IncurredOn          string `json:"incurred_on"`
	DueOn               string `json:"due_on,omitempty"`
	LastPaidOn          string `json:"last_paid_on,omitempty"`
	VendorID            string `json:"vendor_id,omitempty"`
	VendorName          string `json:"vendor_name,omitempty"`
	WorkOrderID         string `json:"work_order_id,omitempty"`
	WorkOrderTitle      string `json:"work_order_title,omitempty"`
	Description         string `json:"description,omitempty"`
	TaxDeductible       bool   `json:"tax_deductible"`
	IsRecurring         bool   `json:"is_recurring"`
	RecurrenceFrequency string `json:"recurrence_frequency,omitempty"`
	AttachmentID        string `json:"attachment_id,omitempty"`
	Source              string `json:"source"`
	// ReadOnly is true for an expense created from a work order: it
	// mirrors the work order's actual cost and is edited there.
	ReadOnly  bool   `json:"read_only"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func NewTransactionResponse(r *domain.TransactionRow, now time.Time) TransactionResponse {
	resp := TransactionResponse{
		ID:               r.ID.String(),
		PropertyID:       r.PropertyID.String(),
		PropertyName:     r.PropertyName,
		UnitID:           optionalUUIDString(r.UnitID),
		UnitName:         r.UnitName,
		TenantName:       r.TenantName,
		LeaseID:          optionalUUIDString(r.LeaseID),
		Kind:             string(r.Kind),
		Type:             string(r.Type),
		AmountCents:      r.AmountCents,
		PaidCents:        r.PaidCents,
		OutstandingCents: r.OutstandingCents(),
		IncurredOn:       formatDate(r.IncurredOn),
		DueOn:            formatOptionalDate(r.DueOn),
		LastPaidOn:       formatOptionalDate(r.LastPaidOn),
		VendorID:         optionalUUIDString(r.VendorID),
		VendorName:       r.VendorName,
		WorkOrderID:      optionalUUIDString(r.WorkOrderID),
		WorkOrderTitle:   r.WorkOrderTitle,
		Description:      r.Description,
		TaxDeductible:    r.TaxDeductible,
		IsRecurring:      r.IsRecurring,
		AttachmentID:     optionalUUIDString(r.AttachmentID),
		Source:           string(r.Source),
		ReadOnly:         r.Source == domain.TransactionSourceWorkOrder,
		CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        r.UpdatedAt.Format(time.RFC3339),
	}
	if r.VendorCompany != "" {
		resp.VendorName = r.VendorCompany
	}
	if r.ChargeType != nil {
		resp.ChargeType = string(*r.ChargeType)
	}
	if r.Category != nil {
		resp.Category = string(*r.Category)
	}
	if r.RecurrenceFrequency != nil {
		resp.RecurrenceFrequency = string(*r.RecurrenceFrequency)
	}
	if r.Kind == domain.TransactionKindExpense {
		resp.Status = string(r.ExpenseStatus(now))
	} else {
		resp.Status = string(r.IncomeStatus(0, now))
	}
	return resp
}

func NewTransactionListResponse(rows []*domain.TransactionRow, now time.Time) []TransactionResponse {
	out := make([]TransactionResponse, len(rows))
	for i, r := range rows {
		out[i] = NewTransactionResponse(r, now)
	}
	return out
}

type PaymentResponse struct {
	ID            string `json:"id"`
	TransactionID string `json:"transaction_id"`
	AmountCents   int64  `json:"amount_cents"`
	PaidOn        string `json:"paid_on"`
	Method        string `json:"method,omitempty"`
	Reference     string `json:"reference,omitempty"`
	BankAccountID string `json:"bank_account_id,omitempty"`
	Voided        bool   `json:"voided"`
	CreatedAt     string `json:"created_at"`
}

func NewPaymentResponse(p *domain.TransactionPayment) PaymentResponse {
	return PaymentResponse{
		ID:            p.ID.String(),
		TransactionID: p.TransactionID.String(),
		AmountCents:   p.AmountCents,
		PaidOn:        formatDate(p.PaidOn),
		Method:        p.Method,
		Reference:     p.Reference,
		BankAccountID: optionalUUIDString(p.BankAccountID),
		Voided:        p.VoidedAt != nil,
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
	}
}

func NewPaymentListResponse(ps []*domain.TransactionPayment) []PaymentResponse {
	out := make([]PaymentResponse, len(ps))
	for i, p := range ps {
		out[i] = NewPaymentResponse(p)
	}
	return out
}

type RentRollRowResponse struct {
	LeaseID          string `json:"lease_id"`
	PropertyID       string `json:"property_id"`
	PropertyName     string `json:"property_name"`
	UnitID           string `json:"unit_id"`
	UnitName         string `json:"unit_name"`
	TenantName       string `json:"tenant_name"`
	Period           string `json:"period"`
	RentCents        int64  `json:"rent_cents"`
	DueOn            string `json:"due_on"`
	BilledCents      int64  `json:"billed_cents"`
	PaidCents        int64  `json:"paid_cents"`
	OutstandingCents int64  `json:"outstanding_cents"`
	Status           string `json:"status"`
	LastPaidOn       string `json:"last_paid_on,omitempty"`
	GraceDays        int    `json:"grace_days"`
}

func NewRentRollListResponse(rows []*domain.RentRollRow, now time.Time) []RentRollRowResponse {
	out := make([]RentRollRowResponse, len(rows))
	for i, r := range rows {
		out[i] = RentRollRowResponse{
			LeaseID:          r.LeaseID.String(),
			PropertyID:       r.PropertyID.String(),
			PropertyName:     r.PropertyName,
			UnitID:           r.UnitID.String(),
			UnitName:         r.UnitName,
			TenantName:       r.TenantName,
			Period:           formatDate(r.Period),
			RentCents:        r.RentCents,
			DueOn:            formatDate(r.DueOn),
			BilledCents:      r.BilledCents,
			PaidCents:        r.PaidCents,
			OutstandingCents: r.OutstandingCents(),
			Status:           string(r.Status(now)),
			LastPaidOn:       formatOptionalDate(r.LastPaidOn),
			GraceDays:        r.GraceDays,
		}
	}
	return out
}

type GenerateResultResponse struct {
	RentCreated    int `json:"rent_created"`
	LateFeeCreated int `json:"late_fee_created"`
}

type AuditEntryResponse struct {
	ID        string                        `json:"id"`
	Action    string                        `json:"action"`
	ActorID   string                        `json:"actor_id"`
	Changes   map[string]domain.FieldChange `json:"changes"`
	CreatedAt string                        `json:"created_at"`
}

func NewAuditListResponse(entries []*domain.AuditEntry) []AuditEntryResponse {
	out := make([]AuditEntryResponse, len(entries))
	for i, e := range entries {
		out[i] = AuditEntryResponse{
			ID:        e.ID.String(),
			Action:    e.Action,
			ActorID:   e.ActorID.String(),
			Changes:   e.Changes,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		}
	}
	return out
}

type AccountingSettingsResponse struct {
	LateFeeKind       string `json:"late_fee_kind"`
	LateFeeValue      int64  `json:"late_fee_value"`
	GraceDays         int    `json:"grace_days"`
	DefaultRentDueDay int    `json:"default_rent_due_day"`
}

func NewAccountingSettingsResponse(s *domain.AccountingSettings) AccountingSettingsResponse {
	return AccountingSettingsResponse{
		LateFeeKind:       string(s.LateFeeKind),
		LateFeeValue:      s.LateFeeValue,
		GraceDays:         s.GraceDays,
		DefaultRentDueDay: s.DefaultRentDueDay,
	}
}

type SeriesPointResponse struct {
	Bucket       string `json:"bucket"`
	IncomeCents  int64  `json:"income_cents"`
	ExpenseCents int64  `json:"expense_cents"`
}

type AccountingDashboardResponse struct {
	Period               string                `json:"period"`
	CollectedCents       int64                 `json:"collected_cents"`
	ExpectedCents        int64                 `json:"expected_cents"`
	OutstandingCents     int64                 `json:"outstanding_cents"`
	ExpensesPaidCents    int64                 `json:"expenses_paid_cents"`
	NetIncomeCents       int64                 `json:"net_income_cents"`
	UpcomingExpenseCents int64                 `json:"upcoming_expense_cents"`
	UpcomingExpenseCount int                   `json:"upcoming_expense_count"`
	OverdueExpenseCount  int                   `json:"overdue_expense_count"`
	LateRentCount        int                   `json:"late_rent_count"`
	Series               []SeriesPointResponse `json:"series"`
}

func NewAccountingDashboardResponse(d *domain.AccountingDashboard) AccountingDashboardResponse {
	series := make([]SeriesPointResponse, len(d.Series))
	for i, p := range d.Series {
		series[i] = SeriesPointResponse{Bucket: formatDate(p.Bucket), IncomeCents: p.IncomeCents, ExpenseCents: p.ExpenseCents}
	}
	return AccountingDashboardResponse{
		Period:               string(d.Period),
		CollectedCents:       d.Totals.CollectedCents,
		ExpectedCents:        d.Totals.ExpectedCents,
		OutstandingCents:     d.Totals.OutstandingCents,
		ExpensesPaidCents:    d.Totals.ExpensesPaidCents,
		NetIncomeCents:       d.NetIncomeCents,
		UpcomingExpenseCents: d.Totals.UpcomingExpenseCents,
		UpcomingExpenseCount: d.Totals.UpcomingExpenseCount,
		OverdueExpenseCount:  d.Totals.OverdueExpenseCount,
		LateRentCount:        d.Totals.LateRentCount,
		Series:               series,
	}
}
