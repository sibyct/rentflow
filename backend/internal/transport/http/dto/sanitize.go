package dto

import (
	"strings"
	"unicode"
)

// Sanitizable is implemented by any request DTO that needs normalization
// before validation runs. decodeAndValidate (see
// internal/transport/http/handlers/decode.go) calls Sanitize on the
// decoded struct before running validate.Struct, so validation tags
// (required, min, ...) see the normalized value rather than raw
// user input — e.g. an address of "   " is trimmed to "" and correctly
// fails "required", instead of passing "min=3" on its untrimmed length
// and only becoming empty later in ToDomain.
type Sanitizable interface {
	Sanitize()
}

// sanitizeString trims surrounding whitespace and strips control
// characters (anything unicode.IsControl reports true for — NUL, escape
// sequences, form feed, etc.) that have no legitimate reason to appear
// in a single-line form field. It does not otherwise alter the text.
func sanitizeString(s string) string {
	s = strings.TrimSpace(s)
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// sanitizeEmail additionally lowercases: email addresses are
// conventionally treated case-insensitively (the domain part always is;
// almost every real provider treats the local part that way too), so
// normalizing case here means "Owner@Example.com" and "owner@example.com"
// are recognized as the same account instead of silently creating two.
func sanitizeEmail(s string) string {
	return strings.ToLower(sanitizeString(s))
}
