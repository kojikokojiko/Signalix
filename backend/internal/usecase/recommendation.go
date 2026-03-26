package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

// RateLimitStore checks per-user rate limits.
type RateLimitStore interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// RefreshPublisher publishes a recommendation refresh request to the scoring worker.
type RefreshPublisher interface {
	PublishRecommendationRefresh(ctx context.Context, userID string) error
}

// RecommendationListInput holds parameters for listing recommendations.
type RecommendationListInput struct {
	UserID   uuid.UUID
	Language *string
	Page     int
	PerPage  int
}

// RecommendationListOutput holds the result of listing recommendations.
type RecommendationListOutput struct {
	Items              []*domain.RecommendedItem
	Total              int
	HasInterestProfile bool
}

// RecommendationUsecase implements recommendation business logic.
type RecommendationUsecase struct {
	repo      repository.RecommendationRepository
	interests repository.InterestRepository
	rateLimit RateLimitStore
	publisher RefreshPublisher
}

// NewRecommendationUsecase creates a new RecommendationUsecase.
func NewRecommendationUsecase(
	repo repository.RecommendationRepository,
	interests repository.InterestRepository,
	rateLimit RateLimitStore,
	publisher RefreshPublisher,
) *RecommendationUsecase {
	return &RecommendationUsecase{repo: repo, interests: interests, rateLimit: rateLimit, publisher: publisher}
}

// List returns paginated recommendations for a user.
func (uc *RecommendationUsecase) List(ctx context.Context, in RecommendationListInput) (RecommendationListOutput, error) {
	items, total, err := uc.repo.List(ctx, in.UserID, in.Language, in.Page, in.PerPage)
	if err != nil {
		return RecommendationListOutput{}, fmt.Errorf("recommendation list: %w", err)
	}
	return RecommendationListOutput{
		Items:              items,
		Total:              total,
		HasInterestProfile: total > 0,
	}, nil
}

// RequestRefresh checks rate limit then publishes to the scoring worker.
func (uc *RecommendationUsecase) RequestRefresh(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("rec:refresh:%s", userID)
	allowed, err := uc.rateLimit.Allow(ctx, key)
	if err != nil {
		return fmt.Errorf("rate limit check: %w", err)
	}
	if !allowed {
		return ErrRateLimitExceeded
	}
	if err := uc.publisher.PublishRecommendationRefresh(ctx, userID.String()); err != nil {
		return fmt.Errorf("publish refresh: %w", err)
	}
	return nil
}

// ComputeAndStore scores candidate articles and upserts recommendation logs.
func (uc *RecommendationUsecase) ComputeAndStore(ctx context.Context, userID uuid.UUID, language *string) error {
	candidates, err := uc.repo.ListCandidates(ctx, userID, language, 200)
	if err != nil {
		return fmt.Errorf("list candidates: %w", err)
	}

	positiveTags, err := uc.repo.GetPositiveFeedbackTagFreq(ctx, userID)
	if err != nil {
		return fmt.Errorf("get positive tag freq: %w", err)
	}

	userInterests, err := uc.interests.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list user interests: %w", err)
	}

	// Build interest weight map by tag ID.
	interestWeights := make(map[uuid.UUID]float64, len(userInterests))
	for _, ui := range userInterests {
		interestWeights[ui.TagID] = ui.Weight
	}

	now := time.Now().UTC()

	for _, article := range candidates {
		relevance := computeRelevance(article.Tags, interestWeights)
		freshness := FreshnessScorePtr(article.Article.PublishedAt)
		trend := article.Article.TrendScore
		sourceQuality := 0.0
		if article.Source != nil {
			sourceQuality = article.Source.QualityScore
		}
		personalization := PersonalizationBoost(article.Tags, positiveTags)

		scores := NewScoreBreakdown(relevance, freshness, trend, sourceQuality, personalization)
		totalScore := scores.Total()

		topTagName := ""
		if len(article.Tags) > 0 {
			topTagName = article.Tags[0].Name
		}
		sourceName := ""
		if article.Source != nil {
			sourceName = article.Source.Name
		}
		explanation := GenerateExplanation(scores, topTagName, sourceName)

		log := &domain.RecommendationLog{
			ID:                   uuid.New(),
			UserID:               userID,
			ArticleID:            article.Article.ID,
			TotalScore:           totalScore,
			RelevanceScore:       relevance,
			FreshnessScore:       freshness,
			TrendScore:           trend,
			SourceQualityScore:   sourceQuality,
			PersonalizationBoost: personalization,
			Explanation:          explanation,
			GeneratedAt:          now,
			ExpiresAt:            now.Add(24 * time.Hour),
		}

		if err := uc.repo.Upsert(ctx, log); err != nil {
			return fmt.Errorf("upsert recommendation log: %w", err)
		}
	}
	return nil
}

// computeRelevance returns a relevance score based on user interest tag overlap.
func computeRelevance(tags []domain.TagWithConfidence, interestWeights map[uuid.UUID]float64) float64 {
	if len(tags) == 0 || len(interestWeights) == 0 {
		return 0.0
	}
	score := 0.0
	matched := 0
	for _, tag := range tags {
		if w, ok := interestWeights[tag.ID]; ok {
			score += w * tag.Confidence
			matched++
		}
	}
	if matched == 0 {
		return 0.0
	}
	v := score / float64(matched)
	if v > 1.0 {
		return 1.0
	}
	return v
}
