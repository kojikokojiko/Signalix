package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/worker"
	"go.uber.org/zap"
)

// --- mocks ---

type mockProcessingArticleRepo struct {
	article      *domain.Article
	statusUpdates []string
	savedSummary  *domain.ArticleSummary
	savedEmbedding []float32
	savedTags    []domain.TagWithConfidence
	tagNames     []string
	tagIDs       map[string]uuid.UUID
}

func (m *mockProcessingArticleRepo) GetRawForProcessing(_ context.Context, _ uuid.UUID) (*domain.Article, error) {
	return m.article, nil
}
func (m *mockProcessingArticleRepo) UpdateStatus(_ context.Context, _ uuid.UUID, status string) error {
	m.statusUpdates = append(m.statusUpdates, status)
	return nil
}
func (m *mockProcessingArticleRepo) UpdateCleanContent(_ context.Context, _ uuid.UUID, _ string, _ *string) error {
	return nil
}
func (m *mockProcessingArticleRepo) UpdateTrendScore(_ context.Context, _ uuid.UUID, _ float64) error {
	return nil
}
func (m *mockProcessingArticleRepo) SaveSummary(_ context.Context, s *domain.ArticleSummary) error {
	m.savedSummary = s
	return nil
}
func (m *mockProcessingArticleRepo) SaveEmbedding(_ context.Context, _ uuid.UUID, emb []float32) error {
	m.savedEmbedding = emb
	return nil
}
func (m *mockProcessingArticleRepo) SaveTags(_ context.Context, _ uuid.UUID, tags []domain.TagWithConfidence) error {
	m.savedTags = tags
	return nil
}
func (m *mockProcessingArticleRepo) ListAllTagNames(_ context.Context) ([]string, error) {
	return m.tagNames, nil
}
func (m *mockProcessingArticleRepo) FindTagIDsByName(_ context.Context, names []string) (map[string]uuid.UUID, error) {
	if m.tagIDs != nil {
		return m.tagIDs, nil
	}
	result := make(map[string]uuid.UUID)
	for _, n := range names {
		result[n] = uuid.New()
	}
	return result, nil
}

// mock AI client
type mockAIClient struct {
	embedding []float32
	summary   string
	tokens    int
	tags      []worker.ExtractedTag
}

func (m *mockAIClient) CreateEmbedding(_ context.Context, _ string) ([]float32, error) {
	if m.embedding != nil {
		return m.embedding, nil
	}
	return make([]float32, 1536), nil
}
func (m *mockAIClient) CreateSummary(_ context.Context, _, _ string) (string, int, error) {
	s := m.summary
	if s == "" {
		s = "これはテスト要約です。重要な技術的ポイントを含んでいます。実装の影響を説明します。"
	}
	return s, m.tokens, nil
}
func (m *mockAIClient) CreateTags(_ context.Context, _, _ string, _ []string) ([]worker.ExtractedTag, int, error) {
	if m.tags != nil {
		return m.tags, m.tokens, nil
	}
	return []worker.ExtractedTag{{Name: "go", Confidence: 0.9}}, m.tokens, nil
}

// mock stream publisher
type mockProcessingStream struct {
	published []string
}

func (m *mockProcessingStream) Publish(_ context.Context, id string) error {
	m.published = append(m.published, id)
	return nil
}

func makeArticleForProcessing() *domain.Article {
	content := "<p>これは十分な長さのテスト記事本文です。GoとRustのパフォーマンス比較について詳しく説明します。メモリ管理の違い、実行速度の差異、開発体験の違いなど多角的に分析します。両言語の特徴を理解することでシステム設計に役立てることができます。</p>"
	pub := time.Now().Add(-3 * time.Hour)
	return &domain.Article{
		ID:         uuid.New(),
		SourceID:   uuid.New(),
		Title:      "GoとRustのパフォーマンス比較",
		RawContent: &content,
		PublishedAt: &pub,
		Status:     "pending",
	}
}

// --- tests ---

func TestProcessingWorker_ProcessArticle_Success(t *testing.T) {
	repo := &mockProcessingArticleRepo{
		article:  makeArticleForProcessing(),
		tagNames: []string{"go", "rust", "performance"},
		tagIDs:   map[string]uuid.UUID{"go": uuid.New()},
	}
	ai := &mockAIClient{}
	stream := &mockProcessingStream{}
	logger, _ := zap.NewDevelopment()

	w := worker.NewProcessingWorker(repo, ai, stream, logger)
	if err := w.ProcessArticle(context.Background(), repo.article.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ステータスが 'processed' に更新されること
	if len(repo.statusUpdates) == 0 {
		t.Error("expected at least one status update")
	}
	last := repo.statusUpdates[len(repo.statusUpdates)-1]
	if last != "processed" {
		t.Errorf("expected final status 'processed', got %q", last)
	}

	// 要約が保存されること
	if repo.savedSummary == nil {
		t.Error("expected summary to be saved")
	}

	// 埋め込みが保存されること
	if len(repo.savedEmbedding) == 0 {
		t.Error("expected embedding to be saved")
	}

	// タグが保存されること
	if len(repo.savedTags) == 0 {
		t.Error("expected tags to be saved")
	}
}

func TestProcessingWorker_ProcessArticle_SkipsShortContent(t *testing.T) {
	short := "<p>短い</p>"
	article := &domain.Article{
		ID:         uuid.New(),
		Title:      "Short",
		RawContent: &short,
		Status:     "pending",
	}
	repo := &mockProcessingArticleRepo{article: article}
	ai := &mockAIClient{}
	stream := &mockProcessingStream{}
	logger, _ := zap.NewDevelopment()

	w := worker.NewProcessingWorker(repo, ai, stream, logger)
	if err := w.ProcessArticle(context.Background(), article.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// スキップ時は 'skipped' ステータスになること
	if len(repo.statusUpdates) == 0 {
		t.Error("expected status update")
	}
	if repo.statusUpdates[len(repo.statusUpdates)-1] != "skipped" {
		t.Errorf("expected 'skipped', got %q", repo.statusUpdates[len(repo.statusUpdates)-1])
	}
}

func TestProcessingWorker_ProcessArticle_NilRawContent(t *testing.T) {
	article := &domain.Article{
		ID:     uuid.New(),
		Title:  "No Content",
		Status: "pending",
	}
	repo := &mockProcessingArticleRepo{article: article}
	ai := &mockAIClient{}
	stream := &mockProcessingStream{}
	logger, _ := zap.NewDevelopment()

	w := worker.NewProcessingWorker(repo, ai, stream, logger)
	if err := w.ProcessArticle(context.Background(), article.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.statusUpdates) == 0 || repo.statusUpdates[len(repo.statusUpdates)-1] != "skipped" {
		t.Errorf("expected 'skipped', got %v", repo.statusUpdates)
	}
}
