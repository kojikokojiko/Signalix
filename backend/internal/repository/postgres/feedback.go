package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

type FeedbackRepository struct {
	db *pgxpool.Pool
}

func NewFeedbackRepository(db *pgxpool.Pool) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

func (r *FeedbackRepository) Upsert(ctx context.Context, fb *domain.UserFeedback) error {
	if fb.FeedbackType == "click" {
		// click は複数記録可のため単純 INSERT
		_, err := r.db.Exec(ctx,
			`INSERT INTO user_feedback (id, user_id, article_id, feedback_type, created_at)
			 VALUES ($1,$2,$3,$4,$5)`,
			fb.ID, fb.UserID, fb.ArticleID, fb.FeedbackType, fb.CreatedAt,
		)
		return err
	}
	// その他は UPSERT (unique constraint: user_id + article_id で非 click は 1 件)
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_feedback (id, user_id, article_id, feedback_type, created_at)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (user_id, article_id) WHERE feedback_type != 'click'
		DO UPDATE SET feedback_type = EXCLUDED.feedback_type, created_at = EXCLUDED.created_at, id = EXCLUDED.id
	`, fb.ID, fb.UserID, fb.ArticleID, fb.FeedbackType, fb.CreatedAt)
	return err
}

func (r *FeedbackRepository) Delete(ctx context.Context, userID, articleID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_feedback WHERE user_id=$1 AND article_id=$2 AND feedback_type != 'click'`,
		userID, articleID,
	)
	return err
}

func (r *FeedbackRepository) FindByUserAndArticle(ctx context.Context, userID, articleID uuid.UUID) (*domain.UserFeedback, error) {
	fb := &domain.UserFeedback{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, article_id, feedback_type, created_at
		FROM user_feedback
		WHERE user_id=$1 AND article_id=$2 AND feedback_type != 'click'
		LIMIT 1
	`, userID, articleID).Scan(&fb.ID, &fb.UserID, &fb.ArticleID, &fb.FeedbackType, &fb.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return fb, err
}

// InterestRepository implementation

type InterestRepository struct {
	db *pgxpool.Pool
}

func NewInterestRepository(db *pgxpool.Pool) *InterestRepository {
	return &InterestRepository{db: db}
}

func (r *InterestRepository) AdjustWeight(ctx context.Context, userID, tagID uuid.UUID, delta float64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_interests (id, user_id, tag_id, weight, source, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, GREATEST(0.0, LEAST(1.0, 0.5 + $3)), 'inferred', NOW())
		ON CONFLICT (user_id, tag_id)
		DO UPDATE SET
			weight = GREATEST(0.0, LEAST(1.0, user_interests.weight + $3)),
			source = 'inferred',
			updated_at = NOW()
	`, userID, tagID, delta)
	return err
}

func (r *InterestRepository) GetTagsByArticle(ctx context.Context, articleID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx,
		`SELECT tag_id FROM article_tags WHERE article_id = $1`, articleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *InterestRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.UserInterest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, tag_id, weight, source, updated_at
		FROM user_interests
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interests []domain.UserInterest
	for rows.Next() {
		var ui domain.UserInterest
		if err := rows.Scan(&ui.ID, &ui.UserID, &ui.TagID, &ui.Weight, &ui.Source, &ui.UpdatedAt); err != nil {
			return nil, err
		}
		interests = append(interests, ui)
	}
	return interests, rows.Err()
}

func (r *InterestRepository) ListWithTags(ctx context.Context, userID uuid.UUID) ([]domain.InterestItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.name, t.id, t.category, ui.weight, ui.source
		FROM user_interests ui
		JOIN tags t ON t.id = ui.tag_id
		WHERE ui.user_id = $1
		ORDER BY ui.weight DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.InterestItem
	for rows.Next() {
		var item domain.InterestItem
		if err := rows.Scan(&item.TagName, &item.TagID, &item.Category, &item.Weight, &item.Source); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []domain.InterestItem{}
	}
	return items, rows.Err()
}

func (r *InterestRepository) ReplaceAll(ctx context.Context, userID uuid.UUID, entries []repository.InterestEntry) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 既存の interests を削除
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}

	// 新しい interests を挿入
	for _, e := range entries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (id, user_id, tag_id, weight, source, updated_at)
			VALUES (uuid_generate_v4(), $1, $2, $3, 'manual', NOW())
		`, userID, e.TagID, e.Weight); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// TagRepository implementation

type TagRepository struct {
	db *pgxpool.Pool
}

func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) FindByName(ctx context.Context, name string) (*domain.Tag, error) {
	tag := &domain.Tag{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, category FROM tags WHERE name = $1`, name,
	).Scan(&tag.ID, &tag.Name, &tag.Category)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tag, nil
}
