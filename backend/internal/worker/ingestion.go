package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"

	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/repository/postgres"
)

const (
	maxConcurrent  = 10
	fetchTimeout   = 30 * time.Second
	lockTTL        = 5 * time.Minute
	minContentLen  = 100
	fetchBatchSize = 20
)

// IngestionJobStore persists ingestion job records.
type IngestionJobStore interface {
	Begin(ctx context.Context, sourceID string) (jobID string, err error)
	Complete(ctx context.Context, jobID string, found, newCount, skipped int) error
	Fail(ctx context.Context, jobID string, errMsg string) error
}

// FetchLock prevents concurrent fetches of the same source.
type FetchLock interface {
	Acquire(ctx context.Context, sourceID string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, sourceID string) error
}

// StreamPublisher queues article processing jobs.
type StreamPublisher interface {
	Publish(ctx context.Context, articleID string) error
}

type IngestionWorker struct {
	sources   repository.SourceRepository
	articles  repository.ArticleRepository
	jobs      IngestionJobStore
	lock      FetchLock
	stream    StreamPublisher
	httpClient *http.Client
	logger    *zap.Logger
}

func NewIngestionWorker(
	sources repository.SourceRepository,
	articles repository.ArticleRepository,
	jobs IngestionJobStore,
	lock FetchLock,
	stream StreamPublisher,
	logger *zap.Logger,
) *IngestionWorker {
	return &IngestionWorker{
		sources:  sources,
		articles: articles,
		jobs:     jobs,
		lock:     lock,
		stream:   stream,
		httpClient: &http.Client{
			Timeout: fetchTimeout,
		},
		logger: logger,
	}
}

// Run executes one ingestion cycle: fetches all due sources concurrently.
func (w *IngestionWorker) Run(ctx context.Context) error {
	sources, err := w.sources.ListDueForFetch(ctx, fetchBatchSize)
	if err != nil {
		return fmt.Errorf("list due sources: %w", err)
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, src := range sources {
		src := src
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			w.fetchSource(ctx, src)
		}()
	}

	wg.Wait()
	return nil
}

func (w *IngestionWorker) fetchSource(ctx context.Context, src *domain.Source) {
	start := time.Now()

	acquired, err := w.lock.Acquire(ctx, src.ID, lockTTL)
	if err != nil || !acquired {
		w.logger.Debug("skipping locked source", zap.String("source_id", src.ID))
		return
	}
	defer w.lock.Release(ctx, src.ID)

	jobID, err := w.jobs.Begin(ctx, src.ID)
	if err != nil {
		w.logger.Error("failed to begin ingestion job", zap.String("source_id", src.ID), zap.Error(err))
		return
	}

	found, newCount, skipped, fetchErr := w.processFeed(ctx, src)

	if fetchErr != nil {
		w.logger.Error("ingestion job failed",
			zap.String("source_id", src.ID),
			zap.String("source_name", src.Name),
			zap.Error(fetchErr),
		)
		_ = w.jobs.Fail(ctx, jobID, fetchErr.Error())
		_ = w.sources.UpdateAfterFetch(ctx, src.ID, false)
		return
	}

	_ = w.jobs.Complete(ctx, jobID, found, newCount, skipped)
	_ = w.sources.UpdateAfterFetch(ctx, src.ID, true)

	w.logger.Info("ingestion_job_completed",
		zap.String("source_id", src.ID),
		zap.String("source_name", src.Name),
		zap.Int("articles_found", found),
		zap.Int("articles_new", newCount),
		zap.Int("articles_skipped", skipped),
		zap.Int64("duration_ms", time.Since(start).Milliseconds()),
	)
}

func (w *IngestionWorker) processFeed(ctx context.Context, src *domain.Source) (found, newCount, skipped int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.FeedURL, nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Signalix-Bot/1.0")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("http fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, 0, 0, fmt.Errorf("http %d from %s", resp.StatusCode, src.FeedURL)
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse feed: %w", err)
	}

	sourceID, err := uuid.Parse(src.ID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid source id: %w", err)
	}

	for _, item := range feed.Items {
		found++
		article, skip := itemToArticle(item, sourceID, src.Language)
		if skip {
			skipped++
			continue
		}

		if err := w.articles.Insert(ctx, article); err != nil {
			// DB error: stop processing this feed
			return found, newCount, skipped, fmt.Errorf("insert article: %w", err)
		}

		// Publish processing job for new articles
		_ = w.stream.Publish(ctx, article.ID.String())
		newCount++
	}
	return found, newCount, skipped, nil
}

func itemToArticle(item *gofeed.Item, sourceID uuid.UUID, defaultLang string) (*domain.Article, bool) {
	if item.Title == "" || item.Link == "" {
		return nil, true
	}

	content := pickContent(item)

	// Skip thin content
	if len(strings.TrimSpace(content)) < minContentLen {
		return nil, true
	}

	normalURL := normalizeURL(item.Link)
	hash := articleURLHash(normalURL)

	var publishedAt *time.Time
	if item.PublishedParsed != nil {
		t := item.PublishedParsed.UTC()
		publishedAt = &t
	} else if item.UpdatedParsed != nil {
		t := item.UpdatedParsed.UTC()
		publishedAt = &t
	}

	var author *string
	if item.Author != nil && item.Author.Name != "" {
		author = &item.Author.Name
	}

	lang := defaultLang
	var langPtr *string
	if lang != "" {
		langPtr = &lang
	}

	return &domain.Article{
		ID:         uuid.New(),
		SourceID:   sourceID,
		URL:        normalURL,
		URLHash:    hash,
		Title:      item.Title,
		RawContent: &content,
		Author:     author,
		Language:   langPtr,
		PublishedAt: publishedAt,
		Status:     "pending",
	}, false
}

func pickContent(item *gofeed.Item) string {
	// 仕様: content:encoded > content > description > summary
	if v, ok := item.Extensions["content"]["encoded"]; ok && len(v) > 0 {
		return v[0].Value
	}
	if item.Content != "" {
		return item.Content
	}
	if item.Description != "" {
		return item.Description
	}
	return ""
}

func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Fragment = ""
	q := u.Query()
	for key := range q {
		if strings.HasPrefix(key, "utm_") {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func articleURLHash(normalizedURL string) string {
	h := sha256.Sum256([]byte(normalizedURL))
	return hex.EncodeToString(h[:])
}

// Ensure ArticleRepository satisfies the required interface at compile time.
var _ repository.ArticleRepository = (*postgres.ArticleRepository)(nil)
