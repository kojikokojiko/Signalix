package worker

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"go.uber.org/zap"
	"golang.org/x/net/html"

	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

const minCleanLen = 50

// ExtractedTag is exposed so test mocks can reference it.
type ExtractedTag struct {
	Name       string
	Confidence float64
}

// ProcessingArticleRepository defines what the processing worker needs.
type ProcessingArticleRepository interface {
	GetRawForProcessing(ctx context.Context, id uuid.UUID) (*domain.Article, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateCleanContent(ctx context.Context, id uuid.UUID, clean string, language *string) error
	UpdateTrendScore(ctx context.Context, id uuid.UUID, score float64) error
	SaveSummary(ctx context.Context, s *domain.ArticleSummary) error
	SaveEmbedding(ctx context.Context, articleID uuid.UUID, embedding []float32) error
	SaveTags(ctx context.Context, articleID uuid.UUID, tags []domain.TagWithConfidence) error
	ListAllTagNames(ctx context.Context) ([]string, error)
	FindTagIDsByName(ctx context.Context, names []string) (map[string]uuid.UUID, error)
}

// AIClient defines the AI operations needed by the worker.
type AIClient interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
	CreateSummary(ctx context.Context, title, cleanContent string) (string, int, error)
	CreateTags(ctx context.Context, title, cleanContent string, allowedTags []string) ([]ExtractedTag, int, error)
}

// ProcessingWorker runs the 5-stage article processing pipeline.
type ProcessingWorker struct {
	articles ProcessingArticleRepository
	ai       AIClient
	stream   StreamPublisher
	policy   *bluemonday.Policy
	logger   *zap.Logger
}

// NewProcessingWorker creates a ProcessingWorker.
func NewProcessingWorker(
	articles ProcessingArticleRepository,
	ai AIClient,
	stream StreamPublisher,
	logger *zap.Logger,
) *ProcessingWorker {
	return &ProcessingWorker{
		articles: articles,
		ai:       ai,
		stream:   stream,
		policy:   bluemonday.StrictPolicy(),
		logger:   logger,
	}
}

// ProcessArticle runs all 5 stages for a single article.
func (w *ProcessingWorker) ProcessArticle(ctx context.Context, articleID uuid.UUID) error {
	start := time.Now()

	article, err := w.articles.GetRawForProcessing(ctx, articleID)
	if err != nil {
		return fmt.Errorf("get article: %w", err)
	}

	// Stage 1: Normalize
	clean, skipped := w.normalize(article)
	if skipped {
		_ = w.articles.UpdateStatus(ctx, articleID, "skipped")
		w.logger.Info("article_skipped", zap.String("article_id", articleID.String()), zap.String("reason", "content_too_short"))
		return nil
	}

	if err := w.articles.UpdateCleanContent(ctx, articleID, clean, nil); err != nil {
		return fmt.Errorf("update clean content: %w", err)
	}

	// Stage 2: Embed
	embedText := article.Title + "\n\n" + clean
	embedding, err := w.ai.CreateEmbedding(ctx, embedText)
	if err != nil {
		w.logger.Warn("embedding failed, continuing", zap.String("article_id", articleID.String()), zap.Error(err))
	} else {
		if err := w.articles.SaveEmbedding(ctx, articleID, embedding); err != nil {
			w.logger.Warn("save embedding failed", zap.String("article_id", articleID.String()), zap.Error(err))
		}
	}

	// Stage 3: Summarize
	totalTokens := 0
	summary, tokens, err := w.ai.CreateSummary(ctx, article.Title, clean)
	if err != nil {
		w.logger.Warn("summarize failed, continuing", zap.String("article_id", articleID.String()), zap.Error(err))
	} else if validateSummary(summary) == nil {
		totalTokens += tokens
		cnt := tokens
		if err := w.articles.SaveSummary(ctx, &domain.ArticleSummary{
			ID:            uuid.New(),
			ArticleID:     articleID,
			SummaryText:   summary,
			ModelName:     "gpt-4o-mini",
			ModelVersion:  "2024-07-18",
			PromptVersion: "v1.0",
			TokenCount:    &cnt,
		}); err != nil {
			w.logger.Warn("save summary failed", zap.String("article_id", articleID.String()), zap.Error(err))
		}
	}

	// Stage 4: Tag
	allowedTags, err := w.articles.ListAllTagNames(ctx)
	if err != nil {
		w.logger.Warn("list tags failed, continuing", zap.String("article_id", articleID.String()), zap.Error(err))
	} else if len(allowedTags) > 0 {
		extracted, tagTokens, err := w.ai.CreateTags(ctx, article.Title, clean, allowedTags)
		if err != nil {
			w.logger.Warn("tag extraction failed, continuing", zap.String("article_id", articleID.String()), zap.Error(err))
		} else {
			totalTokens += tagTokens
			if err := w.saveTags(ctx, articleID, extracted); err != nil {
				w.logger.Warn("save tags failed", zap.String("article_id", articleID.String()), zap.Error(err))
			}
		}
	}

	// Stage 5: Trend score
	trendScore := computeTrendScore(article)
	if err := w.articles.UpdateTrendScore(ctx, articleID, trendScore); err != nil {
		w.logger.Warn("update trend score failed", zap.String("article_id", articleID.String()), zap.Error(err))
	}

	// Finalize
	if err := w.articles.UpdateStatus(ctx, articleID, "processed"); err != nil {
		return fmt.Errorf("update status processed: %w", err)
	}

	w.logger.Info("article_processed",
		zap.String("article_id", articleID.String()),
		zap.Int("llm_tokens_used", totalTokens),
		zap.Int64("total_duration_ms", time.Since(start).Milliseconds()),
	)
	return nil
}

// normalize strips HTML, extracts clean text. Returns ("", true) if content is too short.
func (w *ProcessingWorker) normalize(article *domain.Article) (string, bool) {
	if article.RawContent == nil {
		return "", true
	}
	// Strip all HTML tags
	stripped := w.policy.Sanitize(*article.RawContent)
	// Extract text nodes
	clean := extractText(stripped)
	// Collapse whitespace
	clean = collapseWhitespace(clean)
	if utf8.RuneCountInString(strings.TrimSpace(clean)) < minCleanLen {
		return "", true
	}
	return clean, false
}

// saveTags resolves tag IDs and persists them.
func (w *ProcessingWorker) saveTags(ctx context.Context, articleID uuid.UUID, extracted []ExtractedTag) error {
	if len(extracted) == 0 {
		return nil
	}
	names := make([]string, len(extracted))
	for i, t := range extracted {
		names[i] = t.Name
	}
	tagIDs, err := w.articles.FindTagIDsByName(ctx, names)
	if err != nil {
		return err
	}
	var tags []domain.TagWithConfidence
	for _, t := range extracted {
		id, ok := tagIDs[t.Name]
		if !ok {
			continue
		}
		tags = append(tags, domain.TagWithConfidence{
			Tag:        domain.Tag{ID: id, Name: t.Name},
			Confidence: t.Confidence,
		})
	}
	if len(tags) == 0 {
		return nil
	}
	return w.articles.SaveTags(ctx, articleID, tags)
}

// computeTrendScore applies the spec formula: 0.7 * time_decay + 0.3 * source_quality.
// Since we don't have source quality here, default to 0.7 (neutral).
func computeTrendScore(article *domain.Article) float64 {
	decay := usecase.FreshnessScorePtr(article.PublishedAt)
	return 0.7*decay + 0.3*0.7
}

// extractText walks an HTML tree and extracts visible text nodes.
func extractText(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}
	var sb strings.Builder
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
			sb.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	return sb.String()
}

var multiSpaceRe = regexp.MustCompile(`\s+`)

func collapseWhitespace(s string) string {
	return strings.TrimSpace(multiSpaceRe.ReplaceAllString(s, " "))
}

// validateSummary checks that a summary meets quality requirements.
func validateSummary(text string) error {
	if len(text) < 50 {
		return fmt.Errorf("summary too short: %d chars", len(text))
	}
	if len(text) > 1000 {
		return fmt.Errorf("summary too long: %d chars", len(text))
	}
	if strings.Contains(text, "```") || strings.Contains(text, `{"`) {
		return fmt.Errorf("summary contains code or JSON")
	}
	return nil
}
