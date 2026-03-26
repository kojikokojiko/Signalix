package handler_test

import (
	"bytes"
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

type mockFeedbackUC struct {
	submitFn func(ctx context.Context, in usecase.FeedbackInput) (*domain.UserFeedback, error)
	deleteFn func(ctx context.Context, userID, articleID uuid.UUID) error
}

func (m *mockFeedbackUC) Submit(ctx context.Context, in usecase.FeedbackInput) (*domain.UserFeedback, error) {
	return m.submitFn(ctx, in)
}
func (m *mockFeedbackUC) Delete(ctx context.Context, userID, articleID uuid.UUID) error {
	return m.deleteFn(ctx, userID, articleID)
}

func newFeedbackRouter(uc handler.FeedbackUsecaseIface) http.Handler {
	r := chi.NewRouter()
	h := handler.NewFeedbackHandler(uc)
	r.Post("/api/v1/feedback", h.Submit)
	r.Delete("/api/v1/feedback/{article_id}", h.Delete)
	return r
}

func TestFeedbackHandler_Submit_Success(t *testing.T) {
	userID := uuid.New()
	articleID := uuid.New()
	uc := &mockFeedbackUC{
		submitFn: func(_ context.Context, _ usecase.FeedbackInput) (*domain.UserFeedback, error) {
			return &domain.UserFeedback{ID: uuid.New(), ArticleID: articleID, FeedbackType: "like", CreatedAt: time.Now()}, nil
		},
	}
	body, _ := json.Marshal(map[string]string{"article_id": articleID.String(), "feedback_type": "like"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, userID)
	w := httptest.NewRecorder()
	newFeedbackRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFeedbackHandler_Submit_InvalidType(t *testing.T) {
	uc := &mockFeedbackUC{
		submitFn: func(_ context.Context, _ usecase.FeedbackInput) (*domain.UserFeedback, error) {
			return nil, usecase.ErrInvalidFeedbackType
		},
	}
	body, _ := json.Marshal(map[string]string{"article_id": uuid.New().String(), "feedback_type": "invalid"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	newFeedbackRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFeedbackHandler_Submit_ArticleNotFound(t *testing.T) {
	uc := &mockFeedbackUC{
		submitFn: func(_ context.Context, _ usecase.FeedbackInput) (*domain.UserFeedback, error) {
			return nil, usecase.ErrArticleNotFound
		},
	}
	body, _ := json.Marshal(map[string]string{"article_id": uuid.New().String(), "feedback_type": "like"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	newFeedbackRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFeedbackHandler_Submit_Unauthenticated(t *testing.T) {
	uc := &mockFeedbackUC{}
	body, _ := json.Marshal(map[string]string{"article_id": uuid.New().String(), "feedback_type": "like"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newFeedbackRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestFeedbackHandler_Delete_Success(t *testing.T) {
	uc := &mockFeedbackUC{
		deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/feedback/"+uuid.New().String(), nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	newFeedbackRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestFeedbackHandler_Delete_NotFound(t *testing.T) {
	uc := &mockFeedbackUC{
		deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return usecase.ErrFeedbackNotFound },
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/feedback/"+uuid.New().String(), nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	newFeedbackRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
