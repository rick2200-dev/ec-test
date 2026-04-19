package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
)

// CouponStore abstracts persistence for the coupons table. All methods
// honor the ambient DB transaction (pulled from context).
type CouponStore interface {
	Insert(ctx context.Context, c *domain.Coupon) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Coupon, error)

	// GetByCodeForUpdate locks the matching coupon row with FOR UPDATE
	// for the duration of the current transaction. Used by Reserve so
	// the usage_count check-and-increment is atomic against concurrent
	// reservations of the last available seat.
	GetByCodeForUpdate(ctx context.Context, issuerType domain.IssuerType, issuerID *uuid.UUID, code string) (*domain.Coupon, error)

	// List returns a paginated view. issuerType / issuerID narrow the
	// result set: pass a seller UUID to list that seller's coupons;
	// leave both empty to list everything (admin view).
	List(ctx context.Context, filter ListCouponsFilter) ([]domain.Coupon, int, error)

	// IncrementUsageIfBelowLimit atomically bumps usage_count if it is
	// still below usage_limit_total (or that limit is NULL). Returns
	// ErrUsageLimitExceeded when the limit was already reached. Callers
	// hold the coupon row lock (via GetByCodeForUpdate) so the race
	// against a concurrent redeem is resolved by Postgres.
	IncrementUsageIfBelowLimit(ctx context.Context, id uuid.UUID) error

	// DecrementUsage reverses IncrementUsageIfBelowLimit on release or
	// expiry. Never goes below zero (CHECK constraint enforces this).
	DecrementUsage(ctx context.Context, id uuid.UUID) error

	// SetStatus updates status + updated_at. Used by RevokeCoupon.
	SetStatus(ctx context.Context, id uuid.UUID, status domain.CouponStatus) error

	// Update applies the editable-field subset of a coupon row. The
	// columns that identify the deal (code, issuer, discount type +
	// amount, currency, valid_from) are not touched; nil pointers
	// clear their nullable columns. Returns the updated row.
	Update(ctx context.Context, id uuid.UUID, patch CouponPatch) (*domain.Coupon, error)
}

// CouponPatch is the repo-level view of an update: every writable
// column, with nil representing "clear" on nullable columns and the
// zero value on non-nullable columns meaning "set to zero". The app
// layer is responsible for validating before calling.
type CouponPatch struct {
	Title             string
	Description       string
	MinOrderAmount    int64
	MaxDiscountAmount *int64
	ExpiresAt         *time.Time
	UsageLimitTotal   *int
	UsageLimitPerUser *int
}

// ReservationStore abstracts coupon_reservations.
type ReservationStore interface {
	Insert(ctx context.Context, r *domain.CouponReservation) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CouponReservation, error)

	// UpdateStatus transitions a reservation. Returns (false, nil) when
	// the row is not in the expected current status — app translates
	// that to ErrReservationAlreadyTerminal so callers can distinguish
	// the race from a true missing-row error.
	UpdateStatus(ctx context.Context, id uuid.UUID, expect, next domain.ReservationStatus) (bool, error)

	// ClaimExpired marks at most limit pending rows whose expires_at is
	// in the past as expired in one round-trip. Returns the coupon IDs
	// of affected rows so the reaper can decrement their usage_count.
	ClaimExpired(ctx context.Context, limit int) ([]ReservationExpiry, error)

	// CountPendingByCoupon is the admin stats "pending_reservation_count".
	CountPendingByCoupon(ctx context.Context, couponID uuid.UUID) (int, error)
}

// ReservationExpiry is the reaper's per-row result.
type ReservationExpiry struct {
	ReservationID uuid.UUID
	CouponID      uuid.UUID
}

// RedemptionStore abstracts coupon_redemptions (the finalized,
// immutable ledger).
type RedemptionStore interface {
	// Insert writes a new redemption. UNIQUE (coupon_id, order_id)
	// surfaces as ErrDuplicateRedemption — callers treat that as
	// "already committed" and return the existing row.
	Insert(ctx context.Context, r *domain.CouponRedemption) error

	GetByCouponAndOrder(ctx context.Context, couponID, orderID uuid.UUID) (*domain.CouponRedemption, error)

	// GetByReservationID loads the redemption that was written when
	// the reservation committed. Used by the cancellation subscriber
	// to find the cart's redemption from any seller order's
	// cancellation event — not just the anchor order's.
	GetByReservationID(ctx context.Context, reservationID uuid.UUID) (*domain.CouponRedemption, error)

	ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.CouponRedemption, int, error)

	// CountByBuyerAndCoupon is the per-user usage limit check.
	CountByBuyerAndCoupon(ctx context.Context, buyerAuth0ID string, couponID uuid.UUID) (int, error)

	// StatsByCoupon aggregates the admin dashboard numbers (count +
	// total discount applied) — cheaper than scanning the raw rows.
	StatsByCoupon(ctx context.Context, couponID uuid.UUID) (count int, totalDiscount int64, err error)

	// ApplyPartialRefund credits `share` worth of refund against the
	// redemption. Called by the app layer AFTER a successful
	// InsertRefundEvent so the two steps together are per-order
	// idempotent.
	//
	// Semantics:
	//   - applied:       amount actually added (capped at the unrefunded
	//                    remainder so a wrong-sized share is clipped
	//                    safely, not rejected)
	//   - fullyRefunded: true on the transition that takes the row to
	//                    discount_applied (→ caller releases the seat)
	//   - alreadyFull:   true when the row was already fully refunded
	//                    before this call (replay, returns without
	//                    touching anything)
	ApplyPartialRefund(ctx context.Context, redemptionID uuid.UUID, share int64, reason string) (applied int64, fullyRefunded, alreadyFull bool, err error)

	// InsertRefundEvent records that a specific cancelled order has
	// contributed its share to the redemption's refunded_amount.
	// UNIQUE (redemption_id, cancelled_order_id) makes the first
	// INSERT the arbiter — a redelivered order.cancelled event for
	// the same order hits the constraint and the caller returns
	// AlreadyRefunded without re-applying the share.
	//
	// Returns ErrDuplicateRefundEvent when the (redemption, order)
	// pair has already been recorded.
	InsertRefundEvent(ctx context.Context, redemptionID, cancelledOrderID uuid.UUID, shareApplied int64, reason string) error
}
