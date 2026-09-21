package service_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/service"
)

// fakeWorkOrderRepository is an in-memory stand-in for domain.WorkOrderRepository.
type fakeWorkOrderRepository struct {
	orders   map[uuid.UUID]*domain.WorkOrder
	activity map[uuid.UUID][]*domain.MaintenanceActivity
}

func newFakeWorkOrderRepository() *fakeWorkOrderRepository {
	return &fakeWorkOrderRepository{
		orders:   make(map[uuid.UUID]*domain.WorkOrder),
		activity: make(map[uuid.UUID][]*domain.MaintenanceActivity),
	}
}

func (f *fakeWorkOrderRepository) Create(_ context.Context, w *domain.WorkOrder) error {
	f.orders[w.ID] = w
	return nil
}

func (f *fakeWorkOrderRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	w, ok := f.orders[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return w, nil
}

func (f *fakeWorkOrderRepository) ListForOwner(_ context.Context, _ domain.WorkOrderListOptions) ([]*domain.WorkOrderWithProperty, int, error) {
	out := make([]*domain.WorkOrderWithProperty, 0, len(f.orders))
	for _, w := range f.orders {
		out = append(out, &domain.WorkOrderWithProperty{WorkOrder: *w})
	}
	return out, len(out), nil
}

func (f *fakeWorkOrderRepository) Update(_ context.Context, w *domain.WorkOrder) error {
	if _, ok := f.orders[w.ID]; !ok {
		return domain.ErrNotFound
	}
	f.orders[w.ID] = w
	return nil
}

func (f *fakeWorkOrderRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.orders[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.orders, id)
	return nil
}

func (f *fakeWorkOrderRepository) BulkUpdateStatus(_ context.Context, _ uuid.UUID, ids []uuid.UUID, status domain.WorkOrderStatus) (int, error) {
	n := 0
	for _, id := range ids {
		if w, ok := f.orders[id]; ok {
			w.Status = status
			n++
		}
	}
	return n, nil
}

func (f *fakeWorkOrderRepository) BulkReassign(_ context.Context, _ uuid.UUID, ids []uuid.UUID, assignedTo string) (int, error) {
	n := 0
	for _, id := range ids {
		if w, ok := f.orders[id]; ok {
			w.AssignedTo = assignedTo
			n++
		}
	}
	return n, nil
}

func (f *fakeWorkOrderRepository) GetSummary(_ context.Context, _ uuid.UUID) (*domain.WorkOrderSummary, error) {
	s := &domain.WorkOrderSummary{}
	now := time.Now().UTC()
	for _, w := range f.orders {
		if w.Status.IsOpen() {
			s.Open++
		}
		if w.IsOverdue(now) {
			s.Overdue++
		}
		if w.AssignedTo == "" && w.Status.IsOpen() {
			s.Unassigned++
		}
		if w.Priority == domain.WorkOrderPriorityEmergency && w.Status.IsOpen() {
			s.Emergency++
		}
		if w.Priority == domain.WorkOrderPriorityHigh && w.Status.IsOpen() {
			s.High++
		}
		if w.Priority == domain.WorkOrderPriorityMedium && w.Status.IsOpen() {
			s.Medium++
		}
		if w.Priority == domain.WorkOrderPriorityLow && w.Status.IsOpen() {
			s.Low++
		}
	}
	return s, nil
}

func (f *fakeWorkOrderRepository) ListActivity(_ context.Context, workOrderID uuid.UUID) ([]*domain.MaintenanceActivity, error) {
	return f.activity[workOrderID], nil
}

func (f *fakeWorkOrderRepository) AddActivity(_ context.Context, a *domain.MaintenanceActivity) error {
	f.activity[a.WorkOrderID] = append(f.activity[a.WorkOrderID], a)
	return nil
}

func (f *fakeWorkOrderRepository) ListRecentActivityForOwner(_ context.Context, _ uuid.UUID, limit, offset int) ([]*domain.MaintenanceActivityWithContext, error) {
	all := make([]*domain.MaintenanceActivityWithContext, 0)
	for workOrderID, entries := range f.activity {
		title := ""
		if w, ok := f.orders[workOrderID]; ok {
			title = w.Title
		}
		for _, a := range entries {
			all = append(all, &domain.MaintenanceActivityWithContext{MaintenanceActivity: *a, WorkOrderTitle: title})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })

	if offset >= len(all) {
		return []*domain.MaintenanceActivityWithContext{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

// fakeRecurringRuleRepository is an in-memory stand-in for domain.RecurringRuleRepository.
type fakeRecurringRuleRepository struct {
	rules map[uuid.UUID]*domain.RecurringRule
}

func newFakeRecurringRuleRepository() *fakeRecurringRuleRepository {
	return &fakeRecurringRuleRepository{rules: make(map[uuid.UUID]*domain.RecurringRule)}
}

func (f *fakeRecurringRuleRepository) Create(_ context.Context, r *domain.RecurringRule) error {
	f.rules[r.ID] = r
	return nil
}

func (f *fakeRecurringRuleRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.RecurringRule, error) {
	r, ok := f.rules[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return r, nil
}

func (f *fakeRecurringRuleRepository) ListForOwner(_ context.Context, _ uuid.UUID) ([]*domain.RecurringRuleWithProperty, error) {
	out := make([]*domain.RecurringRuleWithProperty, 0, len(f.rules))
	for _, r := range f.rules {
		out = append(out, &domain.RecurringRuleWithProperty{RecurringRule: *r})
	}
	return out, nil
}

func (f *fakeRecurringRuleRepository) Update(_ context.Context, r *domain.RecurringRule) error {
	if _, ok := f.rules[r.ID]; !ok {
		return domain.ErrNotFound
	}
	f.rules[r.ID] = r
	return nil
}

func (f *fakeRecurringRuleRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.rules[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.rules, id)
	return nil
}

// fakeVendorRepository is an in-memory stand-in for domain.VendorRepository.
// propertiesServed tracks the last value passed to Create/SetPropertiesServed
// per vendor, so VendorService tests can assert on it.
type fakeVendorRepository struct {
	vendors          map[uuid.UUID]*domain.Vendor
	propertiesServed map[uuid.UUID][]uuid.UUID
}

func newFakeVendorRepository() *fakeVendorRepository {
	return &fakeVendorRepository{
		vendors:          make(map[uuid.UUID]*domain.Vendor),
		propertiesServed: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (f *fakeVendorRepository) Create(_ context.Context, v *domain.Vendor, propertiesServed []uuid.UUID) error {
	f.vendors[v.ID] = v
	f.propertiesServed[v.ID] = propertiesServed
	return nil
}

func (f *fakeVendorRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Vendor, error) {
	v, ok := f.vendors[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return v, nil
}

func (f *fakeVendorRepository) GetByIDWithStats(_ context.Context, id uuid.UUID) (*domain.VendorWithStats, error) {
	v, ok := f.vendors[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.VendorWithStats{Vendor: *v}, nil
}

func (f *fakeVendorRepository) ListForOwner(_ context.Context, _ domain.VendorListOptions) ([]*domain.VendorWithStats, int, error) {
	out := make([]*domain.VendorWithStats, 0, len(f.vendors))
	for _, v := range f.vendors {
		out = append(out, &domain.VendorWithStats{Vendor: *v})
	}
	return out, len(out), nil
}

func (f *fakeVendorRepository) ListForCategory(_ context.Context, _ uuid.UUID, _ domain.WorkOrderCategory) ([]*domain.VendorWithStats, error) {
	return nil, nil
}

func (f *fakeVendorRepository) Update(_ context.Context, v *domain.Vendor) error {
	if _, ok := f.vendors[v.ID]; !ok {
		return domain.ErrNotFound
	}
	f.vendors[v.ID] = v
	return nil
}

func (f *fakeVendorRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.vendors[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.vendors, id)
	return nil
}

func (f *fakeVendorRepository) GetPropertiesServed(_ context.Context, vendorID uuid.UUID) ([]uuid.UUID, error) {
	return f.propertiesServed[vendorID], nil
}

func (f *fakeVendorRepository) SetPropertiesServed(_ context.Context, vendorID uuid.UUID, propertyIDs []uuid.UUID) error {
	f.propertiesServed[vendorID] = propertyIDs
	return nil
}

func (f *fakeVendorRepository) GetSpendSummary(_ context.Context, _ uuid.UUID) (*domain.VendorSpendSummary, error) {
	return &domain.VendorSpendSummary{}, nil
}

type workOrderTestFixture struct {
	svc           *service.WorkOrderService
	workOrderRepo *fakeWorkOrderRepository
	propertyRepo  *fakePropertyRepository
	unitRepo      *fakeUnitRepository
	vendorRepo    *fakeVendorRepository
	ownerID       uuid.UUID
	property      *domain.Property
	unit          *domain.Unit
}

func setupWorkOrderTest(t *testing.T) workOrderTestFixture {
	t.Helper()
	propertyRepo := newFakePropertyRepository()
	unitRepo := newFakeUnitRepository()
	workOrderRepo := newFakeWorkOrderRepository()
	vendorRepo := newFakeVendorRepository()
	svc := service.NewWorkOrderService(workOrderRepo, unitRepo, propertyRepo, vendorRepo, noopLogger())

	ownerID := uuid.New()
	property := &domain.Property{ID: uuid.New(), Name: "Willow Creek Apartments", Type: domain.PropertyTypeResidentialMultiUnit, AddressLine1: "123 Main St", OwnerID: ownerID}
	propertyRepo.properties[property.ID] = property

	unit := &domain.Unit{ID: uuid.New(), PropertyID: property.ID, UnitName: "1A", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusOccupied}
	unitRepo.units[unit.ID] = unit

	return workOrderTestFixture{
		svc: svc, workOrderRepo: workOrderRepo, propertyRepo: propertyRepo, unitRepo: unitRepo, vendorRepo: vendorRepo,
		ownerID: ownerID, property: property, unit: unit,
	}
}

func validCreateWorkOrderInput(propertyID uuid.UUID) domain.CreateWorkOrderInput {
	return domain.CreateWorkOrderInput{
		PropertyID: propertyID,
		Title:      "Leaking faucet",
		Category:   domain.WorkOrderCategoryPlumbing,
	}
}

func TestWorkOrderService_CreateWorkOrder(t *testing.T) {
	t.Run("valid input creates a work order with default status and priority", func(t *testing.T) {
		f := setupWorkOrderTest(t)

		got, err := f.svc.CreateWorkOrder(context.Background(), f.ownerID, validCreateWorkOrderInput(f.property.ID))
		if err != nil {
			t.Fatalf("CreateWorkOrder() unexpected error = %v", err)
		}
		if got.Status != domain.WorkOrderStatusNew {
			t.Errorf("CreateWorkOrder() status = %q, want %q", got.Status, domain.WorkOrderStatusNew)
		}
		if got.Priority != domain.WorkOrderPriorityMedium {
			t.Errorf("CreateWorkOrder() priority = %q, want %q", got.Priority, domain.WorkOrderPriorityMedium)
		}
	})

	t.Run("property belongs to a different owner reads as not found", func(t *testing.T) {
		f := setupWorkOrderTest(t)

		_, err := f.svc.CreateWorkOrder(context.Background(), uuid.New(), validCreateWorkOrderInput(f.property.ID))
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("CreateWorkOrder() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("unit belonging to a different property is rejected", func(t *testing.T) {
		f := setupWorkOrderTest(t)

		otherProperty := &domain.Property{ID: uuid.New(), Name: "Other Property", Type: domain.PropertyTypeResidentialMultiUnit, AddressLine1: "456 Side St", OwnerID: f.ownerID}
		otherUnit := &domain.Unit{ID: uuid.New(), PropertyID: otherProperty.ID, UnitName: "2B", Type: domain.UnitTypeOneBed, Status: domain.UnitStatusVacant}
		f.propertyRepo.properties[otherProperty.ID] = otherProperty
		f.unitRepo.units[otherUnit.ID] = otherUnit

		input := validCreateWorkOrderInput(f.property.ID)
		input.UnitID = &otherUnit.ID
		_, err := f.svc.CreateWorkOrder(context.Background(), f.ownerID, input)
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "unit_id") {
			t.Fatalf("CreateWorkOrder() error = %v, want a ValidationErrors failure for field %q", err, "unit_id")
		}
	})

	t.Run("emergency priority without a due date is rejected", func(t *testing.T) {
		f := setupWorkOrderTest(t)

		input := validCreateWorkOrderInput(f.property.ID)
		input.Priority = domain.WorkOrderPriorityEmergency
		_, err := f.svc.CreateWorkOrder(context.Background(), f.ownerID, input)
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "due_date") {
			t.Fatalf("CreateWorkOrder() error = %v, want a ValidationErrors failure for field %q", err, "due_date")
		}
	})

	t.Run("emergency priority with a due date beyond the SLA window is rejected", func(t *testing.T) {
		f := setupWorkOrderTest(t)

		tooLate := time.Now().UTC().Add(72 * time.Hour)
		input := validCreateWorkOrderInput(f.property.ID)
		input.Priority = domain.WorkOrderPriorityEmergency
		input.DueDate = &tooLate
		_, err := f.svc.CreateWorkOrder(context.Background(), f.ownerID, input)
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "due_date") {
			t.Fatalf("CreateWorkOrder() error = %v, want a ValidationErrors failure for field %q", err, "due_date")
		}
	})

	t.Run("emergency priority with a due date inside the SLA window succeeds", func(t *testing.T) {
		f := setupWorkOrderTest(t)

		soon := time.Now().UTC().Add(12 * time.Hour)
		input := validCreateWorkOrderInput(f.property.ID)
		input.Priority = domain.WorkOrderPriorityEmergency
		input.DueDate = &soon
		if _, err := f.svc.CreateWorkOrder(context.Background(), f.ownerID, input); err != nil {
			t.Fatalf("CreateWorkOrder() unexpected error = %v", err)
		}
	})

	t.Run("choosing a vendor auto-fills assigned_to and assigned_to_contact", func(t *testing.T) {
		f := setupWorkOrderTest(t)
		vendor := &domain.Vendor{ID: uuid.New(), OwnerID: f.ownerID, CompanyName: "Ace Plumbing", Phone: "555-0100"}
		f.vendorRepo.vendors[vendor.ID] = vendor

		input := validCreateWorkOrderInput(f.property.ID)
		input.VendorID = &vendor.ID
		got, err := f.svc.CreateWorkOrder(context.Background(), f.ownerID, input)
		if err != nil {
			t.Fatalf("CreateWorkOrder() unexpected error = %v", err)
		}
		if got.AssignedTo != vendor.CompanyName {
			t.Errorf("CreateWorkOrder() assigned_to = %q, want %q", got.AssignedTo, vendor.CompanyName)
		}
		if got.AssignedToContact != vendor.Phone {
			t.Errorf("CreateWorkOrder() assigned_to_contact = %q, want %q", got.AssignedToContact, vendor.Phone)
		}
	})

	t.Run("vendor belonging to a different owner is rejected", func(t *testing.T) {
		f := setupWorkOrderTest(t)
		vendor := &domain.Vendor{ID: uuid.New(), OwnerID: uuid.New(), CompanyName: "Someone Else's Vendor"}
		f.vendorRepo.vendors[vendor.ID] = vendor

		input := validCreateWorkOrderInput(f.property.ID)
		input.VendorID = &vendor.ID
		_, err := f.svc.CreateWorkOrder(context.Background(), f.ownerID, input)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("CreateWorkOrder() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}

func TestWorkOrderService_UpdateWorkOrder(t *testing.T) {
	f := setupWorkOrderTest(t)

	created, err := f.svc.CreateWorkOrder(context.Background(), f.ownerID, validCreateWorkOrderInput(f.property.ID))
	if err != nil {
		t.Fatalf("CreateWorkOrder() unexpected error = %v", err)
	}

	t.Run("status change logs activity and sets completed_at", func(t *testing.T) {
		completed := domain.WorkOrderStatusCompleted
		got, err := f.svc.UpdateWorkOrder(context.Background(), created.ID, f.ownerID, domain.UpdateWorkOrderInput{Status: &completed})
		if err != nil {
			t.Fatalf("UpdateWorkOrder() unexpected error = %v", err)
		}
		if got.Status != domain.WorkOrderStatusCompleted {
			t.Errorf("UpdateWorkOrder() status = %q, want %q", got.Status, domain.WorkOrderStatusCompleted)
		}
		if got.CompletedAt == nil {
			t.Error("UpdateWorkOrder() completed_at not set after marking completed")
		}

		activity := f.workOrderRepo.activity[created.ID]
		found := false
		for _, a := range activity {
			if a.Kind == domain.MaintenanceActivityKindStatusChange && a.NewValue != nil && *a.NewValue == string(domain.WorkOrderStatusCompleted) {
				found = true
			}
		}
		if !found {
			t.Error("UpdateWorkOrder() did not log a status_change activity entry for the completion")
		}
	})

	t.Run("belongs to a different owner", func(t *testing.T) {
		title := "Different title"
		_, err := f.svc.UpdateWorkOrder(context.Background(), created.ID, uuid.New(), domain.UpdateWorkOrderInput{Title: &title})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("UpdateWorkOrder() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("setting a vendor auto-fills assigned_to and assigned_to_contact", func(t *testing.T) {
		vendor := &domain.Vendor{ID: uuid.New(), OwnerID: f.ownerID, CompanyName: "Ace Plumbing", Phone: "555-0100"}
		f.vendorRepo.vendors[vendor.ID] = vendor

		got, err := f.svc.UpdateWorkOrder(context.Background(), created.ID, f.ownerID, domain.UpdateWorkOrderInput{VendorID: &vendor.ID, VendorIDSet: true})
		if err != nil {
			t.Fatalf("UpdateWorkOrder() unexpected error = %v", err)
		}
		if got.AssignedTo != vendor.CompanyName {
			t.Errorf("UpdateWorkOrder() assigned_to = %q, want %q", got.AssignedTo, vendor.CompanyName)
		}
		if got.AssignedToContact != vendor.Phone {
			t.Errorf("UpdateWorkOrder() assigned_to_contact = %q, want %q", got.AssignedToContact, vendor.Phone)
		}
	})

	t.Run("vendor belonging to a different owner is rejected", func(t *testing.T) {
		vendor := &domain.Vendor{ID: uuid.New(), OwnerID: uuid.New(), CompanyName: "Someone Else's Vendor"}
		f.vendorRepo.vendors[vendor.ID] = vendor

		_, err := f.svc.UpdateWorkOrder(context.Background(), created.ID, f.ownerID, domain.UpdateWorkOrderInput{VendorID: &vendor.ID, VendorIDSet: true})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("UpdateWorkOrder() error = %v, want %v", err, domain.ErrNotFound)
		}
	})

	t.Run("rating out of range is rejected", func(t *testing.T) {
		bad := 6
		_, err := f.svc.UpdateWorkOrder(context.Background(), created.ID, f.ownerID, domain.UpdateWorkOrderInput{Rating: &bad})
		var verrs domain.ValidationErrors
		if !errors.As(err, &verrs) || !hasField(verrs, "rating") {
			t.Fatalf("UpdateWorkOrder() error = %v, want a ValidationErrors failure for field %q", err, "rating")
		}
	})

	t.Run("valid rating is persisted", func(t *testing.T) {
		good := 5
		got, err := f.svc.UpdateWorkOrder(context.Background(), created.ID, f.ownerID, domain.UpdateWorkOrderInput{Rating: &good})
		if err != nil {
			t.Fatalf("UpdateWorkOrder() unexpected error = %v", err)
		}
		if got.Rating == nil || *got.Rating != good {
			t.Errorf("UpdateWorkOrder() rating = %v, want %d", got.Rating, good)
		}
	})
}

func TestWorkOrderService_DeleteWorkOrder(t *testing.T) {
	f := setupWorkOrderTest(t)

	created, err := f.svc.CreateWorkOrder(context.Background(), f.ownerID, validCreateWorkOrderInput(f.property.ID))
	if err != nil {
		t.Fatalf("CreateWorkOrder() unexpected error = %v", err)
	}

	t.Run("belongs to a different owner", func(t *testing.T) {
		if err := f.svc.DeleteWorkOrder(context.Background(), created.ID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("DeleteWorkOrder() error = %v, want %v", err, domain.ErrNotFound)
		}
		if _, ok := f.workOrderRepo.orders[created.ID]; !ok {
			t.Error("DeleteWorkOrder() removed a work order belonging to a different owner")
		}
	})

	if err := f.svc.DeleteWorkOrder(context.Background(), created.ID, f.ownerID); err != nil {
		t.Fatalf("DeleteWorkOrder() unexpected error = %v", err)
	}
}

func TestMaintenanceRuleService_GenerateNow(t *testing.T) {
	propertyRepo := newFakePropertyRepository()
	unitRepo := newFakeUnitRepository()
	workOrderRepo := newFakeWorkOrderRepository()
	ruleRepo := newFakeRecurringRuleRepository()
	svc := service.NewMaintenanceRuleService(ruleRepo, workOrderRepo, unitRepo, propertyRepo, noopLogger())

	ownerID := uuid.New()
	property := &domain.Property{ID: uuid.New(), Name: "Willow Creek Apartments", Type: domain.PropertyTypeResidentialMultiUnit, AddressLine1: "123 Main St", OwnerID: ownerID}
	propertyRepo.properties[property.ID] = property

	nextDue := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	rule, err := svc.CreateRule(context.Background(), ownerID, domain.CreateRecurringRuleInput{
		PropertyID:        property.ID,
		Title:             "HVAC filter change",
		Category:          domain.WorkOrderCategoryHVAC,
		FrequencyInterval: 3,
		FrequencyUnit:     domain.RecurringRuleFrequencyMonths,
		NextDueDate:       nextDue,
	})
	if err != nil {
		t.Fatalf("CreateRule() unexpected error = %v", err)
	}

	t.Run("generates a work order and advances next_due_date", func(t *testing.T) {
		wo, err := svc.GenerateNow(context.Background(), rule.ID, ownerID)
		if err != nil {
			t.Fatalf("GenerateNow() unexpected error = %v", err)
		}
		if wo.PropertyID != property.ID {
			t.Errorf("GenerateNow() property_id = %s, want %s", wo.PropertyID, property.ID)
		}
		if wo.RecurringRuleID == nil || *wo.RecurringRuleID != rule.ID {
			t.Error("GenerateNow() generated work order isn't linked back to the rule")
		}
		if _, ok := workOrderRepo.orders[wo.ID]; !ok {
			t.Error("GenerateNow() work order was not persisted")
		}

		updatedRule, err := ruleRepo.GetByID(context.Background(), rule.ID)
		if err != nil {
			t.Fatalf("GetByID() unexpected error = %v", err)
		}
		wantNext := nextDue.AddDate(0, 3, 0)
		if !updatedRule.NextDueDate.Equal(wantNext) {
			t.Errorf("GenerateNow() rule next_due_date = %v, want %v", updatedRule.NextDueDate, wantNext)
		}
	})

	t.Run("belongs to a different owner", func(t *testing.T) {
		if _, err := svc.GenerateNow(context.Background(), rule.ID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GenerateNow() error = %v, want %v", err, domain.ErrNotFound)
		}
	})
}
