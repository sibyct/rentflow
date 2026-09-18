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

	AuthService     domain.AuthService
	PropertyService domain.PropertyService
	UnitService     domain.UnitService
	LeaseService    domain.LeaseService

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
		})
	})

	return r
}
