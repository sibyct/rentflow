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

	propertyHandler := handlers.NewPropertyHandler(cfg.PropertyService)

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
			})
		})
	})

	return r
}
