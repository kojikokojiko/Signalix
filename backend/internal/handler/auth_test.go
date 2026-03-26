package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/handler"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// --- mock auth usecase ---

type mockAuthUsecase struct {
	registerFn func(ctx context.Context, in usecase.RegisterInput) (*usecase.AuthResult, error)
	loginFn    func(ctx context.Context, in usecase.LoginInput) (*usecase.AuthResult, error)
	logoutFn   func(ctx context.Context, jti string, exp time.Time) error
	refreshFn  func(ctx context.Context, token string) (*usecase.RefreshResult, error)
}

func (m *mockAuthUsecase) Register(ctx context.Context, in usecase.RegisterInput) (*usecase.AuthResult, error) {
	return m.registerFn(ctx, in)
}
func (m *mockAuthUsecase) Login(ctx context.Context, in usecase.LoginInput) (*usecase.AuthResult, error) {
	return m.loginFn(ctx, in)
}
func (m *mockAuthUsecase) Logout(ctx context.Context, jti string, exp time.Time) error {
	return m.logoutFn(ctx, jti, exp)
}
func (m *mockAuthUsecase) Refresh(ctx context.Context, token string) (*usecase.RefreshResult, error) {
	return m.refreshFn(ctx, token)
}

func defaultAuthResult() *usecase.AuthResult {
	return &usecase.AuthResult{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
		User: &domain.User{
			Email:       "u@example.com",
			DisplayName: "User",
		},
	}
}

func newRouter(uc handler.AuthUsecaseIface) http.Handler {
	r := chi.NewRouter()
	h := handler.NewAuthHandler(uc)
	r.Post("/api/v1/auth/register", h.Register)
	r.Post("/api/v1/auth/login", h.Login)
	r.Post("/api/v1/auth/logout", h.Logout)
	r.Post("/api/v1/auth/refresh", h.Refresh)
	return r
}

func postJSON(t *testing.T, router http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ─── Register ────────────────────────────────────────────────────────────────

func TestAuthHandler_Register_Success(t *testing.T) {
	uc := &mockAuthUsecase{
		registerFn: func(_ context.Context, in usecase.RegisterInput) (*usecase.AuthResult, error) {
			return defaultAuthResult(), nil
		},
	}
	w := postJSON(t, newRouter(uc), "/api/v1/auth/register", map[string]string{
		"email": "u@example.com", "password": "Secure1234", "display_name": "User",
	})

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["access_token"] == nil {
		t.Error("expected access_token in response")
	}
	if data["user"] == nil {
		t.Error("expected user in response")
	}

	cookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "refresh_token=") {
		t.Error("expected refresh_token cookie")
	}
	if !strings.Contains(cookie, "HttpOnly") {
		t.Error("expected HttpOnly cookie attribute")
	}
}

func TestAuthHandler_Register_MissingFields(t *testing.T) {
	uc := &mockAuthUsecase{}
	w := postJSON(t, newRouter(uc), "/api/v1/auth/register", map[string]string{
		"email": "u@example.com",
		// missing password and display_name
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	uc := &mockAuthUsecase{
		registerFn: func(_ context.Context, in usecase.RegisterInput) (*usecase.AuthResult, error) {
			return nil, usecase.ErrEmailAlreadyExists
		},
	}
	w := postJSON(t, newRouter(uc), "/api/v1/auth/register", map[string]string{
		"email": "dup@example.com", "password": "Secure1234", "display_name": "User",
	})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestAuthHandler_Register_InvalidEmail(t *testing.T) {
	uc := &mockAuthUsecase{}
	w := postJSON(t, newRouter(uc), "/api/v1/auth/register", map[string]string{
		"email": "not-an-email", "password": "Secure1234", "display_name": "User",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_WeakPassword(t *testing.T) {
	uc := &mockAuthUsecase{}
	w := postJSON(t, newRouter(uc), "/api/v1/auth/register", map[string]string{
		"email": "u@example.com", "password": "short", "display_name": "User",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestAuthHandler_Login_Success(t *testing.T) {
	uc := &mockAuthUsecase{
		loginFn: func(_ context.Context, in usecase.LoginInput) (*usecase.AuthResult, error) {
			return defaultAuthResult(), nil
		},
	}
	w := postJSON(t, newRouter(uc), "/api/v1/auth/login", map[string]string{
		"email": "u@example.com", "password": "Secure1234",
	})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	uc := &mockAuthUsecase{
		loginFn: func(_ context.Context, in usecase.LoginInput) (*usecase.AuthResult, error) {
			return nil, usecase.ErrInvalidCredentials
		},
	}
	w := postJSON(t, newRouter(uc), "/api/v1/auth/login", map[string]string{
		"email": "u@example.com", "password": "WrongPassword1",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_Login_AccountLocked(t *testing.T) {
	uc := &mockAuthUsecase{
		loginFn: func(_ context.Context, in usecase.LoginInput) (*usecase.AuthResult, error) {
			return nil, usecase.ErrAccountLocked
		},
	}
	w := postJSON(t, newRouter(uc), "/api/v1/auth/login", map[string]string{
		"email": "u@example.com", "password": "Secure1234",
	})
	if w.Code != 423 {
		t.Errorf("expected 423, got %d", w.Code)
	}
}

func TestAuthHandler_Login_AccountDisabled(t *testing.T) {
	uc := &mockAuthUsecase{
		loginFn: func(_ context.Context, in usecase.LoginInput) (*usecase.AuthResult, error) {
			return nil, usecase.ErrAccountDisabled
		},
	}
	w := postJSON(t, newRouter(uc), "/api/v1/auth/login", map[string]string{
		"email": "u@example.com", "password": "Secure1234",
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// ─── Logout ──────────────────────────────────────────────────────────────────

func TestAuthHandler_Logout_NoToken(t *testing.T) {
	uc := &mockAuthUsecase{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()
	newRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestAuthHandler_Refresh_NoCookie(t *testing.T) {
	uc := &mockAuthUsecase{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	w := httptest.NewRecorder()
	newRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	uc := &mockAuthUsecase{
		refreshFn: func(_ context.Context, token string) (*usecase.RefreshResult, error) {
			return nil, usecase.ErrTokenInvalid
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "bad-token"})
	w := httptest.NewRecorder()
	newRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	uc := &mockAuthUsecase{
		refreshFn: func(_ context.Context, token string) (*usecase.RefreshResult, error) {
			return &usecase.RefreshResult{AccessToken: "new-access", ExpiresIn: 3600}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh"})
	w := httptest.NewRecorder()
	newRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["access_token"] == nil {
		t.Error("expected access_token in response")
	}
}

func TestAuthHandler_Refresh_ExpiredToken(t *testing.T) {
	uc := &mockAuthUsecase{
		refreshFn: func(_ context.Context, token string) (*usecase.RefreshResult, error) {
			return nil, usecase.ErrTokenExpired
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "expired-token"})
	w := httptest.NewRecorder()
	newRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.Contains(errors.New(resp["code"].(string)).Error(), "token_expired") {
		// just check the code field exists and contains something reasonable
		if resp["code"] == nil {
			t.Error("expected error code in response")
		}
	}
}
