package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// --- mock source repo ---

type mockSourceRepo struct {
	sources []*domain.Source
}

func (m *mockSourceRepo) List(_ context.Context, f repository.SourceFilter) ([]*domain.Source, int, error) {
	var out []*domain.Source
	for _, s := range m.sources {
		if f.Category != nil && s.Category != *f.Category {
			continue
		}
		if f.Language != nil && s.Language != *f.Language {
			continue
		}
		out = append(out, s)
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	pp := f.PerPage
	if pp < 1 {
		pp = 50
	}
	total := len(out)
	start := (page - 1) * pp
	if start >= total {
		return []*domain.Source{}, total, nil
	}
	end := start + pp
	if end > total {
		end = total
	}
	return out[start:end], total, nil
}

func (m *mockSourceRepo) FindByID(_ context.Context, id string) (*domain.Source, error) {
	for _, s := range m.sources {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockSourceRepo) ListDueForFetch(_ context.Context, limit int) ([]*domain.Source, error) {
	return m.sources, nil
}
func (m *mockSourceRepo) UpdateAfterFetch(_ context.Context, id string, success bool) error {
	return nil
}
func (m *mockSourceRepo) UpdateStatus(_ context.Context, id string, status string) error {
	return nil
}
func (m *mockSourceRepo) ListAll(_ context.Context, _ repository.SourceFilter) ([]*domain.Source, int, error) {
	return m.sources, len(m.sources), nil
}
func (m *mockSourceRepo) Create(_ context.Context, _ *domain.Source) error { return nil }
func (m *mockSourceRepo) Update(_ context.Context, _ string, _ repository.SourceUpdateFields) (*domain.Source, error) {
	return nil, nil
}
func (m *mockSourceRepo) Delete(_ context.Context, _ string) error { return nil }

// --- mock article repo for source tests ---

type mockArticleRepoForSource struct {
	articles []*domain.ArticleWithDetails
}

func (m *mockArticleRepoForSource) List(_ context.Context, f repository.ArticleFilter) ([]*domain.ArticleWithDetails, int, error) {
	return m.articles, len(m.articles), nil
}
func (m *mockArticleRepoForSource) FindByID(_ context.Context, _ interface{ String() string }) (*domain.ArticleWithDetails, error) {
	return nil, nil
}
func (m *mockArticleRepoForSource) Trending(_ context.Context, _ string, _ *string, _, _ int) ([]*domain.ArticleWithDetails, int, error) {
	return m.articles, len(m.articles), nil
}
func (m *mockArticleRepoForSource) Insert(_ context.Context, _ *domain.Article) error {
	return nil
}
func (m *mockArticleRepoForSource) ListRecentBySource(_ context.Context, _ interface{ String() string }, limit int) ([]*domain.ArticleWithDetails, error) {
	if limit > len(m.articles) {
		return m.articles, nil
	}
	return m.articles[:limit], nil
}
func (m *mockArticleRepoForSource) GetRawForProcessing(_ context.Context, _ uuid.UUID) (*domain.Article, error) {
	return nil, nil
}
func (m *mockArticleRepoForSource) UpdateStatus(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockArticleRepoForSource) UpdateCleanContent(_ context.Context, _ uuid.UUID, _ string, _ *string) error {
	return nil
}
func (m *mockArticleRepoForSource) UpdateTrendScore(_ context.Context, _ uuid.UUID, _ float64) error {
	return nil
}
func (m *mockArticleRepoForSource) SaveSummary(_ context.Context, _ *domain.ArticleSummary) error {
	return nil
}
func (m *mockArticleRepoForSource) SaveEmbedding(_ context.Context, _ uuid.UUID, _ []float32) error {
	return nil
}
func (m *mockArticleRepoForSource) SaveTags(_ context.Context, _ uuid.UUID, _ []domain.TagWithConfidence) error {
	return nil
}
func (m *mockArticleRepoForSource) ListAllTagNames(_ context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockArticleRepoForSource) FindTagIDsByName(_ context.Context, _ []string) (map[string]uuid.UUID, error) {
	return nil, nil
}

// --- tests ---

func TestSourceUsecase_List_ReturnsPaginatedResults(t *testing.T) {
	repo := &mockSourceRepo{
		sources: []*domain.Source{
			{ID: "1", Name: "Source A", Category: "tech", Language: "en", Status: "active"},
			{ID: "2", Name: "Source B", Category: "tech", Language: "ja", Status: "active"},
			{ID: "3", Name: "Source C", Category: "science", Language: "en", Status: "active"},
		},
	}
	uc := usecase.NewSourceUsecase(repo)

	result, err := uc.List(context.Background(), usecase.SourceListInput{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
	if len(result.Sources) != 3 {
		t.Errorf("expected 3 sources, got %d", len(result.Sources))
	}
}

func TestSourceUsecase_List_FiltersByCategory(t *testing.T) {
	repo := &mockSourceRepo{
		sources: []*domain.Source{
			{ID: "1", Category: "tech", Language: "en", Status: "active"},
			{ID: "2", Category: "science", Language: "en", Status: "active"},
		},
	}
	uc := usecase.NewSourceUsecase(repo)

	cat := "tech"
	result, err := uc.List(context.Background(), usecase.SourceListInput{Category: &cat, Page: 1, PerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1, got %d", result.Total)
	}
}

func TestSourceUsecase_GetByID_Found(t *testing.T) {
	repo := &mockSourceRepo{
		sources: []*domain.Source{
			{ID: "abc-123", Name: "Go Blog", Status: "active"},
		},
	}
	uc := usecase.NewSourceUsecase(repo)

	s, err := uc.GetByID(context.Background(), "abc-123")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected source, got nil")
	}
	if s.ID != "abc-123" {
		t.Errorf("expected id abc-123, got %s", s.ID)
	}
}

func TestSourceUsecase_GetByID_NotFound(t *testing.T) {
	repo := &mockSourceRepo{}
	uc := usecase.NewSourceUsecase(repo)

	s, err := uc.GetByID(context.Background(), "does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Error("expected nil for unknown id")
	}
}

func TestSourceUsecase_List_Pagination(t *testing.T) {
	sources := make([]*domain.Source, 10)
	for i := range sources {
		sources[i] = &domain.Source{ID: itoa(i), Status: "active"}
	}
	repo := &mockSourceRepo{sources: sources}
	uc := usecase.NewSourceUsecase(repo)

	result, _ := uc.List(context.Background(), usecase.SourceListInput{Page: 2, PerPage: 3})
	if len(result.Sources) != 3 {
		t.Errorf("expected 3, got %d", len(result.Sources))
	}
	if result.TotalPages != 4 {
		t.Errorf("expected 4 pages, got %d", result.TotalPages)
	}
	if !result.HasNext {
		t.Error("expected HasNext=true")
	}
	if !result.HasPrev {
		t.Error("expected HasPrev=true")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [10]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
