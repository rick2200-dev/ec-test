package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	app "github.com/Riku-KANO/ec-test/services/cart/internal/app"
	"github.com/Riku-KANO/ec-test/services/cart/internal/domain"
	"github.com/Riku-KANO/ec-test/services/cart/internal/port"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// mockCartStore implements port.CartStore for testing.
type mockCartStore struct {
	cart      *domain.Cart
	getErr    error
	saveErr   error
	deleteErr error

	savedCart   *domain.Cart
	deleteCalls int
}

func (m *mockCartStore) Get(_ context.Context, _ uuid.UUID, _ string) (*domain.Cart, error) {
	return m.cart, m.getErr
}

func (m *mockCartStore) Save(_ context.Context, cart *domain.Cart) error {
	m.savedCart = cart
	return m.saveErr
}

func (m *mockCartStore) Delete(_ context.Context, _ uuid.UUID, _ string) error {
	m.deleteCalls++
	return m.deleteErr
}

// mockSKULookupClient implements port.SKULookupClient for testing.
type mockSKULookupClient struct {
	sku *port.SKULookup
	err error
}

func (m *mockSKULookupClient) LookupSKU(_ context.Context, _, _ uuid.UUID) (*port.SKULookup, error) {
	return m.sku, m.err
}

// mockCheckoutClient implements port.CheckoutClient for testing.
type mockCheckoutClient struct {
	result *domain.CheckoutResult
	err    error

	calledWith *domain.CheckoutInput
}

func (m *mockCheckoutClient) CreateCheckout(_ context.Context, _ uuid.UUID, in domain.CheckoutInput) (*domain.CheckoutResult, error) {
	m.calledWith = &in
	return m.result, m.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newService(store *mockCartStore, catalog *mockSKULookupClient, order *mockCheckoutClient) *app.CartService {
	return app.NewCartService(store, catalog, order, nil)
}

var (
	testTenantID    = uuid.New()
	testBuyerID     = "auth0|buyer-001"
	testSKUID       = uuid.New()
	testSellerID    = uuid.New()
	testProductName = "Test Product"
	testSKUCode     = "SKU-001"
)

func newTestCart(items ...domain.CartItem) *domain.Cart {
	return &domain.Cart{
		TenantID:     testTenantID,
		BuyerAuth0ID: testBuyerID,
		Items:        items,
		UpdatedAt:    time.Now().UTC(),
	}
}

func newTestCartItem(skuID, sellerID uuid.UUID, qty int, price int64) domain.CartItem {
	return domain.CartItem{
		SKUID:               skuID,
		SellerID:            sellerID,
		Quantity:            qty,
		UnitPriceSnapshot:   price,
		Currency:            "jpy",
		ProductNameSnapshot: testProductName,
		SKUCodeSnapshot:     testSKUCode,
		AddedAt:             time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// GetCart
// ---------------------------------------------------------------------------

func TestGetCart_ReturnsExistingCart(t *testing.T) {
	existing := newTestCart(newTestCartItem(testSKUID, testSellerID, 2, 1000))
	store := &mockCartStore{cart: existing}
	svc := newService(store, nil, nil)

	cart, err := svc.GetCart(context.Background(), testTenantID, testBuyerID)
	if err != nil {
		t.Fatalf("GetCart() error = %v, want nil", err)
	}
	if cart != existing {
		t.Error("GetCart() did not return the stored cart")
	}
	if len(cart.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(cart.Items))
	}
}

func TestGetCart_ReturnsEmptyCartWhenNil(t *testing.T) {
	store := &mockCartStore{cart: nil}
	svc := newService(store, nil, nil)

	cart, err := svc.GetCart(context.Background(), testTenantID, testBuyerID)
	if err != nil {
		t.Fatalf("GetCart() error = %v, want nil", err)
	}
	if cart == nil {
		t.Fatal("GetCart() returned nil, want empty cart")
	}
	if cart.TenantID != testTenantID {
		t.Errorf("TenantID = %v, want %v", cart.TenantID, testTenantID)
	}
	if cart.BuyerAuth0ID != testBuyerID {
		t.Errorf("BuyerAuth0ID = %q, want %q", cart.BuyerAuth0ID, testBuyerID)
	}
	if len(cart.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(cart.Items))
	}
}

func TestGetCart_RepoError(t *testing.T) {
	store := &mockCartStore{getErr: errors.New("redis down")}
	svc := newService(store, nil, nil)

	_, err := svc.GetCart(context.Background(), testTenantID, testBuyerID)
	if err == nil {
		t.Fatal("GetCart() error = nil, want error")
	}
}

// ---------------------------------------------------------------------------
// AddItem
// ---------------------------------------------------------------------------

func TestAddItem_NewSKU_CallsLookupAndSaves(t *testing.T) {
	skuID := uuid.New()
	sellerID := uuid.New()

	store := &mockCartStore{cart: nil} // empty cart
	catalog := &mockSKULookupClient{
		sku: &port.SKULookup{
			SKUID:         skuID,
			ProductID:     uuid.New(),
			SellerID:      sellerID,
			ProductName:   "Widget",
			SKUCode:       "WDG-001",
			PriceAmount:   2500,
			PriceCurrency: "jpy",
			Status:        "active",
		},
	}
	svc := newService(store, catalog, nil)

	cart, err := svc.AddItem(context.Background(), testTenantID, testBuyerID, skuID, 3)
	if err != nil {
		t.Fatalf("AddItem() error = %v, want nil", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(cart.Items))
	}
	item := cart.Items[0]
	if item.SKUID != skuID {
		t.Errorf("SKUID = %v, want %v", item.SKUID, skuID)
	}
	if item.SellerID != sellerID {
		t.Errorf("SellerID = %v, want %v", item.SellerID, sellerID)
	}
	if item.Quantity != 3 {
		t.Errorf("Quantity = %d, want 3", item.Quantity)
	}
	if item.UnitPriceSnapshot != 2500 {
		t.Errorf("UnitPriceSnapshot = %d, want 2500", item.UnitPriceSnapshot)
	}
	if item.Currency != "jpy" {
		t.Errorf("Currency = %q, want %q", item.Currency, "jpy")
	}
	if item.ProductNameSnapshot != "Widget" {
		t.Errorf("ProductNameSnapshot = %q, want %q", item.ProductNameSnapshot, "Widget")
	}
	if item.SKUCodeSnapshot != "WDG-001" {
		t.Errorf("SKUCodeSnapshot = %q, want %q", item.SKUCodeSnapshot, "WDG-001")
	}
	if store.savedCart == nil {
		t.Error("Save was not called")
	}
}

func TestAddItem_ExistingSKU_IncrementsQuantity(t *testing.T) {
	skuID := uuid.New()
	existing := newTestCart(newTestCartItem(skuID, testSellerID, 2, 1000))
	store := &mockCartStore{cart: existing}
	// catalog should NOT be called for an existing SKU
	catalog := &mockSKULookupClient{err: errors.New("should not be called")}
	svc := newService(store, catalog, nil)

	cart, err := svc.AddItem(context.Background(), testTenantID, testBuyerID, skuID, 5)
	if err != nil {
		t.Fatalf("AddItem() error = %v, want nil", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(cart.Items))
	}
	if cart.Items[0].Quantity != 7 {
		t.Errorf("Quantity = %d, want 7 (2 + 5)", cart.Items[0].Quantity)
	}
}

func TestAddItem_InvalidQuantity_Zero(t *testing.T) {
	svc := newService(&mockCartStore{}, nil, nil)

	_, err := svc.AddItem(context.Background(), testTenantID, testBuyerID, testSKUID, 0)
	if !errors.Is(err, domain.ErrInvalidQuantity) {
		t.Errorf("AddItem(qty=0) error = %v, want %v", err, domain.ErrInvalidQuantity)
	}
}

func TestAddItem_InvalidQuantity_Negative(t *testing.T) {
	svc := newService(&mockCartStore{}, nil, nil)

	_, err := svc.AddItem(context.Background(), testTenantID, testBuyerID, testSKUID, -1)
	if !errors.Is(err, domain.ErrInvalidQuantity) {
		t.Errorf("AddItem(qty=-1) error = %v, want %v", err, domain.ErrInvalidQuantity)
	}
}

func TestAddItem_LookupError(t *testing.T) {
	store := &mockCartStore{cart: nil}
	catalog := &mockSKULookupClient{err: errors.New("catalog unavailable")}
	svc := newService(store, catalog, nil)

	_, err := svc.AddItem(context.Background(), testTenantID, testBuyerID, uuid.New(), 1)
	if err == nil {
		t.Fatal("AddItem() error = nil, want error from catalog")
	}
}

func TestAddItem_SaveError(t *testing.T) {
	skuID := uuid.New()
	store := &mockCartStore{
		cart:    nil,
		saveErr: errors.New("redis write failed"),
	}
	catalog := &mockSKULookupClient{
		sku: &port.SKULookup{
			SKUID:         skuID,
			SellerID:      uuid.New(),
			PriceAmount:   100,
			PriceCurrency: "jpy",
		},
	}
	svc := newService(store, catalog, nil)

	_, err := svc.AddItem(context.Background(), testTenantID, testBuyerID, skuID, 1)
	if err == nil {
		t.Fatal("AddItem() error = nil, want save error")
	}
}

// ---------------------------------------------------------------------------
// UpdateItemQuantity
// ---------------------------------------------------------------------------

func TestUpdateItemQuantity_Success(t *testing.T) {
	skuID := uuid.New()
	existing := newTestCart(newTestCartItem(skuID, testSellerID, 2, 1000))
	store := &mockCartStore{cart: existing}
	svc := newService(store, nil, nil)

	cart, err := svc.UpdateItemQuantity(context.Background(), testTenantID, testBuyerID, skuID, 10)
	if err != nil {
		t.Fatalf("UpdateItemQuantity() error = %v, want nil", err)
	}
	if cart.Items[0].Quantity != 10 {
		t.Errorf("Quantity = %d, want 10", cart.Items[0].Quantity)
	}
	if store.savedCart == nil {
		t.Error("Save was not called")
	}
}

func TestUpdateItemQuantity_NegativeQuantity(t *testing.T) {
	svc := newService(&mockCartStore{}, nil, nil)

	_, err := svc.UpdateItemQuantity(context.Background(), testTenantID, testBuyerID, testSKUID, -1)
	if !errors.Is(err, domain.ErrNonNegativeQuantity) {
		t.Errorf("UpdateItemQuantity(qty=-1) error = %v, want %v", err, domain.ErrNonNegativeQuantity)
	}
}

func TestUpdateItemQuantity_ZeroDelegatesToRemoveItem(t *testing.T) {
	skuID := uuid.New()
	otherSKU := uuid.New()
	existing := newTestCart(
		newTestCartItem(skuID, testSellerID, 2, 1000),
		newTestCartItem(otherSKU, testSellerID, 1, 500),
	)
	store := &mockCartStore{cart: existing}
	svc := newService(store, nil, nil)

	cart, err := svc.UpdateItemQuantity(context.Background(), testTenantID, testBuyerID, skuID, 0)
	if err != nil {
		t.Fatalf("UpdateItemQuantity(qty=0) error = %v, want nil", err)
	}
	// The SKU should be removed.
	if cart.FindItem(skuID) >= 0 {
		t.Error("expected SKU to be removed from cart")
	}
	// The other item should remain.
	if cart.FindItem(otherSKU) < 0 {
		t.Error("other SKU should remain in cart")
	}
}

func TestUpdateItemQuantity_SKUNotInCart(t *testing.T) {
	existing := newTestCart(newTestCartItem(uuid.New(), testSellerID, 1, 100))
	store := &mockCartStore{cart: existing}
	svc := newService(store, nil, nil)

	_, err := svc.UpdateItemQuantity(context.Background(), testTenantID, testBuyerID, uuid.New(), 5)
	if !errors.Is(err, domain.ErrSKUNotInCart) {
		t.Errorf("UpdateItemQuantity(missing SKU) error = %v, want %v", err, domain.ErrSKUNotInCart)
	}
}

// ---------------------------------------------------------------------------
// RemoveItem
// ---------------------------------------------------------------------------

func TestRemoveItem_ExistingSKU(t *testing.T) {
	skuID := uuid.New()
	otherSKU := uuid.New()
	existing := newTestCart(
		newTestCartItem(skuID, testSellerID, 2, 1000),
		newTestCartItem(otherSKU, testSellerID, 1, 500),
	)
	store := &mockCartStore{cart: existing}
	svc := newService(store, nil, nil)

	cart, err := svc.RemoveItem(context.Background(), testTenantID, testBuyerID, skuID)
	if err != nil {
		t.Fatalf("RemoveItem() error = %v, want nil", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(cart.Items))
	}
	if cart.Items[0].SKUID != otherSKU {
		t.Errorf("remaining SKUID = %v, want %v", cart.Items[0].SKUID, otherSKU)
	}
	if store.savedCart == nil {
		t.Error("Save was not called")
	}
}

func TestRemoveItem_NonExistentSKU_NoOp(t *testing.T) {
	skuID := uuid.New()
	existing := newTestCart(newTestCartItem(skuID, testSellerID, 2, 1000))
	store := &mockCartStore{cart: existing}
	svc := newService(store, nil, nil)

	cart, err := svc.RemoveItem(context.Background(), testTenantID, testBuyerID, uuid.New())
	if err != nil {
		t.Fatalf("RemoveItem(non-existent) error = %v, want nil", err)
	}
	if len(cart.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1 (unchanged)", len(cart.Items))
	}
	if cart.Items[0].SKUID != skuID {
		t.Errorf("SKUID = %v, want %v (unchanged)", cart.Items[0].SKUID, skuID)
	}
}

// ---------------------------------------------------------------------------
// ClearCart
// ---------------------------------------------------------------------------

func TestClearCart_Success(t *testing.T) {
	store := &mockCartStore{}
	svc := newService(store, nil, nil)

	cart, err := svc.ClearCart(context.Background(), testTenantID, testBuyerID)
	if err != nil {
		t.Fatalf("ClearCart() error = %v, want nil", err)
	}
	if store.deleteCalls != 1 {
		t.Errorf("Delete called %d times, want 1", store.deleteCalls)
	}
	if cart.TenantID != testTenantID {
		t.Errorf("TenantID = %v, want %v", cart.TenantID, testTenantID)
	}
	if cart.BuyerAuth0ID != testBuyerID {
		t.Errorf("BuyerAuth0ID = %q, want %q", cart.BuyerAuth0ID, testBuyerID)
	}
	if len(cart.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(cart.Items))
	}
}

func TestClearCart_DeleteError(t *testing.T) {
	store := &mockCartStore{deleteErr: errors.New("redis down")}
	svc := newService(store, nil, nil)

	_, err := svc.ClearCart(context.Background(), testTenantID, testBuyerID)
	if err == nil {
		t.Fatal("ClearCart() error = nil, want error")
	}
}

// ---------------------------------------------------------------------------
// Checkout
// ---------------------------------------------------------------------------

func TestCheckout_Success(t *testing.T) {
	skuID1 := uuid.New()
	skuID2 := uuid.New()
	sellerID1 := uuid.New()
	sellerID2 := uuid.New()
	orderID1 := uuid.New()
	orderID2 := uuid.New()

	existing := newTestCart(
		newTestCartItem(skuID1, sellerID1, 2, 1000),
		newTestCartItem(skuID2, sellerID2, 1, 3000),
	)
	store := &mockCartStore{cart: existing}
	orderClient := &mockCheckoutClient{
		result: &domain.CheckoutResult{
			OrderIDs:              []uuid.UUID{orderID1, orderID2},
			StripeClientSecret:    "pi_secret_test",
			StripePaymentIntentID: "pi_test_123",
			TotalAmount:           5000,
			Currency:              "jpy",
		},
	}
	svc := newService(store, nil, orderClient)

	shippingAddr := json.RawMessage(`{"city":"Tokyo"}`)
	result, err := svc.Checkout(context.Background(), testTenantID, testBuyerID, shippingAddr, "jpy")
	if err != nil {
		t.Fatalf("Checkout() error = %v, want nil", err)
	}

	// Verify result fields.
	if len(result.OrderIDs) != 2 {
		t.Errorf("len(OrderIDs) = %d, want 2", len(result.OrderIDs))
	}
	if result.TotalAmount != 5000 {
		t.Errorf("TotalAmount = %d, want 5000", result.TotalAmount)
	}
	if result.Currency != "jpy" {
		t.Errorf("Currency = %q, want %q", result.Currency, "jpy")
	}
	if result.StripeClientSecret != "pi_secret_test" {
		t.Errorf("StripeClientSecret = %q, want %q", result.StripeClientSecret, "pi_secret_test")
	}
	if result.StripePaymentIntentID != "pi_test_123" {
		t.Errorf("StripePaymentIntentID = %q, want %q", result.StripePaymentIntentID, "pi_test_123")
	}

	// Verify checkout input was built correctly.
	if orderClient.calledWith == nil {
		t.Fatal("CreateCheckout was not called")
	}
	if orderClient.calledWith.BuyerAuth0ID != testBuyerID {
		t.Errorf("CheckoutInput.BuyerAuth0ID = %q, want %q", orderClient.calledWith.BuyerAuth0ID, testBuyerID)
	}
	if len(orderClient.calledWith.Lines) != 2 {
		t.Errorf("len(CheckoutInput.Lines) = %d, want 2", len(orderClient.calledWith.Lines))
	}

	// Verify cart was deleted after checkout.
	if store.deleteCalls != 1 {
		t.Errorf("Delete called %d times, want 1", store.deleteCalls)
	}
}

func TestCheckout_EmptyCart(t *testing.T) {
	existing := newTestCart() // no items
	store := &mockCartStore{cart: existing}
	svc := newService(store, nil, nil)

	_, err := svc.Checkout(context.Background(), testTenantID, testBuyerID, nil, "jpy")
	if !errors.Is(err, domain.ErrEmptyCart) {
		t.Errorf("Checkout(empty cart) error = %v, want %v", err, domain.ErrEmptyCart)
	}
}

func TestCheckout_NilCart(t *testing.T) {
	store := &mockCartStore{cart: nil}
	svc := newService(store, nil, nil)

	_, err := svc.Checkout(context.Background(), testTenantID, testBuyerID, nil, "jpy")
	if !errors.Is(err, domain.ErrEmptyCart) {
		t.Errorf("Checkout(nil cart) error = %v, want %v", err, domain.ErrEmptyCart)
	}
}

func TestCheckout_OrderServiceError(t *testing.T) {
	existing := newTestCart(newTestCartItem(testSKUID, testSellerID, 1, 1000))
	store := &mockCartStore{cart: existing}
	orderClient := &mockCheckoutClient{err: errors.New("order service unavailable")}
	svc := newService(store, nil, orderClient)

	_, err := svc.Checkout(context.Background(), testTenantID, testBuyerID, nil, "jpy")
	if err == nil {
		t.Fatal("Checkout() error = nil, want error from order service")
	}
	// Cart should NOT be deleted on checkout failure.
	if store.deleteCalls != 0 {
		t.Errorf("Delete called %d times, want 0 (cart preserved on failure)", store.deleteCalls)
	}
}

func TestCheckout_RepoGetError(t *testing.T) {
	store := &mockCartStore{getErr: errors.New("redis read failed")}
	svc := newService(store, nil, nil)

	_, err := svc.Checkout(context.Background(), testTenantID, testBuyerID, nil, "jpy")
	if err == nil {
		t.Fatal("Checkout() error = nil, want error from repo")
	}
}

func TestCheckout_DefaultCurrency(t *testing.T) {
	skuID := uuid.New()
	item := newTestCartItem(skuID, testSellerID, 1, 500)
	item.Currency = "usd"
	existing := newTestCart(item)
	store := &mockCartStore{cart: existing}
	orderClient := &mockCheckoutClient{
		result: &domain.CheckoutResult{
			OrderIDs:    []uuid.UUID{uuid.New()},
			TotalAmount: 500,
			Currency:    "usd",
		},
	}
	svc := newService(store, nil, orderClient)

	// Pass empty currency -- service should pick from cart item.
	_, err := svc.Checkout(context.Background(), testTenantID, testBuyerID, nil, "")
	if err != nil {
		t.Fatalf("Checkout() error = %v, want nil", err)
	}
	if orderClient.calledWith.Currency != "usd" {
		t.Errorf("CheckoutInput.Currency = %q, want %q (from cart item)", orderClient.calledWith.Currency, "usd")
	}
}
