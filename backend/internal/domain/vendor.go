package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type VendorRateType string

const (
	VendorRateTypeHourly VendorRateType = "hourly"
	VendorRateTypeFlat   VendorRateType = "flat"
)

func (t VendorRateType) Valid() bool {
	switch t {
	case VendorRateTypeHourly, VendorRateTypeFlat:
		return true
	default:
		return false
	}
}

type VendorPaymentTerms string

const (
	VendorPaymentTermsNet15 VendorPaymentTerms = "net_15"
	VendorPaymentTermsNet30 VendorPaymentTerms = "net_30"
	VendorPaymentTermsNet45 VendorPaymentTerms = "net_45"
)

func (t VendorPaymentTerms) Valid() bool {
	switch t {
	case VendorPaymentTermsNet15, VendorPaymentTermsNet30, VendorPaymentTermsNet45:
		return true
	default:
		return false
	}
}

// InsuranceStatus is computed from InsuranceExpiry, not stored — same
// rationale as LeaseDisplayStatus: a stored "expiring_soon" would go
// stale the day nobody happens to re-save the row.
type InsuranceStatus string

const (
	InsuranceStatusValid        InsuranceStatus = "valid"
	InsuranceStatusExpiringSoon InsuranceStatus = "expiring_soon"
	InsuranceStatusExpired      InsuranceStatus = "expired"
	InsuranceStatusUnknown      InsuranceStatus = "unknown"
)

// InsuranceExpiringSoonDays is the window (from today) within which a
// vendor's insurance_expiry makes it "expiring soon" rather than plain
// "valid" — mirrors LeaseExpiringSoonDays.
const InsuranceExpiringSoonDays = 30

// Vendor is the core business entity — a contractor/service provider
// that can be assigned to work orders. Unlike Unit/Lease, it belongs
// directly to OwnerID (like Property), since a vendor typically serves
// many properties rather than living inside one.
//
// Categories reuses WorkOrderCategory's values so a vendor's trades
// line up 1:1 with the work order categories they can be matched
// against (see UnitService... — VendorService.ListVendorsForCategory).
type Vendor struct {
	ID                  uuid.UUID
	OwnerID             uuid.UUID
	CompanyName         string
	Categories          []WorkOrderCategory
	ContactPerson       string
	Phone               string
	Email               string
	Address             string
	ServesAllProperties bool
	InsuranceExpiry     *time.Time
	LicenseNumber       string
	LicenseExpiry       *time.Time
	COIAttachmentID     *uuid.UUID
	TaxDocAttachmentID  *uuid.UUID
	RateType            *VendorRateType
	RateAmount          *float64
	PaymentTerms        *VendorPaymentTerms
	InternalNotes       string
	Active              bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// InsuranceStatus computes the richer status the list/profile actually
// shows from InsuranceExpiry relative to today.
func (v *Vendor) InsuranceStatus(now time.Time) InsuranceStatus {
	if v.InsuranceExpiry == nil {
		return InsuranceStatusUnknown
	}
	today := truncateToDate(now)
	expiry := truncateToDate(*v.InsuranceExpiry)
	if expiry.Before(today) {
		return InsuranceStatusExpired
	}
	if int(expiry.Sub(today).Hours()/24) <= InsuranceExpiringSoonDays {
		return InsuranceStatusExpiringSoon
	}
	return InsuranceStatusValid
}

type CreateVendorInput struct {
	OwnerID             uuid.UUID
	CompanyName         string
	Categories          []WorkOrderCategory
	ContactPerson       string
	Phone               string
	Email               string
	Address             string
	ServesAllProperties bool
	PropertiesServed    []uuid.UUID // only meaningful when ServesAllProperties is false
	InsuranceExpiry     *time.Time
	LicenseNumber       string
	LicenseExpiry       *time.Time
	COIAttachmentID     *uuid.UUID
	TaxDocAttachmentID  *uuid.UUID
	RateType            *VendorRateType
	RateAmount          *float64
	PaymentTerms        *VendorPaymentTerms
	InternalNotes       string
}

// UpdateVendorInput fields are all optional (nil = leave unchanged) —
// mirrors UpdateLeaseInput/UpdateWorkOrderInput. PropertiesServed is
// replaced wholesale when non-nil (an empty, non-nil slice clears it),
// the same nil-vs-empty distinction CoResidents/Amenities already use
// elsewhere in this codebase.
type UpdateVendorInput struct {
	CompanyName           *string
	Categories            []WorkOrderCategory
	ContactPerson         *string
	Phone                 *string
	Email                 *string
	Address               *string
	ServesAllProperties   *bool
	PropertiesServed      []uuid.UUID
	PropertiesServedSet   bool
	InsuranceExpiry       *time.Time
	LicenseNumber         *string
	LicenseExpiry         *time.Time
	COIAttachmentID       *uuid.UUID
	COIAttachmentIDSet    bool // COIAttachmentID is itself nullable, so "remove the file" needs its own flag
	TaxDocAttachmentID    *uuid.UUID
	TaxDocAttachmentIDSet bool // same as COIAttachmentIDSet, for the tax document
	RateType              *VendorRateType
	RateAmount            *float64
	PaymentTerms          *VendorPaymentTerms
	InternalNotes         *string
	Active                *bool
}

type VendorSortKey string

const (
	VendorSortName   VendorSortKey = "name"
	VendorSortRating VendorSortKey = "rating"
)

func (k VendorSortKey) Valid() bool {
	switch k {
	case VendorSortName, VendorSortRating:
		return true
	default:
		return false
	}
}

// VendorListFilter narrows ListForOwner results. Zero values (empty
// Search, nil pointers) mean "don't filter on this".
type VendorListFilter struct {
	Search          string
	Category        *WorkOrderCategory
	Active          *bool
	InsuranceStatus *InsuranceStatus
}

type VendorListOptions struct {
	OwnerID  uuid.UUID
	Filter   VendorListFilter
	Sort     VendorSortKey
	SortDesc bool
	Limit    int
	Offset   int
}

// VendorWithStats decorates a Vendor with the computed numbers its list
// row and profile header show: open work orders, average rating (nil
// until at least one completed work order has been rated), and how
// many properties it's scoped to (meaningful only when
// !ServesAllProperties).
type VendorWithStats struct {
	Vendor
	OpenWorkOrders        int
	AverageRating         *float64
	PropertiesServedCount int
}

// VendorSpendSummary is the Financials tab's real, computed-from-actuals
// number — see CreateWorkOrderInput.ActualCost. There is no
// payments/invoicing ledger in this codebase, so this is the honest
// substitute: what work orders assigned to this vendor actually cost,
// not a fabricated invoice/paid-status history.
type VendorSpendSummary struct {
	ThisMonth  float64
	YearToDate float64
}

// VendorRepository is the port implemented by internal/repository/postgres.
type VendorRepository interface {
	Create(ctx context.Context, v *Vendor, propertiesServed []uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Vendor, error)
	// GetByIDWithStats is GetByID decorated with the same computed
	// numbers ListForOwner's rows carry (open work orders, average
	// rating, properties served) — used by the vendor profile page.
	GetByIDWithStats(ctx context.Context, id uuid.UUID) (*VendorWithStats, error)
	ListForOwner(ctx context.Context, opts VendorListOptions) ([]*VendorWithStats, int, error)
	// ListForCategory backs the Vendor Selector: active vendors whose
	// Categories include category, for owner.
	ListForCategory(ctx context.Context, ownerID uuid.UUID, category WorkOrderCategory) ([]*VendorWithStats, error)
	Update(ctx context.Context, v *Vendor) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetPropertiesServed(ctx context.Context, vendorID uuid.UUID) ([]uuid.UUID, error)
	SetPropertiesServed(ctx context.Context, vendorID uuid.UUID, propertyIDs []uuid.UUID) error
	GetSpendSummary(ctx context.Context, vendorID uuid.UUID) (*VendorSpendSummary, error)
}

// VendorService is the port implemented by internal/service and
// consumed by the HTTP transport layer. Every method resolves ownership
// via the vendor's own OwnerID — one hop, like PropertyService, since
// vendors belong directly to an owner rather than through a parent.
type VendorService interface {
	CreateVendor(ctx context.Context, ownerID uuid.UUID, input CreateVendorInput) (*Vendor, error)
	GetVendor(ctx context.Context, id, ownerID uuid.UUID) (*VendorWithStats, error)
	ListVendorsForOwner(ctx context.Context, ownerID uuid.UUID, opts VendorListOptions) ([]*VendorWithStats, int, error)
	ListVendorsForCategory(ctx context.Context, ownerID uuid.UUID, category WorkOrderCategory) ([]*VendorWithStats, error)
	UpdateVendor(ctx context.Context, id, ownerID uuid.UUID, input UpdateVendorInput) (*Vendor, error)
	DeleteVendor(ctx context.Context, id, ownerID uuid.UUID) error
	GetSpendSummary(ctx context.Context, id, ownerID uuid.UUID) (*VendorSpendSummary, error)
	// GetPropertiesServed backs the vendor form's Service Scope section
	// when editing a vendor that doesn't serve all properties — it needs
	// the current selection to prefill the multi-select.
	GetPropertiesServed(ctx context.Context, id, ownerID uuid.UUID) ([]uuid.UUID, error)
}
