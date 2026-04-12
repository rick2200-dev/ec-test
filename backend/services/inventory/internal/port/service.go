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
	GetInventory(ctx context.Context, tenantID, skuID uuid.UUID) (*domain.Inventory, error)
	ListInventory(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Inventory, int, error)
	UpdateStock(ctx context.Context, tenantID uuid.UUID, inv *domain.Inventory) error
	ReserveStock(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	ReleaseStock(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	ReleaseStockForOrderCancellation(ctx context.Context, tenantID, orderID uuid.UUID, lines []domain.CancellationLine) error
	ConfirmSold(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
}
