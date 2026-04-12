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

	pkgmiddleware "github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/cart/internal/adapter/http"
	"github.com/Riku-KANO/ec-test/services/cart/internal/adapter/httpclient"
	"github.com/Riku-KANO/ec-test/services/cart/internal/adapter/postgres"
	cartredis "github.com/Riku-KANO/ec-test/services/cart/internal/adapter/redis"
	"github.com/Riku-KANO/ec-test/services/cart/internal/app"
	"github.com/Riku-KANO/ec-test/services/cart/internal/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	redisClient, err := cartredis.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Warn("failed to close redis client", "error", err)
		}
	}()

	slog.Info("connected to redis", "url", cfg.RedisURL)

	// Pub/Sub publisher (optional — cart events are best-effort).
	var publisher pubsub.Publisher
	if cfg.PubSubProjectID != "" {
		pub, pubErr := pubsub.NewGCPPublisher(ctx, cfg.PubSubProjectID)
		if pubErr != nil {
			slog.Warn("failed to create pubsub publisher, events will not be published", "error", pubErr)
		} else {
			publisher = pub
			defer func() {
				if err := pub.Close(); err != nil {
					slog.Warn("failed to close pubsub publisher", "error", err)
				}
			}()
			slog.Info("pubsub publisher created", "project_id", cfg.PubSubProjectID)
		}
	} else {
		slog.Info("PUBSUB_PROJECT_ID not set, event publishing disabled")
	}

	// Wire dependencies.
	cartRepo := repository.NewCartRepository(redisClient, cfg.CartTTLSeconds)
	catalogClient := httpclient.NewCatalogClient(cfg.CatalogServiceURL, cfg.CatalogInternalToken)
	orderClient := httpclient.NewOrderClient(cfg.OrderServiceURL, cfg.OrderInternalToken)
	cartSvc := app.NewCartService(cartRepo, catalogClient, orderClient, publisher)

	cartHandler := handler.NewCartHandler(cartSvc)
	healthHandler := handler.NewHealthHandler(redisClient)

	// Router.
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(pkgmiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(pkgmiddleware.InternalContext)

	// Health.
	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// Cart API (tenant-scoped via X-Tenant-ID / X-User-ID headers).
	r.Mount("/cart", cartHandler.Routes())

	// HTTP server.
	addr := ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 35 * time.Second, // checkout may take longer than default
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting cart service", "addr", addr)
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

	slog.Info("cart service stopped")
}
