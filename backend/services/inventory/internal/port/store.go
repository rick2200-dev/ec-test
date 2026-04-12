// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the inventory service.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
)

// InventoryStore is the driven port for inventory persistence.
// *repository.InventoryRepository satisfies this interface.
type InventoryStore interface {
	GetBySKUID(ctx context.Context, tenantID, skuID uuid.UUID) (*domain.Inventory, error)
	List(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Inventory, int, error)
	Upsert(ctx context.Context, tenantID uuid.UUID, inv *domain.Inventory) error
	Reserve(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	Release(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	ConfirmSold(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	RecordMovement(ctx context.Context, tenantID uuid.UUID, m *domain.StockMovement) error
	ReleaseForOrderCancellation(ctx context.Context, tenantID, orderID uuid.UUID, lines []domain.CancellationLine) (alreadyReleased bool, err error)
}
