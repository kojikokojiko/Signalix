package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// UserUsecaseIface defines the operations needed by UserHandler.
type UserUsecaseIface interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	UpdateProfile(ctx context.Context, in usecase.UpdateProfileInput) (*domain.User, error)
	GetInterests(ctx context.Context, userID uuid.UUID) ([]domain.InterestItem, error)
	SetInterests(ctx context.Context, userID uuid.UUID, inputs []usecase.SetInterestInput) ([]domain.InterestItem, error)
	ListSources(ctx context.Context, userID uuid.UUID) ([]*domain.Source, error)
	SubscribeSource(ctx context.Context, userID uuid.UUID, sourceID string) (*domain.Source, error)
	UnsubscribeSource(ctx context.Context, userID uuid.UUID, sourceID string) error
}

type UserHandler struct {
	uc UserUsecaseIface
}

func NewUserHandler(uc UserUsecaseIface) *UserHandler {
	return &UserHandler{uc: uc}
}

// GET /api/v1/users/me
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	user, err := h.uc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "user_not_found", "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get profile")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"data": userToMap(user)})
}

// PATCH /api/v1/users/me
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var body struct {
		DisplayName       *string `json:"display_name"`
		PreferredLanguage *string `json:"preferred_language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	user, err := h.uc.UpdateProfile(r.Context(), usecase.UpdateProfileInput{
		UserID:            userID,
		DisplayName:       body.DisplayName,
		PreferredLanguage: body.PreferredLanguage,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "user_not_found", "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to update profile")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"data": userToMap(user)})
}

// GET /api/v1/users/me/interests
func (h *UserHandler) GetInterests(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	items, err := h.uc.GetInterests(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get interests")
		return
	}

	data := make([]map[string]any, len(items))
	for i, item := range items {
		data[i] = map[string]any{
			"tag_name": item.TagName,
			"tag_id":   item.TagID,
			"category": item.Category,
			"weight":   item.Weight,
			"source":   item.Source,
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": data})
}

// PUT /api/v1/users/me/interests
func (h *UserHandler) SetInterests(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var body struct {
		Interests []struct {
			TagName string  `json:"tag_name"`
			Weight  float64 `json:"weight"`
		} `json:"interests"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if len(body.Interests) > 20 {
		respondError(w, http.StatusBadRequest, "validation_error", "too many interests: max 20")
		return
	}

	inputs := make([]usecase.SetInterestInput, len(body.Interests))
	for i, inp := range body.Interests {
		inputs[i] = usecase.SetInterestInput{TagName: inp.TagName, Weight: inp.Weight}
	}

	items, err := h.uc.SetInterests(r.Context(), userID, inputs)
	if err != nil {
		if errors.Is(err, usecase.ErrTagNotFound) {
			respondError(w, http.StatusUnprocessableEntity, "tag_not_found", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to set interests")
		return
	}

	data := make([]map[string]any, len(items))
	for i, item := range items {
		data[i] = map[string]any{
			"tag_name": item.TagName,
			"tag_id":   item.TagID,
			"category": item.Category,
			"weight":   item.Weight,
			"source":   item.Source,
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": data})
}

// GET /api/v1/users/me/sources
func (h *UserHandler) GetSources(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	sources, err := h.uc.ListSources(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list sources")
		return
	}

	data := make([]map[string]any, len(sources))
	for i, s := range sources {
		data[i] = sourceToMap(s)
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": data})
}

// POST /api/v1/users/me/sources
func (h *UserHandler) SubscribeSource(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var body struct {
		SourceID string `json:"source_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if body.SourceID == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "source_id is required")
		return
	}

	source, err := h.uc.SubscribeSource(r.Context(), userID, body.SourceID)
	if err != nil {
		if errors.Is(err, usecase.ErrSourceNotFound) {
			respondError(w, http.StatusNotFound, "source_not_found", "source not found")
			return
		}
		if errors.Is(err, usecase.ErrAlreadySubscribed) {
			respondError(w, http.StatusConflict, "already_subscribed", "already subscribed to this source")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to subscribe")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"data": sourceToMap(source)})
}

// DELETE /api/v1/users/me/sources/{source_id}
func (h *UserHandler) UnsubscribeSource(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	sourceID := chi.URLParam(r, "source_id")
	if sourceID == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "source_id is required")
		return
	}

	if err := h.uc.UnsubscribeSource(r.Context(), userID, sourceID); err != nil {
		if errors.Is(err, usecase.ErrNotSubscribed) {
			respondError(w, http.StatusNotFound, "not_subscribed", "not subscribed to this source")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to unsubscribe")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func userToMap(u *domain.User) map[string]any {
	return map[string]any{
		"id":                 u.ID,
		"email":              u.Email,
		"display_name":       u.DisplayName,
		"preferred_language": u.PreferredLanguage,
		"is_admin":           u.IsAdmin,
		"created_at":         u.CreatedAt,
	}
}
