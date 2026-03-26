package domain_test

import (
	"testing"

	"github.com/kojikokojiko/signalix/internal/domain"
)

// ─── FeedbackWeightDelta ──────────────────────────────────────────────────────

func TestFeedbackWeightDelta(t *testing.T) {
	tests := []struct {
		feedbackType string
		want         float64
	}{
		{"like", +0.05},
		{"save", +0.05},
		{"click", +0.05},
		{"dislike", -0.10},
		{"hide", -0.10},
		{"unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.feedbackType, func(t *testing.T) {
			got := domain.FeedbackWeightDelta(tt.feedbackType)
			if got != tt.want {
				t.Errorf("FeedbackWeightDelta(%q) = %v, want %v", tt.feedbackType, got, tt.want)
			}
		})
	}
}

// ─── ValidFeedbackTypes ───────────────────────────────────────────────────────

func TestValidFeedbackTypes(t *testing.T) {
	valid := []string{"like", "dislike", "save", "click", "hide"}
	for _, ft := range valid {
		if !domain.ValidFeedbackTypes[ft] {
			t.Errorf("expected %q to be valid", ft)
		}
	}

	invalid := []string{"bookmark", "share", "", "LIKE"}
	for _, ft := range invalid {
		if domain.ValidFeedbackTypes[ft] {
			t.Errorf("expected %q to be invalid", ft)
		}
	}
}

// ─── ScoreBreakdown.Total ─────────────────────────────────────────────────────

func TestScoreBreakdown_Total(t *testing.T) {
	tests := []struct {
		name string
		s    domain.ScoreBreakdown
		want float64
	}{
		{
			name: "all zeros",
			s:    domain.ScoreBreakdown{},
			want: 0.0,
		},
		{
			name: "all ones",
			s: domain.ScoreBreakdown{
				Relevance:       1.0,
				Freshness:       1.0,
				Trend:           1.0,
				SourceQuality:   1.0,
				Personalization: 1.0,
			},
			// 0.35 + 0.20 + 0.20 + 0.10 + 0.15 = 1.0
			want: 1.0,
		},
		{
			name: "weights sum to 1.0",
			s: domain.ScoreBreakdown{
				Relevance:       0.8,
				Freshness:       0.6,
				Trend:           0.5,
				SourceQuality:   0.9,
				Personalization: 0.7,
			},
			// 0.8*0.35 + 0.6*0.20 + 0.5*0.20 + 0.9*0.10 + 0.7*0.15
			// = 0.28 + 0.12 + 0.10 + 0.09 + 0.105 = 0.695
			want: 0.695,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Total()
			diff := got - tt.want
			if diff < -0.0001 || diff > 0.0001 {
				t.Errorf("Total() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── ScoreBreakdown.DominantComponent ────────────────────────────────────────

func TestScoreBreakdown_DominantComponent(t *testing.T) {
	tests := []struct {
		name string
		s    domain.ScoreBreakdown
		want string
	}{
		{
			name: "relevance dominates",
			s:    domain.ScoreBreakdown{Relevance: 0.9, Freshness: 0.5, Trend: 0.3, SourceQuality: 0.2, Personalization: 0.4},
			want: "relevance",
		},
		{
			name: "freshness dominates",
			s:    domain.ScoreBreakdown{Relevance: 0.3, Freshness: 0.9, Trend: 0.4, SourceQuality: 0.2, Personalization: 0.5},
			want: "freshness",
		},
		{
			name: "trend dominates",
			s:    domain.ScoreBreakdown{Relevance: 0.3, Freshness: 0.4, Trend: 0.9, SourceQuality: 0.2, Personalization: 0.5},
			want: "trend",
		},
		{
			name: "personalization dominates",
			s:    domain.ScoreBreakdown{Relevance: 0.3, Freshness: 0.4, Trend: 0.5, SourceQuality: 0.2, Personalization: 0.9},
			want: "personalization",
		},
		{
			name: "all zeros defaults to relevance",
			s:    domain.ScoreBreakdown{},
			want: "relevance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.DominantComponent()
			if got != tt.want {
				t.Errorf("DominantComponent() = %q, want %q", got, tt.want)
			}
		})
	}
}
