package port

import (
	"context"

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

	List(ctx context.Context, status string, limit, offset int) ([]domain.Coupon, int, error)

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

	ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.CouponRedemption, int, error)

	// CountByBuyerAndCoupon is the per-user usage limit check.
	CountByBuyerAndCoupon(ctx context.Context, buyerAuth0ID string, couponID uuid.UUID) (int, error)

	// StatsByCoupon aggregates the admin dashboard numbers (count +
	// total discount applied) — cheaper than scanning the raw rows.
	StatsByCoupon(ctx context.Context, couponID uuid.UUID) (count int, totalDiscount int64, err error)

	// MarkRefunded stamps refunded_at / refunded_reason on a
	// redemption. Returns (true, nil) on the first call and
	// (false, nil) on a replay (row already had refunded_at set).
	MarkRefunded(ctx context.Context, redemptionID uuid.UUID, reason string) (bool, error)
}
