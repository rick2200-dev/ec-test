// Package port holds the loyalty service's inbound/outbound interfaces.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
)

// LoyaltyUseCase is the inbound interface consumed by HTTP and gRPC
// handlers. All side effects (ledger writes, publishing) happen behind
// this boundary.
type LoyaltyUseCase interface {
	// Buyer-facing reads.
	GetBalance(ctx context.Context, buyerAuth0ID string) (*domain.Account, error)
	ListTransactions(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.Transaction, int, error)

	// --- Internal (called by order-svc) ---

	// AwardEarn is the pure-earn form used when the buyer did not
	// redeem points at checkout. Idempotent via
	// (order_paid, orderID, earn).
	AwardEarn(ctx context.Context, buyerAuth0ID, orderID string, paidSubtotal int64, currency string) (*domain.Transaction, error)

	// ReservePoints holds `amount` points in pending_redemption for
	// the buyer. Returns ErrInsufficientPoints when balance -
	// pending < amount.
	ReservePoints(ctx context.Context, buyerAuth0ID string, amount int64) (*domain.Reservation, error)

	// CommitPointsReservation burns the reserved points (writes a
	// redeem ledger row) and atomically appends an earn row for
	// `paidSubtotal × earnRateBps / 10_000`. Both ledger writes are
	// idempotent via their own (source_type, source_id, type) key.
	CommitPointsReservation(ctx context.Context, reservationID uuid.UUID, orderID, stripePaymentIntentID string, paidSubtotal int64, currency string) (*CommitPointsResult, error)

	// ReleasePointsReservation returns pending points to the
	// buyer's spendable balance without ever writing a ledger row.
	// No-op on already-terminal reservations.
	ReleasePointsReservation(ctx context.Context, reservationID uuid.UUID, reason string) (bool, error)

	// ExpireStaleReservations is the reaper entrypoint — releases
	// pending reservations whose TTL has elapsed.
	ExpireStaleReservations(ctx context.Context) (int, error)

	// ApplyCancellation reverses the effects of a paid order on
	// cancellation: credits back any redeemed points (refund) and
	// reverses previously earned points (reverse_earn). Idempotent
	// via (order_cancelled, orderID, refund) and (order_cancelled,
	// orderID, reverse_earn).
	ApplyCancellation(ctx context.Context, input CancelPointsInput) (*CancelPointsResult, error)

	// AdjustPoints applies a signed admin-initiated balance delta.
	// Positive credits, negative debits. reason is required (audit).
	// adminUserID is recorded on the ledger row for traceability.
	// A debit that would push spendable negative returns
	// domain.ErrInsufficientPoints.
	AdjustPoints(ctx context.Context, input AdjustPointsInput) (*AdjustPointsResult, error)

	// ExpirePoints is the expiration reaper entrypoint. It scans for
	// buyers with unprocessed expired earn rows, rebuilds per-buyer
	// FIFO earn-bucket state by walking the chronological ledger, and
	// writes one `expire` row per still-unconsumed expired bucket.
	// Returns the total number of expire rows written this pass.
	// Safe to re-run: each expire row is idempotent via
	// (expiration_job, earn_txn_id, expire).
	ExpirePoints(ctx context.Context) (int, error)
}

// CancelPointsInput is the payload extracted from OrderCancelled for
// the loyalty subscriber. Both amount fields may be zero — e.g. an
// order that earned points but did not redeem, or vice versa.
type CancelPointsInput struct {
	BuyerAuth0ID        string
	OrderID             string
	PointDiscountAmount int64 // the share of this order that was paid with points
	PointsEarned        int64 // the earn credited on order.paid
}

// CancelPointsResult reports whether each reversal actually wrote a
// row (false = already compensated on a previous delivery).
type CancelPointsResult struct {
	Refunded   int64
	Reversed   int64
	NewBalance int64
}

// CommitPointsResult is the return value of CommitPointsReservation.
// Redeemed is the amount that was actually burned (0 on an already-
// committed replay because the ledger row existed). Earned is the
// fresh accrual from paidSubtotal × earnRateBps / 10_000.
type CommitPointsResult struct {
	Redeemed         int64
	Earned           int64
	NewBalance       int64
	AlreadyCommitted bool
}

// AdjustPointsInput carries a signed manual adjustment initiated by an
// admin. Amount is non-zero; reason is required; AdminUserID is the
// caller's identity, persisted on the ledger row for audit.
type AdjustPointsInput struct {
	BuyerAuth0ID string
	Amount       int64
	Reason       string
	AdminUserID  string
}

// AdjustPointsResult returns the ledger row that was written and the
// buyer's post-adjustment balance.
type AdjustPointsResult struct {
	Transaction *domain.Transaction
	NewBalance  int64
}
