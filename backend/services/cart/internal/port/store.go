// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the cart service. Adapters in adapter/* implement or consume these
// interfaces; the app layer (service/) depends only on these contracts.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/cart/internal/domain"
)

// CartStore is the driven port for cart persistence.
// *repository.CartRepository satisfies this interface.
type CartStore interface {
	// Get retrieves the cart for the given buyer within the tenant.
	// Returns a nil pointer and no error when the cart does not yet exist.
	Get(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.Cart, error)
	// Save persists (insert-or-update) the given cart.
	Save(ctx context.Context, cart *domain.Cart) error
	// Delete removes the cart for the given buyer within the tenant.
	Delete(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) error
}

// SKULookup is the catalog SKU data the cart needs when adding an item.
// It is defined here — on the port boundary — so both the app layer and the
// httpclient adapter refer to the same type without a circular import.
type SKULookup struct {
	SKUID         uuid.UUID `json:"id"`
	ProductID     uuid.UUID `json:"product_id"`
	SellerID      uuid.UUID `json:"seller_id"`
	ProductName   string    `json:"product_name"`
	SKUCode       string    `json:"sku_code"`
	PriceAmount   int64     `json:"price_amount"`
	PriceCurrency string    `json:"price_currency"`
	Status        string    `json:"status"`
}

// SKULookupClient is the driven port for catalog SKU lookups.
// *httpclient.CatalogClient satisfies this interface.
type SKULookupClient interface {
	// LookupSKU fetches SKU metadata from the catalog service for the given tenant and SKU ID.
	LookupSKU(ctx context.Context, tenantID, skuID uuid.UUID) (*SKULookup, error)
}

// CheckoutClient is the driven port for order creation.
// *httpclient.OrderClient satisfies this interface.
type CheckoutClient interface {
	// CreateCheckout sends a checkout request to the order service and returns the resulting order summary.
	CreateCheckout(ctx context.Context, tenantID uuid.UUID, in domain.CheckoutInput) (*domain.CheckoutResult, error)
}
