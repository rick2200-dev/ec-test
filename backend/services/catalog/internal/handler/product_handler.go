package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/repository"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/service"
)

// ProductHandler handles HTTP requests for product operations.
type ProductHandler struct {
	svc *service.CatalogService
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(svc *service.CatalogService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// Routes returns the chi router for product endpoints.
func (h *ProductHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{slug}", h.GetBySlug)
	r.Put("/{id}", h.Update)
	r.Put("/{id}/status", h.UpdateStatus)
	return r
}

// createProductRequest is the request body for creating a product.
type createProductRequest struct {
	SellerID    uuid.UUID       `json:"seller_id"`
	CategoryID  *uuid.UUID      `json:"category_id,omitempty"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description"`
	Attributes  json.RawMessage `json:"attributes,omitempty"`
	SKUs        []createSKUReq  `json:"skus,omitempty"`
}

// createSKUReq is the SKU portion of a create product request.
type createSKUReq struct {
	SKUCode       string          `json:"sku_code"`
	PriceAmount   int64           `json:"price_amount"`
	PriceCurrency string          `json:"price_currency"`
	Attributes    json.RawMessage `json:"attributes,omitempty"`
}

// Create handles POST /products.
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	var req createProductRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	p := &domain.Product{
		SellerID:    req.SellerID,
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Attributes:  req.Attributes,
	}

	var skus []domain.SKU
	for _, s := range req.SKUs {
		skus = append(skus, domain.SKU{
			SKUCode:       s.SKUCode,
			PriceAmount:   s.PriceAmount,
			PriceCurrency: s.PriceCurrency,
			Attributes:    s.Attributes,
		})
	}

	if err := h.svc.CreateProduct(r.Context(), tenantID, p, skus); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, p)
}

// GetBySlug handles GET /products/{slug}.
func (h *ProductHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	slug := chi.URLParam(r, "slug")
	p, err := h.svc.GetProduct(r.Context(), tenantID, slug)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, p)
}

// List handles GET /products.
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	p := pagination.FromRequest(r)

	filter := repository.ProductFilter{
		TenantID: tenantID,
	}

	// Optional query filters.
	if sellerStr := r.URL.Query().Get("seller_id"); sellerStr != "" {
		sid, err := uuid.Parse(sellerStr)
		if err == nil {
			filter.SellerID = &sid
		}
	}
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := domain.ProductStatus(statusStr)
		filter.Status = &status
	}
	if catStr := r.URL.Query().Get("category_id"); catStr != "" {
		cid, err := uuid.Parse(catStr)
		if err == nil {
			filter.CategoryID = &cid
		}
	}

	products, total, err := h.svc.ListProducts(r.Context(), filter, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	resp := pagination.Response[domain.Product]{
		Items:  products,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// updateProductRequest is the request body for updating a product.
type updateProductRequest struct {
	CategoryID  *uuid.UUID      `json:"category_id,omitempty"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description"`
	Attributes  json.RawMessage `json:"attributes,omitempty"`
}

// Update handles PUT /products/{id}.
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	var req updateProductRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	p := &domain.Product{
		ID:          id,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Attributes:  req.Attributes,
	}

	if err := h.svc.UpdateProduct(r.Context(), tenantID, p); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, p)
}

// updateStatusRequest is the request body for updating product status.
type updateStatusRequest struct {
	Status domain.ProductStatus `json:"status"`
}

// UpdateStatus handles PUT /products/{id}/status.
func (h *ProductHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	var req updateStatusRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.svc.UpdateProductStatus(r.Context(), tenantID, id, req.Status); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": string(req.Status)})
}
