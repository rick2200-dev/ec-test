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
	"github.com/Riku-KANO/ec-test/services/auth/internal/adapter/http"
	"github.com/Riku-KANO/ec-test/services/auth/internal/adapter/postgres"
	"github.com/Riku-KANO/ec-test/services/auth/internal/app"
	"github.com/Riku-KANO/ec-test/services/auth/internal/config"
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
	sellerRepo := repository.NewSellerRepository(pool)
	sellerUserRepo := repository.NewSellerUserRepository(pool)
	platformAdminRepo := repository.NewPlatformAdminRepository(pool)
	rbacAuditRepo := repository.NewRBACAuditRepository(pool)
	apiTokenRepo := repository.NewAPITokenRepository(pool)
	buyerRepo := repository.NewBuyerRepository(pool)

	// Services — split along domain boundaries so handlers depend only on
	// the surface they actually need. A future extraction of Subscription
	// into its own deployable is a matter of dropping the constructor here
	// and pointing the handlers at a gRPC client instead.
	txRunner := &database.PoolTxRunner{Pool: pool}
	identitySvc := app.NewIdentityService(txRunner, sellerRepo, sellerUserRepo)
	rbacSvc := app.NewRBACService(txRunner, sellerUserRepo, platformAdminRepo, rbacAuditRepo)
	credentialSvc := app.NewCredentialService(txRunner, apiTokenRepo, sellerUserRepo)
	buyerSvc := app.NewBuyerService(buyerRepo)

	// Bootstrap the initial super_admin if requested via environment.
	if cfg.BootstrapSuperAdminSub != "" {
		bootCtx, bootCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if bootErr := rbacSvc.BootstrapSuperAdmin(bootCtx, cfg.BootstrapSuperAdminSub); bootErr != nil {
			slog.Error("failed to bootstrap super_admin", "error", bootErr)
		}
		bootCancel()
	}

	// Handlers
	sellerTeamHandler := handler.NewSellerTeamHandler(rbacSvc)
	apiTokenHandler := handler.NewAPITokenHandler(credentialSvc, cfg.APITokenPrefix)
	sellerHandler := handler.NewSellerHandler(identitySvc, sellerTeamHandler, apiTokenHandler)
	platformAdminHandler := handler.NewPlatformAdminHandler(rbacSvc)
	internalAuthzHandler := handler.NewInternalAuthzHandler(rbacSvc, credentialSvc, cfg.InternalToken)
	internalBuyerHandler := handler.NewInternalBuyerHandler(buyerSvc, cfg.InternalToken)
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

	// Seller endpoints. Note: seller team management is nested at
	// /sellers/{sellerID}/team via SellerHandler.
	r.Mount("/sellers", sellerHandler.Routes())

	// Platform admin management (super_admin operations).
	r.Mount("/platform-admins", platformAdminHandler.Routes())

	// RBAC audit log. Mounted on a separate prefix to avoid
	// conflicting with the /platform-admins subtree above.
	r.Mount("/rbac-audit", platformAdminHandler.AuditRoutes())

	// Internal authz endpoints used by the gateway for fine-grained role
	// lookups. Protected by a shared secret middleware.
	r.Mount("/internal/authz", internalAuthzHandler.Routes())

	// Internal buyer profile upsert, called by the Next.js BFF on Auth0
	// callback. Same shared-secret protection as /internal/authz.
	r.Mount("/internal/buyers", internalBuyerHandler.Routes())

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
