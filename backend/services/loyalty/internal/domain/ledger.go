// Package domain defines the loyalty service's core types. int64 minor
// units (JPY yen) are used throughout so the ledger math never passes
// through a float or decimal library.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Account is a per-buyer point balance with denormalized counters. One
// row per buyer, created lazily on the first earn or reservation.
type Account struct {
	BuyerAuth0ID       string    `json:"buyer_auth0_id"`
	Balance            int64     `json:"balance"`
	PendingRedemption  int64     `json:"pending_redemption"`
	LifetimeEarned     int64     `json:"lifetime_earned"`
	LifetimeRedeemed   int64     `json:"lifetime_redeemed"`
	Version            int       `json:"version"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Spendable returns the balance that can be redeemed right now (i.e.
// total balance minus anything already reserved at checkout).
func (a Account) Spendable() int64 {
	return a.Balance - a.PendingRedemption
}

// TransactionType enumerates the ledger entry kinds. earn and redeem
// are the primary flows; refund/reverse_earn support cancellations;
// adjust is admin-initiated; expire will be used when a point
// expiration job ships in a later phase.
type TransactionType string

const (
	TransactionTypeEarn         TransactionType = "earn"
	TransactionTypeRedeem       TransactionType = "redeem"
	TransactionTypeRefund       TransactionType = "refund"
	TransactionTypeReverseEarn  TransactionType = "reverse_earn"
	TransactionTypeAdjust       TransactionType = "adjust"
	TransactionTypeExpire       TransactionType = "expire"
)

// SourceType enumerates where a ledger row originated. Combined with
// SourceID it forms the idempotency key that prevents duplicate writes
// from webhook replays and Pub/Sub redeliveries.
type SourceType string

const (
	SourceTypeOrderPaid        SourceType = "order_paid"
	SourceTypeReservationCommit SourceType = "reservation_commit"
	SourceTypeOrderCancelled   SourceType = "order_cancelled"
	SourceTypeAdminAdjust      SourceType = "admin_adjust"
	SourceTypeExpirationJob    SourceType = "expiration_job"
)

// Transaction is one row in the append-only point ledger. Amount is
// signed: positive for earn/refund/adjust-credit, negative for
// redeem/reverse_earn/adjust-debit/expire. BalanceAfter is captured at
// write time so history queries don't need to re-accumulate.
type Transaction struct {
	ID             uuid.UUID       `json:"id"`
	BuyerAuth0ID   string          `json:"buyer_auth0_id"`
	Type           TransactionType `json:"type"`
	Amount         int64           `json:"amount"`
	BalanceAfter   int64           `json:"balance_after"`
	SourceType     SourceType      `json:"source_type"`
	SourceID       string          `json:"source_id"`
	ReservationID  *uuid.UUID      `json:"reservation_id,omitempty"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
	Note           string          `json:"note,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ReservationStatus is the lifecycle of a point-redeem reservation.
type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "pending"
	ReservationStatusCommitted ReservationStatus = "committed"
	ReservationStatusReleased  ReservationStatus = "released"
	ReservationStatusExpired   ReservationStatus = "expired"
)

// Reservation is a pending redeem hold created at checkout. Committed
// when the order pays; released on failure; expired by the reaper.
type Reservation struct {
	ID                    uuid.UUID         `json:"id"`
	BuyerAuth0ID          string            `json:"buyer_auth0_id"`
	Amount                int64             `json:"amount"`
	OrderCandidateID      *uuid.UUID        `json:"order_candidate_id,omitempty"`
	StripePaymentIntentID string            `json:"stripe_payment_intent_id,omitempty"`
	Status                ReservationStatus `json:"status"`
	ExpiresAt             time.Time         `json:"expires_at"`
	CreatedAt             time.Time         `json:"created_at"`
	CommittedAt           *time.Time        `json:"committed_at,omitempty"`
	ReleasedAt            *time.Time        `json:"released_at,omitempty"`
}

// ComputeEarnAmount floors the earn on the paid subtotal by the given
// rate in basis points (100 = 1%). Used at order.paid handling.
// paidSubtotal is the product subtotal actually paid — excluding
// shipping fee (pass-through cost) and commission (not buyer-facing).
func ComputeEarnAmount(paidSubtotal int64, earnRateBps int) int64 {
	if paidSubtotal <= 0 || earnRateBps <= 0 {
		return 0
	}
	// Integer-only math to avoid FP rounding: floor(subtotal * bps / 10_000).
	return paidSubtotal * int64(earnRateBps) / 10000
}
