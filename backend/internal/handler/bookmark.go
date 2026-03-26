package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/ctxkey"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

type BookmarkUsecaseIface interface {
	List(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.BookmarkListResult, error)
	Add(ctx context.Context, userID, articleID uuid.UUID) (*domain.Bookmark, error)
	Remove(ctx context.Context, userID, articleID uuid.UUID) error
}

type BookmarkHandler struct {
	uc BookmarkUsecaseIface
}

func NewBookmarkHandler(uc BookmarkUsecaseIface) *BookmarkHandler {
	return &BookmarkHandler{uc: uc}
}

func (h *BookmarkHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	q := r.URL.Query()
	page := queryInt(q.Get("page"), 1)
	perPage := queryInt(q.Get("per_page"), 20)

	result, err := h.uc.List(r.Context(), userID, page, perPage)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list bookmarks")
		return
	}

	items := make([]map[string]any, len(result.Bookmarks))
	for i, bm := range result.Bookmarks {
		items[i] = bookmarkToMap(bm)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"pagination": map[string]any{
			"page":        result.Page,
			"per_page":    result.PerPage,
			"total":       result.Total,
			"total_pages": result.TotalPages,
			"has_next":    result.HasNext,
			"has_prev":    result.HasPrev,
		},
	})
}

func (h *BookmarkHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req struct {
		ArticleID string `json:"article_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	articleID, err := uuid.Parse(req.ArticleID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", "invalid article_id")
		return
	}

	bm, err := h.uc.Add(r.Context(), userID, articleID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrArticleNotFound):
			respondError(w, http.StatusNotFound, "article_not_found", "article not found")
		case errors.Is(err, usecase.ErrAlreadyBookmarked):
			respondError(w, http.StatusConflict, "already_bookmarked", "article already bookmarked")
		default:
			respondError(w, http.StatusInternalServerError, "internal_error", "failed to add bookmark")
		}
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"bookmark_id":   bm.ID,
			"article_id":    bm.ArticleID,
			"bookmarked_at": bm.CreatedAt.Format(time.RFC3339),
		},
	})
}

func (h *BookmarkHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	articleID, err := uuid.Parse(chi.URLParam(r, "article_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", "invalid article_id")
		return
	}

	if err := h.uc.Remove(r.Context(), userID, articleID); err != nil {
		if errors.Is(err, usecase.ErrBookmarkNotFound) {
			respondError(w, http.StatusNotFound, "bookmark_not_found", "bookmark not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to remove bookmark")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func bookmarkToMap(bm *repository.BookmarkWithArticle) map[string]any {
	m := map[string]any{
		"bookmark_id":   bm.Bookmark.ID,
		"bookmarked_at": bm.Bookmark.CreatedAt.Format(time.RFC3339),
	}
	if bm.Article != nil {
		m["article"] = articleSummaryToMap(bm.Article)
	}
	return m
}

func userIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	v := r.Context().Value(ctxkey.UserID)
	if v == nil {
		return uuid.Nil, false
	}
	s, ok := v.(string)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	return id, err == nil
}
