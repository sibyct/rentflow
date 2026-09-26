// Command seed-demo resets every operational table (properties, units,
// leases, vendors, work orders, and the whole accounting module) and
// repopulates them with a small, coherent demo portfolio, so every
// screen in the app — Rent Roll, Expenses, Charges, Bank Accounts,
// Security Deposits, Owner Statements, the dashboards — has real,
// varied data to show. It does NOT touch the users table: run
// `go run ./cmd/seed` first if the demo account doesn't exist yet.
//
// Local dev only. Safe to re-run: it truncates before it inserts.
//
//	go run ./cmd/seed-demo
//	SEED_ADMIN_EMAIL=owner@rentflow.local go run ./cmd/seed-demo
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"propertymanagement/internal/config"
	"propertymanagement/internal/domain"
	"propertymanagement/internal/infra/pdf"
	"propertymanagement/internal/repository/postgres"
	"propertymanagement/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// today anchors every relative date (N days ago / from now) in the demo
// data to the moment the seed runs, so "3 days overdue" etc. stays true
// however many days after the fact this is actually run.
var today = time.Now().UTC()

func daysAgo(n int) time.Time  { return dateOnly(today.AddDate(0, 0, -n)) }
func daysFrom(n int) time.Time { return dateOnly(today.AddDate(0, 0, n)) }
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func ym(monthsAgo int) time.Time {
	return domain.FirstOfMonth(today.AddDate(0, -monthsAgo, 0))
}
func ptr[T any](v T) *T           { return &v }
func cents(dollars float64) int64 { return domain.DollarsToCents(dollars) }

const seedEmail = "owner@rentflow.local"

func run() error {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	email := getEnv("SEED_ADMIN_EMAIL", seedEmail)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConn)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	user, err := userRepo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("looking up %s (run `go run ./cmd/seed` first): %w", email, err)
	}
	ownerID := user.ID

	fmt.Println("resetting operational and accounting tables...")
	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE properties, vendors, property_owners, bank_accounts, attachments,
			accounting_settings, accounting_audit_log, worker_runs, email_outbox, maintenance_recurring_rules
		RESTART IDENTITY CASCADE`); err != nil {
		return fmt.Errorf("truncating tables: %w", err)
	}

	// Wiring mirrors cmd/api/main.go, minus the HTTP layer and Redis
	// (PropertyService works with a nil domain.Cache — see cacheProperty).
	propertyRepo := postgres.NewPropertyRepository(pool)
	unitRepo := postgres.NewUnitRepository(pool)
	leaseRepo := postgres.NewLeaseRepository(pool)
	workOrderRepo := postgres.NewWorkOrderRepository(pool)
	vendorRepo := postgres.NewVendorRepository(pool)
	ledgerRepo := postgres.NewLedgerRepository(pool)
	bankRepo := postgres.NewBankAccountRepository(pool)
	depositRepo := postgres.NewDepositRepository(pool)
	ownerRepo := postgres.NewPropertyOwnerRepository(pool)
	statementRepo := postgres.NewOwnerStatementRepository(pool)
	outboxRepo := postgres.NewEmailOutboxRepository(pool)
	attachmentRepo := postgres.NewAttachmentRepository(pool)
	unitDocumentRepo := postgres.NewUnitDocumentRepository(pool)

	var cache domain.Cache
	propertyService := service.NewPropertyService(propertyRepo, unitRepo, cache, log)
	unitService := service.NewUnitService(unitRepo, propertyRepo, unitDocumentRepo, attachmentRepo, log)
	leaseService := service.NewLeaseService(leaseRepo, unitRepo, propertyRepo, log)
	workOrderService := service.NewWorkOrderService(workOrderRepo, unitRepo, propertyRepo, vendorRepo, attachmentRepo, log)
	vendorService := service.NewVendorService(vendorRepo, propertyRepo, attachmentRepo, log)
	ledgerService := service.NewLedgerService(ledgerRepo, leaseRepo, unitRepo, propertyRepo, log)
	rentRollService := service.NewRentRollService(ledgerRepo, log)
	expenseService := service.NewExpenseService(ledgerRepo, propertyRepo, unitRepo, vendorRepo, log)
	chargeService := service.NewChargeService(ledgerRepo, unitRepo, propertyRepo, log)
	bankAccountService := service.NewBankAccountService(bankRepo, propertyRepo, log)
	depositService := service.NewDepositService(depositRepo, ledgerRepo, log)
	propertyOwnerService := service.NewPropertyOwnerService(ownerRepo, propertyRepo, log)
	statementService := service.NewOwnerStatementService(statementRepo, ownerRepo, propertyRepo, outboxRepo, pdf.NewStatementRenderer(), cfg.MailEnabled(), cfg.PublicAppURL, log)
	workOrderService.SetExpenseSyncer(expenseService)
	leaseService.SetDepositEnsurer(depositService)

	d := &demo{
		ctx: ctx, ownerID: ownerID, log: log,
		properties: propertyService, units: unitService, leases: leaseService,
		workOrders: workOrderService, vendors: vendorService, ledger: ledgerService,
		ledgerRepo: ledgerRepo, rentRoll: rentRollService, expenses: expenseService,
		charges: chargeService, bankAccounts: bankAccountService, deposits: depositService,
		propertyOwners: propertyOwnerService, statements: statementService,
	}
	if err := d.build(); err != nil {
		return err
	}

	fmt.Println("\ndemo data seeded — log in as", email, "and explore.")
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// demo holds every service the seed needs and the one ownerID everything
// belongs to (this app has exactly one account per owner — see
// domain/user.go).
type demo struct {
	ctx     context.Context
	ownerID uuid.UUID
	log     *slog.Logger

	properties *service.PropertyService
	units      *service.UnitService
	leases     *service.LeaseService
	workOrders *service.WorkOrderService
	vendors    *service.VendorService

	ledger       *service.LedgerService
	ledgerRepo   domain.LedgerRepository
	rentRoll     *service.RentRollService
	expenses     *service.ExpenseService
	charges      *service.ChargeService
	bankAccounts *service.BankAccountService
	deposits     *service.DepositService

	propertyOwners *service.PropertyOwnerService
	statements     *service.OwnerStatementService
}

func (d *demo) must(label string, err error) {
	if err != nil {
		fmt.Printf("FAILED %s: %v\n", label, err)
		os.Exit(1)
	}
	fmt.Println("  ✓", label)
}

func (d *demo) build() error {
	fmt.Println("creating property owners...")
	jane, err := d.propertyOwners.Create(d.ctx, d.ownerID, domain.PropertyOwnerInput{
		Name: "Jane Sullivan", Email: "jane.sullivan@example.com", Phone: "512-555-0142", ManagementFeeBps: 800,
	})
	d.must("owner: Jane Sullivan (8% fee)", err)
	marcus, err := d.propertyOwners.Create(d.ctx, d.ownerID, domain.PropertyOwnerInput{
		Name: "Marcus Chen", Email: "marcus.chen@example.com", Phone: "737-555-0198", ManagementFeeBps: 1000,
	})
	d.must("owner: Marcus Chen (10% fee)", err)

	fmt.Println("creating properties...")
	maple, err := d.properties.CreateProperty(d.ctx, domain.CreatePropertyInput{
		Name: "Maple Street Duplex", Type: domain.PropertyTypeResidentialMultiUnit,
		AddressLine1: "214 Maple Street", City: "Austin", StateProvince: "TX", PostalCode: "78704",
		Units: 2, Ownership: ptr(domain.PropertyOwnershipManaged), OwnerName: "Jane Sullivan",
		YearBuilt: ptr(1978), Status: domain.PropertyStatusActive, OwnerID: d.ownerID,
		OnboardDate: ptr(daysAgo(320)),
	})
	d.must("property: Maple Street Duplex", err)

	riverside, err := d.properties.CreateProperty(d.ctx, domain.CreatePropertyInput{
		Name: "Riverside Bungalow", Type: domain.PropertyTypeResidentialSingleUnit,
		AddressLine1: "48 Riverside Court", City: "Austin", StateProvince: "TX", PostalCode: "78702",
		Units: 1, Ownership: ptr(domain.PropertyOwnershipOwned), YearBuilt: ptr(2004),
		Status: domain.PropertyStatusActive, OwnerID: d.ownerID, OnboardDate: ptr(daysAgo(480)),
	})
	d.must("property: Riverside Bungalow", err)

	oakview, err := d.properties.CreateProperty(d.ctx, domain.CreatePropertyInput{
		Name: "Oakview Apartments", Type: domain.PropertyTypeResidentialMultiUnit,
		AddressLine1: "900 Oakview Lane", City: "Round Rock", StateProvince: "TX", PostalCode: "78664",
		Units: 3, Ownership: ptr(domain.PropertyOwnershipManaged), OwnerName: "Marcus Chen",
		YearBuilt: ptr(1995), Status: domain.PropertyStatusActive, OwnerID: d.ownerID,
		Amenities: []string{"Pool", "Storage"}, OnboardDate: ptr(daysAgo(600)),
	})
	d.must("property: Oakview Apartments", err)

	// Link the properties to their real-world owners now that both exist.
	_, err = d.propertyOwners.Update(d.ctx, d.ownerID, jane.ID, domain.PropertyOwnerInput{
		Name: jane.Name, Email: jane.Email, Phone: jane.Phone, ManagementFeeBps: jane.ManagementFeeBps,
		AutoStatements: false, PropertyIDs: []uuid.UUID{maple.ID},
	})
	d.must("link Maple Street Duplex to Jane Sullivan", err)
	_, err = d.propertyOwners.Update(d.ctx, d.ownerID, marcus.ID, domain.PropertyOwnerInput{
		Name: marcus.Name, Email: marcus.Email, Phone: marcus.Phone, ManagementFeeBps: marcus.ManagementFeeBps,
		AutoStatements: false, PropertyIDs: []uuid.UUID{oakview.ID},
	})
	d.must("link Oakview Apartments to Marcus Chen", err)

	fmt.Println("creating units...")
	mapleA, err := d.units.CreateUnit(d.ctx, d.ownerID, domain.CreateUnitInput{
		PropertyID: maple.ID, UnitName: "Unit A", Type: domain.UnitTypeTwoBed, Bedrooms: ptr(2), Bathrooms: ptr(1.0),
		Sqft: ptr(950), Status: domain.UnitStatusOccupied, MarketRent: ptr(1900.0), CurrentRent: ptr(1850.0), RentDueDay: ptr(1),
	}, domain.AllPropertyAccess())
	d.must("unit: Maple St / Unit A", err)
	mapleB, err := d.units.CreateUnit(d.ctx, d.ownerID, domain.CreateUnitInput{
		PropertyID: maple.ID, UnitName: "Unit B", Type: domain.UnitTypeTwoBed, Bedrooms: ptr(2), Bathrooms: ptr(1.0),
		Sqft: ptr(975), Status: domain.UnitStatusOccupied, MarketRent: ptr(1950.0), CurrentRent: ptr(1900.0), RentDueDay: ptr(1),
	}, domain.AllPropertyAccess())
	d.must("unit: Maple St / Unit B", err)

	riversideMain, err := d.units.CreateUnit(d.ctx, d.ownerID, domain.CreateUnitInput{
		PropertyID: riverside.ID, UnitName: "Main House", Type: domain.UnitTypeThreeBedPlus, Bedrooms: ptr(3), Bathrooms: ptr(2.0),
		Sqft: ptr(1650), Status: domain.UnitStatusOccupied, MarketRent: ptr(2500.0), CurrentRent: ptr(2400.0), RentDueDay: ptr(5),
	}, domain.AllPropertyAccess())
	d.must("unit: Riverside Bungalow / Main House", err)

	oak101, err := d.units.CreateUnit(d.ctx, d.ownerID, domain.CreateUnitInput{
		PropertyID: oakview.ID, UnitName: "Unit 101", Type: domain.UnitTypeOneBed, Bedrooms: ptr(1), Bathrooms: ptr(1.0),
		Sqft: ptr(700), Status: domain.UnitStatusOccupied, MarketRent: ptr(1250.0), CurrentRent: ptr(1200.0), RentDueDay: ptr(1),
	}, domain.AllPropertyAccess())
	d.must("unit: Oakview / Unit 101", err)
	oak102, err := d.units.CreateUnit(d.ctx, d.ownerID, domain.CreateUnitInput{
		PropertyID: oakview.ID, UnitName: "Unit 102", Type: domain.UnitTypeOneBed, Bedrooms: ptr(1), Bathrooms: ptr(1.0),
		Sqft: ptr(700), Status: domain.UnitStatusVacant, MarketRent: ptr(1250.0),
	}, domain.AllPropertyAccess())
	d.must("unit: Oakview / Unit 102 (vacant)", err)
	oak103, err := d.units.CreateUnit(d.ctx, d.ownerID, domain.CreateUnitInput{
		PropertyID: oakview.ID, UnitName: "Unit 103", Type: domain.UnitTypeTwoBed, Bedrooms: ptr(2), Bathrooms: ptr(2.0),
		Sqft: ptr(1050), Status: domain.UnitStatusOccupied, MarketRent: ptr(2150.0), CurrentRent: ptr(2100.0), RentDueDay: ptr(1),
	}, domain.AllPropertyAccess())
	d.must("unit: Oakview / Unit 103", err)
	_ = oak102

	fmt.Println("creating vendors...")
	plumbing, err := d.vendors.CreateVendor(d.ctx, d.ownerID, domain.CreateVendorInput{
		OwnerID: d.ownerID, CompanyName: "Pro Plumbing Co", Categories: []domain.WorkOrderCategory{domain.WorkOrderCategoryPlumbing},
		ContactPerson: "Hector Reyes", Phone: "512-555-0110", Email: "dispatch@proplumbingco.example",
		ServesAllProperties: true, InsuranceExpiry: ptr(daysFrom(200)), LicenseNumber: "TX-PL-88213", LicenseExpiry: ptr(daysFrom(400)),
		RateType: ptr(domain.VendorRateTypeHourly), RateAmount: ptr(95.0), PaymentTerms: ptr(domain.VendorPaymentTermsNet30),
	})
	d.must("vendor: Pro Plumbing Co", err)
	electric, err := d.vendors.CreateVendor(d.ctx, d.ownerID, domain.CreateVendorInput{
		OwnerID: d.ownerID, CompanyName: "Bright Spark Electric", Categories: []domain.WorkOrderCategory{domain.WorkOrderCategoryElectrical},
		ContactPerson: "Nadia Farouk", Phone: "512-555-0166", Email: "office@brightsparkelectric.example",
		ServesAllProperties: true, InsuranceExpiry: ptr(daysFrom(20)), LicenseNumber: "TX-EL-44190",
		RateType: ptr(domain.VendorRateTypeFlat), RateAmount: ptr(350.0), PaymentTerms: ptr(domain.VendorPaymentTermsNet15),
	})
	d.must("vendor: Bright Spark Electric (insurance expiring soon)", err)
	_, err = d.vendors.CreateVendor(d.ctx, d.ownerID, domain.CreateVendorInput{
		OwnerID: d.ownerID, CompanyName: "GreenScape Landscaping", Categories: []domain.WorkOrderCategory{domain.WorkOrderCategoryGeneral},
		ContactPerson: "Owen Brooks", Phone: "512-555-0177", Email: "owen@greenscape.example",
		ServesAllProperties: true, InsuranceExpiry: ptr(daysAgo(10)), PaymentTerms: ptr(domain.VendorPaymentTermsNet30),
	})
	d.must("vendor: GreenScape Landscaping (insurance expired)", err)

	fmt.Println("creating leases (security deposits are tracked automatically)...")
	leaseA, err := d.leases.CreateLease(d.ctx, d.ownerID, domain.CreateLeaseInput{
		UnitID: mapleA.ID, Type: domain.LeaseTypeMonthToMonth, StartDate: daysAgo(320), MonthlyRent: 1850,
		SecurityDeposit: ptr(1850.0), RentDueDay: ptr(1), PrimaryResidentName: "Sarah Chen", EmergencyContact: "Mia Chen · 512-555-0301",
	}, domain.AllPropertyAccess())
	d.must("lease: Maple St / Unit A — Sarah Chen", err)
	leaseB, err := d.leases.CreateLease(d.ctx, d.ownerID, domain.CreateLeaseInput{
		UnitID: mapleB.ID, Type: domain.LeaseTypeFixed, StartDate: daysAgo(230), EndDate: ptr(daysFrom(120)), MonthlyRent: 1900,
		SecurityDeposit: ptr(1900.0), RentDueDay: ptr(1), LateFeeAmount: ptr(75.0), LateFeeGraceDays: ptr(3),
		PrimaryResidentName: "David Okafor", EmergencyContact: "Grace Okafor · 512-555-0322",
	}, domain.AllPropertyAccess())
	d.must("lease: Maple St / Unit B — David Okafor (flat $75 late fee override)", err)
	leaseRiverside, err := d.leases.CreateLease(d.ctx, d.ownerID, domain.CreateLeaseInput{
		UnitID: riversideMain.ID, Type: domain.LeaseTypeFixed, StartDate: daysAgo(480), EndDate: ptr(daysFrom(23)), MonthlyRent: 2400,
		SecurityDeposit: ptr(2400.0), RentDueDay: ptr(5), PrimaryResidentName: "The Martinez Family",
		EmergencyContact: "Elena Martinez · 512-555-0455", Notes: "Lease expiring soon — renewal conversation pending.",
	}, domain.AllPropertyAccess())
	d.must("lease: Riverside Bungalow — The Martinez Family (expiring soon)", err)
	lease101, err := d.leases.CreateLease(d.ctx, d.ownerID, domain.CreateLeaseInput{
		UnitID: oak101.ID, Type: domain.LeaseTypeMonthToMonth, StartDate: daysAgo(250), MonthlyRent: 1200,
		SecurityDeposit: ptr(1200.0), RentDueDay: ptr(1), PrimaryResidentName: "Priya Patel",
	}, domain.AllPropertyAccess())
	d.must("lease: Oakview / Unit 101 — Priya Patel", err)
	lease103, err := d.leases.CreateLease(d.ctx, d.ownerID, domain.CreateLeaseInput{
		UnitID: oak103.ID, Type: domain.LeaseTypeFixed, StartDate: daysAgo(400), EndDate: ptr(daysFrom(330)), MonthlyRent: 2100,
		SecurityDeposit: ptr(2100.0), RentDueDay: ptr(1), PrimaryResidentName: "Tom & Lisa Nguyen",
		CoResidents: []string{"Lisa Nguyen"},
	}, domain.AllPropertyAccess())
	d.must("lease: Oakview / Unit 103 — Tom & Lisa Nguyen", err)

	fmt.Println("setting the account's late-fee rule...")
	_, err = d.ledger.UpdateSettings(d.ctx, d.ownerID, domain.UpdateAccountingSettingsInput{
		LateFeeKind: domain.LateFeeKindPercent, LateFeeValue: 500, GraceDays: 5, DefaultRentDueDay: 1,
	})
	d.must("accounting settings: 5% late fee, 5-day grace", err)

	fmt.Println("creating bank accounts...")
	operating, err := d.bankAccounts.Create(d.ctx, d.ownerID, domain.CreateBankAccountInput{
		Nickname: "Operating — Chase", BankName: "Chase", Type: domain.BankAccountTypeOperating,
		BalanceCents: cents(15000), BalanceAsOf: ptr(today), Last4: "4521",
	})
	d.must("bank account: Operating — Chase", err)
	_, err = d.bankAccounts.Create(d.ctx, d.ownerID, domain.CreateBankAccountInput{
		Nickname: "Security Deposit Trust", BankName: "Chase", Type: domain.BankAccountTypeDepositTrust,
		BalanceCents: cents(9450), BalanceAsOf: ptr(today), Last4: "8890",
	})
	d.must("bank account: Security Deposit Trust", err)

	fmt.Println("generating rent roll for the last 4 months and recording payments...")
	for i := 3; i >= 0; i-- {
		_, err := d.rentRoll.GenerateForPeriod(d.ctx, d.ownerID, ym(i))
		d.must(fmt.Sprintf("generate rent for %s", ym(i).Format("Jan 2006")), err)
	}
	// Maple A: current month unpaid (and past its grace period) — shows "late".
	d.payLease(leaseA.ID, ym(3), 1850, "ach", "RENT-3821")
	d.payLease(leaseA.ID, ym(2), 1850, "ach", "RENT-3902")
	d.payLease(leaseA.ID, ym(1), 1850, "ach", "RENT-3977")
	// Maple B: paid every month, always on time.
	for i := 3; i >= 0; i-- {
		d.payLease(leaseB.ID, ym(i), 1900, "ach", fmt.Sprintf("RENT-B-%d", i))
	}
	// Riverside: two clean months, then a partial and an unpaid month.
	d.payLease(leaseRiverside.ID, ym(3), 2400, "check", "CHK-1042")
	d.payLease(leaseRiverside.ID, ym(2), 2400, "check", "CHK-1077")
	d.payPeriod(leaseRiverside.ID, ym(1), 1200, "check", "CHK-1098") // partial
	// Oakview 101: model tenant.
	for i := 3; i >= 0; i-- {
		d.payLease(lease101.ID, ym(i), 1200, "ach", fmt.Sprintf("RENT-101-%d", i))
	}
	// Oakview 103: skips a month in the middle (arrears), catches back up.
	d.payPeriod(lease103.ID, ym(3), 2100, "ach", "RENT-103-3")
	d.payPeriod(lease103.ID, ym(1), 2100, "ach", "RENT-103-1")

	fmt.Println("creating one-off charges...")
	_, err = d.charges.Create(d.ctx, d.ownerID, domain.CreateChargeInput{
		UnitID: mapleA.ID, ChargeType: domain.ChargeTypeUtilityRebill, AmountCents: cents(45.00),
		Description: "Water bill rebill — August", DueOn: daysFrom(5),
	})
	d.must("charge: Maple St A — utility rebill (unpaid)", err)
	chargeDamage, err := d.charges.Create(d.ctx, d.ownerID, domain.CreateChargeInput{
		UnitID: oak103.ID, ChargeType: domain.ChargeTypeDamage, AmountCents: cents(150.00),
		Description: "Hallway wall scuff repair", DueOn: daysAgo(10),
	})
	d.must("charge: Oakview 103 — damage (past due, shows as late)", err)
	_ = chargeDamage
	chargeAmenity, err := d.charges.Create(d.ctx, d.ownerID, domain.CreateChargeInput{
		UnitID: riversideMain.ID, ChargeType: domain.ChargeTypeAmenity, AmountCents: cents(75.00),
		Description: "Storage shed key deposit", DueOn: today,
	})
	d.must("charge: Riverside — amenity fee", err)
	_, err = d.ledger.RecordPayment(d.ctx, d.ownerID, chargeAmenity.ID, domain.RecordPaymentInput{AmountCents: cents(75.00), PaidOn: today, Method: "cash"})
	d.must("  → paid in full", err)

	fmt.Println("creating expenses...")
	paidExpense := func(propertyID uuid.UUID, category domain.ExpenseCategory, amount float64, desc string, incurredDaysAgo int, taxDeductible bool) {
		row, err := d.expenses.Create(d.ctx, d.ownerID, domain.CreateExpenseInput{
			PropertyID: propertyID, Category: category, AmountCents: cents(amount), IncurredOn: daysAgo(incurredDaysAgo),
			Description: desc, TaxDeductible: taxDeductible, PaidOn: ptr(daysAgo(incurredDaysAgo)), PaymentMethod: "ach",
		})
		d.must(fmt.Sprintf("expense (paid): %s", desc), err)
		_ = row
	}
	unpaidExpense := func(propertyID uuid.UUID, category domain.ExpenseCategory, amount float64, desc string, incurredDaysAgo, dueInDays int, taxDeductible bool) {
		_, err := d.expenses.Create(d.ctx, d.ownerID, domain.CreateExpenseInput{
			PropertyID: propertyID, Category: category, AmountCents: cents(amount), IncurredOn: daysAgo(incurredDaysAgo),
			DueOn: ptr(daysFrom(dueInDays)), Description: desc, TaxDeductible: taxDeductible,
		})
		d.must(fmt.Sprintf("expense (unpaid): %s", desc), err)
	}

	paidExpense(maple.ID, domain.ExpenseCategoryUtilities, 210.40, "City Water & Sewer — August", 12, false)
	unpaidExpense(maple.ID, domain.ExpenseCategoryInsurance, 480.00, "Landlord insurance premium", 20, 10, true)
	unpaidExpense(riverside.ID, domain.ExpenseCategoryPropertyTax, 1850.00, "Travis County property tax — Q3", 40, -5, true) // already overdue
	unpaidExpense(maple.ID, domain.ExpenseCategorySupplies, 65.00, "Replacement smoke detectors", 3, 27, false)
	paidExpense(oakview.ID, domain.ExpenseCategoryLegal, 350.00, "Lease review — Oakview 102 re-listing", 6, true)

	vendorExpense, err := d.expenses.Create(d.ctx, d.ownerID, domain.CreateExpenseInput{
		PropertyID: riverside.ID, Category: domain.ExpenseCategoryRepairs, VendorID: &plumbing.ID,
		AmountCents: cents(600.00), IncurredOn: daysAgo(15), DueOn: ptr(daysFrom(15)), Description: "Water heater replacement",
	})
	d.must("expense (unpaid, vendor-linked): Riverside — water heater replacement", err)
	_ = vendorExpense

	// Recurring template dated far enough back that the worker's daily
	// sweep (see cmd/worker) has already materialized a couple of real
	// monthly occurrences from it — the same mechanism a real recurring
	// bill would go through, not special-cased here.
	_, err = d.expenses.Create(d.ctx, d.ownerID, domain.CreateExpenseInput{
		PropertyID: oakview.ID, Category: domain.ExpenseCategoryLandscaping, AmountCents: cents(320.00),
		IncurredOn: daysAgo(65), Description: "GreenScape monthly service", VendorID: nil,
		IsRecurring: true, RecurrenceFrequency: ptr(domain.RecurrenceMonthly),
	})
	d.must("expense (recurring template): Oakview — landscaping", err)
	n, err := d.expenses.GenerateRecurring(d.ctx, d.ownerID, today)
	d.must(fmt.Sprintf("generate recurring occurrences (%d created)", n), err)

	fmt.Println("creating work orders (2 completed ones auto-create expenses)...")
	woFaucet, err := d.workOrders.CreateWorkOrder(d.ctx, d.ownerID, domain.CreateWorkOrderInput{
		PropertyID: maple.ID, UnitID: &mapleA.ID, Title: "Leaking kitchen faucet", Category: domain.WorkOrderCategoryPlumbing,
		Priority: domain.WorkOrderPriorityMedium, Status: domain.WorkOrderStatusNew, ReportedBy: "Sarah Chen",
		VendorID: &plumbing.ID,
	}, domain.AllPropertyAccess())
	d.must("work order: Maple A — leaking kitchen faucet", err)
	_, err = d.workOrders.UpdateWorkOrder(d.ctx, woFaucet.ID, d.ownerID, domain.UpdateWorkOrderInput{
		Status: ptr(domain.WorkOrderStatusCompleted), ActualCost: ptr(145.00),
	}, domain.AllPropertyAccess())
	d.must("  → completed, $145.00 (auto-expense created)", err)

	woBreaker, err := d.workOrders.CreateWorkOrder(d.ctx, d.ownerID, domain.CreateWorkOrderInput{
		PropertyID: maple.ID, UnitID: &mapleB.ID, Title: "Circuit breaker keeps tripping", Category: domain.WorkOrderCategoryElectrical,
		Priority: domain.WorkOrderPriorityHigh, Status: domain.WorkOrderStatusInProgress, ReportedBy: "David Okafor",
		VendorID: &electric.ID, DueDate: ptr(daysFrom(3)),
	}, domain.AllPropertyAccess())
	d.must("work order: Maple B — circuit breaker tripping (in progress)", err)
	_ = woBreaker

	_, err = d.workOrders.CreateWorkOrder(d.ctx, d.ownerID, domain.CreateWorkOrderInput{
		PropertyID: riverside.ID, UnitID: &riversideMain.ID, Title: "AC not cooling", Category: domain.WorkOrderCategoryHVAC,
		Priority: domain.WorkOrderPriorityEmergency, Status: domain.WorkOrderStatusAssigned, ReportedBy: "The Martinez Family",
		AssignedTo: "Cool Air HVAC (not a tracked vendor)", DueDate: ptr(daysFrom(1)),
	}, domain.AllPropertyAccess())
	d.must("work order: Riverside — AC not cooling (emergency)", err)

	_, err = d.workOrders.CreateWorkOrder(d.ctx, d.ownerID, domain.CreateWorkOrderInput{
		PropertyID: oakview.ID, UnitID: &oak101.ID, Title: "Squeaky door hinge", Category: domain.WorkOrderCategoryGeneral,
		Priority: domain.WorkOrderPriorityLow, Status: domain.WorkOrderStatusNew, ReportedBy: "Priya Patel",
	}, domain.AllPropertyAccess())
	d.must("work order: Oakview 101 — squeaky door hinge", err)

	woDishwasher, err := d.workOrders.CreateWorkOrder(d.ctx, d.ownerID, domain.CreateWorkOrderInput{
		PropertyID: oakview.ID, UnitID: &oak103.ID, Title: "Dishwasher not draining", Category: domain.WorkOrderCategoryAppliance,
		Priority: domain.WorkOrderPriorityMedium, Status: domain.WorkOrderStatusNew, ReportedBy: "Tom Nguyen",
		AssignedTo: "Ace Appliance Repair",
	}, domain.AllPropertyAccess())
	d.must("work order: Oakview 103 — dishwasher not draining", err)
	_, err = d.workOrders.UpdateWorkOrder(d.ctx, woDishwasher.ID, d.ownerID, domain.UpdateWorkOrderInput{
		Status: ptr(domain.WorkOrderStatusCompleted), ActualCost: ptr(210.00),
	}, domain.AllPropertyAccess())
	d.must("  → completed, $210.00 (auto-expense created)", err)

	_, err = d.workOrders.CreateWorkOrder(d.ctx, d.ownerID, domain.CreateWorkOrderInput{
		PropertyID: oakview.ID, Title: "Quarterly landscaping service", Category: domain.WorkOrderCategoryGeneral,
		Priority: domain.WorkOrderPriorityLow, Status: domain.WorkOrderStatusNew, ReportedBy: "Property Manager",
		AssignedTo: "GreenScape Landscaping", DueDate: ptr(daysFrom(15)),
	}, domain.AllPropertyAccess())
	d.must("work order: Oakview (property-wide) — quarterly landscaping", err)

	fmt.Println("settling security deposits (one per lease was created automatically)...")
	depA := d.depositForLease(leaseA.ID)
	_, err = d.deposits.AddDeduction(d.ctx, d.ownerID, depA.ID, domain.AddDeductionInput{Description: "Carpet cleaning", AmountCents: cents(150.00)})
	d.must("  Maple A deposit: add $150 deduction", err)
	_, err = d.deposits.Settle(d.ctx, d.ownerID, depA.ID, domain.SettleDepositInput{RefundCents: cents(1850.00 - 150.00), Method: "check", RefundedOn: daysAgo(2)})
	d.must("  Maple A deposit: settled (partially refunded)", err)

	depRiverside := d.depositForLease(leaseRiverside.ID)
	_, err = d.deposits.Settle(d.ctx, d.ownerID, depRiverside.ID, domain.SettleDepositInput{RefundCents: cents(2400.00), Method: "ach", RefundedOn: daysAgo(1)})
	d.must("  Riverside deposit: settled (fully refunded)", err)

	dep103 := d.depositForLease(lease103.ID)
	_, err = d.deposits.Forfeit(d.ctx, d.ownerID, dep103.ID)
	d.must("  Oakview 103 deposit: forfeited", err)

	fmt.Println("  Maple B and Oakview 101 deposits left Held (tenants still in place).")

	fmt.Println("importing and reconciling bank statement lines...")
	depositLines := []domain.StatementLineInput{
		{PostedOn: ym(3), Description: "ACH DEPOSIT — RENT-3821", AmountCents: cents(1850.00), ExternalID: "stmt-1"},
		{PostedOn: ym(2), Description: "ACH DEPOSIT — RENT-3902", AmountCents: cents(1850.00), ExternalID: "stmt-2"},
		{PostedOn: daysAgo(12), Description: "ACH WITHDRAWAL — CITY WATER SEWER", AmountCents: -cents(210.40), ExternalID: "stmt-3"},
		{PostedOn: daysAgo(2), Description: "UNKNOWN DEPOSIT", AmountCents: cents(60.00), ExternalID: "stmt-4"},
		{PostedOn: daysAgo(1), Description: "BANK FEE", AmountCents: -cents(12.00), ExternalID: "stmt-5"},
	}
	added, err := d.bankAccounts.AddStatementLines(d.ctx, d.ownerID, operating.ID, domain.StatementLineSourceCSV, depositLines)
	d.must(fmt.Sprintf("imported %d statement lines", added), err)

	lines, _, err := d.bankAccounts.ListStatementLines(d.ctx, d.ownerID, domain.StatementLineListOptions{BankAccountID: operating.ID, Limit: 50})
	d.must("list statement lines", err)
	rec, err := d.bankAccounts.Reconciliation(d.ctx, d.ownerID, operating.ID)
	d.must("load reconciliation candidates", err)
	matched := 0
	for _, line := range lines {
		if line.MatchedPaymentID != nil {
			continue
		}
		for _, p := range rec.UnmatchedPayments {
			if p.AmountCents == line.AmountCents {
				if err := d.bankAccounts.Match(d.ctx, d.ownerID, operating.ID, line.ID, p.PaymentID); err == nil {
					matched++
				}
				break
			}
		}
	}
	fmt.Printf("  ✓ matched %d of %d statement lines (the rest stay unmatched for the Reconciliation screen to show)\n", matched, len(lines))

	fmt.Println("generating owner statements for last month...")
	periodStart, periodEnd := ym(1), domain.LastOfMonth(ym(1))
	stJane, err := d.statements.Generate(d.ctx, d.ownerID, maple.ID, periodStart, periodEnd)
	d.must(fmt.Sprintf("statement: Maple Street Duplex → Jane Sullivan (net %s)", money(stJane.Snapshot.NetPayoutCents)), err)
	_, err = d.statements.Send(d.ctx, d.ownerID, stJane.ID)
	d.must("  → queued for email delivery (see Mailpit)", err)

	stMarcus, err := d.statements.Generate(d.ctx, d.ownerID, oakview.ID, periodStart, periodEnd)
	d.must(fmt.Sprintf("statement: Oakview Apartments → Marcus Chen (net %s)", money(stMarcus.Snapshot.NetPayoutCents)), err)
	_, err = d.statements.MarkPaid(d.ctx, d.ownerID, stMarcus.ID, domain.MarkStatementPaidInput{
		AmountCents: stMarcus.Snapshot.NetPayoutCents, PaidOn: daysAgo(3), Method: "wire", Reference: "WIRE-9931",
	})
	d.must("  → marked paid directly (wire)", err)

	return nil
}

// payLease pays a whole month's rent for a lease in one lump sum
// (allocated oldest-open-row-first — see LedgerService.RecordLeasePayment),
// the same path the Rent Roll's "Record Payment" button uses.
func (d *demo) payLease(leaseID uuid.UUID, period time.Time, amount float64, method, reference string) {
	_, err := d.ledger.RecordLeasePayment(d.ctx, d.ownerID, leaseID, domain.RecordPaymentInput{
		AmountCents: cents(amount), PaidOn: period.AddDate(0, 0, 2), Method: method, Reference: reference,
	})
	d.must(fmt.Sprintf("  payment: %s for %s", money(cents(amount)), period.Format("Jan 2006")), err)
}

// payPeriod pays one specific billing period directly (bypassing the
// oldest-first lump-sum allocation), so an older month can stay
// deliberately unpaid while a later one is settled — real arrears, not
// an artifact of payment order.
func (d *demo) payPeriod(leaseID uuid.UUID, period time.Time, amount float64, method, reference string) {
	amountCents := cents(amount)
	rows, err := d.ledgerRepo.ListOpenIncomeForLease(d.ctx, leaseID)
	if err != nil {
		d.must("list open income for lease", err)
		return
	}
	for _, r := range rows {
		if r.Type != domain.TransactionTypeRent || r.Period == nil || !r.Period.Equal(domain.FirstOfMonth(period)) {
			continue
		}
		_, err := d.ledger.RecordPayment(d.ctx, d.ownerID, r.ID, domain.RecordPaymentInput{
			AmountCents: amountCents, PaidOn: period.AddDate(0, 0, 2), Method: method, Reference: reference,
		})
		d.must(fmt.Sprintf("  payment: %s toward %s (partial/targeted)", money(amountCents), period.Format("Jan 2006")), err)
		return
	}
	fmt.Printf("  (skip) no open rent row for %s\n", period.Format("Jan 2006"))
}

func (d *demo) depositForLease(leaseID uuid.UUID) *domain.DepositRow {
	rows, _, err := d.deposits.List(d.ctx, d.ownerID, domain.DepositListOptions{Limit: 100})
	if err != nil {
		d.must("list deposits", err)
		return nil
	}
	for _, r := range rows {
		if r.LeaseID == leaseID {
			return r
		}
	}
	fmt.Printf("  (skip) no deposit found for lease %s\n", leaseID)
	return &domain.DepositRow{}
}

func money(c int64) string { return fmt.Sprintf("$%.2f", float64(c)/100) }
