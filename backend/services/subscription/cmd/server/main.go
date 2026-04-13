package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc"

	subscriptionv1 "github.com/Riku-KANO/ec-test/gen/go/subscription/v1"
	"github.com/Riku-KANO/ec-test/pkg/database"
	pkgmiddleware "github.com/Riku-KANO/ec-test/pkg/middleware"
	grpcserver "github.com/Riku-KANO/ec-test/services/subscription/internal/adapter/grpc"
	handler "github.com/Riku-KANO/ec-test/services/subscription/internal/adapter/http"
	repository "github.com/Riku-KANO/ec-test/services/subscription/internal/adapter/postgres"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/app"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/config"
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

	sellerRepo := repository.NewSellerSubscriptionRepository(pool)
	buyerRepo := repository.NewBuyerSubscriptionRepository(pool)

	svc := app.NewService(sellerRepo, buyerRepo)

	subscriptionHandler := handler.NewSubscriptionHandler(svc)
	buyerSubHandler := handler.NewBuyerSubscriptionHandler(svc)
	healthHandler := handler.NewHealthHandler(pool)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(pkgmiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(pkgmiddleware.InternalContext)

	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// All non-health routes require the shared X-Internal-Token header.
	// The subscription service is purely in-cluster (gateway and order
	// are the only callers) so we gate every business route rather than
	// only a /internal/* subtree. Fails closed on empty secret.
	if cfg.InternalToken == "" {
		slog.Warn("SUBSCRIPTION_INTERNAL_TOKEN is empty — all non-health requests will be rejected with 503")
	}
	r.Group(func(gr chi.Router) {
		gr.Use(pkgmiddleware.RequireInternalToken(cfg.InternalToken))
		gr.Mount("/plans", subscriptionHandler.PlanRoutes())
		gr.Mount("/subscriptions", subscriptionHandler.SubscriptionRoutes())
		gr.Mount("/buyer-plans", buyerSubHandler.BuyerPlanRoutes())
		gr.Mount("/buyer-subscriptions", buyerSubHandler.BuyerSubscriptionRoutes())
	})

	addr := ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	grpcAddr := ":" + cfg.GRPCPort
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		slog.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}
	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(pkgmiddleware.UnaryInternalTokenInterceptor(cfg.InternalToken)),
	)
	subscriptionv1.RegisterSubscriptionServiceServer(grpcSrv, grpcserver.NewServer(svc))

	go func() {
		slog.Info("starting subscription gRPC server", "addr", grpcAddr)
		if err := grpcSrv.Serve(grpcListener); err != nil {
			slog.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		slog.Info("starting subscription service", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	slog.Info("shutting down", "signal", sig.String())

	grpcSrv.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("subscription service stopped")
}
