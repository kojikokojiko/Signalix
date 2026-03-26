//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/kojikokojiko/signalix/internal/config"
	"github.com/kojikokojiko/signalix/internal/db"
	"github.com/kojikokojiko/signalix/internal/server"
)

const (
	defaultDatabaseURL = "postgres://signalix:dev_password@localhost:5432/signalix_dev?sslmode=disable"
	defaultRedisURL    = "redis://localhost:6379"
)

type testServer struct {
	srv    *httptest.Server
	client *http.Client
}

func setupTestServer(t *testing.T) *testServer {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = defaultRedisURL
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "test-integration-secret-key"
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("failed to parse redis url: %v", err)
	}
	rdb := redis.NewClient(opt)
	t.Cleanup(func() { rdb.Close() })

	// Ping to verify connections
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("redis ping failed: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		APIPort:     "8080",
		DatabaseURL: databaseURL,
		RedisURL:    redisURL,
		JWTSecret:   jwtSecret,
	}

	srv := server.New(cfg, pool, rdb, logger)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	return &testServer{
		srv:    ts,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (ts *testServer) url(path string) string {
	return fmt.Sprintf("%s%s", ts.srv.URL, path)
}

func (ts *testServer) POST(path string, body any, token string) *http.Response {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, ts.url(path), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := ts.client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("POST %s failed: %v", path, err))
	}
	return resp
}

func (ts *testServer) GET(path string, token string) *http.Response {
	req, _ := http.NewRequest(http.MethodGet, ts.url(path), nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := ts.client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("GET %s failed: %v", path, err))
	}
	return resp
}

func (ts *testServer) DELETE(path string, token string) *http.Response {
	req, _ := http.NewRequest(http.MethodDelete, ts.url(path), nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := ts.client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("DELETE %s failed: %v", path, err))
	}
	return resp
}

func (ts *testServer) POSTWithCookie(path string, body any, cookie *http.Cookie) *http.Response {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, ts.url(path), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := ts.client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("POST %s failed: %v", path, err))
	}
	return resp
}

// decodeBody decodes the JSON body of a response into v.
func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}

// uniqueEmail returns a unique email using the current time to avoid conflicts.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s_%d@integration-test.example.com", prefix, time.Now().UnixNano())
}
