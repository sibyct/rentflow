// This file is the centralized HTTP error translator for the whole API.
// Every handler, and every piece of transport-level middleware that can
// reject a request (auth, panic recovery), calls WriteError on any
// error path instead of writing its own error response. It is the only
// place in the codebase that:
//
//   - knows domain sentinel/typed errors map to HTTP status codes
//   - logs errors at all — repositories and services only wrap and
//     return (see internal/domain/errors.go, internal/service,
//     internal/repository); logging happens exactly once, here
//
// It intentionally placed in this package (internal/transport/http/response)
// rather than internal/transport/http/errors.go: handlers live in the
// sibling package internal/transport/http/handlers, which the http
// (router) package already imports, so a function here needing to live
// directly in package http and be called from handlers would form an
// import cycle (http -> handlers -> http). Keeping it in its own leaf
// package avoids that while still giving every handler one function to
// call, which is the actual requirement.
package response

import (
	"errors"
	"log/slog"
	"net/http"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/reqctx"
)

// ErrorResponse is the JSON body written for every error response.
// Error is always a safe, generic, client-facing message — never raw
// internal error text, a stack trace, or SQL details. Code is a stable
// machine-readable identifier a client can switch on. Details carries
// optional structured extra context that is itself safe to show (e.g.
// per-field validation failures); it is never derived from an error's
// raw text.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

// fieldError is the wire shape for one domain.ValidationError. Kept
// here, not on the domain type, so domain stays free of JSON/transport
// concerns (see internal/domain/errors.go).
type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// classification is the internal result of deciding what to do with an
// error: what to tell the client, and how loudly to log it.
type classification struct {
	status   int
	code     string
	message  string
	details  any
	severity slog.Level
}

// WriteError classifies err, writes the safe ErrorResponse envelope,
// and logs the real err — full wrapped context included — exactly once,
// tagged with the request's ID/trace ID (via the request-scoped logger
// in context). Expected errors (not found, conflict, invalid input,
// auth) log at Info, since they're normal business-flow outcomes, not
// bugs; anything that doesn't match a known domain error logs at Error,
// since — by definition — it's unexpected.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	c := classify(err)

	reqctx.Logger(r.Context()).LogAttrs(r.Context(), c.severity, "request error",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", c.status),
		slog.String("code", c.code),
		slog.Any("error", err),
	)

	writeJSON(w, c.status, ErrorResponse{Error: c.message, Code: c.code, Details: c.details})
}

// classify switches purely on domain sentinel/typed errors via
// errors.Is/errors.As. Anything that doesn't match one — a raw pgx
// error that slipped through unwrapped, a third-party library error, a
// genuine bug — falls through to a generic 500 with a generic message.
// It never inspects err.Error() text to decide status or to build the
// client-visible message, so an accidental change to an internal wrap
// message can't change response shape or leak detail.
func classify(err error) classification {
	var verrs domain.ValidationErrors
	switch {
	case errors.As(err, &verrs):
		details := make([]fieldError, len(verrs))
		for i, fe := range verrs {
			details[i] = fieldError{Field: fe.Field, Message: fe.Message}
		}
		return classification{
			status:   http.StatusBadRequest,
			code:     "invalid_input",
			message:  "One or more fields failed validation.",
			details:  details,
			severity: slog.LevelInfo,
		}

	case errors.Is(err, domain.ErrNotFound):
		return classification{
			status:   http.StatusNotFound,
			code:     "not_found",
			message:  "The requested resource was not found.",
			severity: slog.LevelInfo,
		}

	case errors.Is(err, domain.ErrAlreadyExists):
		return classification{
			status:   http.StatusConflict,
			code:     "already_exists",
			message:  "The resource already exists.",
			severity: slog.LevelInfo,
		}

	case errors.Is(err, domain.ErrUnavailable):
		return classification{
			status:   http.StatusServiceUnavailable,
			code:     "unavailable",
			message:  "This feature isn't configured on the server.",
			severity: slog.LevelWarn,
		}

	case errors.Is(err, domain.ErrConflict):
		return classification{
			status:   http.StatusConflict,
			code:     "conflict",
			message:  "The request conflicts with the current state of the resource.",
			severity: slog.LevelInfo,
		}

	case errors.Is(err, domain.ErrInvalidInput):
		return classification{
			status:   http.StatusBadRequest,
			code:     "invalid_input",
			message:  "The request was invalid.",
			severity: slog.LevelInfo,
		}

	case errors.Is(err, domain.ErrUnauthorized):
		return classification{
			status:   http.StatusUnauthorized,
			code:     "unauthorized",
			message:  "Authentication is required or credentials are invalid.",
			severity: slog.LevelInfo,
		}

	case errors.Is(err, domain.ErrForbidden):
		return classification{
			status:   http.StatusForbidden,
			code:     "forbidden",
			message:  "You do not have permission to perform this action.",
			severity: slog.LevelInfo,
		}

	default:
		return classification{
			status:   http.StatusInternalServerError,
			code:     "internal_error",
			message:  "An unexpected error occurred.",
			severity: slog.LevelError,
		}
	}
}
