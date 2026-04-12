package port

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/cart/internal/domain"
)

// CartUseCase is the driving port (inbound) for cart operations.
// Handlers depend on this interface; *service.CartService satisfies it.
type CartUseCase interface {
	// GetCart returns the current cart for the buyer; an empty cart is returned if none exists yet.
	GetCart(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.Cart, error)
	// AddItem adds the specified SKU to the cart, creating the cart if necessary.
	AddItem(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, skuID uuid.UUID, quantity int) (*domain.Cart, error)
	// UpdateItemQuantity sets the quantity of the given SKU line; a quantity of zero removes the line.
	UpdateItemQuantity(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, skuID uuid.UUID, quantity int) (*domain.Cart, error)
	// RemoveItem deletes the specified SKU line from the cart.
	RemoveItem(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, skuID uuid.UUID) (*domain.Cart, error)
	// ClearCart removes all items from the cart.
	ClearCart(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.Cart, error)
	// Checkout converts the cart into one or more orders via the order service and returns the checkout result.
	Checkout(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, shippingAddress json.RawMessage, currency string) (*domain.CheckoutResult, error)
}
