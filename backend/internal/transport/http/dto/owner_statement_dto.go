package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// ---- Property owners ----

type PropertyOwnerRequest struct {
	Name             string   `json:"name" validate:"required,min=1,max=200,noctrl"`
	Email            string   `json:"email" validate:"omitempty,email,max=200"`
	Phone            string   `json:"phone" validate:"omitempty,max=50,noctrl"`
	ManagementFeeBps int      `json:"management_fee_bps" validate:"gte=0,lte=10000"`
	AutoStatements   bool     `json:"auto_statements"`
	PropertyIDs      []string `json:"property_ids" validate:"max=200,dive,uuid4"`
}

func (r *PropertyOwnerRequest) Sanitize() {
	r.Name = sanitizeString(r.Name)
	r.Email = sanitizeString(r.Email)
	r.Phone = sanitizeString(r.Phone)
}

func (r *PropertyOwnerRequest) ToDomain() (domain.PropertyOwnerInput, error) {
	ids := make([]uuid.UUID, 0, len(r.PropertyIDs))
	for _, s := range r.PropertyIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return domain.PropertyOwnerInput{}, fmt.Errorf("property_ids: %w", err)
		}
		ids = append(ids, id)
	}
	return domain.PropertyOwnerInput{
		Name: r.Name, Email: r.Email, Phone: r.Phone, ManagementFeeBps: r.ManagementFeeBps,
		AutoStatements: r.AutoStatements, PropertyIDs: ids,
	}, nil
}

type PropertyOwnerResponse struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Email            string   `json:"email,omitempty"`
	Phone            string   `json:"phone,omitempty"`
	ManagementFeeBps int      `json:"management_fee_bps"`
	AutoStatements   bool     `json:"auto_statements"`
	PropertyIDs      []string `json:"property_ids"`
	PropertyNames    []string `json:"property_names"`
}

func NewPropertyOwnerResponse(o *domain.PropertyOwnerRow) PropertyOwnerResponse {
	ids := make([]string, len(o.PropertyIDs))
	for i, id := range o.PropertyIDs {
		ids[i] = id.String()
	}
	names := o.PropertyNames
	if names == nil {
		names = []string{}
	}
	return PropertyOwnerResponse{
		ID: o.ID.String(), Name: o.Name, Email: o.Email, Phone: o.Phone, ManagementFeeBps: o.ManagementFeeBps,
		AutoStatements: o.AutoStatements, PropertyIDs: ids, PropertyNames: names,
	}
}

func NewPropertyOwnerListResponse(rows []*domain.PropertyOwnerRow) []PropertyOwnerResponse {
	out := make([]PropertyOwnerResponse, len(rows))
	for i, r := range rows {
		out[i] = NewPropertyOwnerResponse(r)
	}
	return out
}

// ---- Owner statements ----

type GenerateStatementRequest struct {
	PropertyID  string `json:"property_id" validate:"required,uuid4"`
	PeriodStart string `json:"period_start" validate:"required,datetime=2006-01-02"`
	PeriodEnd   string `json:"period_end" validate:"required,datetime=2006-01-02"`
}

func (r *GenerateStatementRequest) ToDomain() (propertyID uuid.UUID, start, end time.Time, err error) {
	if propertyID, err = uuid.Parse(r.PropertyID); err != nil {
		return uuid.UUID{}, time.Time{}, time.Time{}, fmt.Errorf("property_id: %w", err)
	}
	if start, err = parseRequiredDate("period_start", r.PeriodStart); err != nil {
		return uuid.UUID{}, time.Time{}, time.Time{}, err
	}
	if end, err = parseRequiredDate("period_end", r.PeriodEnd); err != nil {
		return uuid.UUID{}, time.Time{}, time.Time{}, err
	}
	return propertyID, start, end, nil
}

type MarkStatementPaidRequest struct {
	AmountCents int64  `json:"amount_cents" validate:"required,gt=0,lte=100000000000"`
	PaidOn      string `json:"paid_on" validate:"required,datetime=2006-01-02"`
	Method      string `json:"method" validate:"omitempty,max=40,noctrl"`
	Reference   string `json:"reference" validate:"omitempty,max=120,noctrl"`
}

func (r *MarkStatementPaidRequest) Sanitize() {
	r.Method = sanitizeString(r.Method)
	r.Reference = sanitizeString(r.Reference)
}

func (r *MarkStatementPaidRequest) ToDomain() (domain.MarkStatementPaidInput, error) {
	paidOn, err := parseRequiredDate("paid_on", r.PaidOn)
	if err != nil {
		return domain.MarkStatementPaidInput{}, err
	}
	return domain.MarkStatementPaidInput{AmountCents: r.AmountCents, PaidOn: paidOn, Method: r.Method, Reference: r.Reference}, nil
}

type StatementResponse struct {
	ID             string                     `json:"id"`
	PropertyID     string                     `json:"property_id"`
	PropertyName   string                     `json:"property_name"`
	OwnerName      string                     `json:"owner_name"`
	OwnerEmail     string                     `json:"owner_email,omitempty"`
	PeriodStart    string                     `json:"period_start"`
	PeriodEnd      string                     `json:"period_end"`
	Status         string                     `json:"status"`
	EmailQueued    bool                       `json:"email_queued"`
	IncomeCents    int64                      `json:"income_cents"`
	ExpensesCents  int64                      `json:"expenses_cents"`
	FeeCents       int64                      `json:"fee_cents"`
	FeeBps         int                        `json:"fee_bps"`
	NetPayoutCents int64                      `json:"net_payout_cents"`
	GeneratedAt    string                     `json:"generated_at"`
	SentAt         string                     `json:"sent_at,omitempty"`
	PaidCents      *int64                     `json:"paid_cents,omitempty"`
	PaidOn         string                     `json:"paid_on,omitempty"`
	PayMethod      string                     `json:"pay_method,omitempty"`
	PayReference   string                     `json:"pay_reference,omitempty"`
	Income         []domain.StatementLineItem `json:"income"`
	Expenses       []domain.StatementLineItem `json:"expenses"`
}

func NewStatementResponse(s *domain.OwnerStatementRow, withLines bool) StatementResponse {
	resp := StatementResponse{
		ID: s.ID.String(), PropertyID: s.PropertyID.String(), PropertyName: s.PropertyName, OwnerName: s.OwnerName,
		OwnerEmail: s.Snapshot.OwnerEmail, PeriodStart: formatDate(s.PeriodStart), PeriodEnd: formatDate(s.PeriodEnd),
		Status: string(s.Status), EmailQueued: s.EmailQueued, IncomeCents: s.Snapshot.IncomeCents, ExpensesCents: s.Snapshot.ExpensesCents,
		FeeCents: s.Snapshot.FeeCents, FeeBps: s.Snapshot.FeeBps, NetPayoutCents: s.Snapshot.NetPayoutCents,
		GeneratedAt: s.GeneratedAt.Format(time.RFC3339), PaidCents: s.PaidCents, PaidOn: formatOptionalDate(s.PaidOn),
		PayMethod: s.PayMethod, PayReference: s.PayReference,
		Income: []domain.StatementLineItem{}, Expenses: []domain.StatementLineItem{},
	}
	if s.SentAt != nil {
		resp.SentAt = s.SentAt.Format(time.RFC3339)
	}
	if withLines {
		if s.Snapshot.Income != nil {
			resp.Income = s.Snapshot.Income
		}
		if s.Snapshot.Expenses != nil {
			resp.Expenses = s.Snapshot.Expenses
		}
	}
	return resp
}

func NewStatementListResponse(rows []*domain.OwnerStatementRow) []StatementResponse {
	out := make([]StatementResponse, len(rows))
	for i, r := range rows {
		out[i] = NewStatementResponse(r, false)
	}
	return out
}
