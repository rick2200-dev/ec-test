package grpcserver

import (
	"encoding/json"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	catalogv1 "github.com/Riku-KANO/ec-test/gen/go/catalog/v1"
	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
)

// domainProductToProto converts a domain Product to a proto Product.
func domainProductToProto(p *domain.Product) *catalogv1.Product {
	pb := &catalogv1.Product{
		Id:          p.ID.String(),
		TenantId:    p.TenantID.String(),
		SellerId:    p.SellerID.String(),
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Status:      string(p.Status),
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
	if len(p.Attributes) > 0 {
		pb.AttributesJson = string(p.Attributes)
	}
	if p.ImageURL != nil {
		pb.ImageUrl = *p.ImageURL
	}
	return pb
}

// domainProductWithSKUsToProto converts a domain ProductWithSKUs to a proto Product.
func domainProductWithSKUsToProto(p *domain.ProductWithSKUs) *catalogv1.Product {
	pb := domainProductToProto(&p.Product)
	for _, s := range p.SKUs {
		pb.Skus = append(pb.Skus, domainSKUToProto(&s))
	}
	return pb
}

// domainSKUToProto converts a domain SKU to a proto SKU.
func domainSKUToProto(s *domain.SKU) *catalogv1.SKU {
	pb := &catalogv1.SKU{
		Id:        s.ID.String(),
		TenantId:  s.TenantID.String(),
		ProductId: s.ProductID.String(),
		SellerId:  s.SellerID.String(),
		SkuCode:   s.SKUCode,
		Price: &commonv1.Money{
			Amount:   s.PriceAmount,
			Currency: s.PriceCurrency,
		},
		Status:    string(s.Status),
		CreatedAt: timestamppb.New(s.CreatedAt),
		UpdatedAt: timestamppb.New(s.UpdatedAt),
	}
	if len(s.Attributes) > 0 {
		pb.AttributesJson = string(s.Attributes)
	}
	return pb
}

// protoCreateProductToDomain converts a CreateProductRequest to domain Product and SKUs.
func protoCreateProductToDomain(req *catalogv1.CreateProductRequest) (*domain.Product, []domain.SKU) {
	sellerID, _ := uuid.Parse(req.GetSellerId())

	p := &domain.Product{
		SellerID:    sellerID,
		Name:        req.GetName(),
		Slug:        req.GetSlug(),
		Description: req.GetDescription(),
	}
	if req.GetAttributesJson() != "" {
		p.Attributes = json.RawMessage(req.GetAttributesJson())
	}

	var skus []domain.SKU
	for _, s := range req.GetSkus() {
		sku := domain.SKU{
			SKUCode: s.GetSkuCode(),
		}
		if s.GetPrice() != nil {
			sku.PriceAmount = s.GetPrice().GetAmount()
			sku.PriceCurrency = s.GetPrice().GetCurrency()
		}
		if s.GetAttributesJson() != "" {
			sku.Attributes = json.RawMessage(s.GetAttributesJson())
		}
		skus = append(skus, sku)
	}
	return p, skus
}

// protoUpdateProductToDomain converts an UpdateProductRequest to a domain Product.
func protoUpdateProductToDomain(req *catalogv1.UpdateProductRequest) *domain.Product {
	id, _ := uuid.Parse(req.GetId())
	p := &domain.Product{
		ID:          id,
		Name:        req.GetName(),
		Description: req.GetDescription(),
	}
	if req.GetAttributesJson() != "" {
		p.Attributes = json.RawMessage(req.GetAttributesJson())
	}
	return p
}

// domainCategoryToProto converts a domain Category to a proto Category.
func domainCategoryToProto(c *domain.Category) *catalogv1.Category {
	pb := &catalogv1.Category{
		Id:        c.ID.String(),
		TenantId:  c.TenantID.String(),
		Name:      c.Name,
		Slug:      c.Slug,
		SortOrder: int32(c.SortOrder),
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
	if c.ParentID != nil {
		pb.ParentId = c.ParentID.String()
	}
	return pb
}

// protoCreateCategoryToDomain converts a CreateCategoryRequest to a domain Category.
func protoCreateCategoryToDomain(req *catalogv1.CreateCategoryRequest) *domain.Category {
	c := &domain.Category{
		Name:      req.GetName(),
		Slug:      req.GetSlug(),
		SortOrder: int(req.GetSortOrder()),
	}
	if req.GetParentId() != "" {
		parentID, err := uuid.Parse(req.GetParentId())
		if err == nil {
			c.ParentID = &parentID
		}
	}
	return c
}
