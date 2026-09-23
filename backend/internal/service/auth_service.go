package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"propertymanagement/internal/domain"
)

// authClaims is the JWT-library-specific claims shape. It stays private
// to the service package; callers only ever see domain.AuthClaims.
//
// UserID is the ACCOUNT id (see domain.AuthClaims's doc comment) —
// always the signed-in user's own id for a root account, and always the
// root's id for staff, so every existing ownership check elsewhere in
// the codebase keeps working unchanged now that staff logins exist.
type authClaims struct {
	UserID    uuid.UUID         `json:"user_id"`
	Role      domain.UserRole   `json:"role"`
	ActorID   uuid.UUID         `json:"actor_id"`
	StaffRole *domain.StaffRole `json:"staff_role,omitempty"`
	jwt.RegisteredClaims
}

type AuthService struct {
	users domain.UserRepository
	cache domain.Cache // may be nil; used only to support immediate refresh-token revocation

	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewAuthService(users domain.UserRepository, cache domain.Cache, accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		users:         users,
		cache:         cache,
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

var _ domain.AuthService = (*AuthService)(nil)

const minPasswordLength = 8

func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	var verrs domain.ValidationErrors
	if email == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "email", Message: "is required"})
	}
	if password == "" {
		verrs = append(verrs, &domain.ValidationError{Field: "password", Message: "is required"})
	} else if len(password) < minPasswordLength {
		verrs = append(verrs, &domain.ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("must be at least %d characters", minPasswordLength),
		})
	}
	if len(verrs) > 0 {
		return nil, fmt.Errorf("register %s: %w", email, verrs)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("register %s: hash password: %w", email, err)
	}

	now := time.Now().UTC()
	u := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.UserRoleManager,
		// A self-registered user is always a root account (AccountOwnerID
		// stays nil — see domain.User.AccountID) and always active: there
		// is no invite step in this path.
		Status:    domain.StaffStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, *domain.User, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", "", nil, fmt.Errorf("login %s: %w", email, domain.ErrUnauthorized)
		}
		return "", "", nil, fmt.Errorf("login %s: %w", email, err)
	}

	if u.Status == domain.StaffStatusDeactivated {
		return "", "", nil, fmt.Errorf("login %s: account deactivated: %w", email, domain.ErrUnauthorized)
	}
	if u.Status == domain.StaffStatusInvited {
		// No usable password has been set yet — accepting the invite is
		// what sets one (see StaffService.AcceptInvite).
		return "", "", nil, fmt.Errorf("login %s: invite not yet accepted: %w", email, domain.ErrUnauthorized)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", "", nil, fmt.Errorf("login %s: wrong password: %w", email, domain.ErrUnauthorized)
	}

	loginAt := time.Now().UTC()
	_ = s.users.TouchLastLogin(ctx, u.ID, loginAt) // best-effort — never fails a login
	u.LastLoginAt = &loginAt

	access, refresh, err := s.issueTokenPair(u)
	if err != nil {
		return "", "", nil, fmt.Errorf("login %s: %w", email, err)
	}
	return access, refresh, u, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, string, *domain.User, error) {
	claims, err := s.parseToken(refreshToken, s.refreshSecret)
	if err != nil {
		return "", "", nil, fmt.Errorf("refresh token: %w: %w", domain.ErrUnauthorized, err)
	}

	if s.cache != nil {
		revoked, err := s.cache.Get(ctx, revokedTokenKey(claims.ID))
		if err == nil && revoked != "" {
			return "", "", nil, fmt.Errorf("refresh token %s: already revoked: %w", claims.ID, domain.ErrUnauthorized)
		}
	}

	// By ActorID, not UserID (the account id) — for a staff login those
	// differ, and it's the staff member's own row whose current status
	// (still active? still the role it was issued with?) needs rechecking.
	u, err := s.users.GetByID(ctx, claims.ActorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", "", nil, fmt.Errorf("refresh token: user %s: %w", claims.ActorID, domain.ErrUnauthorized)
		}
		return "", "", nil, fmt.Errorf("refresh token: %w", err)
	}
	if u.Status != domain.StaffStatusActive {
		return "", "", nil, fmt.Errorf("refresh token: user %s: not active: %w", claims.ActorID, domain.ErrUnauthorized)
	}

	// Rotate: the presented refresh token is single-use.
	s.revokeToken(ctx, claims)

	access, refresh, err := s.issueTokenPair(u)
	if err != nil {
		return "", "", nil, fmt.Errorf("refresh token: %w", err)
	}
	return access, refresh, u, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.parseToken(refreshToken, s.refreshSecret)
	if err != nil {
		// Already invalid/expired — logout is idempotent either way.
		return nil
	}
	s.revokeToken(ctx, claims)
	return nil
}

func (s *AuthService) ValidateAccessToken(_ context.Context, tokenString string) (*domain.AuthClaims, error) {
	claims, err := s.parseToken(tokenString, s.accessSecret)
	if err != nil {
		return nil, fmt.Errorf("validate access token: %w: %w", domain.ErrUnauthorized, err)
	}
	return &domain.AuthClaims{UserID: claims.UserID, Role: claims.Role, ActorID: claims.ActorID, StaffRole: claims.StaffRole}, nil
}

func (s *AuthService) issueTokenPair(u *domain.User) (string, string, error) {
	now := time.Now().UTC()
	accountID := u.AccountID()

	access := authClaims{
		UserID:    accountID,
		Role:      u.Role,
		ActorID:   u.ID,
		StaffRole: u.StaffRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, access).SignedString(s.accessSecret)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	refresh := authClaims{
		UserID:    accountID,
		Role:      u.Role,
		ActorID:   u.ID,
		StaffRole: u.StaffRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refresh).SignedString(s.refreshSecret)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) parseToken(tokenString string, secret []byte) (*authClaims, error) {
	claims := &authClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("token is invalid")
	}
	return claims, nil
}

// revokeToken records a refresh token's jti as revoked until its natural
// expiry, so RefreshToken/Logout take effect immediately instead of
// waiting out the token's remaining lifetime. Best-effort: if the cache
// is unavailable, rotation still proceeds on JWT expiry alone.
func (s *AuthService) revokeToken(ctx context.Context, claims *authClaims) {
	if s.cache == nil || claims.ExpiresAt == nil {
		return
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return
	}
	_ = s.cache.Set(ctx, revokedTokenKey(claims.ID), "1", ttl)
}

func revokedTokenKey(jti string) string {
	return fmt.Sprintf("revoked_refresh_token:%s", jti)
}
