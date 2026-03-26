package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kojikokojiko/signalix/internal/ctxkey"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/handler"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// --- mock admin usecase ---

type mockAdminUC struct {
	createSource *domain.Source
	createErr    error
	updateSource *domain.Source
	updateErr    error
	deleteErr    error
	jobs         []*domain.IngestionJob
	jobsTotal    int
	stats        *domain.AdminStats
	triggerJobID string
	triggerErr   error
	sources      []*domain.Source
}

func (m *mockAdminUC) CreateSource(_ context.Context, _ usecase.CreateSourceInput) (*domain.Source, error) {
	return m.createSource, m.createErr
}
func (m *mockAdminUC) UpdateSource(_ context.Context, _ string, _ usecase.UpdateSourceInput) (*domain.Source, error) {
	return m.updateSource, m.updateErr
}
func (m *mockAdminUC) DeleteSource(_ context.Context, _ string) error { return m.deleteErr }
func (m *mockAdminUC) ListAdminSources(_ context.Context, _ repository.SourceFilter) ([]*domain.Source, int, error) {
	return m.sources, len(m.sources), nil
}
func (m *mockAdminUC) TriggerFetch(_ context.Context, _ string) (string, error) {
	return m.triggerJobID, m.triggerErr
}
func (m *mockAdminUC) ListIngestionJobs(_ context.Context, _ usecase.IngestionJobListInput) ([]*domain.IngestionJob, int, error) {
	return m.jobs, m.jobsTotal, nil
}
func (m *mockAdminUC) GetStats(_ context.Context) (*domain.AdminStats, error) {
	return m.stats, nil
}

// --- helpers ---

func adminReq(method, path string, body any) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.UserID, "admin-user-id"))
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.IsAdmin, true))
	return req
}

func withURLParam(req *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// --- tests ---

func TestAdminHandler_CreateSource_Returns201(t *testing.T) {
	src := &domain.Source{ID: "src-1", Name: "Test Blog", Status: "active"}
	uc := &mockAdminUC{createSource: src}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodPost, "/api/v1/admin/sources", map[string]any{
		"name": "Test Blog", "feed_url": "https://test.com/feed.atom",
		"site_url": "https://test.com", "category": "tech", "language": "en",
	})
	rr := httptest.NewRecorder()
	h.CreateSource(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminHandler_CreateSource_Returns409OnDuplicate(t *testing.T) {
	uc := &mockAdminUC{createErr: usecase.ErrFeedURLAlreadyExists}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodPost, "/api/v1/admin/sources", map[string]any{
		"name": "X", "feed_url": "https://dup.com/feed", "site_url": "https://dup.com",
		"category": "tech", "language": "en",
	})
	rr := httptest.NewRecorder()
	h.CreateSource(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestAdminHandler_CreateSource_Returns400OnValidation(t *testing.T) {
	uc := &mockAdminUC{createErr: fmt.Errorf("%w: test", usecase.ErrValidation)}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodPost, "/api/v1/admin/sources", map[string]any{})
	rr := httptest.NewRecorder()
	h.CreateSource(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAdminHandler_UpdateSource_Returns200(t *testing.T) {
	src := &domain.Source{ID: "src-1", Name: "Updated"}
	uc := &mockAdminUC{updateSource: src}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodPatch, "/api/v1/admin/sources/src-1", map[string]any{"status": "paused"})
	req = withURLParam(req, "id", "src-1")
	rr := httptest.NewRecorder()
	h.UpdateSource(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminHandler_UpdateSource_Returns404(t *testing.T) {
	uc := &mockAdminUC{updateErr: usecase.ErrSourceNotFound}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodPatch, "/api/v1/admin/sources/none", map[string]any{})
	req = withURLParam(req, "id", "none")
	rr := httptest.NewRecorder()
	h.UpdateSource(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAdminHandler_DeleteSource_Returns204(t *testing.T) {
	uc := &mockAdminUC{}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodDelete, "/api/v1/admin/sources/src-1", nil)
	req = withURLParam(req, "id", "src-1")
	rr := httptest.NewRecorder()
	h.DeleteSource(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

func TestAdminHandler_DeleteSource_Returns404(t *testing.T) {
	uc := &mockAdminUC{deleteErr: usecase.ErrSourceNotFound}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodDelete, "/api/v1/admin/sources/none", nil)
	req = withURLParam(req, "id", "none")
	rr := httptest.NewRecorder()
	h.DeleteSource(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAdminHandler_TriggerFetch_Returns202(t *testing.T) {
	uc := &mockAdminUC{triggerJobID: "job-1"}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodPost, "/api/v1/admin/sources/src-1/fetch", nil)
	req = withURLParam(req, "id", "src-1")
	rr := httptest.NewRecorder()
	h.TriggerFetch(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
}

func TestAdminHandler_TriggerFetch_Returns404WhenSourceNotFound(t *testing.T) {
	uc := &mockAdminUC{triggerErr: usecase.ErrSourceNotFound}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodPost, "/api/v1/admin/sources/none/fetch", nil)
	req = withURLParam(req, "id", "none")
	rr := httptest.NewRecorder()
	h.TriggerFetch(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAdminHandler_ListIngestionJobs_Returns200(t *testing.T) {
	jobs := []*domain.IngestionJob{{ID: "j1", Status: "completed"}}
	uc := &mockAdminUC{jobs: jobs, jobsTotal: 1}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodGet, "/api/v1/admin/ingestion-jobs", nil)
	rr := httptest.NewRecorder()
	h.ListIngestionJobs(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if _, ok := resp["data"]; !ok {
		t.Error("expected data field")
	}
}

func TestAdminHandler_GetStats_Returns200(t *testing.T) {
	uc := &mockAdminUC{stats: &domain.AdminStats{}}
	h := handler.NewAdminHandler(uc)

	req := adminReq(http.MethodGet, "/api/v1/admin/stats", nil)
	rr := httptest.NewRecorder()
	h.GetStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// compile-time interface check
var _ = errors.New // suppress unused import
