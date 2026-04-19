package reaper

import (
	"context"
	"log/slog"
	"time"

	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// ExpirationReaper periodically invokes the service's ExpirePoints
// method. Structurally identical to the reservation reaper in this
// package, but with separate interval + tick timeout because
// expiration walks the full per-buyer ledger and runs on a slower
// cadence than the 60-second reservation sweep.
type ExpirationReaper struct {
	svc         port.LoyaltyUseCase
	interval    time.Duration
	tickTimeout time.Duration
}

// NewExpirationReaper constructs the reaper. Zero/negative values for
// interval or tickTimeout fall back to conservative defaults so
// misconfiguration doesn't silently disable the loop.
func NewExpirationReaper(svc port.LoyaltyUseCase, interval, tickTimeout time.Duration) *ExpirationReaper {
	if interval <= 0 {
		interval = 60 * time.Minute
	}
	if tickTimeout <= 0 {
		tickTimeout = 120 * time.Second
	}
	return &ExpirationReaper{svc: svc, interval: interval, tickTimeout: tickTimeout}
}

// Run blocks until ctx is cancelled. Tick errors log-and-continue; a
// single transient failure (DB blip, lock timeout) shouldn't disable
// the safety net for the rest of the process's life.
func (r *ExpirationReaper) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	slog.Info("loyalty expiration reaper started", "interval", r.interval, "tick_timeout", r.tickTimeout)
	for {
		select {
		case <-ctx.Done():
			slog.Info("loyalty expiration reaper stopping")
			return
		case <-ticker.C:
			sweepCtx, cancel := context.WithTimeout(ctx, r.tickTimeout)
			n, err := r.svc.ExpirePoints(sweepCtx)
			cancel()
			if err != nil {
				slog.Warn("loyalty expiration sweep failed", "error", err)
				continue
			}
			if n > 0 {
				slog.Debug("loyalty expiration sweep done", "expired_buckets", n)
			}
		}
	}
}
