package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

type FeedbackUsecaseIface interface {
	Submit(ctx context.Context, in usecase.FeedbackInput) (*domain.UserFeedback, error)
	Delete(ctx context.Context, userID, articleID uuid.UUID) error
}

type FeedbackHandler struct {
	uc FeedbackUsecaseIface
}

func NewFeedbackHandler(uc FeedbackUsecaseIface) *FeedbackHandler {
	return &FeedbackHandler{uc: uc}
}

func (h *FeedbackHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req struct {
		ArticleID    string `json:"article_id"`
		FeedbackType string `json:"feedback_type"`
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
	if req.FeedbackType == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "feedback_type is required")
		return
	}

	fb, err := h.uc.Submit(r.Context(), usecase.FeedbackInput{
		UserID:       userID,
		ArticleID:    articleID,
		FeedbackType: req.FeedbackType,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidFeedbackType):
			respondError(w, http.StatusBadRequest, "validation_error", "invalid feedback_type")
		case errors.Is(err, usecase.ErrArticleNotFound):
			respondError(w, http.StatusNotFound, "article_not_found", "article not found")
		default:
			respondError(w, http.StatusInternalServerError, "internal_error", "failed to submit feedback")
		}
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"id":            fb.ID,
			"article_id":    fb.ArticleID,
			"feedback_type": fb.FeedbackType,
			"created_at":    fb.CreatedAt.Format(time.RFC3339),
		},
	})
}

func (h *FeedbackHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	if err := h.uc.Delete(r.Context(), userID, articleID); err != nil {
		if errors.Is(err, usecase.ErrFeedbackNotFound) {
			respondError(w, http.StatusNotFound, "feedback_not_found", "feedback not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to delete feedback")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
