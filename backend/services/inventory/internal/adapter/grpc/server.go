package grpcserver

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	inventoryv1 "github.com/Riku-KANO/ec-test/gen/go/inventory/v1"
	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/port"
)

// InventoryServer implements the InventoryServiceServer gRPC interface.
type InventoryServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
	svc port.InventoryUseCase
}

// NewInventoryServer creates a new InventoryServer.
func NewInventoryServer(svc port.InventoryUseCase) *InventoryServer {
	return &InventoryServer{svc: svc}
}

// GetInventory retrieves inventory for a specific SKU.
func (s *InventoryServer) GetInventory(ctx context.Context, req *inventoryv1.GetInventoryRequest) (*inventoryv1.GetInventoryResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	skuID, err := uuid.Parse(req.GetSkuId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sku_id")
	}

	inv, err := s.svc.GetInventory(ctx, tenantID, skuID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &inventoryv1.GetInventoryResponse{
		Item: domainInventoryToProto(inv),
	}, nil
}

// ListInventory returns a paginated list of inventory for a seller.
func (s *InventoryServer) ListInventory(ctx context.Context, req *inventoryv1.ListInventoryRequest) (*inventoryv1.ListInventoryResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	sellerID, err := uuid.Parse(req.GetSellerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid seller_id")
	}

	limit := int32(20)
	offset := int32(0)
	if req.GetPagination() != nil {
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
		offset = req.GetPagination().GetOffset()
	}

	items, total, err := s.svc.ListInventory(ctx, tenantID, sellerID, int(limit), int(offset))
	if err != nil {
		return nil, toGRPCError(err)
	}

	var pbItems []*inventoryv1.InventoryItem
	for i := range items {
		pbItems = append(pbItems, domainInventoryToProto(&items[i]))
	}

	return &inventoryv1.ListInventoryResponse{
		Items: pbItems,
		Pagination: &commonv1.PaginationResponse{
			Total:  int32(total),
			Limit:  limit,
			Offset: offset,
		},
	}, nil
}

// UpdateStock upserts an inventory record.
func (s *InventoryServer) UpdateStock(ctx context.Context, req *inventoryv1.UpdateStockRequest) (*inventoryv1.UpdateStockResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	inv := protoUpdateStockToDomain(req)

	if err := s.svc.UpdateStock(ctx, tenantID, inv); err != nil {
		return nil, toGRPCError(err)
	}

	// Retrieve updated inventory to return.
	updated, err := s.svc.GetInventory(ctx, tenantID, inv.SKUID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &inventoryv1.UpdateStockResponse{
		Item: domainInventoryToProto(updated),
	}, nil
}

// ReserveStock reserves quantity for a SKU.
func (s *InventoryServer) ReserveStock(ctx context.Context, req *inventoryv1.ReserveStockRequest) (*inventoryv1.ReserveStockResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	skuID, err := uuid.Parse(req.GetSkuId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sku_id")
	}

	if err := s.svc.ReserveStock(ctx, tenantID, skuID, int(req.GetQuantity())); err != nil {
		return nil, toGRPCError(err)
	}

	// Retrieve updated inventory to return.
	inv, err := s.svc.GetInventory(ctx, tenantID, skuID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &inventoryv1.ReserveStockResponse{
		Success: true,
		Item:    domainInventoryToProto(inv),
	}, nil
}

// ReleaseStock releases reserved stock.
func (s *InventoryServer) ReleaseStock(ctx context.Context, req *inventoryv1.ReleaseStockRequest) (*inventoryv1.ReleaseStockResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	skuID, err := uuid.Parse(req.GetSkuId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sku_id")
	}

	if err := s.svc.ReleaseStock(ctx, tenantID, skuID, int(req.GetQuantity())); err != nil {
		return nil, toGRPCError(err)
	}

	// Retrieve updated inventory to return.
	inv, err := s.svc.GetInventory(ctx, tenantID, skuID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &inventoryv1.ReleaseStockResponse{
		Item: domainInventoryToProto(inv),
	}, nil
}

// ConfirmSold confirms that reserved stock has been sold.
func (s *InventoryServer) ConfirmSold(ctx context.Context, req *inventoryv1.ConfirmSoldRequest) (*inventoryv1.ConfirmSoldResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	skuID, err := uuid.Parse(req.GetSkuId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sku_id")
	}

	if err := s.svc.ConfirmSold(ctx, tenantID, skuID, int(req.GetQuantity())); err != nil {
		return nil, toGRPCError(err)
	}

	// Retrieve updated inventory to return.
	inv, err := s.svc.GetInventory(ctx, tenantID, skuID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &inventoryv1.ConfirmSoldResponse{
		Item: domainInventoryToProto(inv),
	}, nil
}

// toGRPCError converts an application error to a gRPC status error.
func toGRPCError(err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		switch {
		case errors.Is(appErr.Err, apperrors.ErrNotFound):
			return status.Error(codes.NotFound, appErr.Message)
		case errors.Is(appErr.Err, apperrors.ErrBadRequest):
			return status.Error(codes.InvalidArgument, appErr.Message)
		case errors.Is(appErr.Err, apperrors.ErrConflict):
			return status.Error(codes.AlreadyExists, appErr.Message)
		case errors.Is(appErr.Err, apperrors.ErrForbidden):
			return status.Error(codes.PermissionDenied, appErr.Message)
		case errors.Is(appErr.Err, apperrors.ErrUnauthorized):
			return status.Error(codes.Unauthenticated, appErr.Message)
		default:
			return status.Error(codes.Internal, appErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
