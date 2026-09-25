package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/reqctx"
	"propertymanagement/internal/transport/http/response"
)

// Authenticate requires a valid "Authorization: Bearer <access-token>"
// header, validates it via the injected AuthService, and stores the
// resulting claims in context. It rejects the request with 401 otherwise.
func Authenticate(authSvc domain.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				response.WriteError(w, r, fmt.Errorf("authenticate %s: missing bearer token: %w", r.URL.Path, domain.ErrUnauthorized))
				return
			}

			claims, err := authSvc.ValidateAccessToken(r.Context(), token)
			if err != nil {
				// err is already wrapped with domain.ErrUnauthorized by
				// AuthService.ValidateAccessToken — pass it straight through
				// so WriteError's log line keeps the real underlying cause
				// (expired vs. malformed vs. wrong signature) instead of a
				// flattened generic message.
				response.WriteError(w, r, fmt.Errorf("authenticate %s: %w", r.URL.Path, err))
				return
			}

			ctx := reqctx.WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ResolvePropertyAccess computes the signed-in user's current property
// scope via StaffService.ResolveAccess and stores it in context. It
// must run after Authenticate (reads claims.UserID/ActorID) and before
// any handler that checks PropertyAccessFromContext. Resolved fresh
// every request — never cached on the JWT — since a staff member's
// scope can change between requests.
func ResolvePropertyAccess(staffSvc domain.StaffService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				response.WriteError(w, r, fmt.Errorf("resolve property access %s: %w", r.URL.Path, domain.ErrUnauthorized))
				return
			}

			access, err := staffSvc.ResolveAccess(r.Context(), claims.UserID, claims.ActorID)
			if err != nil {
				response.WriteError(w, r, fmt.Errorf("resolve property access %s: %w", r.URL.Path, err))
				return
			}

			ctx := reqctx.WithPropertyAccess(r.Context(), access)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole restricts a route to one of the given roles. It must run
// after Authenticate.
func RequireRole(roles ...domain.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[domain.UserRole]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok || !allowed[claims.Role] {
				response.WriteError(w, r, fmt.Errorf("authorize %s: %w", r.URL.Path, domain.ErrForbidden))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAccountAdmin restricts a route to whoever may manage staff on
// this account — the root account owner always, or a staff member with
// the Admin role (see domain.AuthClaims.CanManageStaff). Unlike
// RequireRole, this checks the newer per-account staff role, not the
// legacy coarse UserRole.
func RequireAccountAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || !claims.CanManageStaff() {
			response.WriteError(w, r, fmt.Errorf("authorize %s: %w", r.URL.Path, domain.ErrForbidden))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ClaimsFromContext(ctx context.Context) (*domain.AuthClaims, bool) {
	return reqctx.Claims(ctx)
}

func PropertyAccessFromContext(ctx context.Context) (domain.PropertyAccess, bool) {
	return reqctx.PropertyAccess(ctx)
}
