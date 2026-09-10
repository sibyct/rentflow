package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"propertymanagement/internal/transport/http/response"
)

// Recover converts a panic anywhere downstream into a logged 500 response
// instead of crashing the process or leaking a raw stack trace to the
// client.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				LoggerFromContext(r.Context()).LogAttrs(r.Context(), slog.LevelError, "panic recovered",
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				response.WriteError(w, r, fmt.Errorf("panic: %v", rec))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
