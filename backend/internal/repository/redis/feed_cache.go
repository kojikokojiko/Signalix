package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// FeedCache provides a Redis-backed cache for recommendation feed responses.
type FeedCache struct {
	rdb *redis.Client
}

// NewFeedCache creates a new FeedCache.
func NewFeedCache(rdb *redis.Client) *FeedCache {
	return &FeedCache{rdb: rdb}
}

// Get retrieves cached bytes by key. Returns nil, nil when not found.
func (c *FeedCache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

// Set stores bytes under key with the given TTL.
func (c *FeedCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

// Delete removes all keys matching the given pattern using SCAN + DEL.
func (c *FeedCache) Delete(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
