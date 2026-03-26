package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/handler"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

type mockArticleUC struct {
	listFn     func(ctx context.Context, in usecase.ArticleListInput) (*usecase.ArticleListResult, error)
	getByIDFn  func(ctx context.Context, id uuid.UUID) (*domain.ArticleWithDetails, error)
	trendingFn func(ctx context.Context, in usecase.TrendingInput) (*usecase.TrendingResult, error)
}

func (m *mockArticleUC) List(ctx context.Context, in usecase.ArticleListInput) (*usecase.ArticleListResult, error) {
	return m.listFn(ctx, in)
}
func (m *mockArticleUC) GetByID(ctx context.Context, id uuid.UUID) (*domain.ArticleWithDetails, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockArticleUC) Trending(ctx context.Context, in usecase.TrendingInput) (*usecase.TrendingResult, error) {
	return m.trendingFn(ctx, in)
}

func newArticleRouter(uc handler.ArticleUsecaseIface) http.Handler {
	r := chi.NewRouter()
	h := handler.NewArticleHandler(uc)
	r.Get("/api/v1/articles/trending", h.Trending)
	r.Get("/api/v1/articles/{id}", h.GetByID)
	r.Get("/api/v1/articles", h.List)
	return r
}

func sampleArticle() *domain.ArticleWithDetails {
	now := time.Now().UTC()
	return &domain.ArticleWithDetails{
		Article: domain.Article{
			ID:          uuid.New(),
			Title:       "Test Article",
			URL:         "https://example.com/test",
			Status:      "processed",
			PublishedAt: &now,
		},
		Source: &domain.Source{ID: "src-1", Name: "Source", SiteURL: "https://example.com"},
	}
}

func TestArticleHandler_List_Success(t *testing.T) {
	uc := &mockArticleUC{
		listFn: func(_ context.Context, _ usecase.ArticleListInput) (*usecase.ArticleListResult, error) {
			return &usecase.ArticleListResult{
				Articles: []*domain.ArticleWithDetails{sampleArticle()},
				Page: 1, PerPage: 20, Total: 1, TotalPages: 1,
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	w := httptest.NewRecorder()
	newArticleRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["data"] == nil || resp["pagination"] == nil {
		t.Error("expected data and pagination in response")
	}
}

func TestArticleHandler_List_InvalidSourceID(t *testing.T) {
	uc := &mockArticleUC{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles?source_id=not-a-uuid", nil)
	w := httptest.NewRecorder()
	newArticleRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestArticleHandler_GetByID_Found(t *testing.T) {
	a := sampleArticle()
	uc := &mockArticleUC{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.ArticleWithDetails, error) {
			return a, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles/"+a.Article.ID.String(), nil)
	w := httptest.NewRecorder()
	newArticleRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["data"] == nil {
		t.Error("expected data in response")
	}
}

func TestArticleHandler_GetByID_NotFound(t *testing.T) {
	uc := &mockArticleUC{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.ArticleWithDetails, error) {
			return nil, nil
		},
	}
	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles/"+id.String(), nil)
	w := httptest.NewRecorder()
	newArticleRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestArticleHandler_GetByID_InvalidUUID(t *testing.T) {
	uc := &mockArticleUC{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles/not-a-uuid", nil)
	w := httptest.NewRecorder()
	newArticleRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestArticleHandler_Trending_Success(t *testing.T) {
	uc := &mockArticleUC{
		trendingFn: func(_ context.Context, in usecase.TrendingInput) (*usecase.TrendingResult, error) {
			return &usecase.TrendingResult{
				Articles:    []*domain.ArticleWithDetails{sampleArticle()},
				Period:      "24h",
				GeneratedAt: time.Now().UTC(),
				Page: 1, PerPage: 20, Total: 1, TotalPages: 1,
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles/trending", nil)
	w := httptest.NewRecorder()
	newArticleRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["meta"] == nil {
		t.Error("expected meta in trending response")
	}
}

func TestArticleHandler_Trending_PeriodPassedToUsecase(t *testing.T) {
	var capturedPeriod string
	uc := &mockArticleUC{
		trendingFn: func(_ context.Context, in usecase.TrendingInput) (*usecase.TrendingResult, error) {
			capturedPeriod = in.Period
			return &usecase.TrendingResult{Period: in.Period, GeneratedAt: time.Now().UTC()}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles/trending?period=7d", nil)
	w := httptest.NewRecorder()
	newArticleRouter(uc).ServeHTTP(w, req)

	if capturedPeriod != "7d" {
		t.Errorf("expected period=7d, got %s", capturedPeriod)
	}
}
