//go:build integration

package integration_test

import (
	"net/http"
	"testing"
)

func TestArticles_List_Pagination(t *testing.T) {
	ts := setupTestServer(t)

	// List without authentication (public endpoint)
	resp := ts.GET("/api/v1/articles?page=1&per_page=5", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list articles: expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	decodeBody(t, resp, &body)

	if _, ok := body["data"]; !ok {
		t.Fatal("expected data field in response")
	}
	pagination, ok := body["pagination"].(map[string]any)
	if !ok {
		t.Fatal("expected pagination field in response")
	}
	if _, ok := pagination["page"]; !ok {
		t.Fatal("expected page field in pagination")
	}
	if _, ok := pagination["per_page"]; !ok {
		t.Fatal("expected per_page field in pagination")
	}
	if _, ok := pagination["total"]; !ok {
		t.Fatal("expected total field in pagination")
	}

	// Verify per_page is respected
	perPage, _ := pagination["per_page"].(float64)
	if perPage != 5 {
		t.Errorf("expected per_page=5, got %v", perPage)
	}
}

func TestArticles_GetByID_NotFound(t *testing.T) {
	ts := setupTestServer(t)

	// Use a non-existent UUID
	resp := ts.GET("/api/v1/articles/00000000-0000-0000-0000-000000000000", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestArticles_Trending(t *testing.T) {
	ts := setupTestServer(t)

	// Test 24h period
	resp24h := ts.GET("/api/v1/articles/trending?period=24h", "")
	defer resp24h.Body.Close()

	if resp24h.StatusCode != http.StatusOK {
		t.Fatalf("trending 24h: expected 200, got %d", resp24h.StatusCode)
	}

	var body24h map[string]any
	decodeBody(t, resp24h, &body24h)
	if _, ok := body24h["data"]; !ok {
		t.Fatal("trending 24h: expected data field")
	}

	// Test 7d period
	resp7d := ts.GET("/api/v1/articles/trending?period=7d", "")
	defer resp7d.Body.Close()

	if resp7d.StatusCode != http.StatusOK {
		t.Fatalf("trending 7d: expected 200, got %d", resp7d.StatusCode)
	}

	var body7d map[string]any
	decodeBody(t, resp7d, &body7d)
	if _, ok := body7d["data"]; !ok {
		t.Fatal("trending 7d: expected data field")
	}
}
