package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRoleAdmin   UserRole = "admin"
	UserRoleManager UserRole = "manager"
	UserRoleTenant  UserRole = "tenant"
)

// User is the core business entity for an account holder. PasswordHash is
// never serialized to a transport DTO.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         UserRole
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserRepository is the port implemented by internal/repository/postgres.
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

// AuthClaims is the framework-free result of validating an access token.
// The service layer translates JWT-library claims into this type so the
// domain and transport layers never import a JWT package directly.
type AuthClaims struct {
	UserID uuid.UUID
	Role   UserRole
}

// AuthService is the port implemented by internal/service and consumed by
// the HTTP transport layer (auth handler + auth middleware).
type AuthService interface {
	Register(ctx context.Context, email, password string) (*User, error)
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, user *User, err error)
	RefreshToken(ctx context.Context, oldRefreshToken string) (accessToken, refreshToken string, user *User, err error)
	Logout(ctx context.Context, refreshToken string) error
	ValidateAccessToken(ctx context.Context, tokenString string) (*AuthClaims, error)
}
