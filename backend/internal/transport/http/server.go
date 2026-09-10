package http

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// NewServer wraps handler in an http.Server with production-sane
// timeouts. Handler-level cancellation still flows through request
// context regardless of these values.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// Shutdown gracefully drains in-flight requests within the given
// deadline before returning.
func Shutdown(ctx context.Context, srv *http.Server, timeout time.Duration) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}
