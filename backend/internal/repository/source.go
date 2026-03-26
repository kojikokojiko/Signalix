package repository

import (
	"context"

	"github.com/kojikokojiko/signalix/internal/domain"
)

type SourceFilter struct {
	Category *string
	Language *string
	Page     int
	PerPage  int
}

type SourceRepository interface {
	List(ctx context.Context, filter SourceFilter) ([]*domain.Source, int, error)
	FindByID(ctx context.Context, id string) (*domain.Source, error)
	// Admin用
	ListAll(ctx context.Context, filter SourceFilter) ([]*domain.Source, int, error)
	Create(ctx context.Context, s *domain.Source) error
	Update(ctx context.Context, id string, fields SourceUpdateFields) (*domain.Source, error)
	Delete(ctx context.Context, id string) error
	// Worker用
	ListDueForFetch(ctx context.Context, limit int) ([]*domain.Source, error)
	UpdateAfterFetch(ctx context.Context, id string, success bool) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

// SourceUpdateFields holds the optional fields for PATCH /admin/sources/:id.
type SourceUpdateFields struct {
	Name                 *string
	Description          *string
	Category             *string
	FetchIntervalMinutes *int
	QualityScore         *float64
	Status               *string
}
