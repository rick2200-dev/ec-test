package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/cart/internal/domain"
	"github.com/Riku-KANO/ec-test/services/cart/internal/repository"
)

// CartService implements cart business logic: read/write the buyer's
// cart in Redis and drive the multi-seller checkout through the order
// service.
type CartService struct {
	cartRepo      *repository.CartRepository
	catalogClient *CatalogClient
	orderClient   *OrderClient
	publisher     pubsub.Publisher
}

// NewCartService constructs a CartService.
func NewCartService(
	cartRepo *repository.CartRepository,
	catalogClient *CatalogClient,
	orderClient *OrderClient,
	publisher pubsub.Publisher,
) *CartService {
	return &CartService{
		cartRepo:      cartRepo,
		catalogClient: catalogClient,
		orderClient:   orderClient,
		publisher:     publisher,
	}
}

// GetCart returns the buyer's current cart, or an empty cart if none exists.
func (s *CartService) GetCart(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.Cart, error) {
	cart, err := s.cartRepo.Get(ctx, tenantID, buyerAuth0ID)
	if err != nil {
		return nil, apperrors.Internal("failed to load cart", err)
	}
	if cart == nil {
		return &domain.Cart{
			TenantID:     tenantID,
			BuyerAuth0ID: buyerAuth0ID,
			Items:        []domain.CartItem{},
			UpdatedAt:    time.Now().UTC(),
		}, nil
	}
	return cart, nil
}

// AddItem adds or increments a SKU in the buyer's cart. Quantity must be
// positive. If the SKU is already in the cart, its quantity is increased
// by the supplied amount (price snapshot is NOT refreshed).
func (s *CartService) AddItem(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, skuID uuid.UUID, quantity int) (*domain.Cart, error) {
	if quantity <= 0 {
		return nil, apperrors.BadRequest("quantity must be positive")
	}

	cart, err := s.GetCart(ctx, tenantID, buyerAuth0ID)
	if err != nil {
		return nil, err
	}

	if idx := cart.FindItem(skuID); idx >= 0 {
		cart.Items[idx].Quantity += quantity
	} else {
		sku, err := s.catalogClient.LookupSKU(ctx, tenantID, skuID)
		if err != nil {
			return nil, err
		}
		cart.Items = append(cart.Items, domain.CartItem{
			SKUID:               sku.SKUID,
			SellerID:            sku.SellerID,
			Quantity:            quantity,
			UnitPriceSnapshot:   sku.PriceAmount,
			Currency:            sku.PriceCurrency,
			ProductNameSnapshot: sku.ProductName,
			SKUCodeSnapshot:     sku.SKUCode,
			AddedAt:             time.Now().UTC(),
		})
	}

	if err := s.cartRepo.Save(ctx, cart); err != nil {
		return nil, apperrors.Internal("failed to save cart", err)
	}
	return cart, nil
}

// UpdateItemQuantity sets the absolute quantity of a SKU in the cart.
// Passing quantity == 0 is equivalent to RemoveItem.
func (s *CartService) UpdateItemQuantity(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, skuID uuid.UUID, quantity int) (*domain.Cart, error) {
	if quantity < 0 {
		return nil, apperrors.BadRequest("quantity must be non-negative")
	}
	if quantity == 0 {
		return s.RemoveItem(ctx, tenantID, buyerAuth0ID, skuID)
	}

	cart, err := s.GetCart(ctx, tenantID, buyerAuth0ID)
	if err != nil {
		return nil, err
	}

	idx := cart.FindItem(skuID)
	if idx < 0 {
		return nil, apperrors.NotFound("sku not in cart")
	}
	cart.Items[idx].Quantity = quantity

	if err := s.cartRepo.Save(ctx, cart); err != nil {
		return nil, apperrors.Internal("failed to save cart", err)
	}
	return cart, nil
}

// RemoveItem removes a SKU from the cart. Removing a non-existent SKU is
// a no-op (returns the cart unchanged).
func (s *CartService) RemoveItem(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, skuID uuid.UUID) (*domain.Cart, error) {
	cart, err := s.GetCart(ctx, tenantID, buyerAuth0ID)
	if err != nil {
		return nil, err
	}

	idx := cart.FindItem(skuID)
	if idx < 0 {
		return cart, nil
	}
	cart.Items = append(cart.Items[:idx], cart.Items[idx+1:]...)

	if err := s.cartRepo.Save(ctx, cart); err != nil {
		return nil, apperrors.Internal("failed to save cart", err)
	}
	return cart, nil
}

// ClearCart removes all items from the cart by deleting the Redis key.
func (s *CartService) ClearCart(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.Cart, error) {
	if err := s.cartRepo.Delete(ctx, tenantID, buyerAuth0ID); err != nil {
		return nil, apperrors.Internal("failed to clear cart", err)
	}
	return &domain.Cart{
		TenantID:     tenantID,
		BuyerAuth0ID: buyerAuth0ID,
		Items:        []domain.CartItem{},
		UpdatedAt:    time.Now().UTC(),
	}, nil
}

// Checkout converts the cart into orders. Flow:
//   1. Load cart (must be non-empty)
//   2. POST /internal/checkouts on the order service with all items + shipping
//   3. On success: delete cart, publish cart.checked_out, return result
//   4. On failure: leave cart intact, propagate error
//
// The order service is responsible for grouping by seller, creating one
// Order per seller in a single TenantTx, and calling Stripe exactly once.
func (s *CartService) Checkout(
	ctx context.Context,
	tenantID uuid.UUID,
	buyerAuth0ID string,
	shippingAddress json.RawMessage,
	currency string,
) (*domain.CheckoutResult, error) {
	cart, err := s.cartRepo.Get(ctx, tenantID, buyerAuth0ID)
	if err != nil {
		return nil, apperrors.Internal("failed to load cart", err)
	}
	if cart == nil || cart.IsEmpty() {
		return nil, apperrors.BadRequest("cart is empty")
	}

	lines := make([]domain.CheckoutLine, 0, len(cart.Items))
	for _, item := range cart.Items {
		lines = append(lines, domain.CheckoutLine{
			SKUID:               item.SKUID,
			SellerID:            item.SellerID,
			Quantity:            item.Quantity,
			UnitPriceSnapshot:   item.UnitPriceSnapshot,
			ProductNameSnapshot: item.ProductNameSnapshot,
			SKUCodeSnapshot:     item.SKUCodeSnapshot,
		})
	}

	if currency == "" {
		if len(cart.Items) > 0 && cart.Items[0].Currency != "" {
			currency = cart.Items[0].Currency
		} else {
			currency = "jpy"
		}
	}

	input := domain.CheckoutInput{
		BuyerAuth0ID:        buyerAuth0ID,
		Currency:            currency,
		ShippingAddressJSON: shippingAddress,
		Lines:               lines,
	}

	result, err := s.orderClient.CreateCheckout(ctx, tenantID, input)
	if err != nil {
		return nil, err
	}

	// Clear cart only after the order service has committed.
	if err := s.cartRepo.Delete(ctx, tenantID, buyerAuth0ID); err != nil {
		// Non-fatal: orders exist, PaymentIntent exists — log and carry on.
		slog.Warn("failed to clear cart after checkout",
			"tenant_id", tenantID, "buyer_auth0_id", buyerAuth0ID, "error", err)
	}

	orderIDStrs := make([]string, 0, len(result.OrderIDs))
	for _, id := range result.OrderIDs {
		orderIDStrs = append(orderIDStrs, id.String())
	}

	pubsub.PublishEvent(ctx, s.publisher, tenantID, "cart.checked_out", "cart-events", map[string]any{
		"buyer_auth0_id":           buyerAuth0ID,
		"order_ids":                orderIDStrs,
		"stripe_payment_intent_id": result.StripePaymentIntentID,
		"total_amount":             result.TotalAmount,
		"currency":                 result.Currency,
	})

	slog.Info("cart checked out",
		"tenant_id", tenantID,
		"buyer_auth0_id", buyerAuth0ID,
		"order_count", len(result.OrderIDs),
		"total", result.TotalAmount,
	)

	return result, nil
}
