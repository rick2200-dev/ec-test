package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SellerSubtotal is the per-seller subtotal sent to the coupon service
// at Reserve time so it can decide which seller (if any) the coupon
// applies to. MVP platform coupons ignore the seller_id but still take
// the full list for min-order validation.
type SellerSubtotal struct {
	SellerID uuid.UUID
	Subtotal int64
}

// CouponReservation is the successful Reserve response from coupon-svc.
type CouponReservation struct {
	ReservationID      uuid.UUID
	CouponID           uuid.UUID
	IssuerType         string     // "platform" | "seller" (MVP always "platform")
	ApplicableSellerID *uuid.UUID // nil for platform (cart-wide)
	DiscountAmount     int64
	Currency           string
	ExpiresAt          time.Time
}

// CouponReserver is the order service's outbound interface to coupon-svc.
// The HTTP adapter at adapter/httpclient/coupon_client.go satisfies it.
type CouponReserver interface {
	Reserve(ctx context.Context, code, buyerAuth0ID string, sellerSubtotals []SellerSubtotal, currency string) (*CouponReservation, error)
	Commit(ctx context.Context, reservationID, orderID uuid.UUID, stripePaymentIntentID string) (*CouponCommitResult, error)
	Release(ctx context.Context, reservationID uuid.UUID, reason string) error
}

// CouponCommitResult mirrors coupon-svc's CommitCouponReservationResponse.
type CouponCommitResult struct {
	RedemptionID     uuid.UUID
	AlreadyCommitted bool
}

// PointReservation is loyalty-svc's ReservePoints response.
type PointReservation struct {
	ReservationID uuid.UUID
	Amount        int64
	ExpiresAt     time.Time
}

// PointCommitResult mirrors loyalty-svc's CommitPointsReservationResponse.
type PointCommitResult struct {
	Redeemed         int64
	Earned           int64
	NewBalance       int64
	AlreadyCommitted bool
}

// PointBalance is the spendable / pending snapshot returned by
// loyalty-svc's GetBalance.
type PointBalance struct {
	Balance           int64
	PendingRedemption int64
	Spendable         int64
}

// PointReserver is the outbound interface to loyalty-svc. The HTTP
// adapter at adapter/httpclient/loyalty_client.go satisfies it.
type PointReserver interface {
	Reserve(ctx context.Context, buyerAuth0ID string, amount int64) (*PointReservation, error)
	Commit(ctx context.Context, reservationID *uuid.UUID, buyerAuth0ID, orderID string, stripePaymentIntentID string, paidSubtotal int64, currency string) (*PointCommitResult, error)
	Release(ctx context.Context, reservationID uuid.UUID, reason string) error
	GetBalance(ctx context.Context, buyerAuth0ID string) (*PointBalance, error)

	// AwardEarn is the no-redeem form used by the webhook path when
	// the order had no point reservation but still earns on purchase.
	// A nil reservation_id tells loyalty-svc to skip the redeem branch.
	AwardEarn(ctx context.Context, buyerAuth0ID, orderID string, paidSubtotal int64, currency string) (*PointCommitResult, error)
}
