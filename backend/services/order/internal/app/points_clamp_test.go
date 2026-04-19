package app

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/port"
)

// fakeCouponReserver records the arguments Reserve was called with so
// tests can assert the checkout passes through the expected values.
type fakeCouponReserver struct {
	reservation *port.CouponReservation
	reserveErr  error
	called      bool
}

func (f *fakeCouponReserver) Reserve(ctx context.Context, code, buyer string, subs []port.SellerSubtotal, currency string) (*port.CouponReservation, error) {
	f.called = true
	if f.reserveErr != nil {
		return nil, f.reserveErr
	}
	return f.reservation, nil
}
func (f *fakeCouponReserver) Commit(ctx context.Context, rID, orderID uuid.UUID, piID string) (*port.CouponCommitResult, error) {
	return &port.CouponCommitResult{RedemptionID: uuid.New()}, nil
}
func (f *fakeCouponReserver) Release(ctx context.Context, rID uuid.UUID, reason string) error {
	return nil
}

// fakePointReserver records the effective Reserve amount so the clamp
// regression can be pinned against it.
type fakePointReserver struct {
	reservedAmount int64
	reservedCalled bool
}

func (f *fakePointReserver) Reserve(ctx context.Context, buyer string, amount int64) (*port.PointReservation, error) {
	f.reservedCalled = true
	f.reservedAmount = amount
	return &port.PointReservation{
		ReservationID: uuid.New(),
		Amount:        amount,
		ExpiresAt:     time.Now().Add(30 * time.Minute),
	}, nil
}
func (f *fakePointReserver) Commit(ctx context.Context, rID *uuid.UUID, buyer, orderID, piID string, paidSubtotal int64, currency string) (*port.PointCommitResult, error) {
	return &port.PointCommitResult{Earned: 0}, nil
}
func (f *fakePointReserver) Release(ctx context.Context, rID uuid.UUID, reason string) error {
	return nil
}
func (f *fakePointReserver) GetBalance(ctx context.Context, buyer string) (*port.PointBalance, error) {
	return &port.PointBalance{Balance: 1_000_000, Spendable: 1_000_000}, nil
}
func (f *fakePointReserver) AwardEarn(ctx context.Context, buyer, orderID string, paidSubtotal int64, currency string) (*port.PointCommitResult, error) {
	return &port.PointCommitResult{}, nil
}

// fakeBuyerSub always reports no free shipping so TestClamp's expected
// totals are deterministic.
type fakeBuyerSub struct{}

func (fakeBuyerSub) HasFreeShipping(ctx context.Context, buyerAuth0ID string) (bool, error) {
	return false, nil
}

type fakeSellerLookup struct{}

func (fakeSellerLookup) BatchGetSellerNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	out := map[uuid.UUID]string{}
	for _, id := range ids {
		out[id] = "Seller " + id.String()[:8]
	}
	return out, nil
}

type fakeSKULookup struct{}

func (fakeSKULookup) BatchGetSKUProductIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	out := map[uuid.UUID]uuid.UUID{}
	for _, id := range ids {
		out[id] = uuid.New()
	}
	return out, nil
}

// fakeCommissionStore / fakePayoutStore / fakeOrderStore are trimmed-
// down stubs that just accept writes and return zero values for reads.
// The ruleByBps knob lets tests pin a rate without wiring the full
// (tenant, category, seller) resolution path.
type fakeCommissionStore struct {
	ruleBps int
}

func (f fakeCommissionStore) GetApplicableRule(ctx context.Context, sellerID uuid.UUID, categoryID *uuid.UUID) (*domain.CommissionRule, error) {
	if f.ruleBps == 0 {
		return nil, nil
	}
	return &domain.CommissionRule{RateBps: f.ruleBps}, nil
}
func (fakeCommissionStore) List(ctx context.Context, limit, offset int) ([]domain.CommissionRule, int, error) {
	return nil, 0, nil
}
func (fakeCommissionStore) Create(ctx context.Context, r *domain.CommissionRule) error {
	return nil
}

// fakePayoutStore is a no-op store for checkout tests. Payout fields
// are asserted directly via fakeOrderStore.insertedBatch[i].Payout
// so no persistence capture is needed here.
type fakePayoutStore struct{}

func (f *fakePayoutStore) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payout, error) {
	return nil, nil
}
func (f *fakePayoutStore) UpdateStatus(ctx context.Context, payoutID uuid.UUID, status string, stripeTransferID *string) error {
	return nil
}
func (f *fakePayoutStore) ListBySeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error) {
	return nil, 0, nil
}
func (f *fakePayoutStore) Reverse(ctx context.Context, payoutID uuid.UUID, stripeReversalID string) error {
	return nil
}

type fakeOrderStore struct {
	insertedOrders []*domain.Order
	insertedBatch  []domain.CheckoutBatchItem
}

func (f *fakeOrderStore) Create(ctx context.Context, order *domain.Order, lines []domain.OrderLine) error {
	return nil
}
func (f *fakeOrderStore) CreateCheckoutBatch(ctx context.Context, items []domain.CheckoutBatchItem) error {
	for _, item := range items {
		item.Order.ID = uuid.New()
		f.insertedOrders = append(f.insertedOrders, item.Order)
		f.insertedBatch = append(f.insertedBatch, item)
	}
	return nil
}
func (f *fakeOrderStore) GetByID(ctx context.Context, orderID uuid.UUID) (*domain.OrderWithLines, error) {
	return nil, nil
}
func (f *fakeOrderStore) HasPurchasedSKU(ctx context.Context, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*domain.PurchaseSKURecord, error) {
	return nil, nil
}
func (f *fakeOrderStore) ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error) {
	return nil, 0, nil
}
func (f *fakeOrderStore) ListBySeller(ctx context.Context, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error) {
	return nil, 0, nil
}
func (f *fakeOrderStore) UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error {
	return nil
}
func (f *fakeOrderStore) SetPaid(ctx context.Context, orderID uuid.UUID, paidAt time.Time, stripePaymentIntentID string) error {
	return nil
}
func (f *fakeOrderStore) FindAllByStripePaymentIntentID(ctx context.Context, paymentIntentID string) ([]domain.Order, error) {
	return nil, nil
}
func (f *fakeOrderStore) SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error {
	return nil
}
func (f *fakeOrderStore) SetPointsEarned(ctx context.Context, orderID uuid.UUID, amount int64) error {
	return nil
}
func (f *fakeOrderStore) MarkPaidEventPublished(ctx context.Context, orderID uuid.UUID) (bool, error) {
	return true, nil
}
func (f *fakeOrderStore) ClaimPaidEventPublish(ctx context.Context, orderID uuid.UUID, ttl time.Duration) (bool, bool, error) {
	return true, false, nil
}

// fakeStripe records the amount passed to CreatePlatformPaymentIntent
// so tests can assert post-discount totals.
type fakeStripe struct {
	lastAmount int64
}

func (f *fakeStripe) CreatePaymentIntent(amount int64, currency, sellerAcct string, meta map[string]string) (string, string, error) {
	return "pi_test", "secret_test", nil
}
func (f *fakeStripe) CreatePlatformPaymentIntent(amount int64, currency string, meta map[string]string) (string, string, error) {
	f.lastAmount = amount
	return "pi_test", "secret_test", nil
}
func (f *fakeStripe) CreateTransfer(amount int64, currency, dest, txGroup string) (string, error) {
	return "tr_test", nil
}
func (f *fakeStripe) CreateRefund(piID string, amount int64) (string, error) {
	return "re_test", nil
}
func (f *fakeStripe) CreateTransferReversal(transferID string, amount int64) (string, error) {
	return "trr_test", nil
}

// TestCreateCheckout_ClampsPointsToApplicableSubtotal is the direct
// regression test for the "buyer loses points they didn't spend"
// finding. A buyer asks to redeem 5000 pts on a 1000-yen cart; the
// order service must only reserve 1000 with loyalty-svc — anything
// more would be burned on Commit without applying to the order total.
func TestCreateCheckout_ClampsPointsToApplicableSubtotal(t *testing.T) {
	points := &fakePointReserver{}
	orderStore := &fakeOrderStore{}
	stripe := &fakeStripe{}
	svc := NewOrderService(
		orderStore, fakeCommissionStore{}, &fakePayoutStore{},
		stripe, nil, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{},
		0, // no shipping fee so cartSubtotal == lineTotal
	).WithPointReserver(points, true)

	sellerID := uuid.New()
	skuID := uuid.New()
	_, err := svc.CreateCheckout(context.Background(), domain.CheckoutInput{
		BuyerAuth0ID: "auth0|buyer",
		Currency:     "jpy",
		Lines: []domain.CheckoutLineInput{{
			SKUID:     skuID,
			SellerID:  sellerID,
			Quantity:  1,
			UnitPrice: 1000, // cart subtotal = 1000
		}},
		ShippingAddress: json.RawMessage(`{}`),
		PointsToRedeem:  5000, // buyer requested 5x the cart
	})
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	if !points.reservedCalled {
		t.Fatal("loyalty Reserve was not called")
	}
	if points.reservedAmount != 1000 {
		t.Errorf("reserved amount = %d, want 1000 (clamped to cart subtotal)", points.reservedAmount)
	}
	// Stripe PI total should reflect the clamped discount: 0 yen due.
	if stripe.lastAmount != 0 {
		t.Errorf("Stripe PI amount = %d, want 0 (subtotal 1000 - points 1000)", stripe.lastAmount)
	}
}

// TestCreateCheckout_PointsWithCouponClampsAgainstRemainder covers the
// interaction between coupon and points: if a coupon already absorbed
// 200 yen of a 1000-yen cart, points should clamp at the remaining
// 800 — not the full 1000 — to preserve the "no over-burn" invariant.
func TestCreateCheckout_PointsWithCouponClampsAgainstRemainder(t *testing.T) {
	coupon := &fakeCouponReserver{
		reservation: &port.CouponReservation{
			ReservationID:  uuid.New(),
			CouponID:       uuid.New(),
			IssuerType:     "platform",
			DiscountAmount: 200,
			Currency:       "jpy",
		},
	}
	points := &fakePointReserver{}
	orderStore := &fakeOrderStore{}
	svc := NewOrderService(
		orderStore, fakeCommissionStore{}, &fakePayoutStore{},
		&fakeStripe{}, nil, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{}, 0,
	).
		WithCouponReserver(coupon, true).
		WithPointReserver(points, true)

	sellerID := uuid.New()
	_, err := svc.CreateCheckout(context.Background(), domain.CheckoutInput{
		BuyerAuth0ID: "auth0|buyer",
		Currency:     "jpy",
		Lines: []domain.CheckoutLineInput{{
			SKUID: uuid.New(), SellerID: sellerID, Quantity: 1, UnitPrice: 1000,
		}},
		ShippingAddress: json.RawMessage(`{}`),
		CouponCode:      "WELCOME",
		PointsToRedeem:  5000,
	})
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	if points.reservedAmount != 800 {
		t.Errorf("reserved amount = %d, want 800 (cart 1000 - coupon 200)", points.reservedAmount)
	}
}

// TestCreateCheckout_PointsWithinSubtotalPassesThrough asserts the
// clamp is a ceiling, not a floor — a valid amount must not get
// rounded down.
func TestCreateCheckout_PointsWithinSubtotalPassesThrough(t *testing.T) {
	points := &fakePointReserver{}
	svc := NewOrderService(
		&fakeOrderStore{}, fakeCommissionStore{}, &fakePayoutStore{},
		&fakeStripe{}, nil, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{}, 0,
	).WithPointReserver(points, true)

	_, err := svc.CreateCheckout(context.Background(), domain.CheckoutInput{
		BuyerAuth0ID: "auth0|buyer",
		Currency:     "jpy",
		Lines: []domain.CheckoutLineInput{{
			SKUID: uuid.New(), SellerID: uuid.New(), Quantity: 1, UnitPrice: 5000,
		}},
		ShippingAddress: json.RawMessage(`{}`),
		PointsToRedeem:  1500,
	})
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	if points.reservedAmount != 1500 {
		t.Errorf("reserved amount = %d, want 1500 (no clamp)", points.reservedAmount)
	}
}

// TestCreateCheckout_SellerCouponAdjustsCommissionAndPayout pins the
// seller-funded-coupon accounting. Seller A issues a 200 yen coupon on
// a 1000 yen order with a 10% commission rule: the buyer pays 800, the
// platform takes commission on the *post-discount* revenue, and the
// seller's payout shrinks to match. Prior to this fix, commission +
// payout were frozen on pre-discount subtotal so the seller got
// overpaid by the coupon amount.
func TestCreateCheckout_SellerCouponAdjustsCommissionAndPayout(t *testing.T) {
	sellerA := uuid.New()
	coupon := &fakeCouponReserver{
		reservation: &port.CouponReservation{
			ReservationID:      uuid.New(),
			CouponID:           uuid.New(),
			IssuerType:         "seller",
			ApplicableSellerID: &sellerA,
			DiscountAmount:     200,
			Currency:           "jpy",
		},
	}
	orderStore := &fakeOrderStore{}
	payoutStore := &fakePayoutStore{}
	svc := NewOrderService(
		orderStore, fakeCommissionStore{ruleBps: 1000}, payoutStore,
		&fakeStripe{}, nil, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{}, 0,
	).WithCouponReserver(coupon, true)

	_, err := svc.CreateCheckout(context.Background(), domain.CheckoutInput{
		BuyerAuth0ID: "auth0|buyer",
		Currency:     "jpy",
		Lines: []domain.CheckoutLineInput{{
			SKUID: uuid.New(), SellerID: sellerA, Quantity: 1, UnitPrice: 1000,
		}},
		ShippingAddress: json.RawMessage(`{}`),
		CouponCode:      "SHOPA10",
	})
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	if len(orderStore.insertedBatch) != 1 {
		t.Fatalf("expected 1 batch item, got %d", len(orderStore.insertedBatch))
	}
	bi := orderStore.insertedBatch[0]
	// effective_subtotal = 1000 - 200 = 800. commission = 800*0.10 = 80.
	// payout = 800 - 80 = 720.
	if bi.Order.CouponDiscountAmount != 200 {
		t.Errorf("CouponDiscountAmount = %d, want 200", bi.Order.CouponDiscountAmount)
	}
	if bi.Order.CommissionAmount != 80 {
		t.Errorf("CommissionAmount = %d, want 80 (10%% of post-discount 800)", bi.Order.CommissionAmount)
	}
	if bi.Payout.Amount != 720 {
		t.Errorf("Payout.Amount = %d, want 720 (800 - 80 commission)", bi.Payout.Amount)
	}
}

// TestCreateCheckout_SellerCouponDoesntAffectOtherSeller asserts that
// seller A's coupon leaves seller B's commission + payout intact — the
// discount is seller A's cost, not seller B's.
func TestCreateCheckout_SellerCouponDoesntAffectOtherSeller(t *testing.T) {
	sellerA := uuid.New()
	sellerB := uuid.New()
	coupon := &fakeCouponReserver{
		reservation: &port.CouponReservation{
			ReservationID:      uuid.New(),
			CouponID:           uuid.New(),
			IssuerType:         "seller",
			ApplicableSellerID: &sellerA,
			DiscountAmount:     300,
			Currency:           "jpy",
		},
	}
	orderStore := &fakeOrderStore{}
	svc := NewOrderService(
		orderStore, fakeCommissionStore{ruleBps: 1000}, &fakePayoutStore{},
		&fakeStripe{}, nil, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{}, 0,
	).WithCouponReserver(coupon, true)

	_, err := svc.CreateCheckout(context.Background(), domain.CheckoutInput{
		BuyerAuth0ID: "auth0|buyer",
		Currency:     "jpy",
		Lines: []domain.CheckoutLineInput{
			{SKUID: uuid.New(), SellerID: sellerA, Quantity: 1, UnitPrice: 1000},
			{SKUID: uuid.New(), SellerID: sellerB, Quantity: 1, UnitPrice: 2000},
		},
		ShippingAddress: json.RawMessage(`{}`),
		CouponCode:      "SHOPA300",
	})
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	if len(orderStore.insertedBatch) != 2 {
		t.Fatalf("expected 2 batch items, got %d", len(orderStore.insertedBatch))
	}
	var aBatch, bBatch domain.CheckoutBatchItem
	for _, bi := range orderStore.insertedBatch {
		switch bi.Order.SellerID {
		case sellerA:
			aBatch = bi
		case sellerB:
			bBatch = bi
		}
	}
	// Seller A: effective 700, commission 70, payout 630.
	if aBatch.Order.CouponDiscountAmount != 300 {
		t.Errorf("seller A coupon = %d, want 300", aBatch.Order.CouponDiscountAmount)
	}
	if aBatch.Order.CommissionAmount != 70 {
		t.Errorf("seller A commission = %d, want 70 (10%% of 700)", aBatch.Order.CommissionAmount)
	}
	if aBatch.Payout.Amount != 630 {
		t.Errorf("seller A payout = %d, want 630", aBatch.Payout.Amount)
	}
	// Seller B: no coupon touched its subtotal, so pre-discount math stands.
	if bBatch.Order.CouponDiscountAmount != 0 {
		t.Errorf("seller B coupon = %d, want 0 (different seller's coupon must not leak)", bBatch.Order.CouponDiscountAmount)
	}
	if bBatch.Order.CommissionAmount != 200 {
		t.Errorf("seller B commission = %d, want 200 (10%% of 2000)", bBatch.Order.CommissionAmount)
	}
	if bBatch.Payout.Amount != 1800 {
		t.Errorf("seller B payout = %d, want 1800", bBatch.Payout.Amount)
	}
}

// TestCreateCheckout_PlatformCouponDoesntTouchPayout confirms the
// platform-funded fallback: platform coupons drain from the platform,
// not the seller, so payout + commission stay on pre-discount subtotal.
func TestCreateCheckout_PlatformCouponDoesntTouchPayout(t *testing.T) {
	coupon := &fakeCouponReserver{
		reservation: &port.CouponReservation{
			ReservationID:  uuid.New(),
			CouponID:       uuid.New(),
			IssuerType:     "platform",
			DiscountAmount: 200,
			Currency:       "jpy",
		},
	}
	orderStore := &fakeOrderStore{}
	svc := NewOrderService(
		orderStore, fakeCommissionStore{ruleBps: 1000}, &fakePayoutStore{},
		&fakeStripe{}, nil, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{}, 0,
	).WithCouponReserver(coupon, true)

	_, err := svc.CreateCheckout(context.Background(), domain.CheckoutInput{
		BuyerAuth0ID: "auth0|buyer",
		Currency:     "jpy",
		Lines: []domain.CheckoutLineInput{{
			SKUID: uuid.New(), SellerID: uuid.New(), Quantity: 1, UnitPrice: 1000,
		}},
		ShippingAddress: json.RawMessage(`{}`),
		CouponCode:      "PLATFORM200",
	})
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	bi := orderStore.insertedBatch[0]
	if bi.Order.CommissionAmount != 100 {
		t.Errorf("CommissionAmount = %d, want 100 (platform coupon must not shrink commission)", bi.Order.CommissionAmount)
	}
	if bi.Payout.Amount != 900 {
		t.Errorf("Payout.Amount = %d, want 900 (platform coupon: seller paid on pre-discount)", bi.Payout.Amount)
	}
}

// TestCreateCheckout_SellerCouponPointsClampToResidual pins the Medium
// issue: when a seller coupon has already absorbed most of an order's
// subtotal, subsequent point shares must fit inside the remaining
// per-order residual, not the original subtotal. Otherwise the buyer's
// points get burned against a clamped-to-zero order total.
func TestCreateCheckout_SellerCouponPointsClampToResidual(t *testing.T) {
	sellerA := uuid.New()
	coupon := &fakeCouponReserver{
		reservation: &port.CouponReservation{
			ReservationID:      uuid.New(),
			CouponID:           uuid.New(),
			IssuerType:         "seller",
			ApplicableSellerID: &sellerA,
			DiscountAmount:     800, // absorbs most of the 1000-yen subtotal
			Currency:           "jpy",
		},
	}
	points := &fakePointReserver{}
	orderStore := &fakeOrderStore{}
	svc := NewOrderService(
		orderStore, fakeCommissionStore{}, &fakePayoutStore{},
		&fakeStripe{}, nil, fakeBuyerSub{}, fakeSellerLookup{}, fakeSKULookup{}, 0,
	).
		WithCouponReserver(coupon, true).
		WithPointReserver(points, true)

	_, err := svc.CreateCheckout(context.Background(), domain.CheckoutInput{
		BuyerAuth0ID: "auth0|buyer",
		Currency:     "jpy",
		Lines: []domain.CheckoutLineInput{{
			SKUID: uuid.New(), SellerID: sellerA, Quantity: 1, UnitPrice: 1000,
		}},
		ShippingAddress: json.RawMessage(`{}`),
		CouponCode:      "SHOPA800",
		PointsToRedeem:  500,
	})
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	// Cart-wide clamp keeps points reservation at cart_subtotal - coupon
	// = 1000 - 800 = 200. Buyer asked for 500 but only 200 should be
	// reserved.
	if points.reservedAmount != 200 {
		t.Errorf("reserved points = %d, want 200 (1000 subtotal - 800 coupon)", points.reservedAmount)
	}
	bi := orderStore.insertedBatch[0]
	if bi.Order.PointDiscountAmount != 200 {
		t.Errorf("PointDiscountAmount = %d, want 200 (must fit in residual)", bi.Order.PointDiscountAmount)
	}
	if bi.Order.PointDiscountAmount > bi.Order.SubtotalAmount-bi.Order.CouponDiscountAmount {
		t.Errorf("points %d exceeds residual %d — buyer would over-spend", bi.Order.PointDiscountAmount, bi.Order.SubtotalAmount-bi.Order.CouponDiscountAmount)
	}
}
