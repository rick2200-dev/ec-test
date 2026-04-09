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
	"github.com/Riku-KANO/ec-test/services/auth/internal/config"
	"github.com/Riku-KANO/ec-test/services/auth/internal/handler"
	"github.com/Riku-KANO/ec-test/services/auth/internal/repository"
	"github.com/Riku-KANO/ec-test/services/auth/internal/service"
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

	// Repositories
	tenantRepo := repository.NewTenantRepository(pool)
	sellerRepo := repository.NewSellerRepository(pool)
	subscriptionRepo := repository.NewSubscriptionRepository(pool)
	buyerSubRepo := repository.NewBuyerSubscriptionRepository(pool)

	// Service
	authSvc := service.NewAuthService(tenantRepo, sellerRepo, subscriptionRepo, buyerSubRepo)

	// Handlers
	tenantHandler := handler.NewTenantHandler(authSvc)
	sellerHandler := handler.NewSellerHandler(authSvc)
	subscriptionHandler := handler.NewSubscriptionHandler(authSvc)
	buyerSubHandler := handler.NewBuyerSubscriptionHandler(authSvc)
	healthHandler := handler.NewHealthHandler(pool)

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(pkgmiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(pkgmiddleware.InternalContext)

	// Health endpoints (no auth required)
	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// Tenant endpoints (no tenant context needed for CRUD on tenants themselves)
	r.Mount("/tenants", tenantHandler.Routes())

	// Seller endpoints (tenant-scoped, require JWT)
	r.Mount("/sellers", sellerHandler.Routes())

	// Subscription plan endpoints (tenant-scoped)
	r.Mount("/plans", subscriptionHandler.PlanRoutes())
	r.Mount("/subscriptions", subscriptionHandler.SubscriptionRoutes())

	// Buyer plan and subscription endpoints (tenant-scoped)
	r.Mount("/buyer-plans", buyerSubHandler.BuyerPlanRoutes())
	r.Mount("/buyer-subscriptions", buyerSubHandler.BuyerSubscriptionRoutes())

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
		slog.Info("starting auth service", "addr", addr)
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

	slog.Info("auth service stopped")
}
