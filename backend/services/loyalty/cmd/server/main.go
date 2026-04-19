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

	"github.com/Riku-KANO/ec-test/pkg/database"
	pkgmiddleware "github.com/Riku-KANO/ec-test/pkg/middleware"
	pkgpubsub "github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/pkg/tracing"
	loyaltyv1 "github.com/Riku-KANO/ec-test/services/loyalty/api/gen/go/loyalty/v1"
	grpcserver "github.com/Riku-KANO/ec-test/services/loyalty/internal/adapter/grpc"
	handler "github.com/Riku-KANO/ec-test/services/loyalty/internal/adapter/http"
	repository "github.com/Riku-KANO/ec-test/services/loyalty/internal/adapter/postgres"
	loyaltypubsub "github.com/Riku-KANO/ec-test/services/loyalty/internal/adapter/pubsub"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/app"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/config"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/reaper"
)

// orderEventsLoyaltySubscription is the name of the subscription
// attached to the order-events topic that the loyalty service reads.
// Declared in infra/scripts/pubsub_emulator_init.sh.
const orderEventsLoyaltySubscription = "order-events-loyalty"

func main() {
	slog.SetDefault(slog.New(tracing.NewSlogHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))))

	tShutdown, _ := tracing.Init(context.Background(), tracing.LoadConfig("loyalty"))
	defer func() { _ = tShutdown(context.Background()) }()

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

	// Pub/Sub publisher: required only to emit PointsEarned etc.
	// Empty PubSubProjectID disables outbound events (local dev).
	var publisher pkgpubsub.Publisher
	if cfg.PubSubProjectID != "" {
		pub, pubErr := pkgpubsub.NewGCPPublisher(context.Background(), cfg.PubSubProjectID)
		if pubErr != nil {
			slog.Warn("failed to create pubsub publisher; loyalty events disabled", "error", pubErr)
		} else {
			publisher = pub
			defer func() {
				if err := pub.Close(); err != nil {
					slog.Warn("failed to close pubsub publisher", "error", err)
				}
			}()
		}
	}

	accountRepo := repository.NewAccountRepository(pool)
	transactionRepo := repository.NewTransactionRepository(pool)
	reservationRepo := repository.NewReservationRepository(pool)

	earnExpiration := time.Duration(cfg.PointExpirationDays) * 24 * time.Hour
	svc := app.NewService(
		accountRepo,
		transactionRepo,
		reservationRepo,
		&database.PoolTxRunner{Pool: pool},
		publisher,
		cfg.EarnRateBps,
		time.Duration(cfg.ReservationTTLMinutes)*time.Minute,
		earnExpiration,
		cfg.PointExpirationBatchSize,
	)

	balanceHandler := handler.NewBalanceHandler(svc)
	internalHandler := handler.NewInternalHandler(svc)
	healthHandler := handler.NewHealthHandler(pool)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(pkgmiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(pkgmiddleware.InternalContext)

	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	if cfg.InternalToken == "" {
		slog.Warn("LOYALTY_INTERNAL_TOKEN is empty — all non-health requests will be rejected with 503")
	}
	r.Group(func(gr chi.Router) {
		gr.Use(pkgmiddleware.RequireInternalToken(cfg.InternalToken))
		gr.Mount("/points", balanceHandler.Routes())
		gr.Mount("/internal", internalHandler.Routes())
	})

	addr := ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      otelhttp.NewHandler(r, "http.server"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// gRPC listener. Only AdjustPoints is currently live; other RPCs
	// fall through to UnimplementedLoyaltyServiceServer. Same
	// internal-token gate as HTTP.
	grpcAddr := ":" + cfg.GRPCPort
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		slog.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}
	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(pkgmiddleware.UnaryInternalTokenInterceptor(cfg.InternalToken)),
	)
	loyaltyv1.RegisterLoyaltyServiceServer(grpcSrv, grpcserver.NewServer(svc))

	// Start the reservation reaper goroutine so stale pending holds
	// auto-release their pending_redemption.
	reaperCtx, reaperCancel := context.WithCancel(context.Background())
	defer reaperCancel()
	rp := reaper.New(svc, 60*time.Second)
	go rp.Run(reaperCtx)

	// Start the points-expiration reaper, gated behind the kill
	// switch so we can deploy the code dark and only enable it after
	// the backfill has been verified in prod.
	if cfg.PointExpirationEnabled {
		expInterval := time.Duration(cfg.PointExpirationIntervalMinutes) * time.Minute
		expTimeout := time.Duration(cfg.PointExpirationTickTimeoutSeconds) * time.Second
		exp := reaper.NewExpirationReaper(svc, expInterval, expTimeout)
		go exp.Run(reaperCtx)
	} else {
		slog.Info("loyalty points expiration disabled via POINT_EXPIRATION_ENABLED=false")
	}

	// Start the order.paid Pub/Sub subscriber as a background goroutine.
	// Empty PubSubProjectID disables it (local dev without emulator).
	subscriberCtx, subscriberCancel := context.WithCancel(context.Background())
	defer subscriberCancel()
	if cfg.PubSubProjectID != "" {
		sub, subErr := pkgpubsub.NewGCPSubscriber(context.Background(), cfg.PubSubProjectID)
		if subErr != nil {
			slog.Warn("failed to create pubsub subscriber; order.paid earn fan-in disabled", "error", subErr)
		} else {
			defer func() {
				if err := sub.Close(); err != nil {
					slog.Warn("failed to close pubsub subscriber", "error", err)
				}
			}()
			orderSub := loyaltypubsub.NewOrderPaidSubscriber(svc, sub)
			go func() {
				if err := orderSub.Run(subscriberCtx, orderEventsLoyaltySubscription); err != nil && err != context.Canceled {
					slog.Error("order.paid subscriber exited with error", "error", err)
				}
			}()
		}
	}

	go func() {
		slog.Info("starting loyalty gRPC server", "addr", grpcAddr)
		if err := grpcSrv.Serve(grpcListener); err != nil {
			slog.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		slog.Info("starting loyalty service", "addr", addr)
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
	subscriberCancel()
	reaperCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("loyalty service stopped")
}
