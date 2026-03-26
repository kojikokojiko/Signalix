package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

type ArticleUsecaseIface interface {
	List(ctx context.Context, in usecase.ArticleListInput) (*usecase.ArticleListResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ArticleWithDetails, error)
	Trending(ctx context.Context, in usecase.TrendingInput) (*usecase.TrendingResult, error)
}

type ArticleHandler struct {
	uc ArticleUsecaseIface
}

func NewArticleHandler(uc ArticleUsecaseIface) *ArticleHandler {
	return &ArticleHandler{uc: uc}
}

func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	in := usecase.ArticleListInput{
		Tags:    q["tag"],
		Sort:    q.Get("sort"),
		Order:   q.Get("order"),
		Page:    queryInt(q.Get("page"), 1),
		PerPage: queryInt(q.Get("per_page"), 20),
	}
	if v := q.Get("q"); v != "" {
		in.Query = &v
	}
	if v := q.Get("language"); v != "" {
		in.Language = &v
	}
	if v := q.Get("source_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			respondError(w, http.StatusBadRequest, "validation_error", "invalid source_id")
			return
		}
		in.SourceID = &id
	}

	result, err := h.uc.List(r.Context(), in)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list articles")
		return
	}

	articles := make([]map[string]any, len(result.Articles))
	for i, a := range result.Articles {
		articles[i] = articleSummaryToMap(a)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": articles,
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

func (h *ArticleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", "invalid article id")
		return
	}

	a, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get article")
		return
	}
	if a == nil {
		respondError(w, http.StatusNotFound, "article_not_found", "article not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": articleDetailToMap(a),
	})
}

func (h *ArticleHandler) Trending(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	in := usecase.TrendingInput{
		Period:  q.Get("period"),
		Page:    queryInt(q.Get("page"), 1),
		PerPage: queryInt(q.Get("per_page"), 20),
	}
	if v := q.Get("language"); v != "" {
		in.Language = &v
	}

	result, err := h.uc.Trending(r.Context(), in)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get trending articles")
		return
	}

	articles := make([]map[string]any, len(result.Articles))
	for i, a := range result.Articles {
		articles[i] = articleSummaryToMap(a)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": articles,
		"pagination": map[string]any{
			"page":        result.Page,
			"per_page":    result.PerPage,
			"total":       result.Total,
			"total_pages": result.TotalPages,
			"has_next":    result.HasNext,
			"has_prev":    result.HasPrev,
		},
		"meta": map[string]any{
			"period":       result.Period,
			"generated_at": result.GeneratedAt.Format(time.RFC3339),
		},
	})
}

// ─── response mappers ────────────────────────────────────────────────────────

func articleSummaryToMap(a *domain.ArticleWithDetails) map[string]any {
	m := map[string]any{
		"id":          a.Article.ID,
		"title":       a.Article.Title,
		"url":         a.Article.URL,
		"language":    a.Article.Language,
		"published_at": a.Article.PublishedAt,
		"trend_score": a.Article.TrendScore,
		"tags":        tagsToSlice(a.Tags),
	}
	if a.Source != nil {
		m["source"] = map[string]any{
			"id":       a.Source.ID,
			"name":     a.Source.Name,
			"site_url": a.Source.SiteURL,
		}
	}
	if a.Article.Author != nil {
		m["author"] = *a.Article.Author
	}
	if a.Summary != nil {
		m["summary"] = a.Summary.SummaryText
	}
	return m
}

func articleDetailToMap(a *domain.ArticleWithDetails) map[string]any {
	m := articleSummaryToMap(a)
	if a.Summary != nil {
		m["summary"] = map[string]any{
			"text":          a.Summary.SummaryText,
			"model_name":    a.Summary.ModelName,
			"model_version": a.Summary.ModelVersion,
		}
	}
	if a.Source != nil {
		m["source"] = map[string]any{
			"id":       a.Source.ID,
			"name":     a.Source.Name,
			"site_url": a.Source.SiteURL,
			"category": a.Source.Category,
		}
	}
	m["created_at"] = a.Article.CreatedAt
	return m
}

func tagsToSlice(tags []domain.TagWithConfidence) []map[string]any {
	out := make([]map[string]any, len(tags))
	for i, t := range tags {
		out[i] = map[string]any{
			"id":       t.ID,
			"name":     t.Name,
			"category": t.Category,
		}
	}
	return out
}
