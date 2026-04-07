package grpcserver

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/Riku-KANO/ec-test/gen/go/catalog/v1"
	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/repository"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/service"
)

// CatalogServer implements the CatalogServiceServer gRPC interface.
type CatalogServer struct {
	catalogv1.UnimplementedCatalogServiceServer
	svc *service.CatalogService
}

// NewCatalogServer creates a new CatalogServer.
func NewCatalogServer(svc *service.CatalogService) *CatalogServer {
	return &CatalogServer{svc: svc}
}

// ListProducts returns a paginated list of products for a tenant.
func (s *CatalogServer) ListProducts(ctx context.Context, req *catalogv1.ListProductsRequest) (*catalogv1.ListProductsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	filter := repository.ProductFilter{
		TenantID: tenantID,
	}
	if req.GetSellerId() != "" {
		sellerID, err := uuid.Parse(req.GetSellerId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid seller_id")
		}
		filter.SellerID = &sellerID
	}
	if req.GetStatus() != "" {
		st := domain.ProductStatus(req.GetStatus())
		filter.Status = &st
	}
	if req.GetCategoryId() != "" {
		catID, err := uuid.Parse(req.GetCategoryId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid category_id")
		}
		filter.CategoryID = &catID
	}

	limit := int32(20)
	offset := int32(0)
	if req.GetPagination() != nil {
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
		offset = req.GetPagination().GetOffset()
	}

	products, total, err := s.svc.ListProducts(ctx, filter, int(limit), int(offset))
	if err != nil {
		return nil, toGRPCError(err)
	}

	var pbProducts []*catalogv1.Product
	for i := range products {
		pbProducts = append(pbProducts, domainProductToProto(&products[i]))
	}

	return &catalogv1.ListProductsResponse{
		Products: pbProducts,
		Pagination: &commonv1.PaginationResponse{
			Total:  int32(total),
			Limit:  limit,
			Offset: offset,
		},
	}, nil
}

// GetProduct retrieves a product by ID or slug.
func (s *CatalogServer) GetProduct(ctx context.Context, req *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	switch v := req.GetIdentifier().(type) {
	case *catalogv1.GetProductRequest_Slug:
		p, err := s.svc.GetProduct(ctx, tenantID, v.Slug)
		if err != nil {
			return nil, toGRPCError(err)
		}
		return &catalogv1.GetProductResponse{
			Product: domainProductWithSKUsToProto(p),
		}, nil
	case *catalogv1.GetProductRequest_Id:
		id, err := uuid.Parse(v.Id)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid product id")
		}
		p, err := s.svc.GetProductByID(ctx, tenantID, id)
		if err != nil {
			return nil, toGRPCError(err)
		}
		return &catalogv1.GetProductResponse{
			Product: domainProductToProto(p),
		}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "identifier (id or slug) is required")
	}
}

// CreateProduct creates a new product with optional SKUs.
func (s *CatalogServer) CreateProduct(ctx context.Context, req *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	product, skus := protoCreateProductToDomain(req)

	if err := s.svc.CreateProduct(ctx, tenantID, product, skus); err != nil {
		return nil, toGRPCError(err)
	}

	return &catalogv1.CreateProductResponse{
		Product: domainProductToProto(product),
	}, nil
}

// UpdateProduct updates a product's details.
func (s *CatalogServer) UpdateProduct(ctx context.Context, req *catalogv1.UpdateProductRequest) (*catalogv1.UpdateProductResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	product := protoUpdateProductToDomain(req)

	if err := s.svc.UpdateProduct(ctx, tenantID, product); err != nil {
		return nil, toGRPCError(err)
	}

	return &catalogv1.UpdateProductResponse{
		Product: domainProductToProto(product),
	}, nil
}

// UpdateProductStatus changes a product's status.
func (s *CatalogServer) UpdateProductStatus(ctx context.Context, req *catalogv1.UpdateProductStatusRequest) (*catalogv1.UpdateProductStatusResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	productID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product id")
	}

	st := domain.ProductStatus(req.GetStatus())

	if err := s.svc.UpdateProductStatus(ctx, tenantID, productID, st); err != nil {
		return nil, toGRPCError(err)
	}

	// Retrieve updated product to return.
	p, err := s.svc.GetProductByID(ctx, tenantID, productID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &catalogv1.UpdateProductStatusResponse{
		Product: domainProductToProto(p),
	}, nil
}

// ListCategories returns all categories for a tenant.
func (s *CatalogServer) ListCategories(ctx context.Context, req *catalogv1.ListCategoriesRequest) (*catalogv1.ListCategoriesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	categories, err := s.svc.ListCategories(ctx, tenantID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	var pbCategories []*catalogv1.Category
	for i := range categories {
		pbCategories = append(pbCategories, domainCategoryToProto(&categories[i]))
	}

	return &catalogv1.ListCategoriesResponse{
		Categories: pbCategories,
	}, nil
}

// CreateCategory creates a new category.
func (s *CatalogServer) CreateCategory(ctx context.Context, req *catalogv1.CreateCategoryRequest) (*catalogv1.CreateCategoryResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	category := protoCreateCategoryToDomain(req)

	if err := s.svc.CreateCategory(ctx, tenantID, category); err != nil {
		return nil, toGRPCError(err)
	}

	return &catalogv1.CreateCategoryResponse{
		Category: domainCategoryToProto(category),
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
