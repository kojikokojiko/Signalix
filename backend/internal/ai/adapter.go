package ai

import (
	"context"

	"github.com/kojikokojiko/signalix/internal/worker"
)

// WorkerAdapter wraps Client to satisfy worker.AIClient.
type WorkerAdapter struct {
	c *Client
}

// NewWorkerAdapter creates a WorkerAdapter.
func NewWorkerAdapter(c *Client) *WorkerAdapter {
	return &WorkerAdapter{c: c}
}

func (a *WorkerAdapter) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return a.c.CreateEmbedding(ctx, text)
}

func (a *WorkerAdapter) CreateSummary(ctx context.Context, title, clean string) (string, int, error) {
	return a.c.CreateSummary(ctx, title, clean)
}

func (a *WorkerAdapter) CreateTags(ctx context.Context, title, clean string, allowed []string) ([]worker.ExtractedTag, int, error) {
	tags, tokens, err := a.c.CreateTags(ctx, title, clean, allowed)
	if err != nil {
		return nil, 0, err
	}
	result := make([]worker.ExtractedTag, len(tags))
	for i, t := range tags {
		result[i] = worker.ExtractedTag{Name: t.Name, Confidence: t.Confidence}
	}
	return result, tokens, nil
}
