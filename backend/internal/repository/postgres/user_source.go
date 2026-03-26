package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kojikokojiko/signalix/internal/domain"
)

type UserSourceRepository struct {
	db *pgxpool.Pool
}

func NewUserSourceRepository(db *pgxpool.Pool) *UserSourceRepository {
	return &UserSourceRepository{db: db}
}

func (r *UserSourceRepository) List(ctx context.Context, userID uuid.UUID) ([]*domain.Source, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.name, s.feed_url, s.site_url, s.description, s.category, s.language,
		       s.fetch_interval_minutes, s.quality_score, s.status, s.last_fetched_at,
		       s.consecutive_failures, s.created_at, s.updated_at
		FROM user_sources us
		JOIN sources s ON s.id = us.source_id
		WHERE us.user_id = $1
		ORDER BY us.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query user sources: %w", err)
	}
	defer rows.Close()

	var sources []*domain.Source
	for rows.Next() {
		s := &domain.Source{}
		if err := rows.Scan(
			&s.ID, &s.Name, &s.FeedURL, &s.SiteURL, &s.Description, &s.Category, &s.Language,
			&s.FetchIntervalMinutes, &s.QualityScore, &s.Status, &s.LastFetchedAt,
			&s.ConsecutiveFailures, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user source: %w", err)
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func (r *UserSourceRepository) Subscribe(ctx context.Context, userID uuid.UUID, sourceID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_sources (id, user_id, source_id)
		VALUES (uuid_generate_v4(), $1, $2)
	`, userID, sourceID)
	return err
}

func (r *UserSourceRepository) Unsubscribe(ctx context.Context, userID uuid.UUID, sourceID string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM user_sources WHERE user_id = $1 AND source_id = $2
	`, userID, sourceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserSourceRepository) IsSubscribed(ctx context.Context, userID uuid.UUID, sourceID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM user_sources WHERE user_id = $1 AND source_id = $2)
	`, userID, sourceID).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return exists, err
}
