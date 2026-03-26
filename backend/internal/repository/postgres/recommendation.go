package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kojikokojiko/signalix/internal/domain"
)

type RecommendationRepository struct {
	db *pgxpool.Pool
}

func NewRecommendationRepository(db *pgxpool.Pool) *RecommendationRepository {
	return &RecommendationRepository{db: db}
}

// List returns paginated recommended items for a user, ordered by total_score DESC.
func (r *RecommendationRepository) List(ctx context.Context, userID uuid.UUID, language *string, page, perPage int) ([]*domain.RecommendedItem, int, error) {
	offset := (page - 1) * perPage

	langFilter := ""
	args := []any{userID, perPage, offset}
	if language != nil {
		langFilter = "AND a.language = $4"
		args = append(args, *language)
	}

	query := `
		SELECT
			a.id, a.source_id, a.url, a.title, a.author, a.language, a.published_at, a.trend_score, a.status, a.created_at, a.updated_at,
			s.id, s.name, s.site_url, s.quality_score,
			rl.id, rl.total_score, rl.relevance_score, rl.freshness_score, rl.trend_score, rl.source_quality_score, rl.personalization_boost,
			rl.explanation, rl.generated_at, rl.expires_at,
			COUNT(*) OVER() AS total_count
		FROM recommendation_logs rl
		JOIN articles a ON a.id = rl.article_id
		JOIN sources s ON s.id = a.source_id
		WHERE rl.user_id = $1
		  AND rl.expires_at > NOW()
		` + langFilter + `
		ORDER BY rl.total_score DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*domain.RecommendedItem
	total := 0
	for rows.Next() {
		article := &domain.ArticleWithDetails{
			Source: &domain.Source{},
		}
		log := &domain.RecommendationLog{UserID: userID}

		if err := rows.Scan(
			&article.Article.ID, &article.Article.SourceID, &article.Article.URL, &article.Article.Title,
			&article.Article.Author, &article.Article.Language, &article.Article.PublishedAt,
			&article.Article.TrendScore, &article.Article.Status, &article.Article.CreatedAt, &article.Article.UpdatedAt,
			&article.Source.ID, &article.Source.Name, &article.Source.SiteURL, &article.Source.QualityScore,
			&log.ID, &log.TotalScore, &log.RelevanceScore, &log.FreshnessScore, &log.TrendScore,
			&log.SourceQualityScore, &log.PersonalizationBoost, &log.Explanation, &log.GeneratedAt, &log.ExpiresAt,
			&total,
		); err != nil {
			return nil, 0, err
		}
		log.ArticleID = article.Article.ID
		items = append(items, &domain.RecommendedItem{Article: article, Log: log})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Upsert saves or updates a recommendation log.
func (r *RecommendationRepository) Upsert(ctx context.Context, log *domain.RecommendationLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO recommendation_logs (
			id, user_id, article_id, total_score, relevance_score, freshness_score,
			trend_score, source_quality_score, personalization_boost,
			explanation, generated_at, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (user_id, article_id)
		DO UPDATE SET
			total_score           = EXCLUDED.total_score,
			relevance_score       = EXCLUDED.relevance_score,
			freshness_score       = EXCLUDED.freshness_score,
			trend_score           = EXCLUDED.trend_score,
			source_quality_score  = EXCLUDED.source_quality_score,
			personalization_boost = EXCLUDED.personalization_boost,
			explanation           = EXCLUDED.explanation,
			generated_at          = EXCLUDED.generated_at,
			expires_at            = EXCLUDED.expires_at
	`,
		log.ID, log.UserID, log.ArticleID, log.TotalScore, log.RelevanceScore, log.FreshnessScore,
		log.TrendScore, log.SourceQualityScore, log.PersonalizationBoost,
		log.Explanation, log.GeneratedAt, log.ExpiresAt,
	)
	return err
}

// LastRefreshedAt returns the most recent recommendation log for a user.
func (r *RecommendationRepository) LastRefreshedAt(ctx context.Context, userID uuid.UUID) (*domain.RecommendationLog, error) {
	log := &domain.RecommendationLog{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, article_id, total_score, generated_at, expires_at
		FROM recommendation_logs
		WHERE user_id = $1
		ORDER BY generated_at DESC
		LIMIT 1
	`, userID).Scan(&log.ID, &log.UserID, &log.ArticleID, &log.TotalScore, &log.GeneratedAt, &log.ExpiresAt)
	if err != nil {
		return nil, nil // no rows → no last refresh
	}
	return log, nil
}

// ListCandidates returns candidate articles for scoring (not yet in recommendation_logs or expiring soon).
func (r *RecommendationRepository) ListCandidates(ctx context.Context, userID uuid.UUID, language *string, limit int) ([]*domain.ArticleWithDetails, error) {
	langFilter := ""
	args := []any{userID, limit}
	if language != nil {
		langFilter = "AND a.language = $3"
		args = append(args, *language)
	}

	query := `
		SELECT
			a.id, a.source_id, a.url, a.title, a.author, a.language,
			a.published_at, a.trend_score, a.status, a.created_at, a.updated_at,
			s.id, s.name, s.site_url, s.quality_score
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		WHERE a.status = 'processed'
		  ` + langFilter + `
		  AND NOT EXISTS (
			SELECT 1 FROM recommendation_logs rl
			WHERE rl.article_id = a.id AND rl.user_id = $1 AND rl.expires_at > NOW() + INTERVAL '1 hour'
		  )
		ORDER BY a.trend_score DESC, a.published_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []*domain.ArticleWithDetails
	for rows.Next() {
		a := &domain.ArticleWithDetails{Source: &domain.Source{}}
		if err := rows.Scan(
			&a.Article.ID, &a.Article.SourceID, &a.Article.URL, &a.Article.Title, &a.Article.Author,
			&a.Article.Language, &a.Article.PublishedAt, &a.Article.TrendScore, &a.Article.Status,
			&a.Article.CreatedAt, &a.Article.UpdatedAt,
			&a.Source.ID, &a.Source.Name, &a.Source.SiteURL, &a.Source.QualityScore,
		); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load tags for each article.
	for _, a := range articles {
		tags, err := r.loadTagsForArticle(ctx, a.Article.ID)
		if err != nil {
			return nil, err
		}
		a.Tags = tags
	}
	return articles, nil
}

// GetPositiveFeedbackTagFreq returns tag frequency from positive feedback in past 30 days.
func (r *RecommendationRepository) GetPositiveFeedbackTagFreq(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]float64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT at.tag_id, COUNT(*)::float / NULLIF((
			SELECT COUNT(*) FROM user_feedback
			WHERE user_id = $1 AND feedback_type IN ('like','save','click')
			  AND created_at > NOW() - INTERVAL '30 days'
		), 0)
		FROM user_feedback uf
		JOIN article_tags at ON at.article_id = uf.article_id
		WHERE uf.user_id = $1
		  AND uf.feedback_type IN ('like','save','click')
		  AND uf.created_at > NOW() - INTERVAL '30 days'
		GROUP BY at.tag_id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]float64)
	for rows.Next() {
		var tagID uuid.UUID
		var freq float64
		if err := rows.Scan(&tagID, &freq); err != nil {
			return nil, err
		}
		result[tagID] = freq
	}
	return result, rows.Err()
}

func (r *RecommendationRepository) loadTagsForArticle(ctx context.Context, articleID uuid.UUID) ([]domain.TagWithConfidence, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.name, t.name, t.category, at.confidence
		FROM article_tags at
		JOIN tags t ON t.id = at.tag_id
		WHERE at.article_id = $1
		ORDER BY at.confidence DESC
	`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.TagWithConfidence
	for rows.Next() {
		var twc domain.TagWithConfidence
		if err := rows.Scan(&twc.Tag.ID, &twc.Tag.Name, &twc.Tag.Slug, &twc.Tag.Category, &twc.Confidence); err != nil {
			return nil, err
		}
		tags = append(tags, twc)
	}
	return tags, rows.Err()
}
