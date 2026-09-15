package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
)

func newJSONRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
}

// wantField asserts err is a domain.ValidationErrors containing exactly
// one failure, for the given field.
func assertSingleFieldError(t *testing.T, err error, field string) {
	t.Helper()
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("error = %v, want errors.Is(err, domain.ErrInvalidInput)", err)
	}
	var verrs domain.ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("error = %v, want errors.As to find domain.ValidationErrors", err)
	}
	if len(verrs) != 1 || verrs[0].Field != field {
		t.Fatalf("ValidationErrors = %v, want exactly one failure for field %q", verrs, field)
	}
}

const validCreatePropertyBody = `{"name":"Willow Creek","type":"residential_multi_unit","address_line1":"123 Main St","units":4}`

func TestDecodeAndValidate_CreatePropertyRequest(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantField string
	}{
		{name: "valid input", body: `{"name":"Willow Creek","type":"residential_multi_unit","address_line1":"123 Main St","units":4,"status":"active"}`},
		{name: "valid input without optional status", body: validCreatePropertyBody},
		{
			name: "missing name", body: `{"type":"residential_multi_unit","address_line1":"123 Main St","units":4}`,
			wantErr: true, wantField: "name",
		},
		{
			name: "missing address_line1", body: `{"name":"Willow Creek","type":"residential_multi_unit","units":4}`,
			wantErr: true, wantField: "address_line1",
		},
		{
			name: "missing units", body: `{"name":"Willow Creek","type":"residential_multi_unit","address_line1":"123 Main St"}`,
			wantErr: true, wantField: "units",
		},
		{
			name: "negative units", body: `{"name":"Willow Creek","type":"residential_multi_unit","address_line1":"123 Main St","units":-1}`,
			wantErr: true, wantField: "units",
		},
		{
			name: "unknown type", body: `{"name":"Willow Creek","type":"condemned","address_line1":"123 Main St","units":4}`,
			wantErr: true, wantField: "type",
		},
		{
			name: "unknown status", body: `{"name":"Willow Creek","type":"residential_multi_unit","address_line1":"123 Main St","units":4,"status":"condemned"}`,
			wantErr: true, wantField: "status",
		},
		{name: "malformed json", body: `{not json`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req dto.CreatePropertyRequest
			err := decodeAndValidate(newJSONRequest(tt.body), &req)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("decodeAndValidate() unexpected error = %v", err)
				}
				return
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("decodeAndValidate() error = %v, want errors.Is(err, domain.ErrInvalidInput)", err)
			}
			if tt.wantField != "" {
				assertSingleFieldError(t, err, tt.wantField)
			}
		})
	}
}

func TestDecodeAndValidate_CreatePropertyRequest_SanitizesBeforeValidating(t *testing.T) {
	var req dto.CreatePropertyRequest
	err := decodeAndValidate(newJSONRequest(`{"name":"Willow Creek","type":"residential_multi_unit","address_line1":"  123 Main St  ","units":4}`), &req)
	if err != nil {
		t.Fatalf("decodeAndValidate() unexpected error = %v", err)
	}
	if req.AddressLine1 != "123 Main St" {
		t.Errorf("AddressLine1 = %q, want trimmed %q", req.AddressLine1, "123 Main St")
	}
}

func TestDecodeAndValidate_CreatePropertyRequest_WhitespaceOnlyAddressRejected(t *testing.T) {
	var req dto.CreatePropertyRequest
	err := decodeAndValidate(newJSONRequest(`{"name":"Willow Creek","type":"residential_multi_unit","address_line1":"      ","units":4}`), &req)
	// Sanitize trims "      " to "", which then correctly fails
	// "required" — this is the ordering (sanitize before validate) that
	// makes an all-whitespace address rejected instead of merely
	// happening to be long enough to pass "min=3" pre-trim.
	assertSingleFieldError(t, err, "address_line1")
}

// TestNoctrlValidator exercises the "noctrl" tag directly against
// validate.Struct, independent of any DTO's Sanitize method. It can't
// be exercised through decodeAndValidate for CreatePropertyRequest or
// UpdatePropertyRequest: their Sanitize already strips control
// characters before validate.Struct ever runs, so noctrl never has
// anything left to reject there — it's a backstop for a DTO that has
// the tag but forgets to wire up Sanitize, not the primary defense.
func TestNoctrlValidator(t *testing.T) {
	type noSanitizeStruct struct {
		Value string `validate:"noctrl"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "clean string passes", value: "hello world"},
		{name: "empty string passes", value: ""},
		{name: "control character fails", value: "helloworld", wantErr: true},
		{name: "newline fails", value: "hello\nworld", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(&noSanitizeStruct{Value: tt.value})
			if tt.wantErr && err == nil {
				t.Fatal("validate.Struct() = nil, want a noctrl validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validate.Struct() unexpected error = %v", err)
			}
		})
	}
}

func TestDecodeAndValidate_UpdatePropertyRequest(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantField string
	}{
		{name: "all fields omitted is valid (partial update)", body: `{}`},
		{name: "valid address", body: `{"address_line1":"456 Oak Ave"}`},
		{
			// omitempty on a pointer field only skips a nil pointer, not
			// an explicit "" — so a too-long string is what reliably
			// exercises the tag at the HTTP boundary; whether an
			// explicitly-empty string is itself acceptable on update is
			// a business rule enforced one layer down, in the service
			// (see TestPropertyService_UpdateProperty's "empty address
			// rejected" case), not something omitempty can express here.
			name: "address_line1 too long", body: `{"address_line1":"` + strings.Repeat("a", 256) + `"}`,
			wantErr: true, wantField: "address_line1",
		},
		{
			name: "zero units", body: `{"units":0}`,
			wantErr: true, wantField: "units",
		},
		{
			name: "unknown status", body: `{"status":"condemned"}`,
			wantErr: true, wantField: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req dto.UpdatePropertyRequest
			err := decodeAndValidate(newJSONRequest(tt.body), &req)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("decodeAndValidate() unexpected error = %v", err)
				}
				return
			}
			assertSingleFieldError(t, err, tt.wantField)
		})
	}
}

func TestDecodeAndValidate_RegisterRequest(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantField string
	}{
		{name: "valid input", body: `{"email":"owner@example.com","password":"hunter22222"}`},
		{
			name: "missing email", body: `{"password":"hunter22222"}`,
			wantErr: true, wantField: "email",
		},
		{
			name: "malformed email", body: `{"email":"not-an-email","password":"hunter22222"}`,
			wantErr: true, wantField: "email",
		},
		{
			name: "missing password", body: `{"email":"owner@example.com"}`,
			wantErr: true, wantField: "password",
		},
		{
			name: "password too short", body: `{"email":"owner@example.com","password":"short"}`,
			wantErr: true, wantField: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req dto.RegisterRequest
			err := decodeAndValidate(newJSONRequest(tt.body), &req)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("decodeAndValidate() unexpected error = %v", err)
				}
				return
			}
			assertSingleFieldError(t, err, tt.wantField)
		})
	}
}

func TestDecodeAndValidate_RegisterRequest_NormalizesEmailCaseAndWhitespace(t *testing.T) {
	var req dto.RegisterRequest
	err := decodeAndValidate(newJSONRequest(`{"email":"  Owner@Example.COM  ","password":"hunter22222"}`), &req)
	if err != nil {
		t.Fatalf("decodeAndValidate() unexpected error = %v", err)
	}
	if req.Email != "owner@example.com" {
		t.Errorf("Email = %q, want normalized %q", req.Email, "owner@example.com")
	}
	if req.Password != "hunter22222" {
		t.Errorf("Password = %q, want unchanged (passwords are never sanitized)", req.Password)
	}
}

func TestDecodeAndValidate_LoginRequest(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantField string
	}{
		{name: "valid input", body: `{"email":"owner@example.com","password":"anything"}`},
		{
			name: "missing email", body: `{"password":"anything"}`,
			wantErr: true, wantField: "email",
		},
		{
			name: "missing password", body: `{"email":"owner@example.com"}`,
			wantErr: true, wantField: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req dto.LoginRequest
			err := decodeAndValidate(newJSONRequest(tt.body), &req)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("decodeAndValidate() unexpected error = %v", err)
				}
				return
			}
			assertSingleFieldError(t, err, tt.wantField)
		})
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  int
		wantOffset int
	}{
		{name: "no params uses defaults", query: "", wantLimit: 20, wantOffset: 0},
		{name: "within bounds", query: "?limit=50&offset=10", wantLimit: 50, wantOffset: 10},
		{name: "limit clamped to max", query: "?limit=100000", wantLimit: 100, wantOffset: 0},
		{name: "limit clamped to min", query: "?limit=0", wantLimit: 1, wantOffset: 0},
		{name: "negative limit clamped to min", query: "?limit=-5", wantLimit: 1, wantOffset: 0},
		{name: "negative offset clamped to zero", query: "?offset=-5", wantLimit: 20, wantOffset: 0},
		{name: "non-numeric params fall back to defaults", query: "?limit=abc&offset=xyz", wantLimit: 20, wantOffset: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/"+tt.query, nil)
			limit, offset := parsePagination(r)
			if limit != tt.wantLimit {
				t.Errorf("parsePagination() limit = %d, want %d", limit, tt.wantLimit)
			}
			if offset != tt.wantOffset {
				t.Errorf("parsePagination() offset = %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}
