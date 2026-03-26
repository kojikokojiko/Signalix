//go:build integration

package integration_test

import (
	"net/http"
	"testing"
)

// registerAndLogin is a helper that registers a new user and returns the access token.
func registerAndLogin(t *testing.T, ts *testServer) string {
	t.Helper()
	email := uniqueEmail("bookmark_user")
	password := "SecurePass123!"

	regResp := ts.POST("/api/v1/auth/register", map[string]any{
		"email":        email,
		"password":     password,
		"display_name": "Bookmark Test User",
	}, "")
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regResp.StatusCode)
	}
	var body map[string]any
	decodeBody(t, regResp, &body)
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatal("register: expected data field")
	}
	token, ok := data["access_token"].(string)
	if !ok || token == "" {
		t.Fatal("register: expected access_token")
	}
	return token
}

// getFirstArticleID fetches the first available article ID.
func getFirstArticleID(t *testing.T, ts *testServer) string {
	t.Helper()
	resp := ts.GET("/api/v1/articles?per_page=1", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skip("no articles available, skipping bookmark test")
		return ""
	}

	var body map[string]any
	decodeBody(t, resp, &body)

	items, ok := body["data"].([]any)
	if !ok || len(items) == 0 {
		t.Skip("no articles available, skipping bookmark test")
		return ""
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatal("unexpected article format")
	}
	id, ok := item["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected article id")
	}
	return id
}

func TestBookmark_AddRemoveList(t *testing.T) {
	ts := setupTestServer(t)
	token := registerAndLogin(t, ts)
	articleID := getFirstArticleID(t, ts)

	// List bookmarks (initially empty)
	listResp := ts.GET("/api/v1/bookmarks", token)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list bookmarks: expected 200, got %d", listResp.StatusCode)
	}
	var listBody map[string]any
	decodeBody(t, listResp, &listBody)
	initialData, ok := listBody["data"].([]any)
	if !ok {
		t.Fatal("list bookmarks: expected data array")
	}
	initialCount := len(initialData)

	// Add bookmark
	addResp := ts.POST("/api/v1/bookmarks", map[string]any{
		"article_id": articleID,
	}, token)
	if addResp.StatusCode != http.StatusCreated {
		t.Fatalf("add bookmark: expected 201, got %d", addResp.StatusCode)
	}
	addResp.Body.Close()

	// List bookmarks (should have one more)
	listResp2 := ts.GET("/api/v1/bookmarks", token)
	if listResp2.StatusCode != http.StatusOK {
		t.Fatalf("list after add: expected 200, got %d", listResp2.StatusCode)
	}
	var listBody2 map[string]any
	decodeBody(t, listResp2, &listBody2)
	data2, ok := listBody2["data"].([]any)
	if !ok {
		t.Fatal("list after add: expected data array")
	}
	if len(data2) != initialCount+1 {
		t.Errorf("list after add: expected %d bookmarks, got %d", initialCount+1, len(data2))
	}

	// Remove bookmark
	deleteResp := ts.DELETE("/api/v1/bookmarks/"+articleID, token)
	if deleteResp.StatusCode != http.StatusNoContent && deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("remove bookmark: expected 204 or 200, got %d", deleteResp.StatusCode)
	}
	deleteResp.Body.Close()

	// List bookmarks (should be back to initial count)
	listResp3 := ts.GET("/api/v1/bookmarks", token)
	if listResp3.StatusCode != http.StatusOK {
		t.Fatalf("list after remove: expected 200, got %d", listResp3.StatusCode)
	}
	var listBody3 map[string]any
	decodeBody(t, listResp3, &listBody3)
	data3, ok := listBody3["data"].([]any)
	if !ok {
		t.Fatal("list after remove: expected data array")
	}
	if len(data3) != initialCount {
		t.Errorf("list after remove: expected %d bookmarks, got %d", initialCount, len(data3))
	}
}

func TestBookmark_AlreadyBookmarked(t *testing.T) {
	ts := setupTestServer(t)
	token := registerAndLogin(t, ts)
	articleID := getFirstArticleID(t, ts)

	// Add bookmark
	addResp1 := ts.POST("/api/v1/bookmarks", map[string]any{
		"article_id": articleID,
	}, token)
	if addResp1.StatusCode != http.StatusCreated {
		t.Fatalf("first add: expected 201, got %d", addResp1.StatusCode)
	}
	addResp1.Body.Close()

	// Add same bookmark again - should return 409
	addResp2 := ts.POST("/api/v1/bookmarks", map[string]any{
		"article_id": articleID,
	}, token)
	defer addResp2.Body.Close()

	if addResp2.StatusCode != http.StatusConflict {
		t.Errorf("duplicate bookmark: expected 409, got %d", addResp2.StatusCode)
	}
}

func TestBookmark_Unauthenticated(t *testing.T) {
	ts := setupTestServer(t)

	// GET bookmarks without token
	getResp := ts.GET("/api/v1/bookmarks", "")
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET bookmarks: expected 401, got %d", getResp.StatusCode)
	}

	// POST bookmark without token
	postResp := ts.POST("/api/v1/bookmarks", map[string]any{
		"article_id": "00000000-0000-0000-0000-000000000001",
	}, "")
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("POST bookmarks: expected 401, got %d", postResp.StatusCode)
	}
}
