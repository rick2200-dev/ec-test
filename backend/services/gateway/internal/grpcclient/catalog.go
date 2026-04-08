package grpcclient

import (
	"context"

	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	catalogv1 "github.com/Riku-KANO/ec-test/gen/go/catalog/v1"
)

// ListProducts calls the catalog gRPC service to list products.
func (c *GRPCClients) ListProducts(
	ctx context.Context,
	tenantID, sellerID, status, categoryID string,
	limit, offset int32,
) (*catalogv1.ListProductsResponse, error) {
	req := &catalogv1.ListProductsRequest{
		TenantId:   tenantID,
		SellerId:   sellerID,
		Status:     status,
		CategoryId: categoryID,
		Pagination: &commonv1.PaginationRequest{
			Limit:  limit,
			Offset: offset,
		},
	}
	return c.CatalogClient.ListProducts(ctx, req)
}

// GetProduct calls the catalog gRPC service to get a product by slug.
func (c *GRPCClients) GetProduct(ctx context.Context, tenantID, slug string) (*catalogv1.GetProductResponse, error) {
	req := &catalogv1.GetProductRequest{
		TenantId: tenantID,
		Identifier: &catalogv1.GetProductRequest_Slug{
			Slug: slug,
		},
	}
	return c.CatalogClient.GetProduct(ctx, req)
}
