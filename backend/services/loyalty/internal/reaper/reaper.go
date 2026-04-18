// Package reaper runs the loyalty reservation reaper loop. A background
// goroutine in main.go calls Run; on ticker fire it batches up
// expired pending reservations and decrements each affected account's
// pending_redemption so the held points return to spendable balance.
package reaper

import (
	"context"
	"log/slog"
	"time"

	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

type Reaper struct {
	svc      port.LoyaltyUseCase
	interval time.Duration
}

func New(svc port.LoyaltyUseCase, interval time.Duration) *Reaper {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &Reaper{svc: svc, interval: interval}
}

// Run blocks until ctx is cancelled. Sweep errors are logged but do
// not stop the loop — a transient DB blip shouldn't disable the
// pending-release safety net for the rest of the process's life.
func (r *Reaper) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	slog.Info("loyalty reaper started", "interval", r.interval)
	for {
		select {
		case <-ctx.Done():
			slog.Info("loyalty reaper stopping")
			return
		case <-ticker.C:
			sweepCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			n, err := r.svc.ExpireStaleReservations(sweepCtx)
			cancel()
			if err != nil {
				slog.Warn("loyalty reaper sweep failed", "error", err)
				continue
			}
			if n > 0 {
				slog.Debug("loyalty reaper sweep done", "expired", n)
			}
		}
	}
}
