package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/port"
)

// InventoryService implements business logic for inventory operations.
type InventoryService struct {
	repo      port.InventoryStore
	publisher pubsub.Publisher
}

// NewInventoryService creates a new InventoryService.
func NewInventoryService(repo port.InventoryStore, publisher pubsub.Publisher) *InventoryService {
	return &InventoryService{repo: repo, publisher: publisher}
}

// publishEvent publishes an event if the publisher is configured.
func (s *InventoryService) publishEvent(ctx context.Context, tenantID uuid.UUID, eventType, topic string, data any) {
	pubsub.PublishEvent(ctx, s.publisher, tenantID, eventType, topic, data)
}

// GetInventory retrieves inventory for a specific SKU.
func (s *InventoryService) GetInventory(ctx context.Context, tenantID, skuID uuid.UUID) (*domain.Inventory, error) {
	inv, err := s.repo.GetBySKUID(ctx, tenantID, skuID)
	if err != nil {
		return nil, apperrors.Internal("failed to get inventory", err)
	}
	if inv == nil {
		return nil, domain.ErrInventoryNotFound
	}
	return inv, nil
}

// ListInventory returns a paginated list of inventory for a seller.
func (s *InventoryService) ListInventory(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Inventory, int, error) {
	items, total, err := s.repo.List(ctx, tenantID, sellerID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list inventory", err)
	}
	return items, total, nil
}

// UpdateStock upserts an inventory record and records a movement.
func (s *InventoryService) UpdateStock(ctx context.Context, tenantID uuid.UUID, inv *domain.Inventory) error {
	if err := s.repo.Upsert(ctx, tenantID, inv); err != nil {
		return apperrors.Internal("failed to update stock", err)
	}

	movement := &domain.StockMovement{
		SKUID:         inv.SKUID,
		MovementType:  domain.MovementAdjusted,
		Quantity:      inv.QuantityAvailable,
		ReferenceType: "stock_update",
	}
	if err := s.repo.RecordMovement(ctx, tenantID, movement); err != nil {
		slog.Warn("failed to record stock movement", "error", err, "sku_id", inv.SKUID)
	}

	slog.Info("stock updated", "tenant_id", tenantID, "sku_id", inv.SKUID)

	s.publishEvent(ctx, tenantID, "inventory.updated", "inventory-events", map[string]any{
		"sku_id":             inv.SKUID.String(),
		"seller_id":         inv.SellerID.String(),
		"quantity_available": inv.QuantityAvailable,
		"quantity_reserved":  inv.QuantityReserved,
	})

	return nil
}

// ReserveStock reserves quantity for a SKU and records the movement.
func (s *InventoryService) ReserveStock(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return domain.ErrInvalidQuantity
	}

	if err := s.repo.Reserve(ctx, tenantID, skuID, quantity); err != nil {
		return apperrors.Conflict("failed to reserve stock: " + err.Error())
	}

	movement := &domain.StockMovement{
		SKUID:         skuID,
		MovementType:  domain.MovementReserved,
		Quantity:      quantity,
		ReferenceType: "reservation",
	}
	if err := s.repo.RecordMovement(ctx, tenantID, movement); err != nil {
		slog.Warn("failed to record reserve movement", "error", err, "sku_id", skuID)
	}

	slog.Info("stock reserved", "tenant_id", tenantID, "sku_id", skuID, "quantity", quantity)

	// Check for low stock after reservation.
	inv, invErr := s.repo.GetBySKUID(ctx, tenantID, skuID)
	if invErr == nil && inv != nil {
		eventData := map[string]any{
			"sku_id":             inv.SKUID.String(),
			"seller_id":         inv.SellerID.String(),
			"quantity_available": inv.QuantityAvailable,
			"quantity_reserved":  inv.QuantityReserved,
		}
		if inv.QuantityAvailable <= inv.LowStockThreshold {
			s.publishEvent(ctx, tenantID, "inventory.low_stock", "inventory-events", eventData)
		}
	}

	return nil
}

// ReleaseStock releases reserved stock and records the movement.
func (s *InventoryService) ReleaseStock(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return domain.ErrInvalidQuantity
	}

	if err := s.repo.Release(ctx, tenantID, skuID, quantity); err != nil {
		return apperrors.Conflict("failed to release stock: " + err.Error())
	}

	movement := &domain.StockMovement{
		SKUID:         skuID,
		MovementType:  domain.MovementReleased,
		Quantity:      quantity,
		ReferenceType: "release",
	}
	if err := s.repo.RecordMovement(ctx, tenantID, movement); err != nil {
		slog.Warn("failed to record release movement", "error", err, "sku_id", skuID)
	}

	slog.Info("stock released", "tenant_id", tenantID, "sku_id", skuID, "quantity", quantity)
	return nil
}

// ReleaseStockForOrderCancellation releases stock for every line of an
// order that has been cancelled by the order service. It is idempotent:
// a prior cancellation-movement for the same order_id makes the call a
// no-op, which is required because Pub/Sub delivery is at-least-once.
//
// This method is called from internal/subscriber/order_subscriber.go in
// response to order.cancelled events. See docs/order-cancellation.md for
// the end-to-end flow.
func (s *InventoryService) ReleaseStockForOrderCancellation(
	ctx context.Context,
	tenantID, orderID uuid.UUID,
	lines []domain.CancellationLine,
) error {
	if len(lines) == 0 {
		slog.Info("skipping cancellation release with no lines", "order_id", orderID)
		return nil
	}

	already, err := s.repo.ReleaseForOrderCancellation(ctx, tenantID, orderID, lines)
	if err != nil {
		return apperrors.Internal("release stock for order cancellation", err)
	}
	if already {
		slog.Info("order cancellation already released",
			"tenant_id", tenantID,
			"order_id", orderID,
		)
		return nil
	}

	slog.Info("order cancellation stock released",
		"tenant_id", tenantID,
		"order_id", orderID,
		"line_count", len(lines),
	)
	return nil
}

// ConfirmSold confirms that reserved stock has been sold and records the movement.
func (s *InventoryService) ConfirmSold(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return domain.ErrInvalidQuantity
	}

	if err := s.repo.ConfirmSold(ctx, tenantID, skuID, quantity); err != nil {
		return apperrors.Conflict("failed to confirm sold: " + err.Error())
	}

	movement := &domain.StockMovement{
		SKUID:         skuID,
		MovementType:  domain.MovementSold,
		Quantity:      quantity,
		ReferenceType: "sale",
	}
	if err := s.repo.RecordMovement(ctx, tenantID, movement); err != nil {
		slog.Warn("failed to record sold movement", "error", err, "sku_id", skuID)
	}

	slog.Info("stock sold confirmed", "tenant_id", tenantID, "sku_id", skuID, "quantity", quantity)
	return nil
}
