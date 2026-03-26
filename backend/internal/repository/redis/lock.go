package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type LockStore struct {
	rdb *redis.Client
}

func NewLockStore(rdb *redis.Client) *LockStore {
	return &LockStore{rdb: rdb}
}

func (s *LockStore) GetFailCount(ctx context.Context, email string) (int, error) {
	v, err := s.rdb.Get(ctx, failKey(email)).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return v, err
}

func (s *LockStore) IncrFailCount(ctx context.Context, email string, ttl time.Duration) error {
	key := failKey(email)
	pipe := s.rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *LockStore) ResetFailCount(ctx context.Context, email string) error {
	return s.rdb.Del(ctx, failKey(email)).Err()
}

func (s *LockStore) IsLocked(ctx context.Context, email string) (bool, error) {
	v, err := s.rdb.Exists(ctx, lockKey(email)).Result()
	if err != nil {
		return false, err
	}
	return v > 0, nil
}

func (s *LockStore) Lock(ctx context.Context, email string, ttl time.Duration) error {
	return s.rdb.Set(ctx, lockKey(email), 1, ttl).Err()
}

func (s *LockStore) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	return s.rdb.Set(ctx, blacklistKey(jti), 1, ttl).Err()
}

func (s *LockStore) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	v, err := s.rdb.Exists(ctx, blacklistKey(jti)).Result()
	if err != nil {
		return false, err
	}
	return v > 0, nil
}

func failKey(email string) string     { return fmt.Sprintf("auth:fail:%s", email) }
func lockKey(email string) string     { return fmt.Sprintf("auth:lock:%s", email) }
func blacklistKey(jti string) string  { return fmt.Sprintf("auth:blacklist:%s", jti) }
