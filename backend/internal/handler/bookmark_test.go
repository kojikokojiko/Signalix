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
	"github.com/kojikokojiko/signalix/internal/ctxkey"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/handler"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

type mockBookmarkUC struct {
	listFn   func(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.BookmarkListResult, error)
	addFn    func(ctx context.Context, userID, articleID uuid.UUID) (*domain.Bookmark, error)
	removeFn func(ctx context.Context, userID, articleID uuid.UUID) error
}

func (m *mockBookmarkUC) List(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.BookmarkListResult, error) {
	return m.listFn(ctx, userID, page, perPage)
}
func (m *mockBookmarkUC) Add(ctx context.Context, userID, articleID uuid.UUID) (*domain.Bookmark, error) {
	return m.addFn(ctx, userID, articleID)
}
func (m *mockBookmarkUC) Remove(ctx context.Context, userID, articleID uuid.UUID) error {
	return m.removeFn(ctx, userID, articleID)
}

func newBookmarkRouter(uc handler.BookmarkUsecaseIface) http.Handler {
	r := chi.NewRouter()
	h := handler.NewBookmarkHandler(uc)
	r.Get("/api/v1/bookmarks", h.List)
	r.Post("/api/v1/bookmarks", h.Add)
	r.Delete("/api/v1/bookmarks/{article_id}", h.Remove)
	return r
}

// withUser injects a user ID into context (simulates JWT middleware)
func withUser(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), ctxkey.UserID, userID.String())
	return r.WithContext(ctx)
}

func TestBookmarkHandler_List_Success(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	uc := &mockBookmarkUC{
		listFn: func(_ context.Context, _ uuid.UUID, _, _ int) (*usecase.BookmarkListResult, error) {
			return &usecase.BookmarkListResult{
				Bookmarks: []*repository.BookmarkWithArticle{
					{
						Bookmark: domain.Bookmark{ID: uuid.New(), CreatedAt: now},
						Article:  &domain.ArticleWithDetails{Article: domain.Article{ID: uuid.New(), Title: "Test"}},
					},
				},
				Page: 1, PerPage: 20, Total: 1, TotalPages: 1,
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookmarks", nil)
	req = withUser(req, userID)
	w := httptest.NewRecorder()
	newBookmarkRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["data"] == nil || resp["pagination"] == nil {
		t.Error("expected data and pagination")
	}
}

func TestBookmarkHandler_List_Unauthenticated(t *testing.T) {
	uc := &mockBookmarkUC{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookmarks", nil)
	// no user in context
	w := httptest.NewRecorder()
	newBookmarkRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestBookmarkHandler_Add_Success(t *testing.T) {
	userID := uuid.New()
	articleID := uuid.New()
	uc := &mockBookmarkUC{
		addFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Bookmark, error) {
			return &domain.Bookmark{ID: uuid.New(), ArticleID: articleID, CreatedAt: time.Now()}, nil
		},
	}
	body, _ := json.Marshal(map[string]string{"article_id": articleID.String()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bookmarks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, userID)
	w := httptest.NewRecorder()
	newBookmarkRouter(uc).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBookmarkHandler_Add_ArticleNotFound(t *testing.T) {
	uc := &mockBookmarkUC{
		addFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Bookmark, error) {
			return nil, usecase.ErrArticleNotFound
		},
	}
	body, _ := json.Marshal(map[string]string{"article_id": uuid.New().String()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bookmarks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	newBookmarkRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestBookmarkHandler_Add_AlreadyBookmarked(t *testing.T) {
	uc := &mockBookmarkUC{
		addFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Bookmark, error) {
			return nil, usecase.ErrAlreadyBookmarked
		},
	}
	body, _ := json.Marshal(map[string]string{"article_id": uuid.New().String()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bookmarks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	newBookmarkRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestBookmarkHandler_Add_InvalidUUID(t *testing.T) {
	uc := &mockBookmarkUC{}
	body, _ := json.Marshal(map[string]string{"article_id": "not-a-uuid"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bookmarks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	newBookmarkRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBookmarkHandler_Remove_Success(t *testing.T) {
	uc := &mockBookmarkUC{
		removeFn: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/bookmarks/"+uuid.New().String(), nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	newBookmarkRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestBookmarkHandler_Remove_NotFound(t *testing.T) {
	uc := &mockBookmarkUC{
		removeFn: func(_ context.Context, _, _ uuid.UUID) error { return usecase.ErrBookmarkNotFound },
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/bookmarks/"+uuid.New().String(), nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	newBookmarkRouter(uc).ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
