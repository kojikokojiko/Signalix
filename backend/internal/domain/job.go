package domain

import "time"

type IngestionJob struct {
	ID              string
	SourceID        string
	SourceName      string // JOIN で取得
	Status          string // running | completed | failed
	ArticlesFound   int
	ArticlesNew     int
	ArticlesSkipped int
	ErrorMessage    *string
	StartedAt       time.Time
	CompletedAt     *time.Time
}

type AdminStats struct {
	Sources struct {
		Total    int
		Active   int
		Degraded int
		Disabled int
	}
	Articles struct {
		Total     int
		Processed int
		Pending   int
		Failed    int
	}
	IngestionJobs struct {
		Last24hCompleted int
		Last24hFailed    int
	}
	Users struct {
		Total         int
		ActiveLast7d  int
	}
}
