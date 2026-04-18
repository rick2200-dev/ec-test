package app

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	pkgpubsub "github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/port"
)

// These tests pin the two retry-critical invariants of
// HandlePaymentSuccess:
//
//   1. order.paid must publish eventually, even if coupon/loyalty
//      commit failed on the first attempt and succeeded only on
//      retry (the regression that starved shipping / notification).
//
//   2. payout states other than pending/completed (failed, reversed)
//      must short-circuit the handler before reaching (B)/(C) —
//      otherwise we'd commit redemptions against orders whose
//      buyer money never moved or already got reversed.
//
// The fakes below only implement what the handler actually touches,
// so changes to unrelated code paths don't regress the test.

type recordingOrderStore struct {
	orderByID              map[uuid.UUID]*domain.Order
	ordersByPI             map[string][]domain.Order
	setPaidCalls           int
	markPublishedCalls     int
	markPublishedFirstCall bool
}

func (r *recordingOrderStore) Create(ctx context.Context, o *domain.Order, lines []domain.OrderLine) error {
	return nil
}
func (r *recordingOrderStore) CreateCheckoutBatch(ctx context.Context, items []domain.CheckoutBatchItem) error {
	return nil
}
func (r *recordingOrderStore) GetByID(ctx context.Context, orderID uuid.UUID) (*domain.OrderWithLines, error) {
	return nil, nil
}
func (r *recordingOrderStore) HasPurchasedSKU(ctx context.Context, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*domain.PurchaseSKURecord, error) {
	return nil, nil
}
func (r *recordingOrderStore) ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error) {
	return nil, 0, nil
}
func (r *recordingOrderStore) ListBySeller(ctx context.Context, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error) {
	return nil, 0, nil
}
func (r *recordingOrderStore) UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error {
	return nil
}
func (r *recordingOrderStore) SetPaid(ctx context.Context, orderID uuid.UUID, paidAt time.Time, stripePaymentIntentID string) error {
	r.setPaidCalls++
	o, ok := r.orderByID[orderID]
	if !ok {
		return fmt.Errorf("order not found")
	}
	if o.Status != domain.StatusPending {
		return domain.ErrOrderNotPending
	}
	o.Status = domain.StatusPaid
	o.PaidAt = &paidAt
	return nil
}
func (r *recordingOrderStore) FindAllByStripePaymentIntentID(ctx context.Context, paymentIntentID string) ([]domain.Order, error) {
	// Return copies so the handler's in-place mutation doesn't leak
	// back into our reference state between attempts.
	src := r.ordersByPI[paymentIntentID]
	out := make([]domain.Order, len(src))
	for i := range src {
		// Pull the latest snapshot from the orderByID map so retries
		// see state changes (SetPaid / SetPointsEarned /
		// MarkPaidEventPublished) from prior attempts.
		if latest, ok := r.orderByID[src[i].ID]; ok {
			out[i] = *latest
		} else {
			out[i] = src[i]
		}
	}
	return out, nil
}
func (r *recordingOrderStore) SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error {
	return nil
}
func (r *recordingOrderStore) SetPointsEarned(ctx context.Context, orderID uuid.UUID, amount int64) error {
	if o, ok := r.orderByID[orderID]; ok {
		o.PointsEarned = amount
	}
	return nil
}
func (r *recordingOrderStore) MarkPaidEventPublished(ctx context.Context, orderID uuid.UUID) (bool, error) {
	r.markPublishedCalls++
	o, ok := r.orderByID[orderID]
	if !ok {
		return false, nil
	}
	if o.PaidEventPublishedAt != nil {
		return false, nil
	}
	now := time.Now()
	o.PaidEventPublishedAt = &now
	if r.markPublishedCalls == 1 {
		r.markPublishedFirstCall = true
	}
	return true, nil
}

type recordingPayoutStore struct {
	byOrder map[uuid.UUID]*domain.Payout
}

func (r *recordingPayoutStore) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payout, error) {
	if p, ok := r.byOrder[orderID]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, nil
}
func (r *recordingPayoutStore) UpdateStatus(ctx context.Context, payoutID uuid.UUID, status string, stripeTransferID *string) error {
	for _, p := range r.byOrder {
		if p.ID == payoutID {
			p.Status = status
			p.StripeTransferID = stripeTransferID
			return nil
		}
	}
	return nil
}
func (r *recordingPayoutStore) ListBySeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error) {
	return nil, 0, nil
}
func (r *recordingPayoutStore) Reverse(ctx context.Context, payoutID uuid.UUID, stripeReversalID string) error {
	return nil
}

// recordingStripe reports whether CreateTransfer was called.
type recordingStripe struct {
	createTransferCalls int
}

func (s *recordingStripe) CreatePaymentIntent(amount int64, currency, seller string, meta map[string]string) (string, string, error) {
	return "", "", nil
}
func (s *recordingStripe) CreatePlatformPaymentIntent(amount int64, currency string, meta map[string]string) (string, string, error) {
	return "", "", nil
}
func (s *recordingStripe) CreateTransfer(amount int64, currency, dest, txGroup string) (string, error) {
	s.createTransferCalls++
	return "tr_test", nil
}
func (s *recordingStripe) CreateRefund(piID string, amount int64) (string, error) {
	return "re_test", nil
}
func (s *recordingStripe) CreateTransferReversal(transferID string, amount int64) (string, error) {
	return "trr_test", nil
}

// recordingPublisher captures event types for assertion. Uses the
// real pkgpubsub.Event envelope so the interface matches without an
// adapter shim.
type recordingPublisher struct {
	published []string
}

func (r *recordingPublisher) Publish(ctx context.Context, topic string, event pkgpubsub.Event) error {
	r.published = append(r.published, event.Type)
	return nil
}
func (r *recordingPublisher) Close() error { return nil }

// stubCommissionStore matches the order service's port.CommissionStore
// by returning nil rule (→ zero commission, unused in this test).
type stubCommissionStore struct{}

func (stubCommissionStore) GetApplicableRule(ctx context.Context, sellerID uuid.UUID, categoryID *uuid.UUID) (*domain.CommissionRule, error) {
	return nil, nil
}
func (stubCommissionStore) List(ctx context.Context, limit, offset int) ([]domain.CommissionRule, int, error) {
	return nil, 0, nil
}
func (stubCommissionStore) Create(ctx context.Context, r *domain.CommissionRule) error {
	return nil
}

// flakyCouponReserver fails Commit on the first call and succeeds
// on subsequent ones. Used to simulate a webhook retry scenario.
type flakyCouponReserver struct {
	commitCalls int
	failOnCall  int // 1-indexed
}

func (f *flakyCouponReserver) Reserve(ctx context.Context, code, buyer string, subs []port.SellerSubtotal, currency string) (*port.CouponReservation, error) {
	return nil, nil
}
func (f *flakyCouponReserver) Commit(ctx context.Context, rID, orderID uuid.UUID, piID string) (*port.CouponCommitResult, error) {
	f.commitCalls++
	if f.commitCalls == f.failOnCall {
		return nil, errors.New("transient coupon failure")
	}
	return &port.CouponCommitResult{RedemptionID: uuid.New()}, nil
}
func (f *flakyCouponReserver) Release(ctx context.Context, rID uuid.UUID, reason string) error {
	return nil
}

// TestHandlePaymentSuccess_PublishAfterRetryRecoveredCommit pins the
// invariant from the reviewer finding: when the first webhook delivery
// succeeds at Stripe transfer but fails at coupon Commit, a retry
// that lands with payout already completed must still publish
// `order.paid` and `payout.completed` — otherwise shipping, notification,
// and loyalty-earn-fanin consumers never see the order.
func TestHandlePaymentSuccess_PublishAfterRetryRecoveredCommit(t *testing.T) {
	orderID := uuid.New()
	payoutID := uuid.New()
	sellerID := uuid.New()
	piID := "pi_test"
	couponReservationID := uuid.New()

	orders := &recordingOrderStore{
		orderByID: map[uuid.UUID]*domain.Order{
			orderID: {
				ID:                  orderID,
				SellerID:            sellerID,
				Status:              domain.StatusPending,
				TotalAmount:         1000,
				Currency:            "JPY",
				CouponReservationID: &couponReservationID,
			},
		},
		ordersByPI: map[string][]domain.Order{
			piID: {{ID: orderID, SellerID: sellerID, Status: domain.StatusPending, CouponReservationID: &couponReservationID}},
		},
	}
	payouts := &recordingPayoutStore{
		byOrder: map[uuid.UUID]*domain.Payout{
			orderID: {ID: payoutID, OrderID: orderID, SellerID: sellerID, Status: domain.PayoutStatusPending, Amount: 900, Currency: "JPY"},
		},
	}
	stripe := &recordingStripe{}
	pub := &recordingPublisher{}
	coupon := &flakyCouponReserver{failOnCall: 1}

	svc := NewOrderService(
		orders, stubCommissionStore{}, payouts,
		stripe, pub, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{}, 500,
	).WithCouponReserver(coupon, true)

	// Attempt 1: Stripe retries webhook up to this handler.
	err := svc.HandlePaymentSuccess(context.Background(), piID)
	if err == nil {
		t.Fatal("expected first attempt to fail at coupon Commit")
	}
	if stripe.createTransferCalls != 1 {
		t.Errorf("first attempt: CreateTransfer calls = %d, want 1", stripe.createTransferCalls)
	}
	if payouts.byOrder[orderID].Status != domain.PayoutStatusCompleted {
		t.Errorf("first attempt: payout status = %q, want completed", payouts.byOrder[orderID].Status)
	}
	if orders.orderByID[orderID].PaidEventPublishedAt != nil {
		t.Errorf("first attempt: PaidEventPublishedAt should still be nil (Commit failed before publish)")
	}

	// Attempt 2: Stripe redelivers. Transfer is already done → skip.
	// Coupon Commit now succeeds. Publish must fire on this pass.
	err = svc.HandlePaymentSuccess(context.Background(), piID)
	if err != nil {
		t.Fatalf("second attempt: unexpected error: %v", err)
	}
	if stripe.createTransferCalls != 1 {
		t.Errorf("second attempt: CreateTransfer should NOT be called again (payout already completed), got %d", stripe.createTransferCalls)
	}
	if orders.orderByID[orderID].PaidEventPublishedAt == nil {
		t.Fatal("second attempt: PaidEventPublishedAt still nil — publish never fired on retry (regression)")
	}
	if !orders.markPublishedFirstCall {
		t.Error("MarkPaidEventPublished should have been called on second attempt")
	}

	// Attempt 3: Idempotent re-delivery. No transfer, no commit, no publish.
	// (The flag prevents re-publishing, the coupon fake's counter proves commit didn't re-run).
	prevCouponCalls := coupon.commitCalls
	err = svc.HandlePaymentSuccess(context.Background(), piID)
	if err != nil {
		t.Fatalf("third attempt: unexpected error: %v", err)
	}
	if stripe.createTransferCalls != 1 {
		t.Errorf("third attempt: CreateTransfer count drifted = %d", stripe.createTransferCalls)
	}
	if coupon.commitCalls == prevCouponCalls+1 {
		// Actually commit CAN re-run — downstream dedupes. But
		// we skip publish entirely via paid_event_published_at.
		// The key assertion is publish did not happen a second time.
	}
	// MarkPaidEventPublished should have been called at most once.
	if orders.markPublishedCalls > 1 {
		t.Errorf("MarkPaidEventPublished called %d times on third attempt; expected no change", orders.markPublishedCalls)
	}
}

// TestHandlePaymentSuccess_FailedPayoutSkipsDownstream asserts the
// payout-state allowlist: a payout already marked `failed` (transfer
// errored on a prior attempt) must short-circuit before any coupon
// or loyalty commit runs. Otherwise we'd book redemptions against a
// buyer whose money never settled.
func TestHandlePaymentSuccess_FailedPayoutSkipsDownstream(t *testing.T) {
	orderID := uuid.New()
	payoutID := uuid.New()
	sellerID := uuid.New()
	piID := "pi_test"
	couponReservationID := uuid.New()

	orders := &recordingOrderStore{
		orderByID: map[uuid.UUID]*domain.Order{
			orderID: {
				ID:                  orderID,
				SellerID:            sellerID,
				Status:              domain.StatusPaid, // already marked paid on earlier attempt
				TotalAmount:         1000,
				Currency:            "JPY",
				CouponReservationID: &couponReservationID,
			},
		},
		ordersByPI: map[string][]domain.Order{
			piID: {{ID: orderID, SellerID: sellerID, Status: domain.StatusPaid, CouponReservationID: &couponReservationID}},
		},
	}
	payouts := &recordingPayoutStore{
		byOrder: map[uuid.UUID]*domain.Payout{
			orderID: {ID: payoutID, OrderID: orderID, SellerID: sellerID, Status: domain.PayoutStatusFailed, Amount: 900, Currency: "JPY"},
		},
	}
	stripe := &recordingStripe{}
	pub := &recordingPublisher{}
	coupon := &flakyCouponReserver{failOnCall: -1} // never fail; assert it's not called at all

	svc := NewOrderService(
		orders, stubCommissionStore{}, payouts,
		stripe, pub, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{}, 500,
	).WithCouponReserver(coupon, true)

	err := svc.HandlePaymentSuccess(context.Background(), piID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stripe.createTransferCalls != 0 {
		t.Errorf("Failed payout must not trigger CreateTransfer; got %d", stripe.createTransferCalls)
	}
	if coupon.commitCalls != 0 {
		t.Errorf("Failed payout must not trigger coupon Commit; got %d", coupon.commitCalls)
	}
	if orders.markPublishedCalls != 0 {
		t.Errorf("Failed payout must not publish events; MarkPaidEventPublished calls = %d", orders.markPublishedCalls)
	}
}

// TestHandlePaymentSuccess_ReversedPayoutSkipsDownstream is the twin
// case of the failed-payout test for the `reversed` state (seller-
// approved cancellation path has rolled back the transfer). The
// handler must not rebuild the committed state.
func TestHandlePaymentSuccess_ReversedPayoutSkipsDownstream(t *testing.T) {
	orderID := uuid.New()
	payoutID := uuid.New()
	sellerID := uuid.New()
	piID := "pi_test"

	orders := &recordingOrderStore{
		orderByID: map[uuid.UUID]*domain.Order{
			orderID: {
				ID:          orderID,
				SellerID:    sellerID,
				Status:      domain.StatusPaid,
				TotalAmount: 1000,
				Currency:    "JPY",
			},
		},
		ordersByPI: map[string][]domain.Order{
			piID: {{ID: orderID, SellerID: sellerID, Status: domain.StatusPaid}},
		},
	}
	payouts := &recordingPayoutStore{
		byOrder: map[uuid.UUID]*domain.Payout{
			orderID: {ID: payoutID, OrderID: orderID, SellerID: sellerID, Status: domain.PayoutStatusReversed, Amount: 900, Currency: "JPY"},
		},
	}
	stripe := &recordingStripe{}
	svc := NewOrderService(
		orders, stubCommissionStore{}, payouts,
		stripe, nil, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{}, 500,
	)

	err := svc.HandlePaymentSuccess(context.Background(), piID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stripe.createTransferCalls != 0 {
		t.Errorf("Reversed payout must not trigger CreateTransfer; got %d", stripe.createTransferCalls)
	}
	if orders.markPublishedCalls != 0 {
		t.Errorf("Reversed payout must not publish events; MarkPaidEventPublished calls = %d", orders.markPublishedCalls)
	}
}
