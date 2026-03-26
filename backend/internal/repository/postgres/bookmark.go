package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

type BookmarkRepository struct {
	db *pgxpool.Pool
}

func NewBookmarkRepository(db *pgxpool.Pool) *BookmarkRepository {
	return &BookmarkRepository{db: db}
}

func (r *BookmarkRepository) List(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*repository.BookmarkWithArticle, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM bookmarks WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.user_id, b.article_id, b.created_at,
		       `+articleSelectCols+`
		FROM bookmarks b
		JOIN articles a ON a.id = b.article_id
		JOIN sources s ON s.id = a.source_id
		LEFT JOIN article_summaries asu ON asu.article_id = a.id
		WHERE b.user_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*repository.BookmarkWithArticle
	for rows.Next() {
		bm := &repository.BookmarkWithArticle{
			Article: &domain.ArticleWithDetails{Source: &domain.Source{}},
		}
		var sumText, sumModel, sumVersion *string
		if err := rows.Scan(
			&bm.Bookmark.ID, &bm.Bookmark.UserID, &bm.Bookmark.ArticleID, &bm.Bookmark.CreatedAt,
			&bm.Article.Article.ID, &bm.Article.Article.SourceID, &bm.Article.Article.URL,
			&bm.Article.Article.URLHash, &bm.Article.Article.Title, &bm.Article.Article.Author,
			&bm.Article.Article.Language, &bm.Article.Article.PublishedAt,
			&bm.Article.Article.TrendScore, &bm.Article.Article.Status,
			&bm.Article.Article.CreatedAt, &bm.Article.Article.UpdatedAt,
			&bm.Article.Source.ID, &bm.Article.Source.Name, &bm.Article.Source.SiteURL, &bm.Article.Source.Category,
			&sumText, &sumModel, &sumVersion,
		); err != nil {
			return nil, 0, err
		}
		if sumText != nil {
			bm.Article.Summary = &domain.ArticleSummary{
				SummaryText:  *sumText,
				ModelName:    strVal(sumModel),
				ModelVersion: strVal(sumVersion),
			}
		}
		results = append(results, bm)
	}
	return results, total, rows.Err()
}

func (r *BookmarkRepository) Add(ctx context.Context, userID, articleID uuid.UUID) (*domain.Bookmark, error) {
	bm := &domain.Bookmark{
		ID:        uuid.New(),
		UserID:    userID,
		ArticleID: articleID,
		CreatedAt: time.Now().UTC(),
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO bookmarks (id, user_id, article_id, created_at) VALUES ($1,$2,$3,$4)`,
		bm.ID, bm.UserID, bm.ArticleID, bm.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errors.New("already bookmarked")
		}
		return nil, err
	}
	return bm, nil
}

func (r *BookmarkRepository) Remove(ctx context.Context, userID, articleID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM bookmarks WHERE user_id = $1 AND article_id = $2`,
		userID, articleID,
	)
	return err
}

func (r *BookmarkRepository) Exists(ctx context.Context, userID, articleID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM bookmarks WHERE user_id=$1 AND article_id=$2)`,
		userID, articleID,
	).Scan(&exists)
	return exists, err
}
