package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

// VendorService resolves ownership via the vendor's own OwnerID — one
// hop, like PropertyService, since a vendor belongs directly to an
// owner rather than through a parent.
type VendorService struct {
	repo         domain.VendorRepository
	propertyRepo domain.PropertyRepository
	log          *slog.Logger
}

func NewVendorService(repo domain.VendorRepository, propertyRepo domain.PropertyRepository, log *slog.Logger) *VendorService {
	return &VendorService{repo: repo, propertyRepo: propertyRepo, log: log}
}

var _ domain.VendorService = (*VendorService)(nil)

func (s *VendorService) CreateVendor(ctx context.Context, ownerID uuid.UUID, input domain.CreateVendorInput) (*domain.Vendor, error) {
	if verrs := validateVendor(input.CompanyName, input.Categories, input.RateType, input.PaymentTerms); len(verrs) > 0 {
		return nil, fmt.Errorf("create vendor: %w", verrs)
	}

	propertiesServed, err := s.resolvePropertiesServed(ctx, ownerID, input.ServesAllProperties, input.PropertiesServed)
	if err != nil {
		return nil, fmt.Errorf("create vendor: %w", err)
	}

	now := time.Now().UTC()
	v := &domain.Vendor{
		ID:                  uuid.New(),
		OwnerID:             ownerID,
		CompanyName:         input.CompanyName,
		Categories:          input.Categories,
		ContactPerson:       input.ContactPerson,
		Phone:               input.Phone,
		Email:               input.Email,
		Address:             input.Address,
		ServesAllProperties: input.ServesAllProperties,
		InsuranceExpiry:     input.InsuranceExpiry,
		LicenseNumber:       input.LicenseNumber,
		LicenseExpiry:       input.LicenseExpiry,
		COILink:             input.COILink,
		TaxDocLink:          input.TaxDocLink,
		RateType:            input.RateType,
		RateAmount:          input.RateAmount,
		PaymentTerms:        input.PaymentTerms,
		InternalNotes:       input.InternalNotes,
		Active:              true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if v.Categories == nil {
		v.Categories = []domain.WorkOrderCategory{}
	}

	if err := s.repo.Create(ctx, v, propertiesServed); err != nil {
		return nil, fmt.Errorf("create vendor: %w", err)
	}
	return v, nil
}

func (s *VendorService) GetVendor(ctx context.Context, id, ownerID uuid.UUID) (*domain.VendorWithStats, error) {
	v, err := s.repo.GetByIDWithStats(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get vendor %s: %w", id, err)
	}
	if v.OwnerID != ownerID {
		return nil, fmt.Errorf("get vendor %s: %w", id, domain.ErrNotFound)
	}
	return v, nil
}

func (s *VendorService) ListVendorsForOwner(ctx context.Context, ownerID uuid.UUID, opts domain.VendorListOptions) ([]*domain.VendorWithStats, int, error) {
	opts.OwnerID = ownerID
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	vendors, total, err := s.repo.ListForOwner(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list vendors for owner %s: %w", ownerID, err)
	}
	return vendors, total, nil
}

func (s *VendorService) ListVendorsForCategory(ctx context.Context, ownerID uuid.UUID, category domain.WorkOrderCategory) ([]*domain.VendorWithStats, error) {
	vendors, err := s.repo.ListForCategory(ctx, ownerID, category)
	if err != nil {
		return nil, fmt.Errorf("list vendors for owner %s category %q: %w", ownerID, category, err)
	}
	return vendors, nil
}

func (s *VendorService) UpdateVendor(ctx context.Context, id, ownerID uuid.UUID, input domain.UpdateVendorInput) (*domain.Vendor, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update vendor %s: %w", id, err)
	}
	if v.OwnerID != ownerID {
		return nil, fmt.Errorf("update vendor %s: %w", id, domain.ErrNotFound)
	}

	var verrs domain.ValidationErrors

	if input.CompanyName != nil {
		if *input.CompanyName == "" {
			verrs = append(verrs, &domain.ValidationError{Field: "company_name", Message: "cannot be empty"})
		} else {
			v.CompanyName = *input.CompanyName
		}
	}
	if input.Categories != nil {
		v.Categories = input.Categories
	}
	if input.ContactPerson != nil {
		v.ContactPerson = *input.ContactPerson
	}
	if input.Phone != nil {
		v.Phone = *input.Phone
	}
	if input.Email != nil {
		v.Email = *input.Email
	}
	if input.Address != nil {
		v.Address = *input.Address
	}
	if input.ServesAllProperties != nil {
		v.ServesAllProperties = *input.ServesAllProperties
	}
	if input.InsuranceExpiry != nil {
		v.InsuranceExpiry = input.InsuranceExpiry
	}
	if input.LicenseNumber != nil {
		v.LicenseNumber = *input.LicenseNumber
	}
	if input.LicenseExpiry != nil {
		v.LicenseExpiry = input.LicenseExpiry
	}
	if input.COILink != nil {
		v.COILink = *input.COILink
	}
	if input.TaxDocLink != nil {
		v.TaxDocLink = *input.TaxDocLink
	}
	if input.RateType != nil {
		if !input.RateType.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "rate_type", Message: fmt.Sprintf("unknown rate type %q", *input.RateType)})
		} else {
			v.RateType = input.RateType
		}
	}
	if input.RateAmount != nil {
		v.RateAmount = input.RateAmount
	}
	if input.PaymentTerms != nil {
		if !input.PaymentTerms.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "payment_terms", Message: fmt.Sprintf("unknown payment terms %q", *input.PaymentTerms)})
		} else {
			v.PaymentTerms = input.PaymentTerms
		}
	}
	if input.InternalNotes != nil {
		v.InternalNotes = *input.InternalNotes
	}
	if input.Active != nil {
		v.Active = *input.Active
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("update vendor %s: %w", id, verrs)
	}

	v.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, fmt.Errorf("update vendor %s: %w", id, err)
	}

	if input.PropertiesServedSet {
		propertiesServed, err := s.resolvePropertiesServed(ctx, ownerID, v.ServesAllProperties, input.PropertiesServed)
		if err != nil {
			return nil, fmt.Errorf("update vendor %s: %w", id, err)
		}
		if err := s.repo.SetPropertiesServed(ctx, id, propertiesServed); err != nil {
			return nil, fmt.Errorf("update vendor %s: %w", id, err)
		}
	}

	return v, nil
}

func (s *VendorService) DeleteVendor(ctx context.Context, id, ownerID uuid.UUID) error {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete vendor %s: %w", id, err)
	}
	if v.OwnerID != ownerID {
		return fmt.Errorf("delete vendor %s: %w", id, domain.ErrNotFound)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete vendor %s: %w", id, err)
	}
	return nil
}

func (s *VendorService) GetPropertiesServed(ctx context.Context, id, ownerID uuid.UUID) ([]uuid.UUID, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get properties served for vendor %s: %w", id, err)
	}
	if v.OwnerID != ownerID {
		return nil, fmt.Errorf("get properties served for vendor %s: %w", id, domain.ErrNotFound)
	}

	ids, err := s.repo.GetPropertiesServed(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get properties served for vendor %s: %w", id, err)
	}
	return ids, nil
}

func (s *VendorService) GetSpendSummary(ctx context.Context, id, ownerID uuid.UUID) (*domain.VendorSpendSummary, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get spend summary for vendor %s: %w", id, err)
	}
	if v.OwnerID != ownerID {
		return nil, fmt.Errorf("get spend summary for vendor %s: %w", id, domain.ErrNotFound)
	}

	summary, err := s.repo.GetSpendSummary(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get spend summary for vendor %s: %w", id, err)
	}
	return summary, nil
}

// resolvePropertiesServed validates every id in propertyIDs actually
// belongs to ownerID before it's ever written — the same IDOR-safe
// check every other cross-reference in this codebase does — and
// normalizes "serves all properties" to an empty list, since
// vendor_properties rows are meaningless (and ignored) in that case.
func (s *VendorService) resolvePropertiesServed(ctx context.Context, ownerID uuid.UUID, servesAll bool, propertyIDs []uuid.UUID) ([]uuid.UUID, error) {
	if servesAll || len(propertyIDs) == 0 {
		return nil, nil
	}
	for _, propertyID := range propertyIDs {
		p, err := s.propertyRepo.GetByID(ctx, propertyID)
		if err != nil {
			return nil, err
		}
		if p.OwnerID != ownerID {
			return nil, domain.ValidationErrors{{Field: "properties_served", Message: "includes a property you don't own"}}
		}
	}
	return propertyIDs, nil
}

func validateVendor(companyName string, categories []domain.WorkOrderCategory, rateType *domain.VendorRateType, paymentTerms *domain.VendorPaymentTerms) domain.ValidationErrors {
	var verrs domain.ValidationErrors
	if companyName == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "company_name", Message: "is required"})
	}
	if len(categories) == 0 {
		verrs = append(verrs, &domain.ValidationError{Field: "categories", Message: "select at least one category"})
	}
	for _, c := range categories {
		if !c.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "categories", Message: fmt.Sprintf("unknown category %q", c)})
		}
	}
	if rateType != nil && !rateType.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "rate_type", Message: fmt.Sprintf("unknown rate type %q", *rateType)})
	}
	if paymentTerms != nil && !paymentTerms.Valid() {
		verrs = append(verrs, &domain.ValidationError{Field: "payment_terms", Message: fmt.Sprintf("unknown payment terms %q", *paymentTerms)})
	}
	return verrs
}
