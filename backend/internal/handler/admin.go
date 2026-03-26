package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// AdminUsecaseIface defines admin handler dependencies.
type AdminUsecaseIface interface {
	CreateSource(ctx context.Context, in usecase.CreateSourceInput) (*domain.Source, error)
	UpdateSource(ctx context.Context, id string, in usecase.UpdateSourceInput) (*domain.Source, error)
	DeleteSource(ctx context.Context, id string) error
	ListAdminSources(ctx context.Context, filter repository.SourceFilter) ([]*domain.Source, int, error)
	TriggerFetch(ctx context.Context, sourceID string) (string, error)
	ListIngestionJobs(ctx context.Context, in usecase.IngestionJobListInput) ([]*domain.IngestionJob, int, error)
	GetStats(ctx context.Context) (*domain.AdminStats, error)
}

type AdminHandler struct {
	uc AdminUsecaseIface
}

func NewAdminHandler(uc AdminUsecaseIface) *AdminHandler {
	return &AdminHandler{uc: uc}
}

// ListSources handles GET /api/v1/admin/sources
func (h *AdminHandler) ListSources(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r.URL.Query().Get("page"), 1)
	perPage := queryInt(r.URL.Query().Get("per_page"), 50)

	var category, language *string
	if v := r.URL.Query().Get("category"); v != "" {
		category = &v
	}
	if v := r.URL.Query().Get("language"); v != "" {
		language = &v
	}

	sources, total, err := h.uc.ListAdminSources(r.Context(), repository.SourceFilter{
		Category: category,
		Language: language,
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list sources")
		return
	}

	items := make([]any, 0, len(sources))
	for _, s := range sources {
		items = append(items, formatAdminSource(s))
	}
	totalPages := 1
	if total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"pagination": map[string]any{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	})
}

// CreateSource handles POST /api/v1/admin/sources
func (h *AdminHandler) CreateSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                 string   `json:"name"`
		FeedURL              string   `json:"feed_url"`
		SiteURL              string   `json:"site_url"`
		Description          *string  `json:"description"`
		Category             string   `json:"category"`
		Language             string   `json:"language"`
		FetchIntervalMinutes int      `json:"fetch_interval_minutes"`
		QualityScore         float64  `json:"quality_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}

	s, err := h.uc.CreateSource(r.Context(), usecase.CreateSourceInput{
		Name:                 req.Name,
		FeedURL:              req.FeedURL,
		SiteURL:              req.SiteURL,
		Description:          req.Description,
		Category:             req.Category,
		Language:             req.Language,
		FetchIntervalMinutes: req.FetchIntervalMinutes,
		QualityScore:         req.QualityScore,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrFeedURLAlreadyExists):
			respondError(w, http.StatusConflict, "feed_url_already_exists", "feed_url already exists")
		case errors.Is(err, usecase.ErrValidation):
			respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "internal_error", "failed to create source")
		}
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"data": formatAdminSource(s)})
}

// UpdateSource handles PATCH /api/v1/admin/sources/:id
func (h *AdminHandler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Name                 *string  `json:"name"`
		Description          *string  `json:"description"`
		Category             *string  `json:"category"`
		FetchIntervalMinutes *int     `json:"fetch_interval_minutes"`
		QualityScore         *float64 `json:"quality_score"`
		Status               *string  `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}

	s, err := h.uc.UpdateSource(r.Context(), id, usecase.UpdateSourceInput{
		Name:                 req.Name,
		Description:          req.Description,
		Category:             req.Category,
		FetchIntervalMinutes: req.FetchIntervalMinutes,
		QualityScore:         req.QualityScore,
		Status:               req.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrSourceNotFound):
			respondError(w, http.StatusNotFound, "source_not_found", "source not found")
		case errors.Is(err, usecase.ErrValidation):
			respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "internal_error", "failed to update source")
		}
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": formatAdminSource(s)})
}

// DeleteSource handles DELETE /api/v1/admin/sources/:id
func (h *AdminHandler) DeleteSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.uc.DeleteSource(r.Context(), id); err != nil {
		if errors.Is(err, usecase.ErrSourceNotFound) {
			respondError(w, http.StatusNotFound, "source_not_found", "source not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to delete source")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TriggerFetch handles POST /api/v1/admin/sources/:id/fetch
func (h *AdminHandler) TriggerFetch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	jobID, err := h.uc.TriggerFetch(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrSourceNotFound) {
			respondError(w, http.StatusNotFound, "source_not_found", "source not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to trigger fetch")
		return
	}
	respondJSON(w, http.StatusAccepted, map[string]any{
		"data": map[string]any{
			"job_id":  jobID,
			"message": "フェッチジョブをキューに追加しました",
		},
	})
}

// ListIngestionJobs handles GET /api/v1/admin/ingestion-jobs
func (h *AdminHandler) ListIngestionJobs(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r.URL.Query().Get("page"), 1)
	perPage := queryInt(r.URL.Query().Get("per_page"), 50)

	var sourceID, status *string
	if v := r.URL.Query().Get("source_id"); v != "" {
		sourceID = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		status = &v
	}

	jobs, total, err := h.uc.ListIngestionJobs(r.Context(), usecase.IngestionJobListInput{
		SourceID: sourceID,
		Status:   status,
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list jobs")
		return
	}

	items := make([]any, 0, len(jobs))
	for _, j := range jobs {
		items = append(items, formatIngestionJob(j))
	}
	totalPages := 1
	if total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"pagination": map[string]any{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	})
}

// GetStats handles GET /api/v1/admin/stats
func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.uc.GetStats(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get stats")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"sources": map[string]any{
				"total":    stats.Sources.Total,
				"active":   stats.Sources.Active,
				"degraded": stats.Sources.Degraded,
				"disabled": stats.Sources.Disabled,
			},
			"articles": map[string]any{
				"total":     stats.Articles.Total,
				"processed": stats.Articles.Processed,
				"pending":   stats.Articles.Pending,
				"failed":    stats.Articles.Failed,
			},
			"ingestion_jobs": map[string]any{
				"last_24h_completed": stats.IngestionJobs.Last24hCompleted,
				"last_24h_failed":    stats.IngestionJobs.Last24hFailed,
			},
			"users": map[string]any{
				"total":          stats.Users.Total,
				"active_last_7d": stats.Users.ActiveLast7d,
			},
		},
	})
}

// --- formatters ---

func formatAdminSource(s *domain.Source) map[string]any {
	m := map[string]any{
		"id":                     s.ID,
		"name":                   s.Name,
		"feed_url":               s.FeedURL,
		"site_url":               s.SiteURL,
		"description":            s.Description,
		"category":               s.Category,
		"language":               s.Language,
		"fetch_interval_minutes": s.FetchIntervalMinutes,
		"quality_score":          s.QualityScore,
		"status":                 s.Status,
		"last_fetched_at":        nil,
		"consecutive_failures":   s.ConsecutiveFailures,
		"created_at":             s.CreatedAt.Format(time.RFC3339),
		"updated_at":             s.UpdatedAt.Format(time.RFC3339),
	}
	if s.LastFetchedAt != nil {
		m["last_fetched_at"] = s.LastFetchedAt.Format(time.RFC3339)
	}
	return m
}

func formatIngestionJob(j *domain.IngestionJob) map[string]any {
	m := map[string]any{
		"id": j.ID,
		"source": map[string]any{
			"id":   j.SourceID,
			"name": j.SourceName,
		},
		"status":           j.Status,
		"articles_found":   j.ArticlesFound,
		"articles_new":     j.ArticlesNew,
		"articles_skipped": j.ArticlesSkipped,
		"error_message":    j.ErrorMessage,
		"started_at":       j.StartedAt.Format(time.RFC3339),
		"completed_at":     nil,
	}
	if j.CompletedAt != nil {
		m["completed_at"] = j.CompletedAt.Format(time.RFC3339)
	}
	return m
}
