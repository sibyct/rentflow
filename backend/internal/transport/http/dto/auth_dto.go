package dto

import (
	"time"

	"propertymanagement/internal/domain"
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// Sanitize normalizes Email only. Password is deliberately left
// untouched — see the package note below.
func (r *RegisterRequest) Sanitize() {
	r.Email = sanitizeEmail(r.Email)
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (r *LoginRequest) Sanitize() {
	r.Email = sanitizeEmail(r.Email)
}

// Password is never sanitized (trimmed, case-normalized, etc.):
// whitespace or exact casing in a password is part of what the user
// actually typed, and silently altering it would either reject a
// correct password or, worse, silently accept a materially different
// one as equivalent. This is a deliberate choice, not an oversight —
// flagged in the PR/task summary as a place where "sanitize string
// inputs" doesn't apply uniformly.

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// AuthResponse is returned on login/register/refresh. The access token is
// returned in the body for the client to hold in memory; the refresh
// token is set as an httpOnly cookie and is never present here.
type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresIn   int          `json:"expires_in"`
	User        UserResponse `json:"user"`
}

func NewUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}
