package grpcserver

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	inventoryv1 "github.com/Riku-KANO/ec-test/gen/go/inventory/v1"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
)

// domainInventoryToProto converts a domain Inventory to a proto InventoryItem.
func domainInventoryToProto(inv *domain.Inventory) *inventoryv1.InventoryItem {
	return &inventoryv1.InventoryItem{
		Id:                inv.ID.String(),
		TenantId:          inv.TenantID.String(),
		SkuId:             inv.SKUID.String(),
		SellerId:          inv.SellerID.String(),
		QuantityAvailable: int32(inv.QuantityAvailable),
		QuantityReserved:  int32(inv.QuantityReserved),
		LowStockThreshold: int32(inv.LowStockThreshold),
		UpdatedAt:         timestamppb.New(inv.UpdatedAt),
	}
}

// protoUpdateStockToDomain converts an UpdateStockRequest to a domain Inventory.
func protoUpdateStockToDomain(req *inventoryv1.UpdateStockRequest) *domain.Inventory {
	skuID, _ := uuid.Parse(req.GetSkuId())
	sellerID, _ := uuid.Parse(req.GetSellerId())

	return &domain.Inventory{
		SKUID:             skuID,
		SellerID:          sellerID,
		QuantityAvailable: int(req.GetQuantity()),
		LowStockThreshold: int(req.GetLowStockThreshold()),
	}
}
