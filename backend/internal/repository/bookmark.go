package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
)

type BookmarkRepository interface {
	List(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*BookmarkWithArticle, int, error)
	Add(ctx context.Context, userID, articleID uuid.UUID) (*domain.Bookmark, error)
	Remove(ctx context.Context, userID, articleID uuid.UUID) error
	Exists(ctx context.Context, userID, articleID uuid.UUID) (bool, error)
}

type BookmarkWithArticle struct {
	domain.Bookmark
	Article *domain.ArticleWithDetails
}
