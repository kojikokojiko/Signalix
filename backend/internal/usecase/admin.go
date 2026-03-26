package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

// FetchTrigger publishes a source ID to the ingestion stream.
type FetchTrigger interface {
	Publish(ctx context.Context, sourceID string) error
}

type AdminUsecase struct {
	sources    repository.SourceRepository
	jobs       repository.IngestionJobRepository
	stats      repository.AdminStatsRepository
	fetchTrigger FetchTrigger
}

func NewAdminUsecase(
	sources repository.SourceRepository,
	jobs repository.IngestionJobRepository,
	stats repository.AdminStatsRepository,
	fetchTrigger FetchTrigger,
) *AdminUsecase {
	return &AdminUsecase{sources: sources, jobs: jobs, stats: stats, fetchTrigger: fetchTrigger}
}

// --- Source CRUD ---

type CreateSourceInput struct {
	Name                 string
	FeedURL              string
	SiteURL              string
	Description          *string
	Category             string
	Language             string
	FetchIntervalMinutes int
	QualityScore         float64
}

func (uc *AdminUsecase) CreateSource(ctx context.Context, in CreateSourceInput) (*domain.Source, error) {
	if in.Name == "" || len(in.Name) > 100 {
		return nil, fmt.Errorf("%w: name must be 1-100 characters", ErrValidation)
	}
	if in.FeedURL == "" {
		return nil, fmt.Errorf("%w: feed_url is required", ErrValidation)
	}
	if in.SiteURL == "" {
		return nil, fmt.Errorf("%w: site_url is required", ErrValidation)
	}
	if in.Category == "" {
		return nil, fmt.Errorf("%w: category is required", ErrValidation)
	}
	if in.Language == "" {
		return nil, fmt.Errorf("%w: language is required", ErrValidation)
	}
	if in.FetchIntervalMinutes == 0 {
		in.FetchIntervalMinutes = 60
	}
	if in.FetchIntervalMinutes < 15 || in.FetchIntervalMinutes > 1440 {
		return nil, fmt.Errorf("%w: fetch_interval_minutes must be 15-1440", ErrValidation)
	}
	if in.QualityScore == 0 {
		in.QualityScore = 0.7
	}
	if in.QualityScore < 0 || in.QualityScore > 1 {
		return nil, fmt.Errorf("%w: quality_score must be 0.0-1.0", ErrValidation)
	}

	s := &domain.Source{
		ID:                   uuid.New().String(),
		Name:                 in.Name,
		FeedURL:              in.FeedURL,
		SiteURL:              in.SiteURL,
		Description:          in.Description,
		Category:             in.Category,
		Language:             in.Language,
		FetchIntervalMinutes: in.FetchIntervalMinutes,
		QualityScore:         in.QualityScore,
		Status:               "active",
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := uc.sources.Create(ctx, s); err != nil {
		if isDuplicateError(err) {
			return nil, ErrFeedURLAlreadyExists
		}
		return nil, fmt.Errorf("create source: %w", err)
	}
	return s, nil
}

type UpdateSourceInput struct {
	Name                 *string
	Description          *string
	Category             *string
	FetchIntervalMinutes *int
	QualityScore         *float64
	Status               *string
}

func (uc *AdminUsecase) UpdateSource(ctx context.Context, id string, in UpdateSourceInput) (*domain.Source, error) {
	if in.Category != nil && *in.Category == "" {
		return nil, fmt.Errorf("%w: category is required", ErrValidation)
	}
	if in.FetchIntervalMinutes != nil && (*in.FetchIntervalMinutes < 15 || *in.FetchIntervalMinutes > 1440) {
		return nil, fmt.Errorf("%w: fetch_interval_minutes must be 15-1440", ErrValidation)
	}
	if in.QualityScore != nil && (*in.QualityScore < 0 || *in.QualityScore > 1) {
		return nil, fmt.Errorf("%w: quality_score must be 0.0-1.0", ErrValidation)
	}

	s, err := uc.sources.Update(ctx, id, repository.SourceUpdateFields{
		Name:                 in.Name,
		Description:          in.Description,
		Category:             in.Category,
		FetchIntervalMinutes: in.FetchIntervalMinutes,
		QualityScore:         in.QualityScore,
		Status:               in.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update source: %w", err)
	}
	if s == nil {
		return nil, ErrSourceNotFound
	}
	return s, nil
}

func (uc *AdminUsecase) DeleteSource(ctx context.Context, id string) error {
	if err := uc.sources.Delete(ctx, id); err != nil {
		if isNoRowsError(err) {
			return ErrSourceNotFound
		}
		return fmt.Errorf("delete source: %w", err)
	}
	return nil
}

func (uc *AdminUsecase) ListAdminSources(ctx context.Context, filter repository.SourceFilter) ([]*domain.Source, int, error) {
	return uc.sources.ListAll(ctx, filter)
}

// TriggerFetch publishes a manual fetch event for a source.
func (uc *AdminUsecase) TriggerFetch(ctx context.Context, sourceID string) (string, error) {
	s, err := uc.sources.FindByID(ctx, sourceID)
	if err != nil {
		return "", fmt.Errorf("find source: %w", err)
	}
	if s == nil {
		return "", ErrSourceNotFound
	}
	jobID := uuid.New().String()
	if err := uc.fetchTrigger.Publish(ctx, sourceID); err != nil {
		return "", fmt.Errorf("publish fetch trigger: %w", err)
	}
	return jobID, nil
}

// --- Ingestion Jobs ---

type IngestionJobListInput struct {
	SourceID *string
	Status   *string
	Page     int
	PerPage  int
}

func (uc *AdminUsecase) ListIngestionJobs(ctx context.Context, in IngestionJobListInput) ([]*domain.IngestionJob, int, error) {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PerPage < 1 || in.PerPage > 100 {
		in.PerPage = 50
	}
	return uc.jobs.List(ctx, repository.IngestionJobFilter{
		SourceID: in.SourceID,
		Status:   in.Status,
		Page:     in.Page,
		PerPage:  in.PerPage,
	})
}

// --- Stats ---

func (uc *AdminUsecase) GetStats(ctx context.Context) (*domain.AdminStats, error) {
	s, err := uc.stats.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return s, nil
}

// --- helpers ---

func isDuplicateError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint")
}

func isNoRowsError(err error) bool {
	return strings.Contains(err.Error(), "no rows")
}
