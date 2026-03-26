package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ProcessingStream      = "stream:article_processing"
	processingGroup       = "article_processing_workers"
	reclaimMinIdle        = 5 * time.Minute
)

// StreamConsumer reads article IDs from a Redis Streams consumer group.
type StreamConsumer struct {
	rdb        *redis.Client
	consumerID string
}

// NewStreamConsumer creates a consumer and ensures the consumer group exists.
func NewStreamConsumer(rdb *redis.Client, consumerID string) (*StreamConsumer, error) {
	ctx := context.Background()
	err := rdb.XGroupCreateMkStream(ctx, ProcessingStream, processingGroup, "0").Err()
	if err != nil && !errors.Is(err, redis.Nil) && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, err
	}
	return &StreamConsumer{rdb: rdb, consumerID: consumerID}, nil
}

// ReadBatch fetches up to n pending messages from the consumer group.
// It first tries to reclaim stale messages, then reads new ones.
func (c *StreamConsumer) ReadBatch(ctx context.Context, n int) ([]string, []string, error) {
	// Reclaim messages idle > reclaimMinIdle
	claimResp, _, err := c.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   ProcessingStream,
		Group:    processingGroup,
		Consumer: c.consumerID,
		MinIdle:  reclaimMinIdle,
		Start:    "0-0",
		Count:    int64(n),
	}).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	var articleIDs []string
	var msgIDs []string

	for _, msg := range claimResp {
		if id, ok := msg.Values["article_id"].(string); ok {
			articleIDs = append(articleIDs, id)
			msgIDs = append(msgIDs, msg.ID)
		}
	}

	// If we got claimed messages, return them
	if len(articleIDs) > 0 {
		return articleIDs, msgIDs, nil
	}

	// Read new messages
	resp, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    processingGroup,
		Consumer: c.consumerID,
		Streams:  []string{ProcessingStream, ">"},
		Count:    int64(n),
		Block:    time.Second,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	for _, stream := range resp {
		for _, msg := range stream.Messages {
			if id, ok := msg.Values["article_id"].(string); ok {
				articleIDs = append(articleIDs, id)
				msgIDs = append(msgIDs, msg.ID)
			}
		}
	}
	return articleIDs, msgIDs, nil
}

// Ack acknowledges a processed message.
func (c *StreamConsumer) Ack(ctx context.Context, msgID string) error {
	return c.rdb.XAck(ctx, ProcessingStream, processingGroup, msgID).Err()
}
