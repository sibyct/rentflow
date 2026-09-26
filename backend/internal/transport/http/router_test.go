package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	transporthttp "propertymanagement/internal/transport/http"
	"propertymanagement/internal/transport/http/handlers"
)

// fakeAuthService and fakePropertyService let the router test exercise
// real routing, middleware, and JSON-envelope behavior without a
// database — the service layer itself is already covered by table-driven
// tests in internal/service.

type fakeAuthService struct {
	userID uuid.UUID
}

func (f *fakeAuthService) Register(_ context.Context, email, _ string) (*domain.User, error) {
	return &domain.User{ID: f.userID, Email: email, Role: domain.UserRoleManager}, nil
}

func (f *fakeAuthService) Login(_ context.Context, _, _ string) (string, string, *domain.User, error) {
	return "fake-access-token", "fake-refresh-token", &domain.User{ID: f.userID, Role: domain.UserRoleManager}, nil
}

func (f *fakeAuthService) RefreshToken(_ context.Context, _ string) (string, string, *domain.User, error) {
	return "fake-access-token-2", "fake-refresh-token-2", &domain.User{ID: f.userID, Role: domain.UserRoleManager}, nil
}

func (f *fakeAuthService) Logout(_ context.Context, _ string) error { return nil }

func (f *fakeAuthService) ValidateAccessToken(_ context.Context, tokenString string) (*domain.AuthClaims, error) {
	if tokenString != "fake-access-token" && tokenString != "fake-access-token-2" {
		return nil, domain.ErrUnauthorized
	}
	return &domain.AuthClaims{UserID: f.userID, Role: domain.UserRoleManager}, nil
}

type fakePropertyService struct{}

func (f *fakePropertyService) CreateProperty(_ context.Context, input domain.CreatePropertyInput) (*domain.Property, error) {
	now := time.Now().UTC()
	return &domain.Property{
		ID: uuid.New(), Name: input.Name, Type: input.Type, AddressLine1: input.AddressLine1, Units: input.Units,
		Status: domain.PropertyStatusActive, OwnerID: input.OwnerID, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (f *fakePropertyService) GetProperty(_ context.Context, id, ownerID uuid.UUID, _ domain.PropertyAccess) (*domain.Property, error) {
	return nil, domain.ErrNotFound
}

func (f *fakePropertyService) ListProperties(_ context.Context, _ domain.PropertyListOptions) ([]*domain.Property, int, error) {
	return []*domain.Property{}, 0, nil
}

func (f *fakePropertyService) UpdateProperty(_ context.Context, _, _ uuid.UUID, _ domain.UpdatePropertyInput, _ domain.PropertyAccess) (*domain.Property, error) {
	return nil, domain.ErrNotFound
}

func (f *fakePropertyService) DeleteProperty(_ context.Context, _, _ uuid.UUID, _ domain.PropertyAccess) error {
	return domain.ErrNotFound
}

func (f *fakePropertyService) BulkUpdateStatus(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ domain.PropertyStatus, _ domain.PropertyAccess) (int, error) {
	return 0, nil
}

// fakeUnitService is a minimal stand-in for domain.UnitService: the unit
// service layer itself is covered by table-driven tests in
// internal/service, so this router test only needs enough behavior for
// PropertyHandler's stats-decoration calls (GetPropertyUnitStats,
// GetPropertyUnitStatsBulk) not to panic.
type fakeUnitService struct{}

func (f *fakeUnitService) CreateUnit(_ context.Context, _ uuid.UUID, _ domain.CreateUnitInput, _ domain.PropertyAccess) (*domain.Unit, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeUnitService) CreateUnitsBulk(_ context.Context, _, _ uuid.UUID, _ []domain.CreateUnitInput, _ domain.PropertyAccess) ([]*domain.Unit, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeUnitService) GetUnit(_ context.Context, _, _ uuid.UUID, _ domain.PropertyAccess) (*domain.Unit, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeUnitService) ListUnitsByProperty(_ context.Context, _, _ uuid.UUID, _ domain.PropertyAccess) ([]*domain.Unit, error) {
	return []*domain.Unit{}, nil
}

func (f *fakeUnitService) ListUnitsForOwner(_ context.Context, _ uuid.UUID, _ domain.UnitListOptions) ([]*domain.UnitWithProperty, int, error) {
	return []*domain.UnitWithProperty{}, 0, nil
}

func (f *fakeUnitService) UpdateUnit(_ context.Context, _, _ uuid.UUID, _ domain.UpdateUnitInput, _ domain.PropertyAccess) (*domain.Unit, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeUnitService) DeleteUnit(_ context.Context, _, _ uuid.UUID, _ domain.PropertyAccess) error {
	return domain.ErrNotFound
}

func (f *fakeUnitService) GetPropertyUnitStats(_ context.Context, _, _ uuid.UUID, _ domain.PropertyAccess) (*domain.PropertyUnitStats, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeUnitService) ListDocuments(_ context.Context, _, _ uuid.UUID, _ domain.PropertyAccess) ([]*domain.UnitDocument, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeUnitService) AddDocument(_ context.Context, _, _, _, _ uuid.UUID, _ domain.UnitDocumentCategory, _ domain.PropertyAccess) (*domain.UnitDocument, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeUnitService) DeleteDocument(_ context.Context, _, _, _ uuid.UUID, _ domain.PropertyAccess) error {
	return domain.ErrNotFound
}

func (f *fakeUnitService) GetPropertyUnitStatsBulk(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*domain.PropertyUnitStats, error) {
	return map[uuid.UUID]*domain.PropertyUnitStats{}, nil
}

// fakeStaffService is a minimal stand-in for domain.StaffService: this
// router test doesn't exercise Users & Roles behavior, it only needs
// ResolveAccess to satisfy the ResolvePropertyAccess middleware every
// authenticated route now runs through, so every other method is an
// unreachable stub.
type fakeStaffService struct{}

func (f *fakeStaffService) Invite(_ context.Context, _, _ uuid.UUID, _ domain.InviteStaffInput) (*domain.StaffInviteResult, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeStaffService) List(_ context.Context, _ uuid.UUID) ([]*domain.StaffMember, error) {
	return nil, nil
}
func (f *fakeStaffService) Update(_ context.Context, _, _, _ uuid.UUID, _ domain.UpdateStaffInput) (*domain.StaffMember, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeStaffService) Deactivate(_ context.Context, _, _, _ uuid.UUID) error {
	return domain.ErrNotFound
}
func (f *fakeStaffService) Reactivate(_ context.Context, _, _, _ uuid.UUID) error {
	return domain.ErrNotFound
}
func (f *fakeStaffService) ResendInvite(_ context.Context, _, _, _ uuid.UUID) (string, error) {
	return "", domain.ErrNotFound
}
func (f *fakeStaffService) AuditLog(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.StaffAuditEntry, int, error) {
	return nil, 0, nil
}
func (f *fakeStaffService) ResetPassword(_ context.Context, _, _, _ uuid.UUID) (string, error) {
	return "", domain.ErrNotFound
}
func (f *fakeStaffService) LookupInvite(_ context.Context, _ string) (*domain.InviteLookup, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeStaffService) AcceptInvite(_ context.Context, _ domain.AcceptInviteInput) error {
	return domain.ErrNotFound
}
func (f *fakeStaffService) LookupPasswordReset(_ context.Context, _ string) (*domain.PasswordResetLookup, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeStaffService) ConfirmPasswordReset(_ context.Context, _ domain.ConfirmPasswordResetInput) error {
	return domain.ErrNotFound
}
func (f *fakeStaffService) ResolveAccess(_ context.Context, _, _ uuid.UUID) (domain.PropertyAccess, error) {
	return domain.AllPropertyAccess(), nil
}

func newTestRouter() http.Handler {
	authSvc := &fakeAuthService{userID: uuid.New()}
	propertySvc := &fakePropertyService{}
	unitSvc := &fakeUnitService{}
	staffSvc := &fakeStaffService{}

	return transporthttp.NewRouter(transporthttp.RouterConfig{
		Logger:          slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
		AllowedOrigins:  []string{"http://localhost:5173"},
		AuthService:     authSvc,
		PropertyService: propertySvc,
		UnitService:     unitSvc,
		StaffService:    staffSvc,
		AuthHandler:     handlers.NewAuthHandler(authSvc, 15*time.Minute, time.Hour, "", false),
		HealthHandler:   handlers.NewHealthHandler(alwaysUpPinger{}, alwaysUpPinger{}),
		VersionHandler:  handlers.NewVersionHandler("test-version", "test-commit", "test-build-date"),
	})
}

type alwaysUpPinger struct{}

func (alwaysUpPinger) Ping(context.Context) error { return nil }

func TestRouter_Healthz(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz error = %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

func TestRouter_Version(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/version")
	if err != nil {
		t.Fatalf("GET /version error = %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /version status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body struct {
		Data struct {
			Version string `json:"version"`
			Commit  string `json:"commit"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Version != "test-version" || body.Data.Commit != "test-commit" {
		t.Errorf("GET /version body = %+v, want version=test-version commit=test-commit", body.Data)
	}
}

func TestRouter_Properties_RequiresAuth(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/v1/properties")
	if err != nil {
		t.Fatalf("GET /api/v1/properties error = %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/v1/properties (no token) status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}

	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "unauthorized" {
		t.Errorf("GET /api/v1/properties (no token) code = %q, want %q", body.Code, "unauthorized")
	}
	if body.Error == "" {
		t.Error("GET /api/v1/properties (no token) error message is empty")
	}
}

func TestRouter_Properties_Create_ValidationErrorDetails(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	defer srv.Close()

	// Empty body fails every "required" tag on CreatePropertyRequest
	// (name, type, address_line1, units), exercising the full path from
	// go-playground validator tags -> domain.ValidationErrors ->
	// response.WriteError's "details" array.
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/properties", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer fake-access-token")
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/v1/properties error = %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/v1/properties (empty body) status = %d, want %d", res.StatusCode, http.StatusBadRequest)
	}

	var body struct {
		Error   string `json:"error"`
		Code    string `json:"code"`
		Details []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "invalid_input" {
		t.Errorf("POST /api/v1/properties (empty body) code = %q, want %q", body.Code, "invalid_input")
	}
	fields := make(map[string]bool, len(body.Details))
	for _, d := range body.Details {
		fields[d.Field] = true
	}
	if !fields["name"] || !fields["type"] || !fields["address_line1"] || !fields["units"] {
		t.Errorf("POST /api/v1/properties (empty body) details = %+v, want failures for name, type, address_line1, and units", body.Details)
	}
}

func TestRouter_Properties_ListWithAuth(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/properties", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer fake-access-token")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/properties error = %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/properties (with token, no trailing slash) status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body struct {
		Data []json.RawMessage `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 0 {
		t.Errorf("expected empty property list, got %d entries", len(body.Data))
	}
}

func TestRouter_Auth_RegisterLoginFlow(t *testing.T) {
	srv := httptest.NewServer(newTestRouter())
	defer srv.Close()

	loginBody, _ := json.Marshal(map[string]string{"email": "owner@example.com", "password": "hunter22222"})
	res, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("POST /api/v1/auth/login error = %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/v1/auth/login status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var cookieFound bool
	for _, c := range res.Cookies() {
		if c.Name == "refresh_token" {
			cookieFound = true
			if !c.HttpOnly {
				t.Error("refresh_token cookie is not HttpOnly")
			}
		}
	}
	if !cookieFound {
		t.Error("login response did not set a refresh_token cookie")
	}

	var body struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.AccessToken == "" {
		t.Error("login response did not include an access_token")
	}
}
