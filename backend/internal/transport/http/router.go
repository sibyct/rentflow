package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/handlers"
	custommw "propertymanagement/internal/transport/http/middleware"
)

type RouterConfig struct {
	Logger         *slog.Logger
	AllowedOrigins []string

	AuthService          domain.AuthService
	PropertyService      domain.PropertyService
	UnitService          domain.UnitService
	LeaseService         domain.LeaseService
	WorkOrderService     domain.WorkOrderService
	RecurringRuleService domain.RecurringRuleService
	VendorService        domain.VendorService
	LedgerService        domain.LedgerService
	RentRollService      domain.RentRollService
	ExpenseService       domain.ExpenseService
	ChargeService        domain.ChargeService
	AttachmentService    domain.AttachmentService
	BankAccountService   domain.BankAccountService
	DepositService       domain.DepositService
	PropertyOwnerService domain.PropertyOwnerService
	StatementService     domain.OwnerStatementService

	AuthHandler    *handlers.AuthHandler
	HealthHandler  *handlers.HealthHandler
	VersionHandler *handlers.VersionHandler
}

// NewRouter wires the full middleware chain and route table. Route
// groups make the auth boundary explicit: everything under the
// "authenticated" group requires a valid access token.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	r.Use(custommw.RequestID)
	r.Use(custommw.WithLogger(cfg.Logger))
	r.Use(custommw.Recover)
	r.Use(custommw.AccessLog)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", custommw.RequestIDHeader, custommw.TraceIDHeader},
		ExposedHeaders:   []string{custommw.RequestIDHeader, custommw.TraceIDHeader},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", cfg.HealthHandler.Healthz)
	r.Get("/readyz", cfg.HealthHandler.Readyz)
	r.Get("/version", cfg.VersionHandler.Version)

	propertyHandler := handlers.NewPropertyHandler(cfg.PropertyService, cfg.UnitService)
	unitHandler := handlers.NewUnitHandler(cfg.UnitService)
	leaseHandler := handlers.NewLeaseHandler(cfg.LeaseService)
	workOrderHandler := handlers.NewWorkOrderHandler(cfg.WorkOrderService)
	maintenanceRuleHandler := handlers.NewMaintenanceRuleHandler(cfg.RecurringRuleService)
	vendorHandler := handlers.NewVendorHandler(cfg.VendorService)
	accountingHandler := handlers.NewAccountingHandler(cfg.LedgerService, cfg.RentRollService)
	expenseHandler := handlers.NewExpenseHandler(cfg.ExpenseService)
	chargeHandler := handlers.NewChargeHandler(cfg.ChargeService)
	attachmentHandler := handlers.NewAttachmentHandler(cfg.AttachmentService)
	bankAccountHandler := handlers.NewBankAccountHandler(cfg.BankAccountService)
	depositHandler := handlers.NewDepositHandler(cfg.DepositService)
	propertyOwnerHandler := handlers.NewPropertyOwnerHandler(cfg.PropertyOwnerService)
	statementHandler := handlers.NewOwnerStatementHandler(cfg.StatementService)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", cfg.AuthHandler.Register)
			r.Post("/login", cfg.AuthHandler.Login)
			r.Post("/refresh", cfg.AuthHandler.Refresh)
			r.Post("/logout", cfg.AuthHandler.Logout)
		})

		r.Group(func(r chi.Router) {
			r.Use(custommw.Authenticate(cfg.AuthService))

			r.Route("/properties", func(r chi.Router) {
				r.Post("/", propertyHandler.Create)
				r.Get("/", propertyHandler.List)
				r.Patch("/status", propertyHandler.BulkUpdateStatus)
				r.Get("/{id}", propertyHandler.Get)
				r.Put("/{id}", propertyHandler.Update)
				r.Delete("/{id}", propertyHandler.Delete)

				// propertyId is set from the URL, never a client-supplied
				// body field: a unit is always created/listed in the
				// context of the property page the user is already on.
				r.Route("/{propertyId}/units", func(r chi.Router) {
					r.Post("/", unitHandler.Create)
					r.Post("/bulk", unitHandler.BulkCreate)
					r.Get("/", unitHandler.List)
				})

				// Same propertyId-from-URL rationale as units above — a
				// work order or recurring rule is always created from the
				// property (or unit) page the user is already on.
				r.Post("/{propertyId}/work-orders", workOrderHandler.Create)
				r.Post("/{propertyId}/maintenance-rules", maintenanceRuleHandler.Create)
			})

			// The Vendors directory: every vendor the caller owns,
			// plus the category-filtered lookup the Vendor Selector
			// (inside the work order drawer) uses.
			r.Route("/vendors", func(r chi.Router) {
				r.Post("/", vendorHandler.Create)
				r.Get("/", vendorHandler.List)
				r.Get("/by-category", vendorHandler.ListForCategory)
				r.Get("/{id}", vendorHandler.Get)
				r.Put("/{id}", vendorHandler.Update)
				r.Delete("/{id}", vendorHandler.Delete)
				r.Get("/{id}/properties", vendorHandler.GetPropertiesServed)
				r.Get("/{id}/spend", vendorHandler.GetSpendSummary)
			})

			// The portfolio-wide Units page: every unit across every
			// property the caller owns, distinct from the property-scoped
			// list nested under /properties/{propertyId}/units above.
			r.Route("/units", func(r chi.Router) {
				r.Get("/", unitHandler.ListForOwner)
				r.Get("/{id}", unitHandler.Get)
				r.Put("/{id}", unitHandler.Update)
				r.Delete("/{id}", unitHandler.Delete)

				// unitId is set from the URL, same rationale as units
				// under /properties/{propertyId}/units above — a lease
				// is always created from the unit page it belongs to.
				r.Post("/{id}/leases", leaseHandler.Create)
			})

			// The portfolio-wide Leases page: every lease across every
			// property the caller owns.
			r.Route("/leases", func(r chi.Router) {
				r.Get("/", leaseHandler.List)
				r.Get("/{id}", leaseHandler.Get)
				r.Put("/{id}", leaseHandler.Update)
				r.Delete("/{id}", leaseHandler.Delete)
			})

			// The global Maintenance page: every work order across every
			// property the caller owns.
			r.Route("/work-orders", func(r chi.Router) {
				r.Get("/", workOrderHandler.List)
				r.Get("/summary", workOrderHandler.GetSummary)
				r.Get("/activity", workOrderHandler.GetRecentActivity)
				r.Patch("/status", workOrderHandler.BulkUpdateStatus)
				r.Patch("/reassign", workOrderHandler.BulkReassign)
				r.Get("/{id}", workOrderHandler.Get)
				r.Put("/{id}", workOrderHandler.Update)
				r.Delete("/{id}", workOrderHandler.Delete)
				r.Get("/{id}/activity", workOrderHandler.ListActivity)
				r.Post("/{id}/activity", workOrderHandler.AddNote)
			})

			// The Scheduled tab: every recurring maintenance rule across
			// every property the caller owns.
			r.Route("/maintenance-rules", func(r chi.Router) {
				r.Get("/", maintenanceRuleHandler.List)
				r.Put("/{id}", maintenanceRuleHandler.Update)
				r.Delete("/{id}", maintenanceRuleHandler.Delete)
				r.Post("/{id}/generate", maintenanceRuleHandler.GenerateNow)
			})

			// Direct-to-storage file uploads: presign → browser POSTs the
			// file to the bucket → confirm. Bytes never transit the API.
			r.Route("/attachments", func(r chi.Router) {
				r.Post("/presign", attachmentHandler.Presign)
				r.Post("/{id}/confirm", attachmentHandler.Confirm)
				r.Get("/{id}/url", attachmentHandler.DownloadURL)
			})

			// Accounting: one ledger (transactions + payments) with Rent
			// Roll, Expenses and Charges as views over it.
			r.Route("/accounting", func(r chi.Router) {
				r.Get("/dashboard", accountingHandler.Dashboard)
				r.Get("/settings", accountingHandler.GetSettings)
				r.Put("/settings", accountingHandler.UpdateSettings)

				r.Get("/rent-roll", accountingHandler.ListRentRoll)
				r.Post("/rent-roll/generate", accountingHandler.GenerateRent)
				r.Post("/rent-roll/leases/{leaseId}/payments", accountingHandler.RecordLeasePayment)

				r.Get("/expenses", expenseHandler.List)
				r.Post("/expenses", expenseHandler.Create)
				r.Get("/expenses/{id}", expenseHandler.Get)
				r.Put("/expenses/{id}", expenseHandler.Update)

				r.Get("/charges", chargeHandler.List)
				r.Post("/charges", chargeHandler.Create)

				r.Get("/transactions/{id}/payments", accountingHandler.ListPayments)
				r.Post("/transactions/{id}/payments", accountingHandler.RecordPayment)
				r.Post("/transactions/{id}/void", accountingHandler.VoidTransaction)
				r.Get("/transactions/{id}/audit", accountingHandler.TransactionAudit)
				r.Post("/payments/{id}/void", accountingHandler.VoidPayment)

				// Bank accounts and their reconciliation against manually
				// entered / imported statement lines.
				r.Get("/accounts", bankAccountHandler.List)
				r.Post("/accounts", bankAccountHandler.Create)
				r.Put("/accounts/{id}", bankAccountHandler.Update)
				r.Delete("/accounts/{id}", bankAccountHandler.Delete)
				r.Get("/accounts/{id}/reconciliation", bankAccountHandler.Reconciliation)
				r.Get("/accounts/{id}/lines", bankAccountHandler.ListLines)
				r.Post("/accounts/{id}/lines", bankAccountHandler.AddLines)
				r.Delete("/accounts/{id}/lines/{lineId}", bankAccountHandler.DeleteLine)
				r.Post("/accounts/{id}/lines/{lineId}/match", bankAccountHandler.Match)
				r.Post("/accounts/{id}/lines/{lineId}/unmatch", bankAccountHandler.Unmatch)

				r.Get("/deposits", depositHandler.List)
				r.Get("/deposits/{id}", depositHandler.Get)
				r.Put("/deposits/{id}/account", depositHandler.SetAccount)
				r.Post("/deposits/{id}/deductions", depositHandler.AddDeduction)
				r.Delete("/deposits/{id}/deductions/{deductionId}", depositHandler.RemoveDeduction)
				r.Post("/deposits/{id}/settle", depositHandler.Settle)
				r.Post("/deposits/{id}/forfeit", depositHandler.Forfeit)

				r.Get("/owners", propertyOwnerHandler.List)
				r.Post("/owners", propertyOwnerHandler.Create)
				r.Put("/owners/{id}", propertyOwnerHandler.Update)
				r.Delete("/owners/{id}", propertyOwnerHandler.Delete)

				r.Get("/statements", statementHandler.List)
				r.Post("/statements", statementHandler.Generate)
				r.Get("/statements/{id}", statementHandler.Get)
				r.Get("/statements/{id}/pdf", statementHandler.PDF)
				r.Post("/statements/{id}/send", statementHandler.Send)
				r.Post("/statements/{id}/mark-sent", statementHandler.MarkSent)
				r.Post("/statements/{id}/mark-paid", statementHandler.MarkPaid)
			})
		})
	})

	return r
}
