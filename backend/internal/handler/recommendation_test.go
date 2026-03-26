package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/ctxkey"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/handler"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// --- mock ---

type mockRecommendationUC struct {
	listOutput usecase.RecommendationListOutput
	listErr    error
	refreshErr error
}

func (m *mockRecommendationUC) List(_ context.Context, _ usecase.RecommendationListInput) (usecase.RecommendationListOutput, error) {
	return m.listOutput, m.listErr
}
func (m *mockRecommendationUC) RequestRefresh(_ context.Context, _ uuid.UUID) error {
	return m.refreshErr
}

// --- helpers ---

func makeRecommendedItem() *domain.RecommendedItem {
	pub := time.Now().Add(-2 * time.Hour)
	return &domain.RecommendedItem{
		Article: &domain.ArticleWithDetails{
			Article: domain.Article{
				ID:          uuid.New(),
				Title:       "Test Article",
				URL:         "https://example.com/test",
				PublishedAt: &pub,
				TrendScore:  0.8,
			},
			Source: &domain.Source{ID: uuid.New().String(), Name: "Test Source"},
		},
		Log: &domain.RecommendationLog{
			TotalScore:           0.75,
			RelevanceScore:       0.7,
			FreshnessScore:       0.85,
			TrendScore:           0.8,
			SourceQualityScore:   0.9,
			PersonalizationBoost: 0.6,
			Explanation:          "あなたの興味に類似した記事です",
			GeneratedAt:          time.Now(),
		},
	}
}

// --- tests ---

func TestRecommendationHandler_List_Returns200(t *testing.T) {
	uc := &mockRecommendationUC{
		listOutput: usecase.RecommendationListOutput{
			Items:              []*domain.RecommendedItem{makeRecommendedItem()},
			Total:              1,
			HasInterestProfile: true,
		},
	}
	h := handler.NewRecommendationHandler(uc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.UserID, uuid.New().String()))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	data, ok := resp["data"].([]any)
	if !ok || len(data) != 1 {
		t.Errorf("expected 1 item in data, got %v", resp["data"])
	}
	meta, ok := resp["meta"].(map[string]any)
	if !ok {
		t.Fatal("expected meta field")
	}
	if meta["has_interest_profile"] != true {
		t.Errorf("expected has_interest_profile=true")
	}
}

func TestRecommendationHandler_List_Returns401WhenNoUserID(t *testing.T) {
	uc := &mockRecommendationUC{}
	h := handler.NewRecommendationHandler(uc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRecommendationHandler_Refresh_Returns202(t *testing.T) {
	uc := &mockRecommendationUC{}
	h := handler.NewRecommendationHandler(uc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/refresh", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.UserID, uuid.New().String()))
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
}

func TestRecommendationHandler_Refresh_Returns429WhenRateLimited(t *testing.T) {
	uc := &mockRecommendationUC{refreshErr: usecase.ErrRateLimitExceeded}
	h := handler.NewRecommendationHandler(uc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/refresh", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.UserID, uuid.New().String()))
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr.Code)
	}
}

func TestRecommendationHandler_Refresh_Returns401WhenNoUserID(t *testing.T) {
	uc := &mockRecommendationUC{}
	h := handler.NewRecommendationHandler(uc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/refresh", nil)
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}
