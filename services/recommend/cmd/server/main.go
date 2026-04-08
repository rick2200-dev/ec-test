package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/Riku-KANO/ec-test/pkg/database"
	pkgmiddleware "github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/config"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/engine"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/handler"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/service"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/subscriber"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, database.Config{
		URL:      cfg.DatabaseURL,
		MaxConns: 20,
		MinConns: 5,
	})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("connected to database")

	// Recommendation engine selection based on configuration.
	var eng engine.RecommendEngine
	if cfg.VertexAIEnabled {
		slog.Info("using Vertex AI recommendation engine")
		eng = engine.NewVertexAIEngine(cfg.GCPProjectID)
	} else {
		slog.Info("using PostgreSQL recommendation engine")
		eng = engine.NewPostgresEngine(pool)
	}

	// Service
	recommendSvc := service.NewRecommendService(eng, pool)

	// Handlers
	recommendHandler := handler.NewRecommendHandler(recommendSvc)
	healthHandler := handler.NewHealthHandler(pool)

	// Pub/Sub subscriber (optional, only if PUBSUB_PROJECT_ID is set).
	if cfg.PubSubProjectID != "" {
		sub, err := newGCPSubscriber(ctx, cfg.PubSubProjectID)
		if err != nil {
			slog.Error("failed to create pubsub subscriber", "error", err)
			os.Exit(1)
		}
		defer sub.Close()

		eventSub := subscriber.NewEventSubscriber(recommendSvc, sub)
		if err := eventSub.Start(ctx); err != nil {
			slog.Error("failed to start event subscribers", "error", err)
			os.Exit(1)
		}
		slog.Info("started event subscribers")
	} else {
		slog.Info("PUBSUB_PROJECT_ID not set, skipping event subscribers")
	}

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(pkgmiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Health endpoints (no auth required)
	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// Recommendation endpoints (require JWT / tenant context)
	jwtMW := pkgmiddleware.NewJWTMiddleware(context.Background(), pkgmiddleware.JWTConfig{})
	r.Group(func(r chi.Router) {
		r.Use(jwtMW.VerifyJWT)
		r.Mount("/recommendations", recommendHandler.Routes())
	})

	// HTTP server
	addr := ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine.
	go func() {
		slog.Info("starting recommend service", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	slog.Info("shutting down", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("recommend service stopped")
}

// newGCPSubscriber creates a GCP Pub/Sub subscriber. This is extracted to keep
// main clean and to allow future replacement with other subscriber backends.
func newGCPSubscriber(ctx context.Context, projectID string) (pubsub.Subscriber, error) {
	// The GCP Pub/Sub library currently only provides GCPPublisher in the shared
	// pkg. We create a minimal subscriber here that wraps the GCP client directly.
	return newGCPSub(ctx, projectID)
}
