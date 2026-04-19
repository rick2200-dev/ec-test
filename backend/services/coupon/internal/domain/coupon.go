// Package domain defines the coupon service core types. Amounts are
// int64 minor units (JPY yen) for consistency with order-svc; percent
// discounts are expressed in basis points (1000 = 10%).
package domain

import (
	"time"

	"github.com/google/uuid"
)

// IssuerType enumerates who minted a coupon. MVP only issues platform
// coupons but the column is kept so a future "seller issues own
// coupons" phase doesn't require a backfill.
type IssuerType string

const (
	IssuerTypePlatform IssuerType = "platform"
	IssuerTypeSeller   IssuerType = "seller"
)

// DiscountType enumerates coupon discount shapes. Exactly one of
// DiscountPercentBps / DiscountAmount is populated per coupon row.
type DiscountType string

const (
	DiscountTypePercent     DiscountType = "percent"
	DiscountTypeFixedAmount DiscountType = "fixed_amount"
)

// CouponStatus is the active/revoked lifecycle. Expired coupons stay
// active=true in DB but fail the validity check at Reserve time.
type CouponStatus string

const (
	CouponStatusActive  CouponStatus = "active"
	CouponStatusRevoked CouponStatus = "revoked"
)

// Coupon is the stored coupon definition. Authoritative copy lives in
// coupon_svc.coupons. usage_count is the denormalized counter that the
// Reserve path increments to give pessimistic-ish concurrency without
// holding a row lock for the duration of the checkout.
type Coupon struct {
	ID                 uuid.UUID    `json:"id"`
	Code               string       `json:"code"`
	IssuerType         IssuerType   `json:"issuer_type"`
	IssuerID           *uuid.UUID   `json:"issuer_id,omitempty"`
	DiscountType       DiscountType `json:"discount_type"`
	DiscountPercentBps *int         `json:"discount_percent_bps,omitempty"`
	DiscountAmount     *int64       `json:"discount_amount,omitempty"`
	Currency           string       `json:"currency"`
	MinOrderAmount     int64        `json:"min_order_amount"`
	MaxDiscountAmount  *int64       `json:"max_discount_amount,omitempty"`
	ValidFrom          time.Time    `json:"valid_from"`
	ExpiresAt          *time.Time   `json:"expires_at,omitempty"`
	UsageLimitTotal    *int         `json:"usage_limit_total,omitempty"`
	UsageLimitPerUser  *int         `json:"usage_limit_per_user,omitempty"`
	UsageCount         int          `json:"usage_count"`
	Status             CouponStatus `json:"status"`
	Title              string       `json:"title"`
	Description        string       `json:"description"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

// ReservationStatus is the coupon reservation lifecycle. See
// CouponReservation below for semantics; mirrors the DB CHECK.
type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "pending"
	ReservationStatusCommitted ReservationStatus = "committed"
	ReservationStatusReleased  ReservationStatus = "released"
	ReservationStatusExpired   ReservationStatus = "expired"
)

// CouponReservation is the pending hold created by order-svc at
// checkout. Committed on order.paid; released on checkout failure;
// expired by the reaper after TTL.
//
// ApplicableSellerID is nil for platform (cart-wide) coupons and set
// to the issuing seller's ID for seller-issued coupons. Frozen at
// Reserve time so later admin edits to the coupon row don't reshape
// the attribution that order-svc already baked into per-seller shares.
type CouponReservation struct {
	ID                    uuid.UUID         `json:"id"`
	CouponID              uuid.UUID         `json:"coupon_id"`
	BuyerAuth0ID          string            `json:"buyer_auth0_id"`
	OrderCandidateID      *uuid.UUID        `json:"order_candidate_id,omitempty"`
	StripePaymentIntentID string            `json:"stripe_payment_intent_id,omitempty"`
	DiscountAmount        int64             `json:"discount_amount"`
	Currency              string            `json:"currency"`
	ApplicableSellerID    *uuid.UUID        `json:"applicable_seller_id,omitempty"`
	Status                ReservationStatus `json:"status"`
	ExpiresAt             time.Time         `json:"expires_at"`
	CreatedAt             time.Time         `json:"created_at"`
	CommittedAt           *time.Time        `json:"committed_at,omitempty"`
	ReleasedAt            *time.Time        `json:"released_at,omitempty"`
}

// CouponRedemption is the ledger entry for a committed coupon use.
// (coupon_id, order_id) is UNIQUE so webhook replays insert nothing
// the second time and the handler reports "already_committed".
//
// One redemption row covers the whole cart — Commit writes it keyed
// on the anchor order_id, and the per-seller-order shares are
// tracked by accumulating RefundedAmount as individual orders get
// cancelled. refunded_at is stamped (and the coupon seat released
// via DecrementUsage) only when RefundedAmount reaches DiscountApplied.
type CouponRedemption struct {
	ID              uuid.UUID  `json:"id"`
	CouponID        uuid.UUID  `json:"coupon_id"`
	BuyerAuth0ID    string     `json:"buyer_auth0_id"`
	OrderID         uuid.UUID  `json:"order_id"`
	DiscountApplied int64      `json:"discount_applied"`
	RefundedAmount  int64      `json:"refunded_amount"`
	ReservationID   uuid.UUID  `json:"reservation_id"`
	CommittedAt     time.Time  `json:"committed_at"`
	RefundedAt      *time.Time `json:"refunded_at,omitempty"`
	RefundedReason  string     `json:"refunded_reason,omitempty"`
}

// SellerSubtotal is the per-seller-group subtotal passed in at Reserve
// time so the service can pick the applicable subset. Platform coupons
// apply against the sum; a future seller-scoped coupon applies against
// a single seller's group.
type SellerSubtotal struct {
	SellerID uuid.UUID `json:"seller_id"`
	Subtotal int64     `json:"subtotal"`
}

// ComputeDiscount resolves the concrete yen discount for this coupon
// against the applicable subtotal. Returns 0 when the coupon's
// preconditions (min_order, currency) aren't met — the caller is
// expected to gate on that before reserving.
//
// Semantics:
//   - fixed_amount: min(DiscountAmount, subtotal)
//   - percent:     floor(subtotal * bps / 10_000), capped by MaxDiscountAmount if set
//   - Never exceeds the subtotal (you can't go negative on a single seller's order)
func (c *Coupon) ComputeDiscount(subtotal int64) int64 {
	if subtotal <= 0 {
		return 0
	}
	var discount int64
	switch c.DiscountType {
	case DiscountTypeFixedAmount:
		if c.DiscountAmount == nil {
			return 0
		}
		discount = *c.DiscountAmount
	case DiscountTypePercent:
		if c.DiscountPercentBps == nil {
			return 0
		}
		discount = subtotal * int64(*c.DiscountPercentBps) / 10000
		if c.MaxDiscountAmount != nil && discount > *c.MaxDiscountAmount {
			discount = *c.MaxDiscountAmount
		}
	default:
		return 0
	}
	if discount > subtotal {
		discount = subtotal
	}
	if discount < 0 {
		return 0
	}
	return discount
}

// IsWithinValidity checks valid_from ≤ now ≤ expires_at. Used by
// Reserve to reject expired coupons even if status is still active.
func (c *Coupon) IsWithinValidity(now time.Time) bool {
	if !c.ValidFrom.IsZero() && now.Before(c.ValidFrom) {
		return false
	}
	if c.ExpiresAt != nil && now.After(*c.ExpiresAt) {
		return false
	}
	return true
}
