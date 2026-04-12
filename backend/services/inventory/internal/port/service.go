package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
)

// InventoryUseCase is the driving port (inbound) for inventory operations.
// Handlers and the gRPC server depend on this interface;
// *service.InventoryService satisfies it.
type InventoryUseCase interface {
	// GetInventory returns the inventory record for the given SKU.
	GetInventory(ctx context.Context, tenantID, skuID uuid.UUID) (*domain.Inventory, error)
	// ListInventory returns a paginated list of inventory records for the seller.
	ListInventory(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Inventory, int, error)
	// UpdateStock sets absolute stock levels for a SKU and records a manual adjustment movement.
	UpdateStock(ctx context.Context, tenantID uuid.UUID, inv *domain.Inventory) error
	// ReserveStock decrements available stock and increments reserved stock; called when an order is created.
	ReserveStock(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	// ReleaseStock moves reserved stock back to available; called when an order is cancelled before payment.
	ReleaseStock(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	// ReleaseStockForOrderCancellation atomically releases reserved stock for all lines of the cancelled order.
	ReleaseStockForOrderCancellation(ctx context.Context, tenantID, orderID uuid.UUID, lines []domain.CancellationLine) error
	// ConfirmSold decrements reserved stock permanently; called after payment is confirmed.
	ConfirmSold(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
}
