package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// --- mocks ---

type mockAdminSourceRepo struct {
	sources  []*domain.Source
	created  *domain.Source
	updated  *domain.Source
	deleted  bool
	createErr error
	updateErr error
	deleteErr error
}

func (m *mockAdminSourceRepo) List(_ context.Context, _ repository.SourceFilter) ([]*domain.Source, int, error) {
	return m.sources, len(m.sources), nil
}
func (m *mockAdminSourceRepo) ListAll(_ context.Context, _ repository.SourceFilter) ([]*domain.Source, int, error) {
	return m.sources, len(m.sources), nil
}
func (m *mockAdminSourceRepo) FindByID(_ context.Context, _ string) (*domain.Source, error) {
	if len(m.sources) > 0 {
		return m.sources[0], nil
	}
	return nil, nil
}
func (m *mockAdminSourceRepo) Create(_ context.Context, s *domain.Source) error {
	m.created = s
	return m.createErr
}
func (m *mockAdminSourceRepo) Update(_ context.Context, _ string, _ repository.SourceUpdateFields) (*domain.Source, error) {
	return m.updated, m.updateErr
}
func (m *mockAdminSourceRepo) Delete(_ context.Context, _ string) error {
	m.deleted = true
	return m.deleteErr
}
func (m *mockAdminSourceRepo) ListDueForFetch(_ context.Context, _ int) ([]*domain.Source, error) { return nil, nil }
func (m *mockAdminSourceRepo) UpdateAfterFetch(_ context.Context, _ string, _ bool) error         { return nil }
func (m *mockAdminSourceRepo) UpdateStatus(_ context.Context, _ string, _ string) error           { return nil }

type mockIngestionJobRepo struct {
	jobs []*domain.IngestionJob
}

func (m *mockIngestionJobRepo) Begin(_ context.Context, _ string) (string, error)          { return "id", nil }
func (m *mockIngestionJobRepo) Complete(_ context.Context, _ string, _, _, _ int) error     { return nil }
func (m *mockIngestionJobRepo) Fail(_ context.Context, _ string, _ string) error            { return nil }
func (m *mockIngestionJobRepo) List(_ context.Context, _ repository.IngestionJobFilter) ([]*domain.IngestionJob, int, error) {
	return m.jobs, len(m.jobs), nil
}

type mockAdminStatsRepo struct{}

func (m *mockAdminStatsRepo) GetStats(_ context.Context) (*domain.AdminStats, error) {
	return &domain.AdminStats{}, nil
}

type mockFetchTrigger struct {
	published string
}

func (m *mockFetchTrigger) Publish(_ context.Context, sourceID string) error {
	m.published = sourceID
	return nil
}

func newAdminUC(src *mockAdminSourceRepo, jobs *mockIngestionJobRepo) *usecase.AdminUsecase {
	return usecase.NewAdminUsecase(src, jobs, &mockAdminStatsRepo{}, &mockFetchTrigger{})
}

// --- tests ---

func TestAdminUsecase_CreateSource_Success(t *testing.T) {
	repo := &mockAdminSourceRepo{}
	uc := newAdminUC(repo, &mockIngestionJobRepo{})

	s, err := uc.CreateSource(context.Background(), usecase.CreateSourceInput{
		Name:     "Test Blog",
		FeedURL:  "https://test.com/feed.atom",
		SiteURL:  "https://test.com",
		Category: "tech",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil || s.Name != "Test Blog" {
		t.Error("expected source to be created")
	}
	if repo.created == nil {
		t.Error("expected Create to be called")
	}
}

func TestAdminUsecase_CreateSource_ValidationError_EmptyName(t *testing.T) {
	uc := newAdminUC(&mockAdminSourceRepo{}, &mockIngestionJobRepo{})
	_, err := uc.CreateSource(context.Background(), usecase.CreateSourceInput{
		FeedURL: "https://test.com/feed", SiteURL: "https://test.com", Category: "tech", Language: "en",
	})
	if !errors.Is(err, usecase.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAdminUsecase_CreateSource_ValidationError_EmptyCategory(t *testing.T) {
	uc := newAdminUC(&mockAdminSourceRepo{}, &mockIngestionJobRepo{})
	_, err := uc.CreateSource(context.Background(), usecase.CreateSourceInput{
		Name: "X", FeedURL: "https://test.com/feed", SiteURL: "https://test.com",
		Category: "", Language: "en",
	})
	if !errors.Is(err, usecase.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestAdminUsecase_CreateSource_DuplicateFeedURL(t *testing.T) {
	repo := &mockAdminSourceRepo{createErr: errors.New("duplicate key value violates unique constraint")}
	uc := newAdminUC(repo, &mockIngestionJobRepo{})
	_, err := uc.CreateSource(context.Background(), usecase.CreateSourceInput{
		Name: "X", FeedURL: "https://test.com/feed", SiteURL: "https://test.com",
		Category: "tech", Language: "en",
	})
	if !errors.Is(err, usecase.ErrFeedURLAlreadyExists) {
		t.Errorf("expected ErrFeedURLAlreadyExists, got %v", err)
	}
}

func TestAdminUsecase_UpdateSource_NotFound(t *testing.T) {
	repo := &mockAdminSourceRepo{updated: nil}
	uc := newAdminUC(repo, &mockIngestionJobRepo{})
	_, err := uc.UpdateSource(context.Background(), "non-existent", usecase.UpdateSourceInput{})
	if !errors.Is(err, usecase.ErrSourceNotFound) {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestAdminUsecase_DeleteSource_Success(t *testing.T) {
	repo := &mockAdminSourceRepo{}
	uc := newAdminUC(repo, &mockIngestionJobRepo{})
	if err := uc.DeleteSource(context.Background(), "some-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.deleted {
		t.Error("expected Delete to be called")
	}
}

func TestAdminUsecase_DeleteSource_NotFound(t *testing.T) {
	repo := &mockAdminSourceRepo{deleteErr: errors.New("no rows in result set")}
	uc := newAdminUC(repo, &mockIngestionJobRepo{})
	err := uc.DeleteSource(context.Background(), "non-existent")
	if !errors.Is(err, usecase.ErrSourceNotFound) {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestAdminUsecase_TriggerFetch_PublishesEvent(t *testing.T) {
	trigger := &mockFetchTrigger{}
	src := &domain.Source{ID: "src-1"}
	repo := &mockAdminSourceRepo{sources: []*domain.Source{src}}
	uc := usecase.NewAdminUsecase(repo, &mockIngestionJobRepo{}, &mockAdminStatsRepo{}, trigger)

	jobID, err := uc.TriggerFetch(context.Background(), "src-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jobID == "" {
		t.Error("expected non-empty job ID")
	}
	if trigger.published != "src-1" {
		t.Errorf("expected published source ID 'src-1', got %q", trigger.published)
	}
}

func TestAdminUsecase_ListIngestionJobs(t *testing.T) {
	jobs := []*domain.IngestionJob{{ID: "j1", Status: "completed"}}
	uc := newAdminUC(&mockAdminSourceRepo{}, &mockIngestionJobRepo{jobs: jobs})
	result, total, err := uc.ListIngestionJobs(context.Background(), usecase.IngestionJobListInput{Page: 1, PerPage: 50})
	if err != nil || total != 1 || len(result) != 1 {
		t.Errorf("unexpected result: %v, %d, %v", result, total, err)
	}
}

func TestAdminUsecase_GetStats(t *testing.T) {
	uc := newAdminUC(&mockAdminSourceRepo{}, &mockIngestionJobRepo{})
	stats, err := uc.GetStats(context.Background())
	if err != nil || stats == nil {
		t.Errorf("unexpected: %v, %v", stats, err)
	}
}

var _ repository.SourceRepository = (*mockAdminSourceRepo)(nil)
var _ repository.IngestionJobRepository = (*mockIngestionJobRepo)(nil)
var _ repository.AdminStatsRepository = (*mockAdminStatsRepo)(nil)
