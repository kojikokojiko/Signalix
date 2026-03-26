package usecase

import (
	"context"
	"fmt"

	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

type SourceListInput struct {
	Category *string
	Language *string
	Page     int
	PerPage  int
}

type SourceListResult struct {
	Sources    []*domain.Source
	Page       int
	PerPage    int
	Total      int
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

type SourceUsecase struct {
	sources repository.SourceRepository
}

func NewSourceUsecase(sources repository.SourceRepository) *SourceUsecase {
	return &SourceUsecase{sources: sources}
}

func (uc *SourceUsecase) List(ctx context.Context, in SourceListInput) (*SourceListResult, error) {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PerPage < 1 || in.PerPage > 100 {
		in.PerPage = 50
	}

	sources, total, err := uc.sources.List(ctx, repository.SourceFilter{
		Category: in.Category,
		Language: in.Language,
		Page:     in.Page,
		PerPage:  in.PerPage,
	})
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}

	totalPages := (total + in.PerPage - 1) / in.PerPage
	return &SourceListResult{
		Sources:    sources,
		Page:       in.Page,
		PerPage:    in.PerPage,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    in.Page < totalPages,
		HasPrev:    in.Page > 1,
	}, nil
}

func (uc *SourceUsecase) GetByID(ctx context.Context, id string) (*domain.Source, error) {
	s, err := uc.sources.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find source: %w", err)
	}
	return s, nil
}
