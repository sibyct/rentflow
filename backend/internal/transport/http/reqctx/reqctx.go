// Package reqctx holds the request-scoped values (request ID, trace ID,
// logger, auth claims) that flow through context.Context, plus the
// plain getter/setter functions for them.
//
// It exists as its own package — rather than living in middleware,
// where these values are first populated — because
// internal/transport/http/response needs to read the logger back out of
// context to log errors at the one centralized point (WriteError), and
// middleware already depends on response (Recover and Authenticate both
// call WriteError on failure). response importing middleware directly
// would form an import cycle; both importing this leaf package instead
// does not.
package reqctx

import (
	"context"
	"log/slog"

	"propertymanagement/internal/domain"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	traceIDKey
	loggerKey
	claimsKey
	propertyAccessKey
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

func TraceID(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

func WithLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

// Logger returns the request-scoped logger, falling back to
// slog.Default() if none was set — e.g. a handler invoked directly in a
// test, without the middleware chain that normally installs one.
func Logger(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

func WithClaims(ctx context.Context, claims *domain.AuthClaims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func Claims(ctx context.Context) (*domain.AuthClaims, bool) {
	claims, ok := ctx.Value(claimsKey).(*domain.AuthClaims)
	return claims, ok
}

func WithPropertyAccess(ctx context.Context, access domain.PropertyAccess) context.Context {
	return context.WithValue(ctx, propertyAccessKey, access)
}

func PropertyAccess(ctx context.Context) (domain.PropertyAccess, bool) {
	access, ok := ctx.Value(propertyAccessKey).(domain.PropertyAccess)
	return access, ok
}
