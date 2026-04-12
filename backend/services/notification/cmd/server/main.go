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

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/notification/internal/config"
	"github.com/Riku-KANO/ec-test/services/notification/internal/email"
	"github.com/Riku-KANO/ec-test/services/notification/internal/adapter/http"
	"github.com/Riku-KANO/ec-test/services/notification/internal/adapter/pubsub"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()

	// Set emulator host if configured.
	if cfg.PubSubEmulatorHost != "" {
		if err := os.Setenv("PUBSUB_EMULATOR_HOST", cfg.PubSubEmulatorHost); err != nil {
			slog.Warn("failed to set PUBSUB_EMULATOR_HOST", "error", err)
		}
	}

	// Email sender (log-only for MVP).
	sender := email.NewLogSender()

	// Pub/Sub subscriber.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub, err := pubsub.NewGCPSubscriber(ctx, cfg.PubSubProjectID)
	if err != nil {
		slog.Error("failed to create pubsub subscriber", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := sub.Close(); err != nil {
			slog.Warn("failed to close pubsub subscriber", "error", err)
		}
	}()

	slog.Info("connected to pubsub", "project_id", cfg.PubSubProjectID)

	// Start event subscribers in goroutines.
	orderSub := subscriber.NewOrderSubscriber(sub, sender)
	inventorySub := subscriber.NewInventorySubscriber(sub, sender)
	inquirySub := subscriber.NewInquirySubscriber(sub, sender)
	reviewSub := subscriber.NewReviewSubscriber(sub, sender)

	go func() {
		if err := orderSub.Start(ctx); err != nil {
			slog.Error("order subscriber error", "error", err)
		}
	}()

	go func() {
		if err := inventorySub.Start(ctx); err != nil {
			slog.Error("inventory subscriber error", "error", err)
		}
	}()

	go func() {
		if err := inquirySub.Start(ctx); err != nil {
			slog.Error("inquiry subscriber error", "error", err)
		}
	}()

	go func() {
		if err := reviewSub.Start(ctx); err != nil {
			slog.Error("review subscriber error", "error", err)
		}
	}()

	// Health handler.
	healthHandler := handler.NewHealthHandler()

	// Router.
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// HTTP server.
	addr := ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting notification service", "addr", addr)
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

	// Cancel Pub/Sub subscribers.
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("notification service stopped")
}
