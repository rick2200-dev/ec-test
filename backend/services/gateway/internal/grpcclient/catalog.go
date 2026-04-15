package grpcclient

import (
	"context"

	catalogv1 "github.com/Riku-KANO/ec-test/services/catalog/api/gen/go/catalog/v1"
	commonv1 "github.com/Riku-KANO/ec-test/shared/api/gen/go/common/v1"
)

// ListProducts calls the catalog gRPC service to list products.
func (c *GRPCClients) ListProducts(
	ctx context.Context,
	sellerID, status, categoryID string,
	limit, offset int32,
) (*catalogv1.ListProductsResponse, error) {
	req := &catalogv1.ListProductsRequest{
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
func (c *GRPCClients) GetProduct(ctx context.Context, slug string) (*catalogv1.GetProductResponse, error) {
	req := &catalogv1.GetProductRequest{
		Identifier: &catalogv1.GetProductRequest_Slug{
			Slug: slug,
		},
	}
	return c.CatalogClient.GetProduct(ctx, req)
}
