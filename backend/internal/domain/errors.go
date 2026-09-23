// Package domain's errors.go defines what can go wrong in business terms
// only. Nothing here knows an HTTP status code exists, what JSON is, or
// what a database driver returns — that translation happens once, at
// the transport boundary (internal/transport/http/response). Repository
// and service code returns these directly (wrapped with %w for context)
// rather than inventing new ad-hoc error values per call site.
package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors covering the common outcomes every layer needs to
// distinguish. errors.Is is the only supported way to check for these —
// never compare error strings.
var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	// ErrConflict means the request is well-formed but the resource's
	// current state (or its dependents) forbids it — e.g. deleting a
	// lease that has ledger history.
	ErrConflict = errors.New("conflict")
	// ErrUnavailable means an optional piece of infrastructure this
	// request needs (object storage, email) isn't configured.
	ErrUnavailable = errors.New("unavailable")
)

// ValidationError carries structured detail about exactly one invalid
// field, for the cases where "invalid input" alone isn't enough — a
// caller (ultimately the HTTP boundary) needs to know *which* field and
// *why*.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors collects one or more ValidationError, e.g. when a
// service validates several fields on a single input struct before
// returning. It satisfies error itself so it can be returned (and
// wrapped) exactly like any other error.
type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 1 {
		return e[0].Error()
	}
	messages := make([]string, len(e))
	for i, fe := range e {
		messages[i] = fe.Error()
	}
	return strings.Join(messages, "; ")
}

// Is reports that every ValidationErrors is also an ErrInvalidInput, so
// callers that only care "was this bad input?" can use the plain
// sentinel via errors.Is, while callers that need the field-level
// detail can errors.As into *ValidationErrors.
func (e ValidationErrors) Is(target error) bool {
	return target == ErrInvalidInput
}
