package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

const feedCacheTTL = 5 * time.Minute

// FeedCacheStore is the interface the recommendation handler needs for caching.
type FeedCacheStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
	Delete(ctx context.Context, pattern string) error
}

// RecommendationUsecaseIface defines the methods used by the handler.
type RecommendationUsecaseIface interface {
	List(ctx context.Context, in usecase.RecommendationListInput) (usecase.RecommendationListOutput, error)
	RequestRefresh(ctx context.Context, userID uuid.UUID) error
}

type RecommendationHandler struct {
	uc    RecommendationUsecaseIface
	cache FeedCacheStore
}

func NewRecommendationHandler(uc RecommendationUsecaseIface, cache FeedCacheStore) *RecommendationHandler {
	return &RecommendationHandler{uc: uc, cache: cache}
}

// List handles GET /api/v1/recommendations
func (h *RecommendationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	page := queryInt(r.URL.Query().Get("page"), 1)
	perPage := queryInt(r.URL.Query().Get("per_page"), 20)
	if perPage > 50 {
		perPage = 50
	}
	var language *string
	if lang := r.URL.Query().Get("language"); lang != "" {
		language = &lang
	}

	// Check cache
	cacheKey := feedCacheKey(userID, page)
	if h.cache != nil {
		if cached, err := h.cache.Get(r.Context(), cacheKey); err == nil && cached != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(cached)
			return
		}
	}

	out, err := h.uc.List(r.Context(), usecase.RecommendationListInput{
		UserID:   userID,
		Language: language,
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to fetch recommendations")
		return
	}

	items := make([]any, 0, len(out.Items))
	for _, item := range out.Items {
		items = append(items, formatRecommendedItem(item))
	}

	totalPages := 1
	if out.Total > 0 {
		totalPages = int(math.Ceil(float64(out.Total) / float64(perPage)))
	}

	body := map[string]any{
		"data": items,
		"pagination": map[string]any{
			"page":        page,
			"per_page":    perPage,
			"total":       out.Total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
		"meta": map[string]any{
			"has_interest_profile": out.HasInterestProfile,
		},
	}

	// Write response and cache it
	if h.cache != nil {
		respondJSONWithCache(w, http.StatusOK, body, func(data []byte) {
			_ = h.cache.Set(r.Context(), cacheKey, data, feedCacheTTL)
		})
		return
	}

	respondJSON(w, http.StatusOK, body)
}

// Refresh handles POST /api/v1/recommendations/refresh
func (h *RecommendationHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	if err := h.uc.RequestRefresh(r.Context(), userID); err != nil {
		if errors.Is(err, usecase.ErrRateLimitExceeded) {
			respondError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "too many refresh requests")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to request refresh")
		return
	}

	// Invalidate all cached pages for this user
	if h.cache != nil {
		pattern := fmt.Sprintf("user_feed:%s:page:*", userID.String())
		_ = h.cache.Delete(r.Context(), pattern)
	}

	respondJSON(w, http.StatusAccepted, map[string]any{
		"data": map[string]any{
			"message":                "フィードの再計算をリクエストしました",
			"estimated_wait_seconds": 30,
		},
	})
}

func feedCacheKey(userID uuid.UUID, page int) string {
	return fmt.Sprintf("user_feed:%s:page:%d", userID.String(), page)
}

func formatRecommendedItem(item *domain.RecommendedItem) map[string]any {
	a := item.Article
	l := item.Log

	tags := make([]map[string]any, 0, len(a.Tags))
	for _, t := range a.Tags {
		tags = append(tags, map[string]any{
			"id":       t.Tag.ID,
			"name":     t.Tag.Name,
			"category": t.Tag.Category,
		})
	}

	articleMap := map[string]any{
		"id":          a.Article.ID,
		"title":       a.Article.Title,
		"url":         a.Article.URL,
		"author":      a.Article.Author,
		"language":    a.Article.Language,
		"trend_score": a.Article.TrendScore,
		"tags":        tags,
	}
	if a.Article.PublishedAt != nil {
		articleMap["published_at"] = a.Article.PublishedAt.Format(time.RFC3339)
	}
	if a.Source != nil {
		articleMap["source"] = map[string]any{
			"id":       a.Source.ID,
			"name":     a.Source.Name,
			"site_url": a.Source.SiteURL,
		}
	}
	if a.Summary != nil {
		articleMap["summary"] = a.Summary.SummaryText
	}

	return map[string]any{
		"article": articleMap,
		"recommendation": map[string]any{
			"total_score": l.TotalScore,
			"explanation": l.Explanation,
			"score_breakdown": map[string]any{
				"relevance":       l.RelevanceScore,
				"freshness":       l.FreshnessScore,
				"trend":           l.TrendScore,
				"source_quality":  l.SourceQualityScore,
				"personalization": l.PersonalizationBoost,
			},
			"generated_at": l.GeneratedAt.Format(time.RFC3339),
		},
	}
}

// respondJSONWithCache writes a JSON response and calls the onEncoded callback with the encoded bytes.
func respondJSONWithCache(w http.ResponseWriter, status int, body any, onEncoded func([]byte)) {
	data, err := json.Marshal(body)
	if err != nil {
		respondJSON(w, status, body)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
	onEncoded(data)
}

