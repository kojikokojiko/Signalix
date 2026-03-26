package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/kojikokojiko/signalix/internal/repository"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

const (
	scoringStream      = "stream:recommendation_refresh"
	scoringGroup       = "recommendation_scoring_workers"
	scoringReclaimIdle = 5 * time.Minute
	scoringBlockTime   = time.Second
)

// ScoringWorker listens to stream:recommendation_refresh and recomputes
// recommendation scores for each user received in the stream.
type ScoringWorker struct {
	rdb             *redis.Client
	consumerID      string
	recommendations repository.RecommendationRepository
	interests       repository.InterestRepository
	articles        repository.ArticleRepository
	logger          *zap.Logger
}

// NewScoringWorker creates a ScoringWorker and ensures the consumer group exists.
func NewScoringWorker(
	rdb *redis.Client,
	consumerID string,
	recommendations repository.RecommendationRepository,
	interests repository.InterestRepository,
	articles repository.ArticleRepository,
	logger *zap.Logger,
) (*ScoringWorker, error) {
	ctx := context.Background()
	err := rdb.XGroupCreateMkStream(ctx, scoringStream, scoringGroup, "0").Err()
	if err != nil && !errors.Is(err, redis.Nil) && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, fmt.Errorf("create scoring consumer group: %w", err)
	}
	return &ScoringWorker{
		rdb:             rdb,
		consumerID:      consumerID,
		recommendations: recommendations,
		interests:       interests,
		articles:        articles,
		logger:          logger,
	}, nil
}

// Run continuously reads from stream:recommendation_refresh and triggers scoring.
// It exits when ctx is cancelled.
func (w *ScoringWorker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		userIDs, msgIDs, err := w.readBatch(ctx, 5)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			w.logger.Error("scoring stream read failed", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}

		for i, userIDStr := range userIDs {
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				w.logger.Warn("invalid user_id in scoring stream", zap.String("user_id", userIDStr))
				_ = w.ack(ctx, msgIDs[i])
				continue
			}

			if err := w.computeAndStore(ctx, userID); err != nil {
				w.logger.Error("scoring failed",
					zap.String("user_id", userIDStr),
					zap.Error(err),
				)
				// Leave message in stream for reclaim
				continue
			}

			_ = w.ack(ctx, msgIDs[i])
			w.logger.Info("recommendation_scoring_completed", zap.String("user_id", userIDStr))
		}
	}
}

// computeAndStore delegates to the recommendation usecase.
func (w *ScoringWorker) computeAndStore(ctx context.Context, userID uuid.UUID) error {
	uc := usecase.NewRecommendationUsecase(w.recommendations, w.interests, &noopRateLimit{}, nil)
	return uc.ComputeAndStore(ctx, userID, nil)
}

// readBatch fetches up to n messages: first tries to reclaim stale, then reads new.
func (w *ScoringWorker) readBatch(ctx context.Context, n int) ([]string, []string, error) {
	// Try to reclaim stale messages
	claimResp, _, err := w.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   scoringStream,
		Group:    scoringGroup,
		Consumer: w.consumerID,
		MinIdle:  scoringReclaimIdle,
		Start:    "0-0",
		Count:    int64(n),
	}).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	var userIDs, msgIDs []string
	for _, msg := range claimResp {
		if id, ok := msg.Values["user_id"].(string); ok {
			userIDs = append(userIDs, id)
			msgIDs = append(msgIDs, msg.ID)
		}
	}
	if len(userIDs) > 0 {
		return userIDs, msgIDs, nil
	}

	// Read new messages
	resp, err := w.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    scoringGroup,
		Consumer: w.consumerID,
		Streams:  []string{scoringStream, ">"},
		Count:    int64(n),
		Block:    scoringBlockTime,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	for _, stream := range resp {
		for _, msg := range stream.Messages {
			if id, ok := msg.Values["user_id"].(string); ok {
				userIDs = append(userIDs, id)
				msgIDs = append(msgIDs, msg.ID)
			}
		}
	}
	return userIDs, msgIDs, nil
}

func (w *ScoringWorker) ack(ctx context.Context, msgID string) error {
	return w.rdb.XAck(ctx, scoringStream, scoringGroup, msgID).Err()
}

// noopRateLimit is used internally by the scoring worker to bypass rate limits.
type noopRateLimit struct{}

func (n *noopRateLimit) Allow(_ context.Context, _ string) (bool, error) {
	return true, nil
}
