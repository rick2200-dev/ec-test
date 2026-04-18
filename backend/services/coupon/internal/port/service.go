// Package port defines the coupon service's in/out interfaces.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
)

// CouponUseCase is the inbound interface consumed by HTTP and gRPC
// handlers. Admin, buyer and internal (order-svc) operations all flow
// through it.
type CouponUseCase interface {
	// --- Admin ---

	CreateCoupon(ctx context.Context, input CreateCouponInput) (*domain.Coupon, error)
	ListCoupons(ctx context.Context, filter ListCouponsFilter) ([]domain.Coupon, int, error)
	GetCoupon(ctx context.Context, id uuid.UUID) (*domain.Coupon, error)
	RevokeCoupon(ctx context.Context, id uuid.UUID) (*domain.Coupon, error)
	GetCouponStats(ctx context.Context, id uuid.UUID) (*CouponStats, error)

	// --- Buyer-facing ---

	// PreviewCoupon dry-runs a coupon against a cart without mutating
	// anything. Returns (nil, nil) if the code is unknown; a domain
	// sentinel error for other validation failures.
	PreviewCoupon(ctx context.Context, code, buyerAuth0ID string, sellerSubtotals []domain.SellerSubtotal, currency string) (*PreviewResult, error)

	ListMyRedemptions(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.CouponRedemption, int, error)

	// --- Internal (order-svc) ---

	ReserveCoupon(ctx context.Context, code, buyerAuth0ID string, sellerSubtotals []domain.SellerSubtotal, currency string) (*domain.CouponReservation, error)
	CommitReservation(ctx context.Context, reservationID, orderID uuid.UUID, stripePaymentIntentID string) (*CommitResult, error)
	ReleaseReservation(ctx context.Context, reservationID uuid.UUID, reason string) (bool, error)

	// --- Reaper (called by the reservation reaper goroutine) ---
	ExpireStaleReservations(ctx context.Context) (int, error)

	// --- Cancellation (called by the order.cancelled subscriber) ---

	// RefundRedemption marks a previously committed redemption as
	// refunded and decrements the coupon's usage_count so the seat
	// returns to the pool. Idempotent: a second call with the same
	// (coupon_id, order_id) no-ops and reports already_refunded.
	RefundRedemption(ctx context.Context, couponID, orderID uuid.UUID, reason string) (*RefundResult, error)
}

// RefundResult tells the subscriber whether compensation actually
// wrote a row. AlreadyRefunded=true on a duplicate delivery.
type RefundResult struct {
	RedemptionID     uuid.UUID
	RefundedAmount   int64
	AlreadyRefunded  bool
	Skipped          bool // no matching redemption (order never redeemed this coupon)
}

// CreateCouponInput carries just the fields the admin form owns —
// system fields (id, usage_count, timestamps) are assigned by the
// service.
type CreateCouponInput struct {
	Code                string
	IssuerType          domain.IssuerType
	IssuerID            *uuid.UUID
	DiscountType        domain.DiscountType
	DiscountPercentBps  *int
	DiscountAmount      *int64
	Currency            string
	MinOrderAmount      int64
	MaxDiscountAmount   *int64
	ValidFromSet        bool
	ValidFromUnix       int64 // epoch seconds, honored only when ValidFromSet
	ExpiresAtUnix       *int64
	UsageLimitTotal     *int
	UsageLimitPerUser   *int
	Title               string
	Description         string
}

// ListCouponsFilter is the admin list query.
type ListCouponsFilter struct {
	Status string
	Limit  int
	Offset int
}

// CouponStats is the admin dashboard summary per coupon.
type CouponStats struct {
	CouponID                uuid.UUID `json:"coupon_id"`
	RedeemedCount           int       `json:"redeemed_count"`
	TotalDiscountApplied    int64     `json:"total_discount_applied"`
	PendingReservationCount int       `json:"pending_reservation_count"`
}

// PreviewResult is the buyer-facing dry-run response.
type PreviewResult struct {
	CouponID           uuid.UUID
	Title              string
	Description        string
	DiscountAmount     int64
	ApplicableSellerID *uuid.UUID
	ExpiresAt          *string // ISO8601 when present
}

// CommitResult tells the caller what happened on Commit.
// AlreadyCommitted=true on idempotent replay.
type CommitResult struct {
	RedemptionID     uuid.UUID
	AlreadyCommitted bool
}
