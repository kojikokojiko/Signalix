package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
)

type RecommendationRepository interface {
	// ユーザーのレコメンド一覧取得（スコア降順）
	List(ctx context.Context, userID uuid.UUID, language *string, page, perPage int) ([]*domain.RecommendedItem, int, error)
	// レコメンドログ保存（UPSERT）
	Upsert(ctx context.Context, log *domain.RecommendationLog) error
	// フィード最終更新日時
	LastRefreshedAt(ctx context.Context, userID uuid.UUID) (*domain.RecommendationLog, error)
	// レコメンド計算用の候補記事取得
	ListCandidates(ctx context.Context, userID uuid.UUID, language *string, limit int) ([]*domain.ArticleWithDetails, error)
	// ユーザーの positive フィードバックタグ（過去 30 日）
	GetPositiveFeedbackTagFreq(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]float64, error)
}
