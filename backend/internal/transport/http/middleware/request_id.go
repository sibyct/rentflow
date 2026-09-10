package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"propertymanagement/internal/transport/http/reqctx"
)

const (
	RequestIDHeader = "X-Request-ID"
	TraceIDHeader   = "X-Trace-ID"
)

// RequestID ensures every request carries a request ID and a trace ID,
// generating them when the caller (or an upstream proxy) didn't supply
// one. Both are echoed back as response headers and stashed in context so
// downstream logging middleware can attach them to every log line.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(RequestIDHeader)
		if reqID == "" {
			reqID = uuid.NewString()
		}
		traceID := r.Header.Get(TraceIDHeader)
		if traceID == "" {
			traceID = uuid.NewString()
		}

		w.Header().Set(RequestIDHeader, reqID)
		w.Header().Set(TraceIDHeader, traceID)

		ctx := reqctx.WithRequestID(r.Context(), reqID)
		ctx = reqctx.WithTraceID(ctx, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	return reqctx.RequestID(ctx)
}

func TraceIDFromContext(ctx context.Context) string {
	return reqctx.TraceID(ctx)
}
