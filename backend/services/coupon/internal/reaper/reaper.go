// Package reaper runs the coupon reservation reaper loop. A
// background goroutine in main.go calls Run; on ticker fire it batches
// up expired pending reservations and decrements the corresponding
// coupons.usage_count so the seat becomes available again.
package reaper

import (
	"context"
	"log/slog"
	"time"

	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

// Reaper periodically expires stale coupon reservations.
type Reaper struct {
	svc      port.CouponUseCase
	interval time.Duration
}

func New(svc port.CouponUseCase, interval time.Duration) *Reaper {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &Reaper{svc: svc, interval: interval}
}

// Run blocks until ctx is cancelled. It runs one expiry sweep per
// tick; exit is silent — cancelled context is the expected shutdown
// path.
func (r *Reaper) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	slog.Info("coupon reaper started", "interval", r.interval)
	for {
		select {
		case <-ctx.Done():
			slog.Info("coupon reaper stopping")
			return
		case <-ticker.C:
			sweepCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			n, err := r.svc.ExpireStaleReservations(sweepCtx)
			cancel()
			if err != nil {
				slog.Warn("coupon reaper sweep failed", "error", err)
				continue
			}
			if n > 0 {
				slog.Debug("coupon reaper sweep done", "expired", n)
			}
		}
	}
}
