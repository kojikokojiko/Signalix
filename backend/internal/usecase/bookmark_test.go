package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// --- mock bookmark repo ---

type mockBookmarkRepo struct {
	bookmarks []*repository.BookmarkWithArticle
	addFn     func(userID, articleID uuid.UUID) (*domain.Bookmark, error)
	removeFn  func(userID, articleID uuid.UUID) error
	existsFn  func(userID, articleID uuid.UUID) bool
}

func (m *mockBookmarkRepo) List(_ context.Context, userID uuid.UUID, page, perPage int) ([]*repository.BookmarkWithArticle, int, error) {
	total := len(m.bookmarks)
	start := (page - 1) * perPage
	if start >= total {
		return []*repository.BookmarkWithArticle{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}
	return m.bookmarks[start:end], total, nil
}

func (m *mockBookmarkRepo) Add(_ context.Context, userID, articleID uuid.UUID) (*domain.Bookmark, error) {
	if m.addFn != nil {
		return m.addFn(userID, articleID)
	}
	bm := &domain.Bookmark{ID: uuid.New(), UserID: userID, ArticleID: articleID, CreatedAt: time.Now()}
	return bm, nil
}

func (m *mockBookmarkRepo) Remove(_ context.Context, userID, articleID uuid.UUID) error {
	if m.removeFn != nil {
		return m.removeFn(userID, articleID)
	}
	return nil
}

func (m *mockBookmarkRepo) Exists(_ context.Context, userID, articleID uuid.UUID) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(userID, articleID), nil
	}
	return false, nil
}

// --- mock article repo for bookmark tests (reuse FindByID) ---

type mockArticleRepoBookmark struct {
	found bool
}

func (m *mockArticleRepoBookmark) List(_ context.Context, _ repository.ArticleFilter) ([]*domain.ArticleWithDetails, int, error) {
	return nil, 0, nil
}
func (m *mockArticleRepoBookmark) FindByID(_ context.Context, _ uuid.UUID) (*domain.ArticleWithDetails, error) {
	if !m.found {
		return nil, nil
	}
	return &domain.ArticleWithDetails{Article: domain.Article{ID: uuid.New()}}, nil
}
func (m *mockArticleRepoBookmark) Trending(_ context.Context, _ string, _ *string, _, _ int) ([]*domain.ArticleWithDetails, int, error) {
	return nil, 0, nil
}
func (m *mockArticleRepoBookmark) Insert(_ context.Context, _ *domain.Article) error { return nil }
func (m *mockArticleRepoBookmark) ListRecentBySource(_ context.Context, _ uuid.UUID, _ int) ([]*domain.ArticleWithDetails, error) {
	return nil, nil
}
func (m *mockArticleRepoBookmark) GetRawForProcessing(_ context.Context, _ uuid.UUID) (*domain.Article, error) {
	return nil, nil
}
func (m *mockArticleRepoBookmark) UpdateStatus(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockArticleRepoBookmark) UpdateCleanContent(_ context.Context, _ uuid.UUID, _ string, _ *string) error {
	return nil
}
func (m *mockArticleRepoBookmark) UpdateTrendScore(_ context.Context, _ uuid.UUID, _ float64) error {
	return nil
}
func (m *mockArticleRepoBookmark) SaveSummary(_ context.Context, _ *domain.ArticleSummary) error {
	return nil
}
func (m *mockArticleRepoBookmark) SaveEmbedding(_ context.Context, _ uuid.UUID, _ []float32) error {
	return nil
}
func (m *mockArticleRepoBookmark) SaveTags(_ context.Context, _ uuid.UUID, _ []domain.TagWithConfidence) error {
	return nil
}
func (m *mockArticleRepoBookmark) ListAllTagNames(_ context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockArticleRepoBookmark) FindTagIDsByName(_ context.Context, _ []string) (map[string]uuid.UUID, error) {
	return nil, nil
}

// --- tests ---

func TestBookmarkUsecase_Add_Success(t *testing.T) {
	repo := &mockBookmarkRepo{}
	artRepo := &mockArticleRepoBookmark{found: true}
	uc := usecase.NewBookmarkUsecase(repo, artRepo)

	bm, err := uc.Add(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bm == nil {
		t.Error("expected bookmark")
	}
}

func TestBookmarkUsecase_Add_ArticleNotFound(t *testing.T) {
	repo := &mockBookmarkRepo{}
	artRepo := &mockArticleRepoBookmark{found: false}
	uc := usecase.NewBookmarkUsecase(repo, artRepo)

	_, err := uc.Add(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, usecase.ErrArticleNotFound) {
		t.Errorf("expected ErrArticleNotFound, got %v", err)
	}
}

func TestBookmarkUsecase_Add_AlreadyBookmarked(t *testing.T) {
	repo := &mockBookmarkRepo{
		existsFn: func(_, _ uuid.UUID) bool { return true },
	}
	artRepo := &mockArticleRepoBookmark{found: true}
	uc := usecase.NewBookmarkUsecase(repo, artRepo)

	_, err := uc.Add(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, usecase.ErrAlreadyBookmarked) {
		t.Errorf("expected ErrAlreadyBookmarked, got %v", err)
	}
}

func TestBookmarkUsecase_Remove_Success(t *testing.T) {
	repo := &mockBookmarkRepo{
		existsFn: func(_, _ uuid.UUID) bool { return true },
	}
	artRepo := &mockArticleRepoBookmark{}
	uc := usecase.NewBookmarkUsecase(repo, artRepo)

	err := uc.Remove(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestBookmarkUsecase_Remove_NotFound(t *testing.T) {
	repo := &mockBookmarkRepo{
		existsFn: func(_, _ uuid.UUID) bool { return false },
	}
	artRepo := &mockArticleRepoBookmark{}
	uc := usecase.NewBookmarkUsecase(repo, artRepo)

	err := uc.Remove(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, usecase.ErrBookmarkNotFound) {
		t.Errorf("expected ErrBookmarkNotFound, got %v", err)
	}
}

func TestBookmarkUsecase_List_ReturnsPaginated(t *testing.T) {
	bms := make([]*repository.BookmarkWithArticle, 5)
	for i := range bms {
		bms[i] = &repository.BookmarkWithArticle{
			Bookmark: domain.Bookmark{ID: uuid.New()},
		}
	}
	repo := &mockBookmarkRepo{bookmarks: bms}
	uc := usecase.NewBookmarkUsecase(repo, &mockArticleRepoBookmark{})

	result, err := uc.List(context.Background(), uuid.New(), 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 5 {
		t.Errorf("expected 5, got %d", result.Total)
	}
}
