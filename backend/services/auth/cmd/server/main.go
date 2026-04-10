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
	"github.com/google/uuid"

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
	sellerUserRepo := repository.NewSellerUserRepository(pool)
	platformAdminRepo := repository.NewPlatformAdminRepository(pool)
	rbacAuditRepo := repository.NewRBACAuditRepository(pool)
	subscriptionRepo := repository.NewSubscriptionRepository(pool)
	buyerSubRepo := repository.NewBuyerSubscriptionRepository(pool)

	// Service
	authSvc := service.NewAuthService(
		pool,
		tenantRepo,
		sellerRepo,
		sellerUserRepo,
		platformAdminRepo,
		rbacAuditRepo,
		subscriptionRepo,
		buyerSubRepo,
	)

	// Bootstrap the initial super_admin if requested via environment.
	if cfg.BootstrapSuperAdminSub != "" && cfg.BootstrapTenantID != "" {
		bootCtx, bootCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if tid, parseErr := uuid.Parse(cfg.BootstrapTenantID); parseErr != nil {
			slog.Error("invalid AUTH_BOOTSTRAP_TENANT_ID, skipping bootstrap", "error", parseErr)
		} else if bootErr := authSvc.BootstrapSuperAdmin(bootCtx, tid, cfg.BootstrapSuperAdminSub); bootErr != nil {
			slog.Error("failed to bootstrap super_admin", "error", bootErr)
		}
		bootCancel()
	}

	// Handlers
	tenantHandler := handler.NewTenantHandler(authSvc)
	sellerTeamHandler := handler.NewSellerTeamHandler(authSvc)
	sellerHandler := handler.NewSellerHandler(authSvc, sellerTeamHandler)
	platformAdminHandler := handler.NewPlatformAdminHandler(authSvc)
	internalAuthzHandler := handler.NewInternalAuthzHandler(authSvc, cfg.InternalToken)
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

	// Seller endpoints (tenant-scoped, require JWT). Note: seller team
	// management is nested at /sellers/{sellerID}/team via SellerHandler.
	r.Mount("/sellers", sellerHandler.Routes())

	// Platform admin management (tenant-scoped, super_admin operations).
	r.Mount("/platform-admins", platformAdminHandler.Routes())

	// RBAC audit log (tenant-scoped). Mounted on a separate prefix to avoid
	// conflicting with the /platform-admins subtree above.
	r.Mount("/rbac-audit", platformAdminHandler.AuditRoutes())

	// Internal authz endpoints used by the gateway for fine-grained role
	// lookups. Protected by a shared secret middleware.
	r.Mount("/internal/authz", internalAuthzHandler.Routes())

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
