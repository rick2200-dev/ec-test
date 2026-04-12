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
	GetCart(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.Cart, error)
	AddItem(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, skuID uuid.UUID, quantity int) (*domain.Cart, error)
	UpdateItemQuantity(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, skuID uuid.UUID, quantity int) (*domain.Cart, error)
	RemoveItem(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, skuID uuid.UUID) (*domain.Cart, error)
	ClearCart(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.Cart, error)
	Checkout(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, shippingAddress json.RawMessage, currency string) (*domain.CheckoutResult, error)
}
