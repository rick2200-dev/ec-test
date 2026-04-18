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
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"

	orderv1 "github.com/Riku-KANO/ec-test/services/order/api/gen/go/order/v1"
	"github.com/Riku-KANO/ec-test/pkg/database"
	pkgmiddleware "github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/pkg/tracing"
	"github.com/Riku-KANO/ec-test/services/order/internal/adapter/grpc"
	"github.com/Riku-KANO/ec-test/services/order/internal/adapter/grpcclient"
	"github.com/Riku-KANO/ec-test/services/order/internal/adapter/http"
	"github.com/Riku-KANO/ec-test/services/order/internal/adapter/httpclient"
	"github.com/Riku-KANO/ec-test/services/order/internal/adapter/postgres"
	stripeClient "github.com/Riku-KANO/ec-test/services/order/internal/adapter/stripe"
	subscriber "github.com/Riku-KANO/ec-test/services/order/internal/adapter/pubsub"
	"github.com/Riku-KANO/ec-test/services/order/internal/app"
	"github.com/Riku-KANO/ec-test/services/order/internal/cancellation"
	"github.com/Riku-KANO/ec-test/services/order/internal/config"
)

func main() {
	slog.SetDefault(slog.New(tracing.NewSlogHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))))

	tShutdown, _ := tracing.Init(context.Background(), tracing.LoadConfig("order"))
	defer func() { _ = tShutdown(context.Background()) }()

	cfg := config.Load()

	// Background context for long-lived goroutines (pub/sub subscribers).
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Pub/Sub emulator host (must be set before subscriber is created).
	if cfg.PubSubEmulatorHost != "" {
		if err := os.Setenv("PUBSUB_EMULATOR_HOST", cfg.PubSubEmulatorHost); err != nil {
			slog.Warn("failed to set PUBSUB_EMULATOR_HOST", "error", err)
		}
	}

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

	// Pub/Sub publisher
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

	// Repositories
	orderRepo := repository.NewOrderRepository(pool)
	commissionRepo := repository.NewCommissionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	// Pub/Sub subscriber — listens for shipping events to update order status.
	if cfg.PubSubProjectID != "" {
		sub, subErr := pubsub.NewGCPSubscriber(bgCtx, cfg.PubSubProjectID)
		if subErr != nil {
			slog.Warn("failed to create pubsub subscriber, shipping status sync disabled", "error", subErr)
		} else {
			defer func() {
				if err := sub.Close(); err != nil {
					slog.Warn("failed to close pubsub subscriber", "error", err)
				}
			}()
			shippingSub := subscriber.NewShippingSubscriber(sub, orderRepo)
			go func() {
				if err := shippingSub.Start(bgCtx); err != nil {
					slog.Error("shipping subscriber error", "error", err)
				}
			}()
			slog.Info("shipping event subscriber started")
		}
	} else {
		slog.Info("PUBSUB_PROJECT_ID not set, shipping status sync disabled")
	}

	// Buyer subscription client (for checking free shipping eligibility).
	// Talks to the dedicated subscription service over gRPC.
	buyerSubClient, err := grpcclient.NewBuyerSubscriptionClient(cfg.SubscriptionServiceGRPCAddr, cfg.SubscriptionInternalToken)
	if err != nil {
		slog.Error("failed to create subscription client", "error", err)
		os.Exit(1)
	}
	defer func() { _ = buyerSubClient.Close() }()

	// Catalog gRPC client: resolves sku -> product_id at checkout so
	// order_svc no longer needs to read catalog_svc directly.
	catalogClient, err := grpcclient.NewCatalogClient(cfg.CatalogServiceGRPCAddr, cfg.CatalogInternalToken)
	if err != nil {
		slog.Error("failed to create catalog client", "error", err)
		os.Exit(1)
	}
	defer func() { _ = catalogClient.Close() }()

	// Auth HTTP client: resolves seller_id -> name at checkout.
	sellerClient := httpclient.NewSellerClient(cfg.AuthServiceURL, cfg.AuthInternalToken)

	// Coupon / loyalty HTTP clients. The feature flags decide whether
	// the service wires them into CreateCheckout — a disabled feature
	// with a configured URL still constructs the client so the
	// zero-value paths stay compilable.
	couponClient := httpclient.NewCouponClient(cfg.CouponServiceURL, cfg.CouponInternalToken)
	loyaltyClient := httpclient.NewLoyaltyClient(cfg.LoyaltyServiceURL, cfg.LoyaltyInternalToken)

	// Service
	orderSvc := app.NewOrderService(orderRepo, commissionRepo, payoutRepo, sc, publisher, buyerSubClient, sellerClient, catalogClient, cfg.DefaultShippingFee).
		WithCouponReserver(couponClient, cfg.EnableCoupons).
		WithPointReserver(loyaltyClient, cfg.EnableLoyalty)

	// Cancellation bounded context — see internal/cancellation/doc.go.
	// Wired in parallel with (not nested under) the order service so
	// the cancellation package owns its own repository, HTTP handler,
	// and error codes without reaching into internal/handler.
	cancellationRepo := cancellation.NewRepository(pool)
	cancellationSvc := cancellation.NewService(orderRepo, payoutRepo, cancellationRepo, sc, publisher)
	cancellationHandler := cancellation.NewHTTPHandler(cancellationSvc)

	// Handlers
	orderHandler := handler.NewOrderHandler(orderSvc)
	commissionHandler := handler.NewCommissionHandler(orderSvc)
	payoutHandler := handler.NewPayoutHandler(orderSvc)
	webhookHandler := handler.NewWebhookHandler(orderSvc, cfg.StripeWebhookSecret)
	internalHandler := handler.NewInternalHandler(orderSvc, cfg.InternalToken)
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

	// Stripe webhooks (no auth required, validated by signature)
	r.Post("/webhooks/stripe", webhookHandler.HandleStripeWebhook)

	// Order endpoints (tenant-scoped).
	// The cancellation package attaches its buyer-facing /{id}/cancellation-request
	// routes onto the same subrouter before mounting, because chi does not
	// allow two Mount() calls at the same prefix. See
	// cancellation.HTTPHandler.AttachOrderScopedRoutes.
	ordersRouter := orderHandler.Routes()
	cancellationHandler.AttachOrderScopedRoutes(ordersRouter)
	r.Mount("/orders", ordersRouter)

	// Seller-facing cancellation endpoints at their own top-level prefix.
	r.Mount("/cancellation-requests", cancellationHandler.SellerRoutes())

	// Commission endpoints (tenant-scoped)
	r.Mount("/commissions", commissionHandler.Routes())

	// Payout endpoints (tenant-scoped)
	r.Mount("/payouts", payoutHandler.Routes())

	// Intra-cluster endpoints (cart service, etc.)
	r.Mount("/internal", internalHandler.Routes())

	// gRPC server
	grpcAddr := ":" + cfg.GRPCPort
	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	orderv1.RegisterOrderServiceServer(grpcSrv, grpcserver.NewServer(orderSvc))

	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			slog.Error("failed to listen for gRPC", "addr", grpcAddr, "error", err)
			os.Exit(1)
		}
		slog.Info("starting order gRPC server", "addr", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	// HTTP server
	addr := ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      otelhttp.NewHandler(r, "http.server"),
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

	// Stop Pub/Sub subscribers before closing servers.
	bgCancel()

	grpcSrv.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("order service stopped")
}
