package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	articleProcessingStream        = "stream:article_processing"
	recommendationRefreshStream    = "stream:recommendation_refresh"
)

type StreamPublisher struct {
	rdb *redis.Client
}

func NewStreamPublisher(rdb *redis.Client) *StreamPublisher {
	return &StreamPublisher{rdb: rdb}
}

func (p *StreamPublisher) Publish(ctx context.Context, articleID string) error {
	return p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: articleProcessingStream,
		Values: map[string]any{
			"article_id": articleID,
			"priority":   "normal",
		},
	}).Err()
}

// PublishRecommendationRefresh publishes a scoring request for the given userID.
func (p *StreamPublisher) PublishRecommendationRefresh(ctx context.Context, userID string) error {
	return p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: recommendationRefreshStream,
		Values: map[string]any{
			"user_id": userID,
		},
	}).Err()
}
