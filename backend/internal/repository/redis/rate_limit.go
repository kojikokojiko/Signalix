package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const refreshRateLimitTTL = 5 * time.Minute

// RateLimitStore implements usecase.RateLimitStore using Redis SET NX.
type RateLimitStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewRateLimitStore creates a RateLimitStore with a 5-minute TTL.
func NewRateLimitStore(rdb *redis.Client) *RateLimitStore {
	return &RateLimitStore{rdb: rdb, ttl: refreshRateLimitTTL}
}

// Allow returns true if the key has not been set (first call in the window),
// and sets the key with TTL to block subsequent calls.
func (s *RateLimitStore) Allow(ctx context.Context, key string) (bool, error) {
	set, err := s.rdb.SetNX(ctx, key, 1, s.ttl).Result()
	if err != nil {
		return false, err
	}
	return set, nil
}
