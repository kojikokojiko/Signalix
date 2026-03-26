package usecase

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
)

// ScoreBreakdown holds individual score components.
type ScoreBreakdown = domain.ScoreBreakdown

// NewScoreBreakdown creates a ScoreBreakdown with all components.
func NewScoreBreakdown(relevance, freshness, trend, sourceQuality, personalization float64) ScoreBreakdown {
	return ScoreBreakdown{
		Relevance:       relevance,
		Freshness:       freshness,
		Trend:           trend,
		SourceQuality:   sourceQuality,
		Personalization: personalization,
	}
}

// FreshnessScore calculates a time-decay score based on article age.
func FreshnessScore(publishedAt time.Time) float64 {
	age := time.Since(publishedAt)
	switch {
	case age < 6*time.Hour:
		return 1.0
	case age < 12*time.Hour:
		return 0.85
	case age < 24*time.Hour:
		return 0.70
	case age < 3*24*time.Hour:
		return 0.50
	case age < 7*24*time.Hour:
		return 0.30
	case age < 30*24*time.Hour:
		return 0.10
	default:
		return 0.05
	}
}

// FreshnessScorePtr handles nil published_at.
func FreshnessScorePtr(publishedAt *time.Time) float64 {
	if publishedAt == nil {
		return 0.05
	}
	return FreshnessScore(*publishedAt)
}

// PersonalizationBoost calculates a boost based on positive feedback tag overlap.
func PersonalizationBoost(tags []domain.TagWithConfidence, positiveTags map[uuid.UUID]float64) float64 {
	if len(tags) == 0 || len(positiveTags) == 0 {
		return 0.0
	}
	boost := 0.0
	matchCount := 0
	for _, tag := range tags {
		if freq, ok := positiveTags[tag.ID]; ok {
			boost += freq * tag.Confidence
			matchCount++
		}
	}
	if matchCount == 0 {
		return 0.0
	}
	v := boost / float64(matchCount)
	if v > 1.0 {
		return 1.0
	}
	return v
}

// GenerateExplanation produces a human-readable explanation based on dominant score.
func GenerateExplanation(scores ScoreBreakdown, topTagName, sourceName string) string {
	dominant := scores.DominantComponent()
	switch dominant {
	case "relevance":
		if topTagName != "" {
			return fmt.Sprintf("あなたがよく読む %s の記事に類似しています", topTagName)
		}
		return "あなたの興味に類似した記事です"
	case "trend":
		return "今週のトレンド上位の記事です"
	case "personalization":
		if topTagName != "" {
			return fmt.Sprintf("「%s」に興味があるあなたへのおすすめです", topTagName)
		}
		return "あなたの過去の行動に基づくおすすめです"
	case "freshness":
		if sourceName != "" {
			return fmt.Sprintf("%s の最新記事です", sourceName)
		}
		return "公開されたばかりの記事です"
	default:
		return "あなたの興味とトレンドの両方に一致しています"
	}
}
