// Package response centralizes how the HTTP boundary writes JSON — both
// the success envelope here and the error envelope in errors.go.
package response

import (
	"encoding/json"
	"net/http"
)

type envelope struct {
	Data any `json:"data,omitempty"`
	Meta any `json:"meta,omitempty"`
}

// JSON writes a successful response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, envelope{Data: data})
}

// JSONWithMeta writes a successful response that also carries pagination
// or other metadata alongside the primary payload.
func JSONWithMeta(w http.ResponseWriter, status int, data, meta any) {
	writeJSON(w, status, envelope{Data: data, Meta: meta})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
