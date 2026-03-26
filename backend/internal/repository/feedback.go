package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
)

type FeedbackRepository interface {
	Upsert(ctx context.Context, fb *domain.UserFeedback) error
	Delete(ctx context.Context, userID, articleID uuid.UUID) error
	FindByUserAndArticle(ctx context.Context, userID, articleID uuid.UUID) (*domain.UserFeedback, error)
}

// InterestEntry is used when replacing all interests for a user.
type InterestEntry struct {
	TagID  uuid.UUID
	Weight float64
}

type InterestRepository interface {
	AdjustWeight(ctx context.Context, userID, tagID uuid.UUID, delta float64) error
	GetTagsByArticle(ctx context.Context, articleID uuid.UUID) ([]uuid.UUID, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.UserInterest, error)
	ListWithTags(ctx context.Context, userID uuid.UUID) ([]domain.InterestItem, error)
	ReplaceAll(ctx context.Context, userID uuid.UUID, entries []InterestEntry) error
}

type TagRepository interface {
	FindByName(ctx context.Context, name string) (*domain.Tag, error)
}
