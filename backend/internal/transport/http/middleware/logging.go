package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"propertymanagement/internal/transport/http/reqctx"
)

// WithLogger derives a request-scoped logger carrying request_id and
// trace_id fields and stores it in context. Handlers and anything they
// call should log via LoggerFromContext so every line — not just the
// access log — carries those fields.
func WithLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqLogger := base.With(
				slog.String("request_id", RequestIDFromContext(r.Context())),
				slog.String("trace_id", TraceIDFromContext(r.Context())),
			)
			ctx := reqctx.WithLogger(r.Context(), reqLogger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoggerFromContext returns the request-scoped logger, falling back to
// slog.Default() if none was set (e.g. in a test calling a handler
// directly without the middleware chain).
func LoggerFromContext(ctx context.Context) *slog.Logger {
	return reqctx.Logger(ctx)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// AccessLog logs one line per completed request: method, path, status,
// and duration, tagged with the request/trace IDs via LoggerFromContext.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		LoggerFromContext(r.Context()).LogAttrs(r.Context(), slog.LevelInfo, "http_request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", sw.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
