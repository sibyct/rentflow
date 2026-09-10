package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
)

// validate is a single shared *validator.Validate for the whole
// process, as recommended by go-playground/validator: it caches
// per-struct-type reflection info internally, so constructing one per
// request (or per handler) would throw that caching away for no benefit.
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())

	// Report the DTO's JSON field name (e.g. "unit_count") rather than
	// the Go struct field name ("UnitCount"), so a client-facing
	// validation error names the field exactly as the client sent it.
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return field.Name
		}
		return name
	})

	registerCustomValidations(v)
	return v
}

// registerCustomValidations adds project-specific tags not covered by
// validator's built-ins, registered once here rather than reimplemented
// per struct.
func registerCustomValidations(v *validator.Validate) {
	// noctrl rejects control characters (NUL, escape sequences, form
	// feed, ...) in free-text fields. It's a backstop, not the primary
	// defense — Sanitize (see dto.Sanitizable) already strips these
	// before validation runs — but a field tagged noctrl stays rejected
	// even if a future DTO forgets to wire up Sanitize.
	_ = v.RegisterValidation("noctrl", func(fl validator.FieldLevel) bool {
		return !strings.ContainsFunc(fl.Field().String(), unicode.IsControl)
	})
}

// decodeAndValidate reads a JSON request body into dst, sanitizes it (if
// it implements dto.Sanitizable) so validation tags see normalized
// values, then runs struct validation tags. Both failure modes return a
// domain error (domain.ErrInvalidInput for malformed JSON,
// domain.ValidationErrors for failed validation tags) so
// response.WriteError — which only ever switches on domain errors —
// handles either one the same way any other handler error is handled.
// Only once both decoding and validation succeed does a handler go on
// to call the service layer.
func decodeAndValidate(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode request: malformed request body: %w: %w", domain.ErrInvalidInput, err)
	}

	if s, ok := dst.(dto.Sanitizable); ok {
		s.Sanitize()
	}

	if err := validate.Struct(dst); err != nil {
		var fieldErrs validator.ValidationErrors
		if errors.As(err, &fieldErrs) {
			return fmt.Errorf("decode request: %w", toDomainValidationErrors(fieldErrs))
		}
		return fmt.Errorf("decode request: %w: %w", domain.ErrInvalidInput, err)
	}
	return nil
}

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

// parsePagination reads limit/offset query params, applying a default
// and clamping to sane bounds, rather than passing raw strconv output
// straight into a repository query. Clamping (over rejecting
// out-of-range values with a 400) is the pagination strategy for every
// paginated endpoint in this API: a client-supplied limit=100000 or
// offset=-5 is almost always a copy-pasted default or an off-by-one,
// not a request that benefits from an error round-trip to correct.
func parsePagination(r *http.Request) (limit, offset int) {
	limit = clampInt(parseIntDefault(r.URL.Query().Get("limit"), defaultPageLimit), 1, maxPageLimit)
	offset = parseIntDefault(r.URL.Query().Get("offset"), 0)
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func clampInt(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func parseIntDefault(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func toDomainValidationErrors(fieldErrs validator.ValidationErrors) domain.ValidationErrors {
	verrs := make(domain.ValidationErrors, len(fieldErrs))
	for i, fe := range fieldErrs {
		verrs[i] = &domain.ValidationError{Field: fe.Field(), Message: humanizeTag(fe)}
	}
	return verrs
}

func humanizeTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "noctrl":
		return "must not contain control characters"
	default:
		return fmt.Sprintf("failed validation: %s", fe.Tag())
	}
}
