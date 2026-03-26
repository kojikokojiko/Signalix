package repository

import (
	"context"

	"github.com/kojikokojiko/signalix/internal/domain"
)

type IngestionJobFilter struct {
	SourceID *string
	Status   *string
	Page     int
	PerPage  int
}

type IngestionJobRepository interface {
	// Worker用 (既存)
	Begin(ctx context.Context, sourceID string) (string, error)
	Complete(ctx context.Context, jobID string, found, newCount, skipped int) error
	Fail(ctx context.Context, jobID string, errMsg string) error
	// Admin用
	List(ctx context.Context, filter IngestionJobFilter) ([]*domain.IngestionJob, int, error)
}

type AdminStatsRepository interface {
	GetStats(ctx context.Context) (*domain.AdminStats, error)
}
