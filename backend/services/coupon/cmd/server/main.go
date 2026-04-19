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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Riku-KANO/ec-test/pkg/database"
	pkgmiddleware "github.com/Riku-KANO/ec-test/pkg/middleware"
	pkgpubsub "github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/pkg/tracing"
	handler "github.com/Riku-KANO/ec-test/services/coupon/internal/adapter/http"
	repository "github.com/Riku-KANO/ec-test/services/coupon/internal/adapter/postgres"
	couponpubsub "github.com/Riku-KANO/ec-test/services/coupon/internal/adapter/pubsub"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/app"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/config"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/reaper"
)

// orderEventsCouponSubscription is the Pub/Sub subscription name on the
// order-events topic that the coupon service reads. Declared in
// infra/scripts/pubsub_emulator_init.sh.
const orderEventsCouponSubscription = "order-events-coupon"

func main() {
	slog.SetDefault(slog.New(tracing.NewSlogHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))))

	tShutdown, _ := tracing.Init(context.Background(), tracing.LoadConfig("coupon"))
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

	var publisher pkgpubsub.Publisher
	if cfg.PubSubProjectID != "" {
		pub, pubErr := pkgpubsub.NewGCPPublisher(context.Background(), cfg.PubSubProjectID)
		if pubErr != nil {
			slog.Warn("failed to create pubsub publisher; coupon events disabled", "error", pubErr)
		} else {
			publisher = pub
			defer func() {
				if err := pub.Close(); err != nil {
					slog.Warn("failed to close pubsub publisher", "error", err)
				}
			}()
		}
	}

	couponRepo := repository.NewCouponRepository(pool)
	reservationRepo := repository.NewReservationRepository(pool)
	redemptionRepo := repository.NewRedemptionRepository(pool)

	reservationTTL := time.Duration(cfg.ReservationTTLMinutes) * time.Minute
	svc := app.NewService(
		couponRepo,
		reservationRepo,
		redemptionRepo,
		&database.PoolTxRunner{Pool: pool},
		publisher,
		reservationTTL,
	)

	adminHandler := handler.NewAdminHandler(svc)
	buyerHandler := handler.NewBuyerHandler(svc)
	sellerHandler := handler.NewSellerHandler(svc)
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
		slog.Warn("COUPON_INTERNAL_TOKEN is empty — all non-health requests will be rejected with 503")
	}
	r.Group(func(gr chi.Router) {
		gr.Use(pkgmiddleware.RequireInternalToken(cfg.InternalToken))
		gr.Mount("/admin/coupons", adminHandler.Routes())
		gr.Mount("/buyer/coupons", buyerHandler.Routes())
		gr.Mount("/seller/coupons", sellerHandler.Routes())
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

	// Start the reservation reaper as a background goroutine. It
	// decrements usage_count on coupons whose reservations expired
	// without a Commit — without this, the final seat of a
	// usage-limited coupon stays held until the admin manually fixes it.
	reaperCtx, reaperCancel := context.WithCancel(context.Background())
	defer reaperCancel()
	rp := reaper.New(svc, 60*time.Second)
	go rp.Run(reaperCtx)

	// Start the order.cancelled subscriber so redemptions on cancelled
	// orders get refunded and the coupon seat returns to the pool.
	// Empty PubSubProjectID disables it (local dev without emulator).
	subscriberCtx, subscriberCancel := context.WithCancel(context.Background())
	defer subscriberCancel()
	if cfg.PubSubProjectID != "" {
		sub, subErr := pkgpubsub.NewGCPSubscriber(context.Background(), cfg.PubSubProjectID)
		if subErr != nil {
			slog.Warn("failed to create pubsub subscriber; order.cancelled fan-in disabled", "error", subErr)
		} else {
			defer func() {
				if err := sub.Close(); err != nil {
					slog.Warn("failed to close pubsub subscriber", "error", err)
				}
			}()
			orderSub := couponpubsub.NewOrderCancelledSubscriber(svc, sub)
			go func() {
				if err := orderSub.Run(subscriberCtx, orderEventsCouponSubscription); err != nil && err != context.Canceled {
					slog.Error("order.cancelled subscriber exited with error", "error", err)
				}
			}()
		}
	}

	go func() {
		slog.Info("starting coupon service", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	slog.Info("shutting down", "signal", sig.String())

	reaperCancel()
	subscriberCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("coupon service stopped")
}
