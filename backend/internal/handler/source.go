package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

type SourceUsecaseIface interface {
	List(ctx context.Context, in usecase.SourceListInput) (*usecase.SourceListResult, error)
	GetByID(ctx context.Context, id string) (*domain.Source, error)
}

type SourceHandler struct {
	uc SourceUsecaseIface
}

func NewSourceHandler(uc SourceUsecaseIface) *SourceHandler {
	return &SourceHandler{uc: uc}
}

func (h *SourceHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	in := usecase.SourceListInput{
		Page:    queryInt(q.Get("page"), 1),
		PerPage: queryInt(q.Get("per_page"), 50),
	}
	if v := q.Get("category"); v != "" {
		in.Category = &v
	}
	if v := q.Get("language"); v != "" {
		in.Language = &v
	}

	result, err := h.uc.List(r.Context(), in)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list sources")
		return
	}

	sources := make([]map[string]any, len(result.Sources))
	for i, s := range result.Sources {
		sources[i] = sourceToMap(s)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": sources,
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

func (h *SourceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	source, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get source")
		return
	}
	if source == nil {
		respondError(w, http.StatusNotFound, "source_not_found", "source not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"source": sourceToMap(source),
		},
	})
}

func sourceToMap(s *domain.Source) map[string]any {
	m := map[string]any{
		"id":              s.ID,
		"name":            s.Name,
		"site_url":        s.SiteURL,
		"category":        s.Category,
		"language":        s.Language,
		"quality_score":   s.QualityScore,
		"status":          s.Status,
		"last_fetched_at": s.LastFetchedAt,
		"created_at":      s.CreatedAt,
	}
	if s.Description != nil {
		m["description"] = *s.Description
	}
	if s.ArticleCount != nil {
		m["article_count"] = *s.ArticleCount
	}
	return m
}

func queryInt(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	return n
}
