package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/kojikokojiko/signalix/internal/config"
	"github.com/kojikokojiko/signalix/internal/handler"
	authmw "github.com/kojikokojiko/signalix/internal/middleware"
	"github.com/kojikokojiko/signalix/internal/repository/postgres"
	redisrepo "github.com/kojikokojiko/signalix/internal/repository/redis"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
}

func New(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client, logger *zap.Logger) *Server {
	r := chi.NewRouter()

	// 標準ミドルウェア
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// ヘルスチェック (認証不要)
	healthHandler := handler.NewHealthHandler(db, rdb)
	r.Get("/health", healthHandler.Health)

	// 依存関係の組み立て
	userRepo := postgres.NewUserRepository(db)
	lockStore := redisrepo.NewLockStore(rdb)
	authUC := usecase.NewAuthUsecase(
		userRepo,
		lockStore,
		cfg.JWTSecret,
		1*time.Hour,
		7*24*time.Hour,
	)

	authHandler := handler.NewAuthHandler(authUC)

	sourceRepo := postgres.NewSourceRepository(db)
	articleRepo := postgres.NewArticleRepository(db)
	bookmarkRepo := postgres.NewBookmarkRepository(db)
	feedbackRepo := postgres.NewFeedbackRepository(db)
	interestRepo := postgres.NewInterestRepository(db)
	recommendationRepo := postgres.NewRecommendationRepository(db)
	rateLimitStore := redisrepo.NewRateLimitStore(rdb)
	ingestionJobStore := postgres.NewIngestionJobStore(db)
	adminStatsStore := postgres.NewAdminStatsStore(db)
	streamPublisher := redisrepo.NewStreamPublisher(rdb)
	feedCache := redisrepo.NewFeedCache(rdb)

	tagRepo := postgres.NewTagRepository(db)

	sourceUC := usecase.NewSourceUsecase(sourceRepo)
	articleUC := usecase.NewArticleUsecase(articleRepo, nil)
	bookmarkUC := usecase.NewBookmarkUsecase(bookmarkRepo, articleRepo)
	feedbackUC := usecase.NewFeedbackUsecase(feedbackRepo, articleRepo, interestRepo)
	recommendationUC := usecase.NewRecommendationUsecase(recommendationRepo, interestRepo, rateLimitStore, streamPublisher)
	adminUC := usecase.NewAdminUsecase(sourceRepo, ingestionJobStore, adminStatsStore, streamPublisher)
	userUC := usecase.NewUserUsecase(userRepo, interestRepo, tagRepo)

	sourceHandler := handler.NewSourceHandler(sourceUC)
	articleHandler := handler.NewArticleHandler(articleUC)
	bookmarkHandler := handler.NewBookmarkHandler(bookmarkUC)
	feedbackHandler := handler.NewFeedbackHandler(feedbackUC)
	recommendationHandler := handler.NewRecommendationHandler(recommendationUC, feedCache)
	adminHandler := handler.NewAdminHandler(adminUC)
	userHandler := handler.NewUserHandler(userUC)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// 認証エンドポイント (認証不要)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.With(authmw.Authenticate(authUC)).Post("/logout", authHandler.Logout)
		})

		// ソース (認証不要)
		r.Get("/sources", sourceHandler.List)
		r.Get("/sources/{id}", sourceHandler.GetByID)

		// 記事 (認証任意)
		r.Get("/articles/trending", articleHandler.Trending)
		r.Get("/articles", articleHandler.List)
		r.Get("/articles/{id}", articleHandler.GetByID)

		// 認証必須エンドポイント
		r.Group(func(r chi.Router) {
			r.Use(authmw.Authenticate(authUC))

			// ユーザー
			r.Get("/users/me", userHandler.GetMe)
			r.Patch("/users/me", userHandler.UpdateMe)
			r.Get("/users/me/interests", userHandler.GetInterests)
			r.Put("/users/me/interests", userHandler.SetInterests)

			// ブックマーク
			r.Get("/bookmarks", bookmarkHandler.List)
			r.Post("/bookmarks", bookmarkHandler.Add)
			r.Delete("/bookmarks/{article_id}", bookmarkHandler.Remove)

			// フィードバック
			r.Post("/feedback", feedbackHandler.Submit)
			r.Delete("/feedback/{article_id}", feedbackHandler.Delete)

			// レコメンデーション
			r.Get("/recommendations", recommendationHandler.List)
			r.Post("/recommendations/refresh", recommendationHandler.Refresh)
		})

		// 管理者エンドポイント
		r.Group(func(r chi.Router) {
			r.Use(authmw.Authenticate(authUC))
			r.Use(authmw.RequireAdmin)

			r.Get("/admin/sources", adminHandler.ListSources)
			r.Post("/admin/sources", adminHandler.CreateSource)
			r.Patch("/admin/sources/{id}", adminHandler.UpdateSource)
			r.Delete("/admin/sources/{id}", adminHandler.DeleteSource)
			r.Post("/admin/sources/{id}/fetch", adminHandler.TriggerFetch)
			r.Get("/admin/ingestion-jobs", adminHandler.ListIngestionJobs)
			r.Get("/admin/stats", adminHandler.GetStats)
		})
	})

	return &Server{
		httpServer: &http.Server{
			Addr:         ":" + cfg.APIPort,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

// Handler returns the underlying HTTP handler (for testing).
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

func (s *Server) Start() error {
	s.logger.Info("starting server", zap.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	return s.httpServer.Shutdown(ctx)
}
