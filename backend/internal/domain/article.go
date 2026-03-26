package domain

import (
	"time"

	"github.com/google/uuid"
)

type Article struct {
	ID          uuid.UUID
	SourceID    uuid.UUID
	URL         string
	URLHash     string
	Title       string
	RawContent  *string
	CleanContent *string
	Author      *string
	Language    *string
	PublishedAt *time.Time
	TrendScore  float64
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ArticleSummary struct {
	ID           uuid.UUID
	ArticleID    uuid.UUID
	SummaryText  string
	ModelName    string
	ModelVersion string
	PromptVersion string
	TokenCount   *int
	CreatedAt    time.Time
}

type Tag struct {
	ID       uuid.UUID
	Name     string
	Slug     string
	Category string
}

// ArticleWithDetails includes joined data for API responses.
type ArticleWithDetails struct {
	Article
	Source  *Source
	Summary *ArticleSummary
	Tags    []TagWithConfidence
}

type TagWithConfidence struct {
	Tag
	Confidence float64
}
