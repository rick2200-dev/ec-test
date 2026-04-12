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

	inventoryv1 "github.com/Riku-KANO/ec-test/gen/go/inventory/v1"
	"github.com/Riku-KANO/ec-test/pkg/database"
	pkgmiddleware "github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/config"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/grpcserver"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/handler"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/repository"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/service"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/subscriber"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()

	// Set emulator host if configured so the GCP client library picks it up.
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

	// Long-lived context used to drive the Pub/Sub subscriber goroutine.
	// Cancelled during graceful shutdown so Subscribe() returns cleanly.
	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()

	// Pub/Sub publisher (for inventory.* outgoing events).
	var publisher pubsub.Publisher
	if cfg.PubSubProjectID != "" {
		pub, pubErr := pubsub.NewGCPPublisher(initCtx, cfg.PubSubProjectID)
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

	// Repository
	inventoryRepo := repository.NewInventoryRepository(pool)

	// Service
	inventorySvc := service.NewInventoryService(inventoryRepo, publisher)

	// Pub/Sub subscriber (for incoming order.cancelled events). Created
	// after the service so the subscriber can be wired with a live
	// service pointer in a single step. The inventory service can still
	// run (without stock-release-on-cancel) if the subscriber fails to
	// initialize — the rest of the service keeps serving requests.
	if cfg.PubSubProjectID != "" {
		sub, subErr := pubsub.NewGCPSubscriber(initCtx, cfg.PubSubProjectID)
		if subErr != nil {
			slog.Warn("failed to create pubsub subscriber, order.cancelled events will not be consumed", "error", subErr)
		} else {
			defer func() {
				if err := sub.Close(); err != nil {
					slog.Warn("failed to close pubsub subscriber", "error", err)
				}
			}()
			orderSub := subscriber.NewOrderSubscriber(sub, inventorySvc)
			go func() {
				if err := orderSub.Start(subCtx); err != nil && subCtx.Err() == nil {
					slog.Error("order subscriber error", "error", err)
				}
			}()
			slog.Info("inventory order subscriber started", "project_id", cfg.PubSubProjectID)
		}
	}

	// Handlers
	inventoryHandler := handler.NewInventoryHandler(inventorySvc)
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

	// Inventory endpoints (tenant-scoped)
	r.Mount("/inventory", inventoryHandler.Routes())

	// HTTP server
	addr := ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// gRPC server
	grpcAddr := ":" + cfg.GRPCPort
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		slog.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}
	grpcSrv := grpc.NewServer()
	inventoryv1.RegisterInventoryServiceServer(grpcSrv, grpcserver.NewInventoryServer(inventorySvc))

	// Start gRPC server in a goroutine.
	go func() {
		slog.Info("starting inventory gRPC server", "addr", grpcAddr)
		if err := grpcSrv.Serve(grpcListener); err != nil {
			slog.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	// Start HTTP server in a goroutine.
	go func() {
		slog.Info("starting inventory service", "addr", addr)
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

	// Stop the Pub/Sub subscriber goroutine so Subscribe() returns
	// cleanly before we tear down the client via the deferred Close().
	subCancel()

	grpcSrv.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("inventory service stopped")
}
