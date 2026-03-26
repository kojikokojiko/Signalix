package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/ctxkey"
	"github.com/kojikokojiko/signalix/internal/middleware"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

const testSecret = "test-jwt-secret"

// makeToken issues a signed JWT for tests.
func makeToken(t *testing.T, userID string, email string, isAdmin bool, tokenType string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      userID,
		"jti":      uuid.New().String(),
		"email":    email,
		"is_admin": isAdmin,
		"type":     tokenType,
		"iat":      now.Unix(),
		"exp":      now.Add(ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	return signed
}

func newAuthUsecase() *usecase.AuthUsecase {
	return usecase.NewAuthUsecase(nil, nil, testSecret, time.Hour, 7*24*time.Hour)
}

// okHandler records that it was called and returns 200.
func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func decodeError(t *testing.T, body []byte) map[string]string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decodeError: %v", err)
	}
	return m
}

// ─── Authenticate middleware ──────────────────────────────────────────────────

func TestAuthenticate_NoHeader(t *testing.T) {
	uc := newAuthUsecase()
	called := false
	handler := middleware.Authenticate(uc)(okHandler(&called))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called")
	}
	m := decodeError(t, w.Body.Bytes())
	if m["code"] != "unauthorized" {
		t.Errorf("want code=unauthorized, got %q", m["code"])
	}
}

func TestAuthenticate_InvalidPrefix(t *testing.T) {
	uc := newAuthUsecase()
	called := false
	handler := middleware.Authenticate(uc)(okHandler(&called))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called")
	}
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	uc := newAuthUsecase()
	called := false
	handler := middleware.Authenticate(uc)(okHandler(&called))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer this.is.not.valid")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called")
	}
}

func TestAuthenticate_ExpiredToken(t *testing.T) {
	uc := newAuthUsecase()
	called := false
	handler := middleware.Authenticate(uc)(okHandler(&called))

	token := makeToken(t, uuid.New().String(), "u@example.com", false, "access", -time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called")
	}
}

func TestAuthenticate_RefreshTokenRejected(t *testing.T) {
	uc := newAuthUsecase()
	called := false
	handler := middleware.Authenticate(uc)(okHandler(&called))

	// refresh token (type="refresh") must be rejected by ParseAccessToken
	token := makeToken(t, uuid.New().String(), "u@example.com", false, "refresh", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called with refresh token")
	}
}

func TestAuthenticate_ValidToken_SetsContext(t *testing.T) {
	uc := newAuthUsecase()

	userID := uuid.New().String()
	email := "alice@example.com"
	var capturedCtx context.Context

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.Authenticate(uc)(next)

	token := makeToken(t, userID, email, false, "access", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	if got := capturedCtx.Value(ctxkey.UserID); got != userID {
		t.Errorf("UserID in ctx = %v, want %v", got, userID)
	}
	if got := capturedCtx.Value(ctxkey.Email); got != email {
		t.Errorf("Email in ctx = %v, want %v", got, email)
	}
	if got, _ := capturedCtx.Value(ctxkey.IsAdmin).(bool); got != false {
		t.Errorf("IsAdmin in ctx = %v, want false", got)
	}
}

func TestAuthenticate_AdminToken_SetsIsAdmin(t *testing.T) {
	uc := newAuthUsecase()

	var capturedCtx context.Context
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.Authenticate(uc)(next)

	token := makeToken(t, uuid.New().String(), "admin@example.com", true, "access", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if got, _ := capturedCtx.Value(ctxkey.IsAdmin).(bool); !got {
		t.Error("IsAdmin should be true for admin token")
	}
}

// ─── RequireAdmin middleware ──────────────────────────────────────────────────

func TestRequireAdmin_NonAdmin(t *testing.T) {
	called := false
	handler := middleware.RequireAdmin(okHandler(&called))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// IsAdmin not set in context (defaults to false)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called for non-admin")
	}
	m := decodeError(t, w.Body.Bytes())
	if m["code"] != "forbidden" {
		t.Errorf("want code=forbidden, got %q", m["code"])
	}
}

func TestRequireAdmin_ExplicitFalse(t *testing.T) {
	called := false
	handler := middleware.RequireAdmin(okHandler(&called))

	ctx := context.WithValue(context.Background(), ctxkey.IsAdmin, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called")
	}
}

func TestRequireAdmin_Admin(t *testing.T) {
	called := false
	handler := middleware.RequireAdmin(okHandler(&called))

	ctx := context.WithValue(context.Background(), ctxkey.IsAdmin, true)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !called {
		t.Error("next handler should be called for admin")
	}
}
