package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/handler"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

type mockSourceUC struct {
	listFn    func(ctx context.Context, in usecase.SourceListInput) (*usecase.SourceListResult, error)
	getByIDFn func(ctx context.Context, id string) (*domain.Source, error)
}

func (m *mockSourceUC) List(ctx context.Context, in usecase.SourceListInput) (*usecase.SourceListResult, error) {
	return m.listFn(ctx, in)
}
func (m *mockSourceUC) GetByID(ctx context.Context, id string) (*domain.Source, error) {
	return m.getByIDFn(ctx, id)
}

func newSourceRouter(uc handler.SourceUsecaseIface) http.Handler {
	r := chi.NewRouter()
	h := handler.NewSourceHandler(uc)
	r.Get("/api/v1/sources", h.List)
	r.Get("/api/v1/sources/{id}", h.GetByID)
	return r
}

func TestSourceHandler_List_Success(t *testing.T) {
	cnt := 5
	uc := &mockSourceUC{
		listFn: func(_ context.Context, in usecase.SourceListInput) (*usecase.SourceListResult, error) {
			return &usecase.SourceListResult{
				Sources: []*domain.Source{
					{ID: "1", Name: "Source A", ArticleCount: &cnt},
				},
				Page: 1, PerPage: 50, Total: 1, TotalPages: 1,
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil)
	w := httptest.NewRecorder()
	newSourceRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["data"] == nil {
		t.Error("expected data field")
	}
	if resp["pagination"] == nil {
		t.Error("expected pagination field")
	}
}

func TestSourceHandler_List_WithCategoryFilter(t *testing.T) {
	var capturedCategory *string
	uc := &mockSourceUC{
		listFn: func(_ context.Context, in usecase.SourceListInput) (*usecase.SourceListResult, error) {
			capturedCategory = in.Category
			return &usecase.SourceListResult{Sources: nil, Total: 0, TotalPages: 0}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources?category=tech", nil)
	w := httptest.NewRecorder()
	newSourceRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedCategory == nil || *capturedCategory != "tech" {
		t.Error("expected category=tech to be passed to usecase")
	}
}

func TestSourceHandler_GetByID_Found(t *testing.T) {
	uc := &mockSourceUC{
		getByIDFn: func(_ context.Context, id string) (*domain.Source, error) {
			return &domain.Source{ID: id, Name: "Go Blog"}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources/abc-123", nil)
	w := httptest.NewRecorder()
	newSourceRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["source"] == nil {
		t.Error("expected source in data")
	}
}

func TestSourceHandler_GetByID_NotFound(t *testing.T) {
	uc := &mockSourceUC{
		getByIDFn: func(_ context.Context, id string) (*domain.Source, error) {
			return nil, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources/unknown", nil)
	w := httptest.NewRecorder()
	newSourceRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
