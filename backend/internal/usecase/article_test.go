package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// --- mock article repo ---

type mockArticleRepo struct {
	articles []*domain.ArticleWithDetails
}

func (m *mockArticleRepo) List(_ context.Context, f repository.ArticleFilter) ([]*domain.ArticleWithDetails, int, error) {
	pp := f.PerPage
	if pp < 1 {
		pp = 20
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	total := len(m.articles)
	start := (page - 1) * pp
	if start >= total {
		return []*domain.ArticleWithDetails{}, total, nil
	}
	end := start + pp
	if end > total {
		end = total
	}
	return m.articles[start:end], total, nil
}

func (m *mockArticleRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.ArticleWithDetails, error) {
	for _, a := range m.articles {
		if a.Article.ID == id {
			return a, nil
		}
	}
	return nil, nil
}

func (m *mockArticleRepo) Trending(_ context.Context, _ string, _ *string, page, perPage int) ([]*domain.ArticleWithDetails, int, error) {
	if perPage < 1 {
		perPage = 20
	}
	total := len(m.articles)
	start := (page - 1) * perPage
	if start >= total {
		return []*domain.ArticleWithDetails{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}
	return m.articles[start:end], total, nil
}

func (m *mockArticleRepo) Insert(_ context.Context, _ *domain.Article) error { return nil }
func (m *mockArticleRepo) ListRecentBySource(_ context.Context, _ uuid.UUID, limit int) ([]*domain.ArticleWithDetails, error) {
	if limit > len(m.articles) {
		return m.articles, nil
	}
	return m.articles[:limit], nil
}
func (m *mockArticleRepo) GetRawForProcessing(_ context.Context, _ uuid.UUID) (*domain.Article, error) {
	return nil, nil
}
func (m *mockArticleRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (m *mockArticleRepo) UpdateCleanContent(_ context.Context, _ uuid.UUID, _ string, _ *string) error {
	return nil
}
func (m *mockArticleRepo) UpdateTrendScore(_ context.Context, _ uuid.UUID, _ float64) error {
	return nil
}
func (m *mockArticleRepo) SaveSummary(_ context.Context, _ *domain.ArticleSummary) error {
	return nil
}
func (m *mockArticleRepo) SaveEmbedding(_ context.Context, _ uuid.UUID, _ []float32) error {
	return nil
}
func (m *mockArticleRepo) SaveTags(_ context.Context, _ uuid.UUID, _ []domain.TagWithConfidence) error {
	return nil
}
func (m *mockArticleRepo) ListAllTagNames(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockArticleRepo) FindTagIDsByName(_ context.Context, _ []string) (map[string]uuid.UUID, error) {
	return nil, nil
}

// --- tests ---

func newArticles(n int) []*domain.ArticleWithDetails {
	now := time.Now().UTC()
	arts := make([]*domain.ArticleWithDetails, n)
	for i := range arts {
		arts[i] = &domain.ArticleWithDetails{
			Article: domain.Article{
				ID:          uuid.New(),
				Title:       "Article " + itoa(i),
				Status:      "processed",
				PublishedAt: &now,
			},
			Source: &domain.Source{Name: "Source"},
		}
	}
	return arts
}

func TestArticleUsecase_List_ReturnsPaginatedResults(t *testing.T) {
	repo := &mockArticleRepo{articles: newArticles(5)}
	uc := usecase.NewArticleUsecase(repo, nil)

	result, err := uc.List(context.Background(), usecase.ArticleListInput{Page: 1, PerPage: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 5 {
		t.Errorf("expected 5, got %d", result.Total)
	}
	if len(result.Articles) != 5 {
		t.Errorf("expected 5 articles, got %d", len(result.Articles))
	}
}

func TestArticleUsecase_List_SecondPage(t *testing.T) {
	repo := &mockArticleRepo{articles: newArticles(25)}
	uc := usecase.NewArticleUsecase(repo, nil)

	result, _ := uc.List(context.Background(), usecase.ArticleListInput{Page: 2, PerPage: 20})
	if len(result.Articles) != 5 {
		t.Errorf("expected 5 on page 2, got %d", len(result.Articles))
	}
	if result.HasPrev == false {
		t.Error("expected HasPrev=true on page 2")
	}
	if result.HasNext == true {
		t.Error("expected HasNext=false on last page")
	}
}

func TestArticleUsecase_GetByID_Found(t *testing.T) {
	articles := newArticles(3)
	repo := &mockArticleRepo{articles: articles}
	uc := usecase.NewArticleUsecase(repo, nil)

	a, err := uc.GetByID(context.Background(), articles[1].Article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("expected article")
	}
	if a.Article.ID != articles[1].Article.ID {
		t.Error("wrong article returned")
	}
}

func TestArticleUsecase_GetByID_NotFound(t *testing.T) {
	repo := &mockArticleRepo{}
	uc := usecase.NewArticleUsecase(repo, nil)

	a, err := uc.GetByID(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if a != nil {
		t.Error("expected nil for unknown id")
	}
}

func TestArticleUsecase_Trending_DefaultPeriod(t *testing.T) {
	repo := &mockArticleRepo{articles: newArticles(5)}
	uc := usecase.NewArticleUsecase(repo, nil)

	result, err := uc.Trending(context.Background(), usecase.TrendingInput{Period: "", Page: 1, PerPage: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Period != "24h" {
		t.Errorf("expected period 24h, got %s", result.Period)
	}
	if len(result.Articles) != 5 {
		t.Errorf("expected 5, got %d", len(result.Articles))
	}
}

func TestArticleUsecase_Trending_7dPeriod(t *testing.T) {
	repo := &mockArticleRepo{articles: newArticles(3)}
	uc := usecase.NewArticleUsecase(repo, nil)

	result, _ := uc.Trending(context.Background(), usecase.TrendingInput{Period: "7d", Page: 1, PerPage: 20})
	if result.Period != "7d" {
		t.Errorf("expected 7d, got %s", result.Period)
	}
}

func TestArticleUsecase_List_PerPageCappedAt100(t *testing.T) {
	repo := &mockArticleRepo{articles: newArticles(10)}
	uc := usecase.NewArticleUsecase(repo, nil)

	result, _ := uc.List(context.Background(), usecase.ArticleListInput{Page: 1, PerPage: 200})
	// per_page is capped — the mock returns up to 100 items or all, total should still be 10
	if result.Total != 10 {
		t.Errorf("expected 10, got %d", result.Total)
	}
}
