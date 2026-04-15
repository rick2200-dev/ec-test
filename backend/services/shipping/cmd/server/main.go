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

	shippingv1 "github.com/Riku-KANO/ec-test/gen/go/shipping/v1"
	"github.com/Riku-KANO/ec-test/pkg/database"
	pkgmiddleware "github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	grpcserver "github.com/Riku-KANO/ec-test/services/shipping/internal/adapter/grpc"
	handler "github.com/Riku-KANO/ec-test/services/shipping/internal/adapter/http"
	repository "github.com/Riku-KANO/ec-test/services/shipping/internal/adapter/postgres"
	subscriber "github.com/Riku-KANO/ec-test/services/shipping/internal/adapter/pubsub"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/app"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/config"
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

	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	// Pub/Sub publisher (best-effort).
	var publisher pubsub.Publisher
	if cfg.PubSubProjectID != "" {
		pub, pubErr := pubsub.NewGCPPublisher(initCtx, cfg.PubSubProjectID)
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

	// Repository + tx runner.
	shipmentRepo := repository.NewShipmentRepository(pool)
	txRunner := &database.PoolTxRunner{Pool: pool}

	// Service — publisher is no longer injected here; events go through the
	// outbox table and are picked up by the relay goroutine below.
	shipmentSvc := app.NewShipmentService(shipmentRepo, txRunner)

	// Background context for long-lived goroutines (subscribers + relay).
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	if cfg.PubSubProjectID != "" {
		sub, subErr := pubsub.NewGCPSubscriber(bgCtx, cfg.PubSubProjectID)
		if subErr != nil {
			slog.Error("failed to create pubsub subscriber", "error", subErr)
			os.Exit(1)
		}
		defer func() {
			if err := sub.Close(); err != nil {
				slog.Warn("failed to close pubsub subscriber", "error", err)
			}
		}()

		orderSub := subscriber.NewOrderSubscriber(sub, shipmentSvc)
		go func() {
			if err := orderSub.Start(bgCtx); err != nil {
				slog.Error("order subscriber error", "error", err)
			}
		}()

		// Outbox relay: publishes events that were written atomically alongside
		// shipment status updates. Uses the publisher that was created above.
		if publisher != nil {
			relay := subscriber.NewOutboxRelay(pool, publisher)
			go relay.Start(bgCtx)
			slog.Info("outbox relay started")
		}
	} else {
		slog.Info("PUBSUB_PROJECT_ID not set, subscribers and relay disabled")
	}

	// HTTP handlers.
	sellerHandler := handler.NewSellerHandler(shipmentSvc)
	sellerOrderHandler := handler.NewSellerOrderShipmentHandler(shipmentSvc)
	buyerHandler := handler.NewBuyerHandler(shipmentSvc)
	healthHandler := handler.NewHealthHandler()

	// Router.
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(pkgmiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(pkgmiddleware.InternalContext)

	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// All non-health routes require the gateway's shared secret so that
	// callers cannot forge X-Seller-ID headers by bypassing the gateway.
	// Fail-closed (503) when SHIPPING_INTERNAL_TOKEN is unset.
	r.Group(func(protected chi.Router) {
		protected.Use(pkgmiddleware.RequireInternalToken(cfg.InternalToken))

		// Seller routes.
		protected.Mount("/seller/shipments", sellerHandler.Routes())
		protected.Get("/seller/orders/{order_id}/shipment", sellerOrderHandler.GetByOrder)

		// Buyer routes.
		protected.Get("/buyer/orders/{order_id}/shipment", buyerHandler.GetByOrder)
	})

	// gRPC server.
	grpcAddr := ":" + cfg.GRPCPort
	grpcSrv := grpc.NewServer()
	shippingv1.RegisterShippingServiceServer(grpcSrv, grpcserver.NewServer())

	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			slog.Error("failed to listen for gRPC", "addr", grpcAddr, "error", err)
			os.Exit(1)
		}
		slog.Info("starting shipping gRPC server", "addr", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

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
		slog.Info("starting shipping service", "addr", addr)
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

	bgCancel()
	grpcSrv.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("shipping service stopped")
}
