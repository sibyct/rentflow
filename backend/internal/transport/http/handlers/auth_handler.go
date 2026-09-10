package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/response"
)

const refreshCookieName = "refresh_token"
const refreshCookiePath = "/api/v1/auth"

type AuthHandler struct {
	svc          domain.AuthService
	accessTTL    time.Duration
	refreshTTL   time.Duration
	cookieDomain string
	cookieSecure bool
}

func NewAuthHandler(svc domain.AuthService, accessTTL, refreshTTL time.Duration, cookieDomain string, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		svc:          svc,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		cookieDomain: cookieDomain,
		cookieSecure: cookieSecure,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	u, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewUserResponse(u))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, refresh, user, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	h.respondWithTokenPair(w, access, refresh, user)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("refresh: missing refresh cookie: %w", domain.ErrUnauthorized))
		return
	}

	access, refresh, user, err := h.svc.RefreshToken(r.Context(), cookie.Value)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	h.respondWithTokenPair(w, access, refresh, user)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		if err := h.svc.Logout(r.Context(), cookie.Value); err != nil && !errors.Is(err, domain.ErrUnauthorized) {
			response.WriteError(w, r, err)
			return
		}
	}

	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// respondWithTokenPair returns the access token in the JSON body (the
// client keeps it in memory only) and sets the refresh token as an
// httpOnly, SameSite=Strict cookie scoped to the auth routes so it is
// never exposed to page JavaScript and is not sent on unrelated requests.
func (h *AuthHandler) respondWithTokenPair(w http.ResponseWriter, accessToken, refreshToken string, user *domain.User) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     refreshCookiePath,
		Domain:   h.cookieDomain,
		Expires:  time.Now().Add(h.refreshTTL),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})

	response.JSON(w, http.StatusOK, dto.AuthResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(h.accessTTL.Seconds()),
		User:        dto.NewUserResponse(user),
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		Domain:   h.cookieDomain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}
