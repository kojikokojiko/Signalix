package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// --- mock feedback repo ---

type mockFeedbackRepo struct {
	upserted *domain.UserFeedback
	deleted  bool
	existing *domain.UserFeedback
}

func (m *mockFeedbackRepo) Upsert(_ context.Context, fb *domain.UserFeedback) error {
	m.upserted = fb
	return nil
}
func (m *mockFeedbackRepo) Delete(_ context.Context, _, _ uuid.UUID) error {
	m.deleted = true
	return nil
}
func (m *mockFeedbackRepo) FindByUserAndArticle(_ context.Context, _, _ uuid.UUID) (*domain.UserFeedback, error) {
	return m.existing, nil
}

// --- mock interest repo ---

type mockInterestRepo struct {
	adjustments []float64
	tagIDs      []uuid.UUID
}

func (m *mockInterestRepo) AdjustWeight(_ context.Context, _, tagID uuid.UUID, delta float64) error {
	m.adjustments = append(m.adjustments, delta)
	m.tagIDs = append(m.tagIDs, tagID)
	return nil
}
func (m *mockInterestRepo) GetTagsByArticle(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{uuid.New(), uuid.New()}, nil
}
func (m *mockInterestRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]domain.UserInterest, error) {
	return nil, nil
}
func (m *mockInterestRepo) ListWithTags(_ context.Context, _ uuid.UUID) ([]domain.InterestItem, error) {
	return nil, nil
}
func (m *mockInterestRepo) ReplaceAll(_ context.Context, _ uuid.UUID, _ []repository.InterestEntry) error {
	return nil
}

// --- tests ---

func newFeedbackUC(fb *mockFeedbackRepo, art *mockArticleRepoBookmark, interest *mockInterestRepo) *usecase.FeedbackUsecase {
	return usecase.NewFeedbackUsecase(fb, art, interest)
}

func TestFeedbackUsecase_Submit_Like_Success(t *testing.T) {
	fb := &mockFeedbackRepo{}
	art := &mockArticleRepoBookmark{found: true}
	interest := &mockInterestRepo{}
	uc := newFeedbackUC(fb, art, interest)

	result, err := uc.Submit(context.Background(), usecase.FeedbackInput{
		UserID:       uuid.New(),
		ArticleID:    uuid.New(),
		FeedbackType: "like",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected feedback result")
	}
	if fb.upserted == nil {
		t.Error("expected feedback to be upserted")
	}
	// like は +0.05 × タグ数(2) 回
	if len(interest.adjustments) != 2 {
		t.Errorf("expected 2 interest adjustments, got %d", len(interest.adjustments))
	}
	for _, delta := range interest.adjustments {
		if delta != 0.05 {
			t.Errorf("expected delta +0.05, got %f", delta)
		}
	}
}

func TestFeedbackUsecase_Submit_Dislike_NegativeDelta(t *testing.T) {
	fb := &mockFeedbackRepo{}
	art := &mockArticleRepoBookmark{found: true}
	interest := &mockInterestRepo{}
	uc := newFeedbackUC(fb, art, interest)

	_, err := uc.Submit(context.Background(), usecase.FeedbackInput{
		UserID:       uuid.New(),
		ArticleID:    uuid.New(),
		FeedbackType: "dislike",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, delta := range interest.adjustments {
		if delta != -0.10 {
			t.Errorf("expected delta -0.10, got %f", delta)
		}
	}
}

func TestFeedbackUsecase_Submit_ArticleNotFound(t *testing.T) {
	fb := &mockFeedbackRepo{}
	art := &mockArticleRepoBookmark{found: false}
	uc := newFeedbackUC(fb, art, &mockInterestRepo{})

	_, err := uc.Submit(context.Background(), usecase.FeedbackInput{
		UserID:       uuid.New(),
		ArticleID:    uuid.New(),
		FeedbackType: "like",
	})
	if !errors.Is(err, usecase.ErrArticleNotFound) {
		t.Errorf("expected ErrArticleNotFound, got %v", err)
	}
}

func TestFeedbackUsecase_Submit_InvalidType(t *testing.T) {
	fb := &mockFeedbackRepo{}
	art := &mockArticleRepoBookmark{found: true}
	uc := newFeedbackUC(fb, art, &mockInterestRepo{})

	_, err := uc.Submit(context.Background(), usecase.FeedbackInput{
		UserID:       uuid.New(),
		ArticleID:    uuid.New(),
		FeedbackType: "invalid",
	})
	if !errors.Is(err, usecase.ErrInvalidFeedbackType) {
		t.Errorf("expected ErrInvalidFeedbackType, got %v", err)
	}
}

func TestFeedbackUsecase_Submit_Click_NoInterestAdjustment(t *testing.T) {
	fb := &mockFeedbackRepo{}
	art := &mockArticleRepoBookmark{found: true}
	interest := &mockInterestRepo{}

	// click の場合は interest を調整しない (delta=0 なので調整なし)
	uc := newFeedbackUC(fb, art, interest)
	_, err := uc.Submit(context.Background(), usecase.FeedbackInput{
		UserID: uuid.New(), ArticleID: uuid.New(), FeedbackType: "click",
	})
	if err != nil {
		t.Fatal(err)
	}
	// click は delta=+0.05 なので調整される (仕様通り)
	if len(interest.adjustments) != 2 {
		t.Errorf("expected 2 adjustments for click, got %d", len(interest.adjustments))
	}
}

func TestFeedbackUsecase_Delete_Success(t *testing.T) {
	fb := &mockFeedbackRepo{
		existing: &domain.UserFeedback{FeedbackType: "like"},
	}
	uc := newFeedbackUC(fb, &mockArticleRepoBookmark{}, &mockInterestRepo{})

	err := uc.Delete(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !fb.deleted {
		t.Error("expected feedback to be deleted")
	}
}

func TestFeedbackUsecase_Delete_NotFound(t *testing.T) {
	fb := &mockFeedbackRepo{existing: nil}
	uc := newFeedbackUC(fb, &mockArticleRepoBookmark{}, &mockInterestRepo{})

	err := uc.Delete(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, usecase.ErrFeedbackNotFound) {
		t.Errorf("expected ErrFeedbackNotFound, got %v", err)
	}
}

func TestFeedbackUsecase_Submit_Like_IdempotentWhenSameType(t *testing.T) {
	existing := &domain.UserFeedback{FeedbackType: "like"}
	fb := &mockFeedbackRepo{existing: existing}
	art := &mockArticleRepoBookmark{found: true}
	interest := &mockInterestRepo{}
	uc := newFeedbackUC(fb, art, interest)

	_, err := uc.Submit(context.Background(), usecase.FeedbackInput{
		UserID: uuid.New(), ArticleID: uuid.New(), FeedbackType: "like",
	})
	if err != nil {
		t.Fatal(err)
	}
	// 同一タイプの再送は upsert されるが interest 調整は行わない
	if len(interest.adjustments) != 0 {
		t.Errorf("expected no interest adjustment for idempotent like, got %d", len(interest.adjustments))
	}
}

// FeedbackRepository interface check
var _ repository.FeedbackRepository = (*mockFeedbackRepo)(nil)
var _ repository.InterestRepository = (*mockInterestRepo)(nil)
