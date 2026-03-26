package domain

import (
	"time"

	"github.com/google/uuid"
)

type RecommendationLog struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	ArticleID           uuid.UUID
	TotalScore          float64
	RelevanceScore      float64
	FreshnessScore      float64
	TrendScore          float64
	SourceQualityScore  float64
	PersonalizationBoost float64
	Explanation         string
	GeneratedAt         time.Time
	ExpiresAt           time.Time
}

type ScoreBreakdown struct {
	Relevance       float64
	Freshness       float64
	Trend           float64
	SourceQuality   float64
	Personalization float64
}

func (s ScoreBreakdown) Total() float64 {
	return s.Relevance*0.35 +
		s.Freshness*0.20 +
		s.Trend*0.20 +
		s.SourceQuality*0.10 +
		s.Personalization*0.15
}

func (s ScoreBreakdown) DominantComponent() string {
	max := s.Relevance
	dominant := "relevance"
	if s.Freshness > max {
		max = s.Freshness
		dominant = "freshness"
	}
	if s.Trend > max {
		max = s.Trend
		dominant = "trend"
	}
	if s.Personalization > max {
		dominant = "personalization"
	}
	return dominant
}

type RecommendedItem struct {
	Article     *ArticleWithDetails
	Log         *RecommendationLog
}
