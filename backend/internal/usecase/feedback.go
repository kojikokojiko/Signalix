package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

type FeedbackInput struct {
	UserID       uuid.UUID
	ArticleID    uuid.UUID
	FeedbackType string
}

type FeedbackUsecase struct {
	feedback  repository.FeedbackRepository
	articles  repository.ArticleRepository
	interests repository.InterestRepository
}

func NewFeedbackUsecase(
	feedback repository.FeedbackRepository,
	articles repository.ArticleRepository,
	interests repository.InterestRepository,
) *FeedbackUsecase {
	return &FeedbackUsecase{feedback: feedback, articles: articles, interests: interests}
}

func (uc *FeedbackUsecase) Submit(ctx context.Context, in FeedbackInput) (*domain.UserFeedback, error) {
	if !domain.ValidFeedbackTypes[in.FeedbackType] {
		return nil, ErrInvalidFeedbackType
	}

	a, err := uc.articles.FindByID(ctx, in.ArticleID)
	if err != nil {
		return nil, fmt.Errorf("find article: %w", err)
	}
	if a == nil {
		return nil, ErrArticleNotFound
	}

	// 既存フィードバックチェック（冪等性: 同タイプは再処理不要）
	existing, err := uc.feedback.FindByUserAndArticle(ctx, in.UserID, in.ArticleID)
	if err != nil {
		return nil, fmt.Errorf("find feedback: %w", err)
	}

	isIdempotent := existing != nil && existing.FeedbackType == in.FeedbackType
	// click は常に新規挿入のため冪等扱いしない
	if in.FeedbackType == "click" {
		isIdempotent = false
	}

	fb := &domain.UserFeedback{
		ID:           uuid.New(),
		UserID:       in.UserID,
		ArticleID:    in.ArticleID,
		FeedbackType: in.FeedbackType,
		CreatedAt:    time.Now().UTC(),
	}

	if err := uc.feedback.Upsert(ctx, fb); err != nil {
		return nil, fmt.Errorf("upsert feedback: %w", err)
	}

	// 冪等でない場合のみ interest を更新
	if !isIdempotent {
		delta := domain.FeedbackWeightDelta(in.FeedbackType)
		if delta != 0 {
			if err := uc.adjustInterests(ctx, in.UserID, in.ArticleID, delta); err != nil {
				// interest 更新失敗はログのみで継続（非致命的）
				_ = err
			}
		}
	}

	return fb, nil
}

func (uc *FeedbackUsecase) Delete(ctx context.Context, userID, articleID uuid.UUID) error {
	existing, err := uc.feedback.FindByUserAndArticle(ctx, userID, articleID)
	if err != nil {
		return fmt.Errorf("find feedback: %w", err)
	}
	if existing == nil {
		return ErrFeedbackNotFound
	}
	if err := uc.feedback.Delete(ctx, userID, articleID); err != nil {
		return fmt.Errorf("delete feedback: %w", err)
	}
	return nil
}

func (uc *FeedbackUsecase) adjustInterests(ctx context.Context, userID, articleID uuid.UUID, delta float64) error {
	tagIDs, err := uc.interests.GetTagsByArticle(ctx, articleID)
	if err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if err := uc.interests.AdjustWeight(ctx, userID, tagID, delta); err != nil {
			return err
		}
	}
	return nil
}
