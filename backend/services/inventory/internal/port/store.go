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
	// GetBySKUID retrieves the inventory record for the given SKU within the tenant.
	GetBySKUID(ctx context.Context, tenantID, skuID uuid.UUID) (*domain.Inventory, error)
	// List returns a paginated list of inventory records owned by the seller.
	List(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Inventory, int, error)
	// Upsert inserts or updates an inventory record (used when setting absolute stock levels).
	Upsert(ctx context.Context, tenantID uuid.UUID, inv *domain.Inventory) error
	// Reserve decrements available stock and increments reserved stock when a buyer places an order.
	Reserve(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	// Release moves reserved stock back to available (e.g. on order cancellation before payment).
	Release(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	// ConfirmSold decrements reserved stock permanently when payment is confirmed.
	ConfirmSold(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error
	// RecordMovement appends an immutable stock movement audit entry.
	RecordMovement(ctx context.Context, tenantID uuid.UUID, m *domain.StockMovement) error
	// ReleaseForOrderCancellation atomically releases reserved stock for all lines of a cancelled order.
	// Returns alreadyReleased=true if the order was already processed (idempotent).
	ReleaseForOrderCancellation(ctx context.Context, tenantID, orderID uuid.UUID, lines []domain.CancellationLine) (alreadyReleased bool, err error)
}
