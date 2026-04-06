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
	"github.com/Riku-KANO/ec-test/services/order/internal/config"
	"github.com/Riku-KANO/ec-test/services/order/internal/handler"
	"github.com/Riku-KANO/ec-test/services/order/internal/repository"
	"github.com/Riku-KANO/ec-test/services/order/internal/service"
	stripeClient "github.com/Riku-KANO/ec-test/services/order/internal/stripe"
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

	// Stripe client
	sc := stripeClient.NewClient(cfg.StripeSecretKey)

	// Repositories
	orderRepo := repository.NewOrderRepository(pool)
	commissionRepo := repository.NewCommissionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	// Service
	orderSvc := service.NewOrderService(orderRepo, commissionRepo, payoutRepo, sc)

	// Handlers
	orderHandler := handler.NewOrderHandler(orderSvc)
	commissionHandler := handler.NewCommissionHandler(orderSvc)
	payoutHandler := handler.NewPayoutHandler(orderSvc)
	webhookHandler := handler.NewWebhookHandler(orderSvc, cfg.StripeWebhookSecret)
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

	// Stripe webhooks (no auth required, validated by signature)
	r.Post("/webhooks/stripe", webhookHandler.HandleStripeWebhook)

	// Order endpoints (tenant-scoped)
	r.Mount("/orders", orderHandler.Routes())

	// Commission endpoints (tenant-scoped)
	r.Mount("/commissions", commissionHandler.Routes())

	// Payout endpoints (tenant-scoped)
	r.Mount("/payouts", payoutHandler.Routes())

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
		slog.Info("starting order service", "addr", addr)
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

	slog.Info("order service stopped")
}
