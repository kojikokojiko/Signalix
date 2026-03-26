package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
)

type ArticleFilter struct {
	Query    *string
	Tags     []string
	SourceID *uuid.UUID
	Language *string
	Sort     string // "published_at" | "trend_score"
	Order    string // "asc" | "desc"
	Page     int
	PerPage  int
}

type ArticleRepository interface {
	List(ctx context.Context, filter ArticleFilter) ([]*domain.ArticleWithDetails, int, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.ArticleWithDetails, error)
	Trending(ctx context.Context, period string, language *string, page, perPage int) ([]*domain.ArticleWithDetails, int, error)
	// Worker用（インジェスション）
	Insert(ctx context.Context, article *domain.Article) error
	ListRecentBySource(ctx context.Context, sourceID uuid.UUID, limit int) ([]*domain.ArticleWithDetails, error)
	// Worker用（処理パイプライン）
	GetRawForProcessing(ctx context.Context, id uuid.UUID) (*domain.Article, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateCleanContent(ctx context.Context, id uuid.UUID, clean string, language *string) error
	UpdateTrendScore(ctx context.Context, id uuid.UUID, score float64) error
	SaveSummary(ctx context.Context, s *domain.ArticleSummary) error
	SaveEmbedding(ctx context.Context, articleID uuid.UUID, embedding []float32) error
	SaveTags(ctx context.Context, articleID uuid.UUID, tags []domain.TagWithConfidence) error
	ListAllTagNames(ctx context.Context) ([]string, error)
	FindTagIDsByName(ctx context.Context, names []string) (map[string]uuid.UUID, error)
}
