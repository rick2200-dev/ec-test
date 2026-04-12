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
	handler "github.com/Riku-KANO/ec-test/services/review/internal/adapter/http"
	"github.com/Riku-KANO/ec-test/services/review/internal/adapter/httpclient"
	repository "github.com/Riku-KANO/ec-test/services/review/internal/adapter/postgres"
	"github.com/Riku-KANO/ec-test/services/review/internal/app"
	"github.com/Riku-KANO/ec-test/services/review/internal/config"
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

	// Pub/Sub publisher (best-effort; nil publisher means events are silently dropped)
	var publisher pubsub.Publisher
	if cfg.PubSubProjectID != "" {
		pub, pubErr := pubsub.NewGCPPublisher(ctx, cfg.PubSubProjectID)
		if pubErr != nil {
			slog.Warn("failed to create pubsub publisher, events will not be published", "error", pubErr)
		} else {
			publisher = pub
			slog.Info("pubsub publisher created", "project_id", cfg.PubSubProjectID)
		}
	} else {
		slog.Info("PUBSUB_PROJECT_ID not set, event publishing disabled")
	}
	defer func() {
		if publisher != nil {
			if err := publisher.Close(); err != nil {
				slog.Error("failed to close pubsub publisher", "error", err)
			}
		}
	}()

	// Repository
	reviewRepo := repository.NewReviewRepository(pool)

	// Internal clients
	catalogClient := httpclient.NewCatalogClient(cfg.CatalogServiceURL, cfg.CatalogInternalToken)
	orderClient := httpclient.NewOrderClient(cfg.OrderServiceURL, cfg.OrderInternalToken)

	// Service
	reviewSvc := app.NewReviewService(reviewRepo, catalogClient, orderClient, publisher)

	// Handlers
	buyerHandler := handler.NewBuyerHandler(reviewSvc)
	sellerHandler := handler.NewSellerHandler(reviewSvc)
	healthHandler := handler.NewHealthHandler(pool)

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(pkgmiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(pkgmiddleware.InternalContext)

	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// Buyer-facing routes (tenant-scoped, mounted under /reviews)
	r.Mount("/reviews", buyerHandler.Routes())

	// Seller-facing routes (tenant + seller-scoped, mounted under /seller/reviews)
	r.Mount("/seller/reviews", sellerHandler.Routes())

	addr := ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting review service", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

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

	slog.Info("review service stopped")
}
