package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/repository"
)

// InventoryService implements business logic for inventory operations.
type InventoryService struct {
	repo *repository.InventoryRepository
}

// NewInventoryService creates a new InventoryService.
func NewInventoryService(repo *repository.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

// GetInventory retrieves inventory for a specific SKU.
func (s *InventoryService) GetInventory(ctx context.Context, tenantID, skuID uuid.UUID) (*domain.Inventory, error) {
	inv, err := s.repo.GetBySKUID(ctx, tenantID, skuID)
	if err != nil {
		return nil, apperrors.Internal("failed to get inventory", err)
	}
	if inv == nil {
		return nil, apperrors.NotFound("inventory not found")
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
	return nil
}

// ReserveStock reserves quantity for a SKU and records the movement.
func (s *InventoryService) ReserveStock(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return apperrors.BadRequest("quantity must be positive")
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
	return nil
}

// ReleaseStock releases reserved stock and records the movement.
func (s *InventoryService) ReleaseStock(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return apperrors.BadRequest("quantity must be positive")
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

// ConfirmSold confirms that reserved stock has been sold and records the movement.
func (s *InventoryService) ConfirmSold(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return apperrors.BadRequest("quantity must be positive")
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
