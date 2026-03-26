package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

type ArticleListInput struct {
	Query    *string
	Tags     []string
	SourceID *uuid.UUID
	Language *string
	Sort     string
	Order    string
	Page     int
	PerPage  int
}

type ArticleListResult struct {
	Articles   []*domain.ArticleWithDetails
	Page       int
	PerPage    int
	Total      int
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

type TrendingInput struct {
	Period   string
	Language *string
	Page     int
	PerPage  int
}

type TrendingResult struct {
	Articles    []*domain.ArticleWithDetails
	Page        int
	PerPage     int
	Total       int
	TotalPages  int
	HasNext     bool
	HasPrev     bool
	Period      string
	GeneratedAt time.Time
}

// CacheStore is an optional Redis cache for trending results.
type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

type ArticleUsecase struct {
	articles repository.ArticleRepository
	cache    CacheStore // may be nil
}

func NewArticleUsecase(articles repository.ArticleRepository, cache CacheStore) *ArticleUsecase {
	return &ArticleUsecase{articles: articles, cache: cache}
}

func (uc *ArticleUsecase) List(ctx context.Context, in ArticleListInput) (*ArticleListResult, error) {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PerPage < 1 || in.PerPage > 100 {
		in.PerPage = 20
	}
	if in.Sort == "" {
		in.Sort = "published_at"
	}
	if in.Order == "" {
		in.Order = "desc"
	}

	articles, total, err := uc.articles.List(ctx, repository.ArticleFilter{
		Query:    in.Query,
		Tags:     in.Tags,
		SourceID: in.SourceID,
		Language: in.Language,
		Sort:     in.Sort,
		Order:    in.Order,
		Page:     in.Page,
		PerPage:  in.PerPage,
	})
	if err != nil {
		return nil, fmt.Errorf("list articles: %w", err)
	}

	totalPages := (total + in.PerPage - 1) / in.PerPage
	return &ArticleListResult{
		Articles:   articles,
		Page:       in.Page,
		PerPage:    in.PerPage,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    in.Page < totalPages,
		HasPrev:    in.Page > 1,
	}, nil
}

func (uc *ArticleUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.ArticleWithDetails, error) {
	a, err := uc.articles.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find article: %w", err)
	}
	return a, nil
}

func (uc *ArticleUsecase) Trending(ctx context.Context, in TrendingInput) (*TrendingResult, error) {
	period := in.Period
	if period != "24h" && period != "7d" {
		period = "24h"
	}
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PerPage < 1 || in.PerPage > 50 {
		in.PerPage = 20
	}

	articles, total, err := uc.articles.Trending(ctx, period, in.Language, in.Page, in.PerPage)
	if err != nil {
		return nil, fmt.Errorf("trending articles: %w", err)
	}

	totalPages := (total + in.PerPage - 1) / in.PerPage
	return &TrendingResult{
		Articles:    articles,
		Page:        in.Page,
		PerPage:     in.PerPage,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     in.Page < totalPages,
		HasPrev:     in.Page > 1,
		Period:      period,
		GeneratedAt: time.Now().UTC(),
	}, nil
}
