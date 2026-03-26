package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

type SourceRepository struct {
	db *pgxpool.Pool
}

func NewSourceRepository(db *pgxpool.Pool) *SourceRepository {
	return &SourceRepository{db: db}
}

func (r *SourceRepository) List(ctx context.Context, f repository.SourceFilter) ([]*domain.Source, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 50
	}

	args := []any{"active"}
	where := "WHERE s.status = $1"
	n := 2

	if f.Category != nil {
		where += " AND s.category = $" + itoa(n)
		args = append(args, *f.Category)
		n++
	}
	if f.Language != nil {
		where += " AND s.language = $" + itoa(n)
		args = append(args, *f.Language)
		n++
	}

	countRow := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM sources s "+where, args...)
	var total int
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PerPage
	query := `
		SELECT s.id, s.name, s.site_url, s.description, s.category, s.language,
		       s.quality_score, s.status, s.last_fetched_at, s.created_at,
		       COUNT(a.id) AS article_count
		FROM sources s
		LEFT JOIN articles a ON a.source_id = s.id AND a.status = 'processed'
		` + where + `
		GROUP BY s.id
		ORDER BY s.name ASC
		LIMIT $` + itoa(n) + ` OFFSET $` + itoa(n+1)
	args = append(args, f.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sources []*domain.Source
	for rows.Next() {
		s := &domain.Source{}
		cnt := 0
		if err := rows.Scan(
			&s.ID, &s.Name, &s.SiteURL, &s.Description, &s.Category, &s.Language,
			&s.QualityScore, &s.Status, &s.LastFetchedAt, &s.CreatedAt, &cnt,
		); err != nil {
			return nil, 0, err
		}
		s.ArticleCount = &cnt
		sources = append(sources, s)
	}
	return sources, total, rows.Err()
}

func (r *SourceRepository) FindByID(ctx context.Context, id string) (*domain.Source, error) {
	s := &domain.Source{}
	cnt := 0
	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.name, s.feed_url, s.site_url, s.description, s.category, s.language,
		       s.fetch_interval_minutes, s.quality_score, s.status, s.last_fetched_at,
		       s.consecutive_failures, s.created_at, s.updated_at,
		       COUNT(a.id) AS article_count
		FROM sources s
		LEFT JOIN articles a ON a.source_id = s.id AND a.status = 'processed'
		WHERE s.id = $1
		GROUP BY s.id
	`, id).Scan(
		&s.ID, &s.Name, &s.FeedURL, &s.SiteURL, &s.Description, &s.Category, &s.Language,
		&s.FetchIntervalMinutes, &s.QualityScore, &s.Status, &s.LastFetchedAt,
		&s.ConsecutiveFailures, &s.CreatedAt, &s.UpdatedAt, &cnt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.ArticleCount = &cnt
	return s, nil
}

func (r *SourceRepository) ListDueForFetch(ctx context.Context, limit int) ([]*domain.Source, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, feed_url, site_url, category, language,
		       fetch_interval_minutes, quality_score, status, last_fetched_at,
		       consecutive_failures, created_at, updated_at
		FROM sources
		WHERE status = 'active'
		  AND (last_fetched_at IS NULL
		       OR last_fetched_at < NOW() - (fetch_interval_minutes * INTERVAL '1 minute'))
		ORDER BY last_fetched_at ASC NULLS FIRST
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*domain.Source
	for rows.Next() {
		s := &domain.Source{}
		if err := rows.Scan(
			&s.ID, &s.Name, &s.FeedURL, &s.SiteURL, &s.Category, &s.Language,
			&s.FetchIntervalMinutes, &s.QualityScore, &s.Status, &s.LastFetchedAt,
			&s.ConsecutiveFailures, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func (r *SourceRepository) UpdateAfterFetch(ctx context.Context, id string, success bool) error {
	now := time.Now().UTC()
	if success {
		_, err := r.db.Exec(ctx, `
			UPDATE sources
			SET last_fetched_at = $1, consecutive_failures = 0,
			    status = CASE WHEN status = 'degraded' THEN 'active' ELSE status END
			WHERE id = $2
		`, now, id)
		return err
	}
	_, err := r.db.Exec(ctx, `
		UPDATE sources
		SET last_fetched_at = $1,
		    consecutive_failures = consecutive_failures + 1,
		    status = CASE
		        WHEN consecutive_failures + 1 >= 10 THEN 'disabled'
		        WHEN consecutive_failures + 1 >= 3  THEN 'degraded'
		        ELSE status
		    END
		WHERE id = $2
	`, now, id)
	return err
}

func (r *SourceRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE sources SET status = $1 WHERE id = $2`, status, id)
	return err
}

// ListAll returns all sources (including non-active) for admin use.
func (r *SourceRepository) ListAll(ctx context.Context, f repository.SourceFilter) ([]*domain.Source, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 50
	}

	args := []any{}
	where := "WHERE 1=1"
	n := 1

	if f.Category != nil {
		where += " AND s.category = $" + itoa(n)
		args = append(args, *f.Category)
		n++
	}
	if f.Language != nil {
		where += " AND s.language = $" + itoa(n)
		args = append(args, *f.Language)
		n++
	}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM sources s "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PerPage
	query := `
		SELECT s.id, s.name, s.feed_url, s.site_url, s.description, s.category, s.language,
		       s.fetch_interval_minutes, s.quality_score, s.status, s.last_fetched_at,
		       s.consecutive_failures, s.created_at, s.updated_at
		FROM sources s
		` + where + `
		ORDER BY s.name ASC
		LIMIT $` + itoa(n) + ` OFFSET $` + itoa(n+1)
	args = append(args, f.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
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
			return nil, 0, err
		}
		sources = append(sources, s)
	}
	return sources, total, rows.Err()
}

// Create inserts a new source.
func (r *SourceRepository) Create(ctx context.Context, s *domain.Source) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sources (id, name, feed_url, site_url, description, category, language,
		                     fetch_interval_minutes, quality_score, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',NOW(),NOW())
	`, s.ID, s.Name, s.FeedURL, s.SiteURL, s.Description, s.Category, s.Language,
		s.FetchIntervalMinutes, s.QualityScore,
	)
	return err
}

// Update applies partial updates to a source and returns the updated record.
func (r *SourceRepository) Update(ctx context.Context, id string, fields repository.SourceUpdateFields) (*domain.Source, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	n := 1

	if fields.Name != nil {
		sets = append(sets, "name = $"+itoa(n))
		args = append(args, *fields.Name)
		n++
	}
	if fields.Description != nil {
		sets = append(sets, "description = $"+itoa(n))
		args = append(args, *fields.Description)
		n++
	}
	if fields.Category != nil {
		sets = append(sets, "category = $"+itoa(n))
		args = append(args, *fields.Category)
		n++
	}
	if fields.FetchIntervalMinutes != nil {
		sets = append(sets, "fetch_interval_minutes = $"+itoa(n))
		args = append(args, *fields.FetchIntervalMinutes)
		n++
	}
	if fields.QualityScore != nil {
		sets = append(sets, "quality_score = $"+itoa(n))
		args = append(args, *fields.QualityScore)
		n++
	}
	if fields.Status != nil {
		sets = append(sets, "status = $"+itoa(n))
		args = append(args, *fields.Status)
		n++
	}

	setClause := ""
	for i, s := range sets {
		if i > 0 {
			setClause += ", "
		}
		setClause += s
	}
	args = append(args, id)
	idPlaceholder := "$" + itoa(n)

	s := &domain.Source{}
	err := r.db.QueryRow(ctx, `
		UPDATE sources SET `+setClause+`
		WHERE id = `+idPlaceholder+`
		RETURNING id, name, feed_url, site_url, description, category, language,
		          fetch_interval_minutes, quality_score, status, last_fetched_at,
		          consecutive_failures, created_at, updated_at
	`, args...).Scan(
		&s.ID, &s.Name, &s.FeedURL, &s.SiteURL, &s.Description, &s.Category, &s.Language,
		&s.FetchIntervalMinutes, &s.QualityScore, &s.Status, &s.LastFetchedAt,
		&s.ConsecutiveFailures, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// Delete removes a source by ID.
func (r *SourceRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM sources WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func itoa(n int) string {
	buf := [20]byte{}
	pos := len(buf)
	for n >= 10 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	pos--
	buf[pos] = byte('0' + n)
	return string(buf[pos:])
}
