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
	"github.com/Riku-KANO/ec-test/services/recommend/internal/adapter/http"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/adapter/pubsub"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/app"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/config"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/engine"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/repository"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()

	// Background context for long-lived tasks (JWKS refresh, etc.).
	// Cancelled during graceful shutdown to stop background goroutines.
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	initCtx, initCancel := context.WithTimeout(bgCtx, 10*time.Second)
	defer initCancel()

	pool, err := database.NewPool(initCtx, database.Config{
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
	viewRefresher := repository.NewPostgresViewRefresher(pool)
	recommendSvc := app.NewRecommendService(eng, viewRefresher)

	// Handlers
	recommendHandler := handler.NewRecommendHandler(recommendSvc)
	healthHandler := handler.NewHealthHandler(pool)

	// Pub/Sub subscriber (optional, only if PUBSUB_PROJECT_ID is set).
	if cfg.PubSubProjectID != "" {
		sub, err := newGCPSubscriber(initCtx, cfg.PubSubProjectID)
		if err != nil {
			slog.Error("failed to create pubsub subscriber", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := sub.Close(); err != nil {
				slog.Warn("failed to close pubsub subscriber", "error", err)
			}
		}()

		eventSub := subscriber.NewEventSubscriber(recommendSvc, sub)
		if err := eventSub.Start(initCtx); err != nil {
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

	// Recommendation endpoints (require JWT / tenant context).
	// JWT config is read from JWT_ISSUER, JWT_AUDIENCE, and JWKS_URL env vars.
	jwtMW := pkgmiddleware.NewJWTMiddleware(bgCtx, pkgmiddleware.JWTConfig{
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		JWKSURL:  cfg.JWKSURL,
	})
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

	// Cancel background context to stop JWKS refresh and other background tasks.
	bgCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("recommend service stopped")
}

// newGCPSubscriber creates a GCP Pub/Sub subscriber via the shared
// pkg/pubsub implementation so every subscriber-owning service sees the
// same ack/nack/log semantics.
func newGCPSubscriber(ctx context.Context, projectID string) (pubsub.Subscriber, error) {
	return pubsub.NewGCPSubscriber(ctx, projectID)
}
