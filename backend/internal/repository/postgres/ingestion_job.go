package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

type IngestionJobStore struct {
	db *pgxpool.Pool
}

func NewIngestionJobStore(db *pgxpool.Pool) *IngestionJobStore {
	return &IngestionJobStore{db: db}
}

func (s *IngestionJobStore) Begin(ctx context.Context, sourceID string) (string, error) {
	jobID := uuid.New().String()
	_, err := s.db.Exec(ctx, `
		INSERT INTO ingestion_jobs (id, source_id, status, started_at)
		VALUES ($1, $2, 'running', $3)
	`, jobID, sourceID, time.Now().UTC())
	if err != nil {
		return "", err
	}
	return jobID, nil
}

func (s *IngestionJobStore) Complete(ctx context.Context, jobID string, found, newCount, skipped int) error {
	_, err := s.db.Exec(ctx, `
		UPDATE ingestion_jobs
		SET status = 'completed',
		    articles_found = $1,
		    articles_new = $2,
		    articles_skipped = $3,
		    completed_at = $4
		WHERE id = $5
	`, found, newCount, skipped, time.Now().UTC(), jobID)
	return err
}

func (s *IngestionJobStore) Fail(ctx context.Context, jobID string, errMsg string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE ingestion_jobs
		SET status = 'failed', error_message = $1, completed_at = $2
		WHERE id = $3
	`, errMsg, time.Now().UTC(), jobID)
	return err
}

// List returns paginated ingestion jobs for admin.
func (s *IngestionJobStore) List(ctx context.Context, f repository.IngestionJobFilter) ([]*domain.IngestionJob, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 50
	}

	args := []any{}
	where := "WHERE 1=1"
	n := 1

	if f.SourceID != nil {
		where += " AND j.source_id = $" + itoa(n)
		args = append(args, *f.SourceID)
		n++
	}
	if f.Status != nil {
		where += " AND j.status = $" + itoa(n)
		args = append(args, *f.Status)
		n++
	}

	var total int
	if err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM ingestion_jobs j "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PerPage
	query := `
		SELECT j.id, j.source_id, COALESCE(src.name, '') AS source_name,
		       j.status, j.articles_found, j.articles_new, j.articles_skipped,
		       j.error_message, j.started_at, j.completed_at
		FROM ingestion_jobs j
		LEFT JOIN sources src ON src.id = j.source_id
		` + where + `
		ORDER BY j.started_at DESC
		LIMIT $` + itoa(n) + ` OFFSET $` + itoa(n+1)
	args = append(args, f.PerPage, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*domain.IngestionJob
	for rows.Next() {
		j := &domain.IngestionJob{}
		if err := rows.Scan(
			&j.ID, &j.SourceID, &j.SourceName,
			&j.Status, &j.ArticlesFound, &j.ArticlesNew, &j.ArticlesSkipped,
			&j.ErrorMessage, &j.StartedAt, &j.CompletedAt,
		); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, j)
	}
	return jobs, total, rows.Err()
}

// AdminStatsStore implements repository.AdminStatsRepository.
type AdminStatsStore struct {
	db *pgxpool.Pool
}

func NewAdminStatsStore(db *pgxpool.Pool) *AdminStatsStore {
	return &AdminStatsStore{db: db}
}

func (s *AdminStatsStore) GetStats(ctx context.Context) (*domain.AdminStats, error) {
	stats := &domain.AdminStats{}

	// Sources
	err := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'active'),
			COUNT(*) FILTER (WHERE status = 'degraded'),
			COUNT(*) FILTER (WHERE status = 'disabled')
		FROM sources
	`).Scan(&stats.Sources.Total, &stats.Sources.Active, &stats.Sources.Degraded, &stats.Sources.Disabled)
	if err != nil {
		return nil, err
	}

	// Articles
	err = s.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'processed'),
			COUNT(*) FILTER (WHERE status IN ('pending','fetched')),
			COUNT(*) FILTER (WHERE status = 'failed')
		FROM articles
	`).Scan(&stats.Articles.Total, &stats.Articles.Processed, &stats.Articles.Pending, &stats.Articles.Failed)
	if err != nil {
		return nil, err
	}

	// Ingestion jobs (last 24h)
	err = s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'completed'),
			COUNT(*) FILTER (WHERE status = 'failed')
		FROM ingestion_jobs
		WHERE started_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.IngestionJobs.Last24hCompleted, &stats.IngestionJobs.Last24hFailed)
	if err != nil {
		return nil, err
	}

	// Users
	err = s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
	`).Scan(&stats.Users.Total)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
