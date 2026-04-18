package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

// Tests in this file pin the multi-seller partial-refund behavior
// that the "cart-wide coupon on order-level cancellation" finding
// surfaced. Cart → 3 seller orders → coupon discount N, split 3 ways.
// Each order's cancellation refunds its share of the single cart-wide
// redemption; the coupon seat (usage_count) is released exactly once,
// on the transition to fully refunded.

type fakeCouponStore struct {
	coupons       map[uuid.UUID]*domain.Coupon
	decrementLog  []uuid.UUID
}

func (f *fakeCouponStore) Insert(ctx context.Context, c *domain.Coupon) error { return nil }
func (f *fakeCouponStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Coupon, error) {
	return f.coupons[id], nil
}
func (f *fakeCouponStore) GetByCodeForUpdate(ctx context.Context, issuerType domain.IssuerType, issuerID *uuid.UUID, code string) (*domain.Coupon, error) {
	return nil, nil
}
func (f *fakeCouponStore) List(ctx context.Context, status string, limit, offset int) ([]domain.Coupon, int, error) {
	return nil, 0, nil
}
func (f *fakeCouponStore) IncrementUsageIfBelowLimit(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (f *fakeCouponStore) DecrementUsage(ctx context.Context, id uuid.UUID) error {
	f.decrementLog = append(f.decrementLog, id)
	if c, ok := f.coupons[id]; ok && c.UsageCount > 0 {
		c.UsageCount--
	}
	return nil
}
func (f *fakeCouponStore) SetStatus(ctx context.Context, id uuid.UUID, status domain.CouponStatus) error {
	return nil
}

type fakeRedemptionStore struct {
	redemption      *domain.CouponRedemption
	refundedAmount  int64
	fullyRefunded   bool
	applyCallCount  int
}

func (f *fakeRedemptionStore) Insert(ctx context.Context, r *domain.CouponRedemption) error {
	return nil
}
func (f *fakeRedemptionStore) GetByCouponAndOrder(ctx context.Context, couponID, orderID uuid.UUID) (*domain.CouponRedemption, error) {
	return nil, nil
}
func (f *fakeRedemptionStore) GetByReservationID(ctx context.Context, reservationID uuid.UUID) (*domain.CouponRedemption, error) {
	if f.redemption == nil {
		return nil, nil
	}
	cp := *f.redemption
	cp.RefundedAmount = f.refundedAmount
	if f.fullyRefunded {
		now := time.Now()
		cp.RefundedAt = &now
	}
	return &cp, nil
}
func (f *fakeRedemptionStore) ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.CouponRedemption, int, error) {
	return nil, 0, nil
}
func (f *fakeRedemptionStore) CountByBuyerAndCoupon(ctx context.Context, buyerAuth0ID string, couponID uuid.UUID) (int, error) {
	return 0, nil
}
func (f *fakeRedemptionStore) StatsByCoupon(ctx context.Context, couponID uuid.UUID) (int, int64, error) {
	return 0, 0, nil
}
func (f *fakeRedemptionStore) ApplyPartialRefund(ctx context.Context, redemptionID uuid.UUID, share int64, reason string) (int64, bool, bool, error) {
	f.applyCallCount++
	if f.fullyRefunded {
		return 0, false, true, nil
	}
	remaining := f.redemption.DiscountApplied - f.refundedAmount
	if remaining <= 0 {
		return 0, false, true, nil
	}
	applied := share
	if applied > remaining {
		applied = remaining
	}
	f.refundedAmount += applied
	justFinished := f.refundedAmount >= f.redemption.DiscountApplied
	if justFinished {
		f.fullyRefunded = true
	}
	return applied, justFinished, false, nil
}

type immediateTx struct{}

func (immediateTx) RunTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type stubReservationStore struct{}

func (stubReservationStore) Insert(ctx context.Context, r *domain.CouponReservation) error {
	return nil
}
func (stubReservationStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.CouponReservation, error) {
	return nil, nil
}
func (stubReservationStore) UpdateStatus(ctx context.Context, id uuid.UUID, expect, next domain.ReservationStatus) (bool, error) {
	return false, nil
}
func (stubReservationStore) ClaimExpired(ctx context.Context, limit int) ([]port.ReservationExpiry, error) {
	return nil, nil
}
func (stubReservationStore) CountPendingByCoupon(ctx context.Context, couponID uuid.UUID) (int, error) {
	return 0, nil
}

func setupService() (*Service, *fakeCouponStore, *fakeRedemptionStore, uuid.UUID, uuid.UUID) {
	couponID := uuid.New()
	reservationID := uuid.New()
	coupons := &fakeCouponStore{
		coupons: map[uuid.UUID]*domain.Coupon{
			couponID: {ID: couponID, UsageCount: 1},
		},
	}
	redemptions := &fakeRedemptionStore{
		redemption: &domain.CouponRedemption{
			ID:              uuid.New(),
			CouponID:        couponID,
			BuyerAuth0ID:    "auth0|buyer",
			OrderID:         uuid.New(),
			DiscountApplied: 1000,
			ReservationID:   reservationID,
			CommittedAt:     time.Now(),
		},
	}
	svc := NewService(coupons, stubReservationStore{}, redemptions, immediateTx{}, nil, 30*time.Minute)
	return svc, coupons, redemptions, reservationID, couponID
}

// TestRefundRedemption_MultiSellerPartialRefunds pins the core fix:
// three seller orders each with a 333/333/334 share cancel one by
// one; the first two refunds accumulate without releasing the seat,
// the third completes the refund and releases it exactly once.
func TestRefundRedemption_MultiSellerPartialRefunds(t *testing.T) {
	svc, coupons, red, reservationID, couponID := setupService()

	// Cancel first order: 333 share, partial.
	res, err := svc.RefundRedemption(context.Background(), reservationID, uuid.New(), 333, "buyer_cancel")
	if err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	if res.AppliedAmount != 333 {
		t.Errorf("applied = %d, want 333", res.AppliedAmount)
	}
	if res.FullyRefunded {
		t.Error("first cancel shouldn't fully refund")
	}
	if len(coupons.decrementLog) != 0 {
		t.Error("seat released too early")
	}

	// Cancel second order: 333 share, partial.
	res, err = svc.RefundRedemption(context.Background(), reservationID, uuid.New(), 333, "buyer_cancel")
	if err != nil {
		t.Fatalf("second cancel: %v", err)
	}
	if res.AppliedAmount != 333 {
		t.Errorf("applied = %d, want 333", res.AppliedAmount)
	}
	if res.FullyRefunded {
		t.Error("second cancel shouldn't fully refund")
	}
	if len(coupons.decrementLog) != 0 {
		t.Error("seat released too early after second cancel")
	}

	// Cancel third order: 334 share, completes refund.
	res, err = svc.RefundRedemption(context.Background(), reservationID, uuid.New(), 334, "buyer_cancel")
	if err != nil {
		t.Fatalf("third cancel: %v", err)
	}
	if res.AppliedAmount != 334 {
		t.Errorf("applied = %d, want 334", res.AppliedAmount)
	}
	if !res.FullyRefunded {
		t.Error("third cancel must fully refund")
	}
	if len(coupons.decrementLog) != 1 {
		t.Errorf("seat should release exactly once, got %d decrements", len(coupons.decrementLog))
	}
	if coupons.decrementLog[0] != couponID {
		t.Errorf("wrong coupon released: got %v want %v", coupons.decrementLog[0], couponID)
	}

	// Redelivered cancel after full refund: idempotent no-op.
	res, err = svc.RefundRedemption(context.Background(), reservationID, uuid.New(), 334, "buyer_cancel")
	if err != nil {
		t.Fatalf("replay cancel: %v", err)
	}
	if !res.AlreadyRefunded {
		t.Error("replay should report AlreadyRefunded")
	}
	if len(coupons.decrementLog) != 1 {
		t.Error("seat released twice on replay — idempotency broken")
	}
	_ = red // ensure the fake redemption ref stays in scope
}

// TestRefundRedemption_SingleOrderCartFullRefund covers the trivial
// single-seller case: one cancel = full refund = seat released on
// first call.
func TestRefundRedemption_SingleOrderCartFullRefund(t *testing.T) {
	svc, coupons, _, reservationID, _ := setupService()
	res, err := svc.RefundRedemption(context.Background(), reservationID, uuid.New(), 1000, "cancel")
	if err != nil {
		t.Fatalf("RefundRedemption: %v", err)
	}
	if !res.FullyRefunded || res.AppliedAmount != 1000 {
		t.Errorf("expected full refund, got applied=%d fully=%v", res.AppliedAmount, res.FullyRefunded)
	}
	if len(coupons.decrementLog) != 1 {
		t.Errorf("seat released %d times, want 1", len(coupons.decrementLog))
	}
}

// TestRefundRedemption_NilReservationSkips exercises the short-circuit
// for non-coupon cancellations (reservation_id empty on the event).
func TestRefundRedemption_NilReservationSkips(t *testing.T) {
	svc, _, _, _, _ := setupService()
	res, err := svc.RefundRedemption(context.Background(), uuid.Nil, uuid.New(), 500, "cancel")
	if err != nil {
		t.Fatalf("RefundRedemption: %v", err)
	}
	if !res.Skipped {
		t.Errorf("expected Skipped=true, got %+v", res)
	}
}

// TestRefundRedemption_MissingRedemptionSkips covers the case where
// the reservation existed but was never committed (payment failed
// before Commit). Cancellation should Skip, not error.
func TestRefundRedemption_MissingRedemptionSkips(t *testing.T) {
	svc, _, red, reservationID, _ := setupService()
	red.redemption = nil // no redemption row
	res, err := svc.RefundRedemption(context.Background(), reservationID, uuid.New(), 500, "cancel")
	if err != nil {
		t.Fatalf("RefundRedemption: %v", err)
	}
	if !res.Skipped {
		t.Errorf("expected Skipped=true, got %+v", res)
	}
}

// TestRefundRedemption_ShareClampedToRemaining asserts that a caller
// sending too-large a share (e.g. rounding bug upstream) doesn't
// over-credit the redemption. The repo caps it at discount_applied;
// the app reports the actually-applied amount.
func TestRefundRedemption_ShareClampedToRemaining(t *testing.T) {
	svc, _, _, reservationID, _ := setupService()
	res, err := svc.RefundRedemption(context.Background(), reservationID, uuid.New(), 5000, "cancel")
	if err != nil {
		t.Fatalf("RefundRedemption: %v", err)
	}
	if res.AppliedAmount != 1000 {
		t.Errorf("applied = %d, want 1000 (clamped to discount_applied)", res.AppliedAmount)
	}
	if !res.FullyRefunded {
		t.Error("clamped full refund should report FullyRefunded=true")
	}
}

// sanity: compile-time check that port.RefundResult has the fields we assert.
var _ = func() error {
	var r port.RefundResult
	_ = r.RedemptionID
	_ = r.AppliedAmount
	_ = r.FullyRefunded
	_ = r.AlreadyRefunded
	_ = r.Skipped
	return errors.New("unused")
}
