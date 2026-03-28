package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

type ArticleRepository struct {
	db *pgxpool.Pool
}

func NewArticleRepository(db *pgxpool.Pool) *ArticleRepository {
	return &ArticleRepository{db: db}
}

const articleSelectCols = `
	a.id, a.source_id, a.url, a.url_hash, a.title, a.author, a.language,
	a.published_at, a.trend_score, a.status, a.created_at, a.updated_at,
	s.id, s.name, s.site_url, s.category,
	asu.summary_text, asu.model_name, asu.model_version`

func scanArticleRow(rows pgx.Rows) (*domain.ArticleWithDetails, error) {
	return scanArticleRowWithOpts(rows, false)
}

func scanArticleRowWithContent(rows pgx.Rows) (*domain.ArticleWithDetails, error) {
	return scanArticleRowWithOpts(rows, true)
}

func scanArticleRowWithOpts(rows pgx.Rows, withContent bool) (*domain.ArticleWithDetails, error) {
	a := &domain.ArticleWithDetails{
		Source: &domain.Source{},
	}
	var sumText, sumModel, sumVersion *string

	dest := []any{
		&a.Article.ID, &a.Article.SourceID, &a.Article.URL, &a.Article.URLHash,
		&a.Article.Title, &a.Article.Author, &a.Article.Language,
		&a.Article.PublishedAt, &a.Article.TrendScore, &a.Article.Status,
		&a.Article.CreatedAt, &a.Article.UpdatedAt,
		&a.Source.ID, &a.Source.Name, &a.Source.SiteURL, &a.Source.Category,
		&sumText, &sumModel, &sumVersion,
	}
	if withContent {
		dest = append(dest, &a.Article.CleanContent)
	}

	if err := rows.Scan(dest...); err != nil {
		return nil, err
	}

	if sumText != nil {
		a.Summary = &domain.ArticleSummary{
			SummaryText:  *sumText,
			ModelName:    strVal(sumModel),
			ModelVersion: strVal(sumVersion),
		}
	}
	return a, nil
}

func (r *ArticleRepository) List(ctx context.Context, f repository.ArticleFilter) ([]*domain.ArticleWithDetails, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
	}

	args := []any{}
	n := 1
	where := "WHERE a.status = 'processed'"

	if f.Language != nil {
		where += fmt.Sprintf(" AND a.language = $%d", n)
		args = append(args, *f.Language)
		n++
	}
	if f.SourceID != nil {
		where += fmt.Sprintf(" AND a.source_id = $%d", n)
		args = append(args, *f.SourceID)
		n++
	}
	if f.Query != nil {
		where += fmt.Sprintf(" AND (a.title ILIKE $%d OR asu.summary_text ILIKE $%d)", n, n)
		args = append(args, "%"+*f.Query+"%")
		n++
	}

	tagJoin := ""
	if len(f.Tags) > 0 {
		tagJoin = `
			JOIN article_tags at2 ON at2.article_id = a.id
			JOIN tags t ON t.id = at2.tag_id AND t.name = ANY($` + itoa(n) + `)`
		args = append(args, f.Tags)
		n++
	}

	sort := "a.published_at"
	if f.Sort == "trend_score" {
		sort = "a.trend_score"
	}
	order := "DESC"
	if f.Order == "asc" {
		order = "ASC"
	}

	countQuery := `SELECT COUNT(DISTINCT a.id)
		FROM articles a
		LEFT JOIN article_summaries asu ON asu.article_id = a.id
		` + tagJoin + ` ` + where
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PerPage
	listQuery := `
		SELECT DISTINCT ON (` + sort + `, a.id) ` + articleSelectCols + `
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		LEFT JOIN article_summaries asu ON asu.article_id = a.id
		` + tagJoin + ` ` + where + `
		ORDER BY ` + sort + ` ` + order + `, a.id
		LIMIT $` + itoa(n) + ` OFFSET $` + itoa(n+1)
	args = append(args, f.PerPage, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	articles, err := r.scanArticleRows(ctx, rows)
	return articles, total, err
}

func (r *ArticleRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.ArticleWithDetails, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+articleSelectCols+`, a.clean_content
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		LEFT JOIN article_summaries asu ON asu.article_id = a.id
		WHERE a.id = $1 AND a.status = 'processed'
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	a, err := scanArticleRowWithContent(rows)
	if err != nil {
		return nil, err
	}
	if err := r.loadTags(ctx, []*domain.ArticleWithDetails{a}); err != nil {
		return nil, err
	}
	return a, nil
}

func (r *ArticleRepository) Trending(ctx context.Context, period string, language *string, page, perPage int) ([]*domain.ArticleWithDetails, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 50 {
		perPage = 20
	}

	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	if period == "7d" {
		cutoff = time.Now().UTC().Add(-7 * 24 * time.Hour)
	}

	args := []any{cutoff}
	n := 2
	where := "WHERE a.status = 'processed' AND a.published_at >= $1"
	if language != nil {
		where += fmt.Sprintf(" AND a.language = $%d", n)
		args = append(args, *language)
		n++
	}

	var total int
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM articles a "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := r.db.Query(ctx, `
		SELECT `+articleSelectCols+`
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		LEFT JOIN article_summaries asu ON asu.article_id = a.id
		`+where+`
		ORDER BY a.trend_score DESC, a.published_at DESC
		LIMIT $`+itoa(n)+` OFFSET $`+itoa(n+1),
		append(args, perPage, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	articles, err := r.scanArticleRows(ctx, rows)
	return articles, total, err
}

func (r *ArticleRepository) Insert(ctx context.Context, a *domain.Article) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO articles
			(id, source_id, url, url_hash, title, raw_content, author, language, published_at, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (url_hash) DO NOTHING
	`, a.ID, a.SourceID, a.URL, a.URLHash, a.Title, a.RawContent,
		a.Author, a.Language, a.PublishedAt, a.Status)
	return err
}

func (r *ArticleRepository) ListRecentBySource(ctx context.Context, sourceID uuid.UUID, limit int) ([]*domain.ArticleWithDetails, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+articleSelectCols+`
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		LEFT JOIN article_summaries asu ON asu.article_id = a.id
		WHERE a.source_id = $1 AND a.status = 'processed'
		ORDER BY a.published_at DESC
		LIMIT $2
	`, sourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanArticleRows(ctx, rows)
}

func (r *ArticleRepository) scanArticleRows(ctx context.Context, rows pgx.Rows) ([]*domain.ArticleWithDetails, error) {
	var articles []*domain.ArticleWithDetails
	for rows.Next() {
		a, err := scanArticleRow(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(articles) > 0 {
		if err := r.loadTags(ctx, articles); err != nil {
			return nil, err
		}
	}
	return articles, nil
}

func (r *ArticleRepository) loadTags(ctx context.Context, articles []*domain.ArticleWithDetails) error {
	if len(articles) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(articles))
	idx := make(map[uuid.UUID]int, len(articles))
	for i, a := range articles {
		ids[i] = a.Article.ID
		idx[a.Article.ID] = i
	}

	rows, err := r.db.Query(ctx, `
		SELECT at2.article_id, t.id, t.name, t.name, t.category, at2.confidence
		FROM article_tags at2
		JOIN tags t ON t.id = at2.tag_id
		WHERE at2.article_id = ANY($1)
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var articleID uuid.UUID
		var tw domain.TagWithConfidence
		if err := rows.Scan(&articleID, &tw.ID, &tw.Name, &tw.Slug, &tw.Category, &tw.Confidence); err != nil {
			return err
		}
		if i, ok := idx[articleID]; ok {
			articles[i].Tags = append(articles[i].Tags, tw)
		}
	}
	return rows.Err()
}

// GetRawForProcessing returns a pending/fetched article's raw fields.
func (r *ArticleRepository) GetRawForProcessing(ctx context.Context, id uuid.UUID) (*domain.Article, error) {
	a := &domain.Article{}
	err := r.db.QueryRow(ctx, `
		SELECT id, source_id, url, title, raw_content, language, published_at, status
		FROM articles WHERE id = $1
	`, id).Scan(&a.ID, &a.SourceID, &a.URL, &a.Title, &a.RawContent, &a.Language, &a.PublishedAt, &a.Status)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// UpdateStatus sets the article status.
func (r *ArticleRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE articles SET status=$1, updated_at=NOW() WHERE id=$2`, status, id)
	return err
}

// UpdateCleanContent saves normalized text and detected language.
func (r *ArticleRepository) UpdateCleanContent(ctx context.Context, id uuid.UUID, clean string, language *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE articles SET clean_content=$1, language=COALESCE($2, language), status='processing', updated_at=NOW() WHERE id=$3`,
		clean, language, id)
	return err
}

// UpdateTrendScore saves the computed trend score.
func (r *ArticleRepository) UpdateTrendScore(ctx context.Context, id uuid.UUID, score float64) error {
	_, err := r.db.Exec(ctx, `UPDATE articles SET trend_score=$1, updated_at=NOW() WHERE id=$2`, score, id)
	return err
}

// SaveSummary upserts an article summary.
func (r *ArticleRepository) SaveSummary(ctx context.Context, s *domain.ArticleSummary) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO article_summaries (id, article_id, summary_text, model_name, model_version, prompt_version, token_count, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		ON CONFLICT (article_id) DO UPDATE SET
			summary_text   = EXCLUDED.summary_text,
			model_name     = EXCLUDED.model_name,
			model_version  = EXCLUDED.model_version,
			prompt_version = EXCLUDED.prompt_version,
			token_count    = EXCLUDED.token_count
	`, s.ID, s.ArticleID, s.SummaryText, s.ModelName, s.ModelVersion, s.PromptVersion, s.TokenCount)
	return err
}

// SaveEmbedding upserts a pgvector embedding.
func (r *ArticleRepository) SaveEmbedding(ctx context.Context, articleID uuid.UUID, embedding []float32) error {
	// pgvector expects format: '[0.1,0.2,...]'
	b := make([]byte, 0, len(embedding)*10+2)
	b = append(b, '[')
	for i, v := range embedding {
		if i > 0 {
			b = append(b, ',')
		}
		b = fmt.Appendf(b, "%.6f", v)
	}
	b = append(b, ']')

	_, err := r.db.Exec(ctx, `
		INSERT INTO article_embeddings (article_id, embedding, model_name, model_version)
		VALUES ($1,$2::vector,'text-embedding-3-small','2024-02-01')
		ON CONFLICT (article_id) DO UPDATE SET
			embedding    = EXCLUDED.embedding,
			model_name   = EXCLUDED.model_name,
			model_version= EXCLUDED.model_version
	`, articleID, string(b))
	return err
}

// SaveTags upserts article tag associations.
func (r *ArticleRepository) SaveTags(ctx context.Context, articleID uuid.UUID, tags []domain.TagWithConfidence) error {
	for _, t := range tags {
		_, err := r.db.Exec(ctx, `
			INSERT INTO article_tags (article_id, tag_id, confidence)
			VALUES ($1,$2,$3)
			ON CONFLICT (article_id, tag_id) DO UPDATE SET confidence = EXCLUDED.confidence
		`, articleID, t.Tag.ID, t.Confidence)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListAllTagNames returns all tag names from the tags master table.
func (r *ArticleRepository) ListAllTagNames(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT name FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

// FindTagIDsByName returns a map of tag name → UUID for the given names.
func (r *ArticleRepository) FindTagIDsByName(ctx context.Context, names []string) (map[string]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name FROM tags WHERE name = ANY($1)`, names)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result[name] = id
	}
	return result, rows.Err()
}

// helpers

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// pgxErrNoRows checks for pgx no-rows error
func pgxErrNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
