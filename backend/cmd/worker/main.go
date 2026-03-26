package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/kojikokojiko/signalix/internal/ai"
	"github.com/kojikokojiko/signalix/internal/config"
	"github.com/kojikokojiko/signalix/internal/db"
	"github.com/kojikokojiko/signalix/internal/repository/postgres"
	redisrepo "github.com/kojikokojiko/signalix/internal/repository/redis"
	"github.com/kojikokojiko/signalix/internal/worker"
)

const ingestionInterval = 5 * time.Minute

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Fatal("failed to parse redis url", zap.Error(err))
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	// 依存関係の組み立て
	sourceRepo := postgres.NewSourceRepository(pool)
	articleRepo := postgres.NewArticleRepository(pool)
	interestRepo := postgres.NewInterestRepository(pool)
	recommendationRepo := postgres.NewRecommendationRepository(pool)
	jobStore := postgres.NewIngestionJobStore(pool)
	fetchLock := redisrepo.NewFetchLock(rdb)
	stream := redisrepo.NewStreamPublisher(rdb)

	ingestionWorker := worker.NewIngestionWorker(
		sourceRepo, articleRepo, jobStore, fetchLock, stream, logger,
	)

	// スコアリングワーカー（レコメンデーション再計算）
	scoringConsumerID := fmt.Sprintf("scoring-%s", uuid.New().String()[:8])
	scoringWorker, err := worker.NewScoringWorker(rdb, scoringConsumerID, recommendationRepo, interestRepo, articleRepo, logger)
	if err != nil {
		logger.Fatal("failed to create scoring worker", zap.Error(err))
	}
	go func() {
		if err := scoringWorker.Run(ctx); err != nil {
			logger.Error("scoring worker failed", zap.Error(err))
		}
	}()
	logger.Info("scoring worker started", zap.String("consumer_id", scoringConsumerID))

	// 処理ワーカー（OpenAI APIキーが設定されている場合のみ起動）
	if cfg.OpenAIKey != "" {
		aiClient := ai.NewWorkerAdapter(ai.NewClient(cfg.OpenAIKey))
		processingWorker := worker.NewProcessingWorker(articleRepo, aiClient, stream, logger)

		consumerID := fmt.Sprintf("worker-%s", uuid.New().String()[:8])
		consumer, err := redisrepo.NewStreamConsumer(rdb, consumerID)
		if err != nil {
			logger.Fatal("failed to create stream consumer", zap.Error(err))
		}

		go runProcessingLoop(ctx, consumer, processingWorker, logger)
		logger.Info("processing worker started", zap.String("consumer_id", consumerID))
	} else {
		logger.Warn("OPENAI_API_KEY not set, processing worker disabled")
	}

	logger.Info("ingestion worker started", zap.Duration("interval", ingestionInterval))

	// 起動直後に1回実行
	go func() {
		if err := ingestionWorker.Run(ctx); err != nil {
			logger.Error("ingestion run failed", zap.Error(err))
		}
	}()

	// 定期実行
	ticker := time.NewTicker(ingestionInterval)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			go func() {
				if err := ingestionWorker.Run(ctx); err != nil {
					logger.Error("ingestion run failed", zap.Error(err))
				}
			}()
		case <-quit:
			logger.Info("worker stopped")
			return
		}
	}
}

func runProcessingLoop(ctx context.Context, consumer *redisrepo.StreamConsumer, w *worker.ProcessingWorker, logger *zap.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		articleIDs, msgIDs, err := consumer.ReadBatch(ctx, 5)
		if err != nil {
			logger.Error("stream read failed", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}

		for i, idStr := range articleIDs {
			articleID, err := uuid.Parse(idStr)
			if err != nil {
				logger.Warn("invalid article_id in stream", zap.String("id", idStr))
				_ = consumer.Ack(ctx, msgIDs[i])
				continue
			}

			if err := w.ProcessArticle(ctx, articleID); err != nil {
				logger.Error("processing failed", zap.String("article_id", idStr), zap.Error(err))
				// Keep message in stream for reclaim
				continue
			}

			_ = consumer.Ack(ctx, msgIDs[i])
		}
	}
}
