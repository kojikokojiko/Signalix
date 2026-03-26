package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

type BookmarkListResult struct {
	Bookmarks  []*repository.BookmarkWithArticle
	Page       int
	PerPage    int
	Total      int
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

type BookmarkUsecase struct {
	bookmarks repository.BookmarkRepository
	articles  repository.ArticleRepository
}

func NewBookmarkUsecase(bookmarks repository.BookmarkRepository, articles repository.ArticleRepository) *BookmarkUsecase {
	return &BookmarkUsecase{bookmarks: bookmarks, articles: articles}
}

func (uc *BookmarkUsecase) List(ctx context.Context, userID uuid.UUID, page, perPage int) (*BookmarkListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	bms, total, err := uc.bookmarks.List(ctx, userID, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}
	totalPages := (total + perPage - 1) / perPage
	return &BookmarkListResult{
		Bookmarks:  bms,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}, nil
}

func (uc *BookmarkUsecase) Add(ctx context.Context, userID, articleID uuid.UUID) (*domain.Bookmark, error) {
	// 記事の存在確認
	a, err := uc.articles.FindByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("find article: %w", err)
	}
	if a == nil {
		return nil, ErrArticleNotFound
	}

	exists, err := uc.bookmarks.Exists(ctx, userID, articleID)
	if err != nil {
		return nil, fmt.Errorf("check bookmark: %w", err)
	}
	if exists {
		return nil, ErrAlreadyBookmarked
	}

	bm, err := uc.bookmarks.Add(ctx, userID, articleID)
	if err != nil {
		return nil, fmt.Errorf("add bookmark: %w", err)
	}
	return bm, nil
}

func (uc *BookmarkUsecase) Remove(ctx context.Context, userID, articleID uuid.UUID) error {
	exists, err := uc.bookmarks.Exists(ctx, userID, articleID)
	if err != nil {
		return fmt.Errorf("check bookmark: %w", err)
	}
	if !exists {
		return ErrBookmarkNotFound
	}
	if err := uc.bookmarks.Remove(ctx, userID, articleID); err != nil {
		return fmt.Errorf("remove bookmark: %w", err)
	}
	return nil
}
