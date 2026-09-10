// Command api is the entrypoint for the PropertyManagement HTTP API.
// main is a thin wrapper around run: it holds the single os.Exit call in
// the program and no other logic. run performs all dependency wiring —
// config, logger, database, cache, repositories, services, handlers,
// router, server — and blocks until the process receives a shutdown
// signal, then drains in-flight requests before returning.
//
// This file (plus the tiny handlers.VersionHandler in
// internal/transport/http/handlers/version_handler.go it constructs) is
// the only place in the codebase allowed to know about every layer at
// once; everything below it is reached only through the domain
// interfaces it wires together.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"propertymanagement/internal/config"
	"propertymanagement/internal/repository/postgres"
	"propertymanagement/internal/repository/rediscache"
	"propertymanagement/internal/service"
	transporthttp "propertymanagement/internal/transport/http"
	"propertymanagement/internal/transport/http/handlers"
)

// Version, Commit, and BuildDate are build metadata, overwritten at
// compile time via -ldflags -X and otherwise left at their zero-value
// defaults for `go run`/local builds. Unlike the *sql.DB-style package
// globals rule (4) below forbids, these are never reassigned once the
// process starts — they're baked in before main() runs — so they don't
// reintroduce shared mutable state.
//
//	go build -ldflags " \
//	    -X main.Version=$(git describe --tags --always) \
//	    -X main.Commit=$(git rev-parse HEAD) \
//	    -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
//	    ./cmd/api
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

const (
	dbConnectTimeout    = 5 * time.Second
	cacheConnectTimeout = 5 * time.Second
	shutdownTimeout     = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		// run() failed before or while building its own logger (e.g. bad
		// config), so this is the one place allowed to fall back to a
		// plain stderr write instead of structured logging.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// Best-effort: loads ./.env into the process environment for local
	// `go run`/`make run`, without overwriting any variable already set
	// (godotenv never overrides real env vars). docker-compose injects
	// .env into containers itself, so this is a no-op there; in a real
	// deployment there's no .env file at all, so Load just returns an
	// error here that's fine to ignore — config.Load below is what
	// actually enforces every required variable is present.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.Info("configuration loaded", "env", cfg.Env, "port", cfg.HTTPPort, "version", Version)

	// One root context for the process lifetime: canceled the moment
	// SIGINT/SIGTERM arrives, which unblocks the shutdown select below.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbCtx, cancelDB := context.WithTimeout(ctx, dbConnectTimeout)
	defer cancelDB()
	pool, err := postgres.NewPool(dbCtx, cfg.DatabaseURL, cfg.DatabaseMaxConn)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()
	log.Info("connected to postgres")

	cache := rediscache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			log.Error("closing redis client", "error", closeErr)
		}
	}()

	cacheCtx, cancelCache := context.WithTimeout(ctx, cacheConnectTimeout)
	defer cancelCache()
	if err := cache.Ping(cacheCtx); err != nil {
		// Redis is optional infrastructure here (see README): caching and
		// immediate refresh-token revocation degrade gracefully rather
		// than blocking startup, so this is a warning, not a fatal error.
		log.Warn("redis not reachable at startup; caching and refresh-token revocation will be degraded", "error", err)
	} else {
		log.Info("connected to redis")
	}

	// Repositories: concrete, DB-backed, injected with the connections above.
	propertyRepo := postgres.NewPropertyRepository(pool)
	userRepo := postgres.NewUserRepository(pool)

	// Services: injected with repositories (as domain interfaces) and the logger.
	propertyService := service.NewPropertyService(propertyRepo, cache, log)
	authService := service.NewAuthService(userRepo, cache, cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	// Handlers: injected with services (as domain interfaces).
	authHandler := handlers.NewAuthHandler(authService, cfg.JWTAccessTTL, cfg.JWTRefreshTTL, cfg.CookieDomain, cfg.CookieSecure)
	healthHandler := handlers.NewHealthHandler(pool, cache)
	versionHandler := handlers.NewVersionHandler(Version, Commit, BuildDate)

	router := transporthttp.NewRouter(transporthttp.RouterConfig{
		Logger:          log,
		AllowedOrigins:  cfg.AllowedOrigins,
		AuthService:     authService,
		PropertyService: propertyService,
		AuthHandler:     authHandler,
		HealthHandler:   healthHandler,
		VersionHandler:  versionHandler,
	})

	srv := transporthttp.NewServer(":"+cfg.HTTPPort, router)

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received, draining in-flight requests")
	}

	// A fresh context, not ctx: ctx is already canceled (that's what got us
	// here), and Shutdown below applies its own bounded timeout on top of it.
	if err := transporthttp.Shutdown(context.Background(), srv, shutdownTimeout); err != nil {
		return fmt.Errorf("shutting down http server: %w", err)
	}

	log.Info("shutdown complete")
	return nil
}
