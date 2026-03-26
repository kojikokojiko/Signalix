package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type FetchLock struct {
	rdb *redis.Client
}

func NewFetchLock(rdb *redis.Client) *FetchLock {
	return &FetchLock{rdb: rdb}
}

func (l *FetchLock) Acquire(ctx context.Context, sourceID string, ttl time.Duration) (bool, error) {
	ok, err := l.rdb.SetNX(ctx, ingestionLockKey(sourceID), 1, ttl).Result()
	return ok, err
}

func (l *FetchLock) Release(ctx context.Context, sourceID string) error {
	return l.rdb.Del(ctx, ingestionLockKey(sourceID)).Err()
}

func ingestionLockKey(sourceID string) string {
	return fmt.Sprintf("lock:ingestion:%s", sourceID)
}
