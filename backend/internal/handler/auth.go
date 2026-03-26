package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/kojikokojiko/signalix/internal/usecase"
)

// AuthUsecaseIface is the subset of AuthUsecase used by the handler.
type AuthUsecaseIface interface {
	Register(ctx context.Context, in usecase.RegisterInput) (*usecase.AuthResult, error)
	Login(ctx context.Context, in usecase.LoginInput) (*usecase.AuthResult, error)
	Logout(ctx context.Context, jti string, exp time.Time) error
	Refresh(ctx context.Context, token string) (*usecase.RefreshResult, error)
}

type AuthHandler struct {
	uc AuthUsecaseIface
}

func NewAuthHandler(uc AuthUsecaseIface) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// ─── Register ────────────────────────────────────────────────────────────────

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}

	if err := validateRegister(req); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	result, err := h.uc.Register(r.Context(), usecase.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrEmailAlreadyExists) {
			respondError(w, http.StatusConflict, "email_already_exists", "email already in use")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "registration failed")
		return
	}

	setRefreshTokenCookie(w, result.RefreshToken, int(7*24*time.Hour/time.Second))
	respondJSON(w, http.StatusCreated, map[string]any{
		"data": authResponseData(result),
	})
}

// ─── Login ───────────────────────────────────────────────────────────────────

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "email and password are required")
		return
	}

	result, err := h.uc.Login(r.Context(), usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidCredentials):
			respondError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		case errors.Is(err, usecase.ErrAccountLocked):
			respondError(w, http.StatusForbidden, "account_locked", "account is temporarily locked")
		case errors.Is(err, usecase.ErrAccountDisabled):
			respondError(w, http.StatusForbidden, "account_disabled", "account is disabled")
		default:
			respondError(w, http.StatusInternalServerError, "internal_error", "login failed")
		}
		return
	}

	setRefreshTokenCookie(w, result.RefreshToken, int(7*24*time.Hour/time.Second))
	respondJSON(w, http.StatusOK, map[string]any{
		"data": authResponseData(result),
	})
}

// ─── Logout ──────────────────────────────────────────────────────────────────

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		respondError(w, http.StatusUnauthorized, "unauthorized", "missing authorization header")
		return
	}

	// The JWT middleware (added later) will provide claims via context.
	// For now, the handler just clears the cookie and returns 204.
	clearRefreshTokenCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", "refresh token not found")
		return
	}

	result, err := h.uc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrTokenExpired):
			respondError(w, http.StatusUnauthorized, "token_expired", "refresh token has expired")
		default:
			respondError(w, http.StatusUnauthorized, "token_invalid", "refresh token is invalid")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"access_token": result.AccessToken,
			"token_type":   "Bearer",
			"expires_in":   result.ExpiresIn,
		},
	})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func authResponseData(result *usecase.AuthResult) map[string]any {
	return map[string]any{
		"access_token": result.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   result.ExpiresIn,
		"user": map[string]any{
			"id":           result.User.ID,
			"email":        result.User.Email,
			"display_name": result.User.DisplayName,
			"is_admin":     result.User.IsAdmin,
			"created_at":   result.User.CreatedAt,
		},
	}
}

func setRefreshTokenCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
		Path:     "/api/v1/auth/refresh",
	})
}

func clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   0,
		Path:     "/api/v1/auth/refresh",
	})
}

func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	RespondError(w, status, code, message)
}

// RespondError is exported for use by middleware packages.
func RespondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
	})
}

func validateRegister(req registerRequest) error {
	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		return errors.New("email, password, and display_name are required")
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return errors.New("invalid email format")
	}
	if len(req.Email) > 255 {
		return errors.New("email too long")
	}
	if err := validatePassword(req.Password); err != nil {
		return err
	}
	if len(req.DisplayName) < 1 || len(req.DisplayName) > 50 {
		return errors.New("display_name must be 1-50 characters")
	}
	return nil
}

func validatePassword(pw string) error {
	if len(pw) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	var hasLetter, hasDigit bool
	for _, c := range pw {
		if unicode.IsLetter(c) {
			hasLetter = true
		}
		if unicode.IsDigit(c) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("password must contain at least one letter and one digit")
	}
	return nil
}
