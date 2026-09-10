package handlers

import (
	"context"
	"net/http"
	"time"

	"propertymanagement/internal/transport/http/response"
)

// Pinger is satisfied by anything the readiness check needs to verify is
// reachable (the Postgres pool, the Redis client). It is defined here,
// at the point of use, rather than in domain, since "pingability" is an
// infrastructure concern.
type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db    Pinger
	cache Pinger // may be nil if Redis is not configured
}

func NewHealthHandler(db, cache Pinger) *HealthHandler {
	return &HealthHandler{db: db, cache: cache}
}

// Healthz reports whether the process itself is up. It never checks
// dependencies — that's Readyz's job — so an orchestrator doesn't kill a
// healthy process just because the database blipped.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readyz reports whether the process can currently serve traffic, i.e.
// whether its dependencies are reachable.
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{}
	ready := true

	if err := h.db.Ping(ctx); err != nil {
		checks["database"] = err.Error()
		ready = false
	} else {
		checks["database"] = "ok"
	}

	if h.cache != nil {
		if err := h.cache.Ping(ctx); err != nil {
			checks["cache"] = err.Error()
			ready = false
		} else {
			checks["cache"] = "ok"
		}
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	response.JSON(w, status, map[string]any{"ready": ready, "checks": checks})
}
