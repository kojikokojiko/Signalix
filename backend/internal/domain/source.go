package domain

import "time"

type Source struct {
	ID                   string
	Name                 string
	FeedURL              string
	SiteURL              string
	Description          *string
	Category             string
	Language             string
	FetchIntervalMinutes int
	QualityScore         float64
	Status               string
	LastFetchedAt        *time.Time
	ConsecutiveFailures  int
	CreatedAt            time.Time
	UpdatedAt            time.Time

	// 記事件数（JOIN で取得する場合のみ設定）
	ArticleCount *int
}
