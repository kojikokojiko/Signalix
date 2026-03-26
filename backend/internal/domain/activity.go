package domain

import (
	"time"

	"github.com/google/uuid"
)

type Bookmark struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ArticleID uuid.UUID
	CreatedAt time.Time
}

type UserFeedback struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	ArticleID    uuid.UUID
	FeedbackType string // like | dislike | save | click | hide
	CreatedAt    time.Time
}

type UserInterest struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TagID     uuid.UUID
	Weight    float64
	Source    string // manual | inferred
	UpdatedAt time.Time
}

// InterestItem is the enriched view of a user interest including tag details.
type InterestItem struct {
	TagName  string
	TagID    uuid.UUID
	Category string
	Weight   float64
	Source   string
}

var ValidFeedbackTypes = map[string]bool{
	"like": true, "dislike": true, "save": true, "click": true, "hide": true,
}

// FeedbackWeightDelta returns the weight delta applied to user_interests.
func FeedbackWeightDelta(feedbackType string) float64 {
	switch feedbackType {
	case "like", "save", "click":
		return +0.05
	case "dislike", "hide":
		return -0.10
	}
	return 0
}
