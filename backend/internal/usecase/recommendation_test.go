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

// --- mock recommendation repo ---

type mockRecommendationRepo struct {
	items      []*domain.RecommendedItem
	upserted   []*domain.RecommendationLog
	candidates []*domain.ArticleWithDetails
	positiveTags map[uuid.UUID]float64
	lastLog    *domain.RecommendationLog
}

func (m *mockRecommendationRepo) List(_ context.Context, _ uuid.UUID, _ *string, page, perPage int) ([]*domain.RecommendedItem, int, error) {
	total := len(m.items)
	start := (page-1)*perPage
	if start >= total { return []*domain.RecommendedItem{}, total, nil }
	end := start + perPage
	if end > total { end = total }
	return m.items[start:end], total, nil
}
func (m *mockRecommendationRepo) Upsert(_ context.Context, log *domain.RecommendationLog) error {
	m.upserted = append(m.upserted, log)
	return nil
}
func (m *mockRecommendationRepo) LastRefreshedAt(_ context.Context, _ uuid.UUID) (*domain.RecommendationLog, error) {
	return m.lastLog, nil
}
func (m *mockRecommendationRepo) ListCandidates(_ context.Context, _ uuid.UUID, _ *string, _ int) ([]*domain.ArticleWithDetails, error) {
	return m.candidates, nil
}
func (m *mockRecommendationRepo) GetPositiveFeedbackTagFreq(_ context.Context, _ uuid.UUID) (map[uuid.UUID]float64, error) {
	if m.positiveTags == nil {
		return map[uuid.UUID]float64{}, nil
	}
	return m.positiveTags, nil
}

// mock interest repo for recommendation
type mockInterestRepoForRec struct {
	interests []domain.UserInterest
}
func (m *mockInterestRepoForRec) AdjustWeight(_ context.Context, _, _ uuid.UUID, _ float64) error { return nil }
func (m *mockInterestRepoForRec) GetTagsByArticle(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) { return nil, nil }
func (m *mockInterestRepoForRec) ListByUser(_ context.Context, _ uuid.UUID) ([]domain.UserInterest, error) {
	return m.interests, nil
}
func (m *mockInterestRepoForRec) ListWithTags(_ context.Context, _ uuid.UUID) ([]domain.InterestItem, error) {
	return nil, nil
}
func (m *mockInterestRepoForRec) ReplaceAll(_ context.Context, _ uuid.UUID, _ []repository.InterestEntry) error {
	return nil
}

// mock rate limit store
type mockRateLimitStore struct {
	allowed bool
}
func (m *mockRateLimitStore) Allow(_ context.Context, _ string) (bool, error) { return m.allowed, nil }

// mock refresh publisher
type mockRefreshPublisher struct{}
func (m *mockRefreshPublisher) PublishRecommendationRefresh(_ context.Context, _ string) error { return nil }

// --- helpers ---

func makeRecommendedItems(n int) []*domain.RecommendedItem {
	now := time.Now().UTC()
	items := make([]*domain.RecommendedItem, n)
	for i := range items {
		pub := now.Add(-time.Duration(i) * time.Hour)
		items[i] = &domain.RecommendedItem{
			Article: &domain.ArticleWithDetails{
				Article: domain.Article{ID: uuid.New(), PublishedAt: &pub},
				Source:  &domain.Source{},
			},
			Log: &domain.RecommendationLog{
				TotalScore:  float64(n-i) / float64(n),
				Explanation: "test",
			},
		}
	}
	return items
}

// --- tests ---

func TestRecommendationUsecase_List_ReturnsPaginatedResults(t *testing.T) {
	repo := &mockRecommendationRepo{items: makeRecommendedItems(5)}
	uc := usecase.NewRecommendationUsecase(repo, &mockInterestRepoForRec{}, &mockRateLimitStore{allowed: true}, &mockRefreshPublisher{})

	result, err := uc.List(context.Background(), usecase.RecommendationListInput{
		UserID: uuid.New(), Page: 1, PerPage: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 5 {
		t.Errorf("expected 5, got %d", result.Total)
	}
	if !result.HasInterestProfile {
		t.Error("expected HasInterestProfile based on repo returning items")
	}
}

func TestRecommendationUsecase_List_HasInterestProfileFalseWhenEmpty(t *testing.T) {
	repo := &mockRecommendationRepo{items: nil}
	uc := usecase.NewRecommendationUsecase(repo, &mockInterestRepoForRec{}, &mockRateLimitStore{}, &mockRefreshPublisher{})

	result, err := uc.List(context.Background(), usecase.RecommendationListInput{
		UserID: uuid.New(), Page: 1, PerPage: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.HasInterestProfile {
		t.Error("expected HasInterestProfile=false when no recommendations")
	}
}

func TestRecommendationUsecase_Refresh_AllowedByRateLimit(t *testing.T) {
	repo := &mockRecommendationRepo{}
	uc := usecase.NewRecommendationUsecase(repo, &mockInterestRepoForRec{}, &mockRateLimitStore{allowed: true}, &mockRefreshPublisher{})

	err := uc.RequestRefresh(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRecommendationUsecase_Refresh_BlockedByRateLimit(t *testing.T) {
	repo := &mockRecommendationRepo{}
	uc := usecase.NewRecommendationUsecase(repo, &mockInterestRepoForRec{}, &mockRateLimitStore{allowed: false}, &mockRefreshPublisher{})

	err := uc.RequestRefresh(context.Background(), uuid.New())
	if err != usecase.ErrRateLimitExceeded {
		t.Errorf("expected ErrRateLimitExceeded, got %v", err)
	}
}

func TestRecommendationUsecase_ComputeScores_SavesToRepo(t *testing.T) {
	now := time.Now().UTC()
	pub := now.Add(-2 * time.Hour)
	candidates := []*domain.ArticleWithDetails{
		{
			Article: domain.Article{
				ID: uuid.New(), TrendScore: 0.8,
				PublishedAt: &pub, Status: "processed",
			},
			Source: &domain.Source{QualityScore: 0.9},
			Tags:   []domain.TagWithConfidence{{Tag: domain.Tag{ID: uuid.New()}, Confidence: 1.0}},
		},
	}
	repo := &mockRecommendationRepo{candidates: candidates}
	uc := usecase.NewRecommendationUsecase(repo, &mockInterestRepoForRec{}, &mockRateLimitStore{}, &mockRefreshPublisher{})

	if err := uc.ComputeAndStore(context.Background(), uuid.New(), nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.upserted) != 1 {
		t.Errorf("expected 1 upserted, got %d", len(repo.upserted))
	}
	log := repo.upserted[0]
	if log.TotalScore <= 0 {
		t.Error("expected positive total_score")
	}
	if log.Explanation == "" {
		t.Error("expected non-empty explanation")
	}
}

var _ repository.RecommendationRepository = (*mockRecommendationRepo)(nil)
