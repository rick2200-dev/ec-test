package grpcclient

import (
	"context"
	"fmt"

	inventoryv1 "github.com/Riku-KANO/ec-test/gen/go/inventory/v1"
)

// ReserveStock calls the inventory gRPC service to reserve stock for a SKU.
func (c *GRPCClients) ReserveStock(ctx context.Context, tenantID, skuID string, quantity int32) error {
	req := &inventoryv1.ReserveStockRequest{
		TenantId: tenantID,
		SkuId:    skuID,
		Quantity: quantity,
	}
	resp, err := c.InventoryClient.ReserveStock(ctx, req)
	if err != nil {
		return fmt.Errorf("reserve stock: %w", err)
	}
	if !resp.GetSuccess() {
		return fmt.Errorf("reserve stock failed for sku %s", skuID)
	}
	return nil
}

// GetInventory calls the inventory gRPC service to get inventory for a SKU.
func (c *GRPCClients) GetInventory(ctx context.Context, tenantID, skuID string) (*inventoryv1.InventoryItem, error) {
	req := &inventoryv1.GetInventoryRequest{
		TenantId: tenantID,
		SkuId:    skuID,
	}
	resp, err := c.InventoryClient.GetInventory(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get inventory: %w", err)
	}
	return resp.GetItem(), nil
}
