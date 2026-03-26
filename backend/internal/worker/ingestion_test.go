package worker_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/worker"
)

// ─── mocks ───────────────────────────────────────────────────────────────────

type mockSourceRepo struct {
	sources []*domain.Source
	updated map[string]bool
}

func (m *mockSourceRepo) List(_ context.Context, _ repository.SourceFilter) ([]*domain.Source, int, error) {
	return m.sources, len(m.sources), nil
}
func (m *mockSourceRepo) FindByID(_ context.Context, id string) (*domain.Source, error) {
	for _, s := range m.sources {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}
func (m *mockSourceRepo) ListDueForFetch(_ context.Context, limit int) ([]*domain.Source, error) {
	return m.sources, nil
}
func (m *mockSourceRepo) UpdateAfterFetch(_ context.Context, id string, success bool) error {
	if m.updated == nil {
		m.updated = make(map[string]bool)
	}
	m.updated[id] = success
	return nil
}
func (m *mockSourceRepo) UpdateStatus(_ context.Context, id string, status string) error { return nil }
func (m *mockSourceRepo) ListAll(_ context.Context, _ repository.SourceFilter) ([]*domain.Source, int, error) {
	return nil, 0, nil
}
func (m *mockSourceRepo) Create(_ context.Context, _ *domain.Source) error { return nil }
func (m *mockSourceRepo) Update(_ context.Context, _ string, _ repository.SourceUpdateFields) (*domain.Source, error) {
	return nil, nil
}
func (m *mockSourceRepo) Delete(_ context.Context, _ string) error { return nil }

type mockArticleRepo struct {
	inserted []*domain.Article
}

func (m *mockArticleRepo) List(_ context.Context, _ repository.ArticleFilter) ([]*domain.ArticleWithDetails, int, error) {
	return nil, 0, nil
}
func (m *mockArticleRepo) FindByID(_ context.Context, _ uuid.UUID) (*domain.ArticleWithDetails, error) {
	return nil, nil
}
func (m *mockArticleRepo) Trending(_ context.Context, _ string, _ *string, _, _ int) ([]*domain.ArticleWithDetails, int, error) {
	return nil, 0, nil
}
func (m *mockArticleRepo) Insert(_ context.Context, a *domain.Article) error {
	m.inserted = append(m.inserted, a)
	return nil
}
func (m *mockArticleRepo) ListRecentBySource(_ context.Context, _ uuid.UUID, _ int) ([]*domain.ArticleWithDetails, error) {
	return nil, nil
}
func (m *mockArticleRepo) GetRawForProcessing(_ context.Context, _ uuid.UUID) (*domain.Article, error) {
	return nil, nil
}
func (m *mockArticleRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (m *mockArticleRepo) UpdateCleanContent(_ context.Context, _ uuid.UUID, _ string, _ *string) error {
	return nil
}
func (m *mockArticleRepo) UpdateTrendScore(_ context.Context, _ uuid.UUID, _ float64) error {
	return nil
}
func (m *mockArticleRepo) SaveSummary(_ context.Context, _ *domain.ArticleSummary) error {
	return nil
}
func (m *mockArticleRepo) SaveEmbedding(_ context.Context, _ uuid.UUID, _ []float32) error {
	return nil
}
func (m *mockArticleRepo) SaveTags(_ context.Context, _ uuid.UUID, _ []domain.TagWithConfidence) error {
	return nil
}
func (m *mockArticleRepo) ListAllTagNames(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockArticleRepo) FindTagIDsByName(_ context.Context, _ []string) (map[string]uuid.UUID, error) {
	return nil, nil
}

type mockJobStore struct {
	completed []string
	failed    []string
}

func (m *mockJobStore) Begin(_ context.Context, _ string) (string, error) {
	return uuid.New().String(), nil
}
func (m *mockJobStore) Complete(_ context.Context, jobID string, _, _, _ int) error {
	m.completed = append(m.completed, jobID)
	return nil
}
func (m *mockJobStore) Fail(_ context.Context, jobID string, _ string) error {
	m.failed = append(m.failed, jobID)
	return nil
}

type mockFetchLock struct{ acquired bool }

func (m *mockFetchLock) Acquire(_ context.Context, _ string, _ time.Duration) (bool, error) {
	return true, nil
}
func (m *mockFetchLock) Release(_ context.Context, _ string) error { return nil }

type mockStream struct{ published []string }

func (m *mockStream) Publish(_ context.Context, articleID string) error {
	m.published = append(m.published, articleID)
	return nil
}

// ─── RSS feed fixtures ────────────────────────────────────────────────────────

const validRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Article One</title>
      <link>https://example.com/article-1</link>
      <description>This is a long enough description for article one to pass the minimum content length check in the ingestion worker.</description>
      <pubDate>Mon, 17 Mar 2025 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Article Two</title>
      <link>https://example.com/article-2?utm_source=rss&amp;utm_medium=feed</link>
      <description>This is a long enough description for article two to pass the minimum content length check in the ingestion worker too.</description>
      <pubDate>Mon, 17 Mar 2025 11:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

const thinRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Thin Feed</title>
    <link>https://thin.example.com</link>
    <item>
      <title>Short</title>
      <link>https://thin.example.com/short</link>
      <description>Too short.</description>
    </item>
  </channel>
</rss>`

// ─── helper ──────────────────────────────────────────────────────────────────

func newWorkerWithServer(t *testing.T, body string, sourceID string) (*worker.IngestionWorker, *mockSourceRepo, *mockArticleRepo, *mockStream) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	srcRepo := &mockSourceRepo{
		sources: []*domain.Source{
			{ID: sourceID, Name: "Test Source", FeedURL: srv.URL, Language: "en", Status: "active"},
		},
	}
	artRepo := &mockArticleRepo{}
	jobs := &mockJobStore{}
	lock := &mockFetchLock{}
	stream := &mockStream{}

	logger, _ := zap.NewDevelopment()
	w := worker.NewIngestionWorker(srcRepo, artRepo, jobs, lock, stream, logger)
	return w, srcRepo, artRepo, stream
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestIngestionWorker_Run_InsertsArticles(t *testing.T) {
	sourceID := uuid.New().String()
	w, srcRepo, artRepo, stream := newWorkerWithServer(t, validRSS, sourceID)

	if err := w.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(artRepo.inserted) != 2 {
		t.Errorf("expected 2 articles inserted, got %d", len(artRepo.inserted))
	}
	if len(stream.published) != 2 {
		t.Errorf("expected 2 jobs published, got %d", len(stream.published))
	}
	if srcRepo.updated[sourceID] != true {
		t.Error("expected source to be updated as success")
	}
}

func TestIngestionWorker_Run_SkipsThinContent(t *testing.T) {
	sourceID := uuid.New().String()
	w, _, artRepo, _ := newWorkerWithServer(t, thinRSS, sourceID)

	if err := w.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(artRepo.inserted) != 0 {
		t.Errorf("expected 0 articles (thin content skipped), got %d", len(artRepo.inserted))
	}
}

func TestIngestionWorker_Run_StripUTMParams(t *testing.T) {
	sourceID := uuid.New().String()
	w, _, artRepo, _ := newWorkerWithServer(t, validRSS, sourceID)
	_ = w.Run(context.Background())

	for _, a := range artRepo.inserted {
		if contains(a.URL, "utm_") {
			t.Errorf("utm_ params should be stripped from URL: %s", a.URL)
		}
	}
}

func TestIngestionWorker_Run_FailsOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	sourceID := uuid.New().String()
	srcRepo := &mockSourceRepo{
		sources: []*domain.Source{
			{ID: sourceID, Name: "Bad Source", FeedURL: srv.URL, Status: "active"},
		},
	}
	artRepo := &mockArticleRepo{}
	jobs := &mockJobStore{}
	lock := &mockFetchLock{}
	stream := &mockStream{}

	logger, _ := zap.NewDevelopment()
	w := worker.NewIngestionWorker(srcRepo, artRepo, jobs, lock, stream, logger)
	_ = w.Run(context.Background())

	if srcRepo.updated[sourceID] != false {
		t.Error("expected source to be updated as failure")
	}
	if len(jobs.failed) == 0 {
		t.Error("expected job to be marked as failed")
	}
}

func TestIngestionWorker_Run_NoSources(t *testing.T) {
	srcRepo := &mockSourceRepo{sources: nil}
	artRepo := &mockArticleRepo{}
	jobs := &mockJobStore{}
	lock := &mockFetchLock{}
	stream := &mockStream{}
	logger, _ := zap.NewDevelopment()

	w := worker.NewIngestionWorker(srcRepo, artRepo, jobs, lock, stream, logger)
	if err := w.Run(context.Background()); err != nil {
		t.Errorf("expected no error with empty source list, got %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
