package usecase_test

import (
	"testing"
	"time"

	"github.com/kojikokojiko/signalix/internal/usecase"
)

func TestFreshnessScore_Returns1_WhenPublishedWithin6Hours(t *testing.T) {
	pub := time.Now().Add(-3 * time.Hour)
	score := usecase.FreshnessScore(pub)
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestFreshnessScore_Returns085_WhenPublishedWithin12Hours(t *testing.T) {
	pub := time.Now().Add(-9 * time.Hour)
	score := usecase.FreshnessScore(pub)
	if score != 0.85 {
		t.Errorf("expected 0.85, got %f", score)
	}
}

func TestFreshnessScore_Returns070_WhenPublishedWithin24Hours(t *testing.T) {
	pub := time.Now().Add(-18 * time.Hour)
	score := usecase.FreshnessScore(pub)
	if score != 0.70 {
		t.Errorf("expected 0.70, got %f", score)
	}
}

func TestFreshnessScore_Returns050_WhenPublishedWithin3Days(t *testing.T) {
	pub := time.Now().Add(-48 * time.Hour)
	score := usecase.FreshnessScore(pub)
	if score != 0.50 {
		t.Errorf("expected 0.50, got %f", score)
	}
}

func TestFreshnessScore_Returns030_WhenPublishedWithin7Days(t *testing.T) {
	pub := time.Now().Add(-5 * 24 * time.Hour)
	score := usecase.FreshnessScore(pub)
	if score != 0.30 {
		t.Errorf("expected 0.30, got %f", score)
	}
}

func TestFreshnessScore_Returns010_WhenPublishedWithin30Days(t *testing.T) {
	pub := time.Now().Add(-15 * 24 * time.Hour)
	score := usecase.FreshnessScore(pub)
	if score != 0.10 {
		t.Errorf("expected 0.10, got %f", score)
	}
}

func TestFreshnessScore_Returns005_WhenPublishedOver30DaysAgo(t *testing.T) {
	pub := time.Now().Add(-45 * 24 * time.Hour)
	score := usecase.FreshnessScore(pub)
	if score != 0.05 {
		t.Errorf("expected 0.05, got %f", score)
	}
}

func TestFreshnessScore_NilPublishedAt_Returns005(t *testing.T) {
	score := usecase.FreshnessScorePtr(nil)
	if score != 0.05 {
		t.Errorf("expected 0.05 for nil, got %f", score)
	}
}

func TestScoreBreakdown_Total_UsesCorrectWeights(t *testing.T) {
	// 全コンポーネントが 1.0 の場合、合計は 1.0
	s := usecase.NewScoreBreakdown(1.0, 1.0, 1.0, 1.0, 1.0)
	total := s.Total()
	if total < 0.999 || total > 1.001 {
		t.Errorf("expected ~1.0, got %f", total)
	}
}

func TestScoreBreakdown_Total_WeightsSum(t *testing.T) {
	// 各コンポーネントを 0 にして個別の重みを確認
	tests := []struct {
		name     string
		scores   usecase.ScoreBreakdown
		expected float64
	}{
		{"relevance only", usecase.NewScoreBreakdown(1, 0, 0, 0, 0), 0.35},
		{"freshness only", usecase.NewScoreBreakdown(0, 1, 0, 0, 0), 0.20},
		{"trend only", usecase.NewScoreBreakdown(0, 0, 1, 0, 0), 0.20},
		{"source only", usecase.NewScoreBreakdown(0, 0, 0, 1, 0), 0.10},
		{"personal only", usecase.NewScoreBreakdown(0, 0, 0, 0, 1), 0.15},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.scores.Total()
			if got < tc.expected-0.001 || got > tc.expected+0.001 {
				t.Errorf("expected %f, got %f", tc.expected, got)
			}
		})
	}
}

func TestScoreBreakdown_DominantComponent(t *testing.T) {
	tests := []struct {
		name      string
		scores    usecase.ScoreBreakdown
		dominates string
	}{
		{"relevance dominates", usecase.NewScoreBreakdown(0.9, 0.1, 0.1, 0.1, 0.1), "relevance"},
		{"freshness dominates", usecase.NewScoreBreakdown(0.1, 0.9, 0.1, 0.1, 0.1), "freshness"},
		{"trend dominates", usecase.NewScoreBreakdown(0.1, 0.1, 0.9, 0.1, 0.1), "trend"},
		{"personalization dominates", usecase.NewScoreBreakdown(0.1, 0.1, 0.1, 0.1, 0.9), "personalization"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.scores.DominantComponent()
			if got != tc.dominates {
				t.Errorf("expected %s, got %s", tc.dominates, got)
			}
		})
	}
}

func TestGenerateExplanation_Relevance(t *testing.T) {
	scores := usecase.NewScoreBreakdown(0.9, 0.1, 0.1, 0.1, 0.1)
	exp := usecase.GenerateExplanation(scores, "Go", "")
	if exp == "" {
		t.Error("expected non-empty explanation")
	}
}

func TestGenerateExplanation_Trend(t *testing.T) {
	scores := usecase.NewScoreBreakdown(0.1, 0.1, 0.9, 0.1, 0.1)
	exp := usecase.GenerateExplanation(scores, "", "")
	if exp == "" {
		t.Error("expected non-empty explanation")
	}
}

func TestGenerateExplanation_Freshness(t *testing.T) {
	scores := usecase.NewScoreBreakdown(0.1, 0.9, 0.1, 0.1, 0.1)
	exp := usecase.GenerateExplanation(scores, "", "Go Blog")
	if exp == "" {
		t.Error("expected non-empty explanation")
	}
}
