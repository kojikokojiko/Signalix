//go:build integration

package integration_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestAuthFlow_RegisterLoginRefreshLogout(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail("auth_flow")
	password := "SecurePass123!"
	displayName := "Integration User"

	// Register
	regResp := ts.POST("/api/v1/auth/register", map[string]any{
		"email":        email,
		"password":     password,
		"display_name": displayName,
	}, "")
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regResp.StatusCode)
	}

	var regBody map[string]any
	decodeBody(t, regResp, &regBody)
	data, ok := regBody["data"].(map[string]any)
	if !ok {
		t.Fatal("register: expected data field")
	}
	accessToken, ok := data["access_token"].(string)
	if !ok || accessToken == "" {
		t.Fatal("register: expected access_token")
	}

	// Get refresh cookie
	var refreshCookie *http.Cookie
	for _, c := range regResp.Cookies() {
		if c.Name == "refresh_token" {
			refreshCookie = c
			break
		}
	}
	if refreshCookie == nil {
		t.Fatal("register: expected refresh_token cookie")
	}

	// Login
	loginResp := ts.POST("/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": password,
	}, "")
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", loginResp.StatusCode)
	}

	var loginBody map[string]any
	decodeBody(t, loginResp, &loginBody)
	loginData, ok := loginBody["data"].(map[string]any)
	if !ok {
		t.Fatal("login: expected data field")
	}
	loginToken, ok := loginData["access_token"].(string)
	if !ok || loginToken == "" {
		t.Fatal("login: expected access_token")
	}

	// Get new refresh cookie from login
	var loginRefreshCookie *http.Cookie
	for _, c := range loginResp.Cookies() {
		if c.Name == "refresh_token" {
			loginRefreshCookie = c
			break
		}
	}
	if loginRefreshCookie == nil {
		t.Fatal("login: expected refresh_token cookie")
	}

	// Refresh
	refreshResp := ts.POSTWithCookie("/api/v1/auth/refresh", nil, loginRefreshCookie)
	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d", refreshResp.StatusCode)
	}

	var refreshBody map[string]any
	decodeBody(t, refreshResp, &refreshBody)
	refreshData, ok := refreshBody["data"].(map[string]any)
	if !ok {
		t.Fatal("refresh: expected data field")
	}
	newToken, ok := refreshData["access_token"].(string)
	if !ok || newToken == "" {
		t.Fatal("refresh: expected new access_token")
	}

	// Logout
	logoutResp := ts.POST("/api/v1/auth/logout", nil, newToken)
	if logoutResp.StatusCode != http.StatusOK && logoutResp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: expected 200 or 204, got %d", logoutResp.StatusCode)
	}
}

func TestLogin_BruteForce_AccountLock(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail("brute_force")
	password := "SecurePass123!"

	// Register first
	regResp := ts.POST("/api/v1/auth/register", map[string]any{
		"email":        email,
		"password":     password,
		"display_name": "Brute Force Test",
	}, "")
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regResp.StatusCode)
	}
	regResp.Body.Close()

	// Attempt login with wrong password 5 times
	for i := 0; i < 5; i++ {
		resp := ts.POST("/api/v1/auth/login", map[string]any{
			"email":    email,
			"password": fmt.Sprintf("wrong_pass_%d", i),
		}, "")
		resp.Body.Close()
		// First 4 should be 401, 5th may trigger lock
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode != 423 {
			t.Logf("attempt %d: got %d (expected 401 or 423)", i+1, resp.StatusCode)
		}
	}

	// 6th attempt should be locked (423)
	lockedResp := ts.POST("/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": "any_password",
	}, "")
	defer lockedResp.Body.Close()

	if lockedResp.StatusCode != 423 && lockedResp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("brute force: expected 423 or 429 after 5 failed attempts, got %d", lockedResp.StatusCode)
	}
}

func TestLogin_DuplicateEmail(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail("duplicate_email")

	// Register once
	resp1 := ts.POST("/api/v1/auth/register", map[string]any{
		"email":        email,
		"password":     "SecurePass123!",
		"display_name": "First User",
	}, "")
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d", resp1.StatusCode)
	}
	resp1.Body.Close()

	// Register again with the same email
	resp2 := ts.POST("/api/v1/auth/register", map[string]any{
		"email":        email,
		"password":     "AnotherPass456!",
		"display_name": "Second User",
	}, "")
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("duplicate email: expected 409, got %d", resp2.StatusCode)
	}
}

func TestAuth_UnauthorizedWithoutToken(t *testing.T) {
	ts := setupTestServer(t)

	// Access authenticated endpoint without token
	resp := ts.GET("/api/v1/recommendations", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}
