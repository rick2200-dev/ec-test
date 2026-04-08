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
	"github.com/Riku-KANO/ec-test/services/search/internal/config"
	"github.com/Riku-KANO/ec-test/services/search/internal/engine"
	"github.com/Riku-KANO/ec-test/services/search/internal/handler"
	"github.com/Riku-KANO/ec-test/services/search/internal/service"
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

	// Search engine selection
	var searchEngine engine.SearchEngine
	if cfg.VertexAIEnabled {
		slog.Info("using Vertex AI search engine", "project_id", cfg.GCPProjectID)
		searchEngine = engine.NewVertexAIEngine(cfg.GCPProjectID)
	} else {
		slog.Info("using PostgreSQL full-text search engine")
		searchEngine = engine.NewPostgresEngine(pool)
	}

	// Service
	searchSvc := service.NewSearchService(searchEngine)

	// Handlers
	searchHandler := handler.NewSearchHandler(searchSvc)
	healthHandler := handler.NewHealthHandler(pool)

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(pkgmiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Health endpoints (no auth required)
	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// Search endpoints
	r.Mount("/search", searchHandler.Routes())

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
		slog.Info("starting search service", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Pub/Sub subscriber for product events (optional)
	// TODO: Start subscriber when GCPSubscriber is implemented in the shared pkg.
	// if cfg.PubSubProjectID != "" {
	//     sub, err := pubsub.NewGCPSubscriber(ctx, cfg.PubSubProjectID)
	//     if err != nil {
	//         slog.Warn("failed to create pubsub subscriber", "error", err)
	//     } else {
	//         defer sub.Close()
	//         productSub := subscriber.NewProductSubscriber(searchEngine, sub)
	//         go func() {
	//             if err := productSub.Start(context.Background()); err != nil {
	//                 slog.Error("product subscriber error", "error", err)
	//             }
	//         }()
	//     }
	// }

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

	slog.Info("search service stopped")
}
