package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	catalogv1 "github.com/Riku-KANO/ec-test/gen/go/catalog/v1"
	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// BuyerHandler handles buyer-facing routes.
//
// The catalog read path (ListProducts, GetProduct) has been migrated to gRPC
// as a pilot — it calls svc.CatalogGRPC directly instead of proxying via
// HTTP. All other buyer routes still proxy HTTP through the ServiceClient
// fields below.
type BuyerHandler struct {
	catalogGRPC catalogv1.CatalogServiceClient
	order       *proxy.ServiceClient
	recommend   *proxy.ServiceClient
	search      *proxy.ServiceClient
	auth        *proxy.ServiceClient
}

// NewBuyerHandler creates a new BuyerHandler.
func NewBuyerHandler(svc *proxy.Services) *BuyerHandler {
	return &BuyerHandler{
		catalogGRPC: svc.CatalogGRPC,
		order:       svc.Order,
		recommend:   svc.Recommend,
		search:      svc.Search,
		auth:        svc.Auth,
	}
}

// ListProducts calls the catalog gRPC service to list active products.
// GET /products
//
// The proto response is converted back to the REST JSON shape that the
// catalog HTTP handler previously produced, so the frontend contract is
// unchanged.
func (h *BuyerHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing tenant context"})
		return
	}

	p := pagination.FromRequest(r)

	req := &catalogv1.ListProductsRequest{
		TenantId: tc.TenantID.String(),
		Status:   "active",
		Pagination: &commonv1.PaginationRequest{
			Limit:  int32(p.Limit),
			Offset: int32(p.Offset),
		},
	}
	if sellerID := r.URL.Query().Get("seller_id"); sellerID != "" {
		req.SellerId = sellerID
	}
	if categoryID := r.URL.Query().Get("category_id"); categoryID != "" {
		req.CategoryId = categoryID
	}

	resp, err := h.catalogGRPC.ListProducts(r.Context(), req)
	if err != nil {
		writeGRPCError(w, "catalog ListProducts", err)
		return
	}

	items := make([]productJSON, 0, len(resp.GetProducts()))
	for _, p := range resp.GetProducts() {
		items = append(items, protoProductToJSON(p))
	}

	httputil.JSON(w, http.StatusOK, pagination.Response[productJSON]{
		Items:  items,
		Total:  int(resp.GetPagination().GetTotal()),
		Limit:  int(resp.GetPagination().GetLimit()),
		Offset: int(resp.GetPagination().GetOffset()),
	})
}

// GetProduct calls the catalog gRPC service to fetch a single product by
// slug, including its SKUs.
// GET /products/{slug}
func (h *BuyerHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing tenant context"})
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		slug = r.PathValue("slug")
	}

	req := &catalogv1.GetProductRequest{
		TenantId:   tc.TenantID.String(),
		Identifier: &catalogv1.GetProductRequest_Slug{Slug: slug},
	}

	resp, err := h.catalogGRPC.GetProduct(r.Context(), req)
	if err != nil {
		writeGRPCError(w, "catalog GetProduct", err)
		return
	}

	httputil.JSON(w, http.StatusOK, protoProductWithSKUsToJSON(resp.GetProduct()))
}

// SearchProducts proxies to the search service.
// GET /search
func (h *BuyerHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.search.Get(r.Context(), "/search", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to search failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "search service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListOrders proxies to the order service to list the buyer's orders.
// GET /orders
func (h *BuyerHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.order.Get(r.Context(), "/orders/buyer", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// TrackEvent proxies to the recommend service to record user behavior events.
// POST /events
func (h *BuyerHandler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.recommend.Post(r.Context(), "/events", r.Body)
	if err != nil {
		slog.Error("proxy to recommend failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "recommend service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// GetRecommendations proxies to the recommend service to fetch recommendations.
// GET /recommendations
func (h *BuyerHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.recommend.Get(r.Context(), "/recommendations", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to recommend failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "recommend service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListBuyerPlans proxies to the auth service to list buyer subscription plans.
// GET /plans
func (h *BuyerHandler) ListBuyerPlans(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.auth.Get(r.Context(), "/buyer-plans", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// GetSubscription proxies to the auth service to get the buyer's current subscription.
// GET /subscription
func (h *BuyerHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing tenant context"})
		return
	}
	body, status, pErr := h.auth.Get(r.Context(), "/buyer-subscriptions/buyers/"+url.PathEscape(tc.UserID), "")
	if pErr != nil {
		slog.Error("proxy to auth failed", "error", pErr)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Subscribe proxies to the auth service to subscribe the buyer to a plan.
// POST /subscription
func (h *BuyerHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing tenant context"})
		return
	}
	body, status, pErr := h.auth.Post(r.Context(), "/buyer-subscriptions/buyers/"+url.PathEscape(tc.UserID), r.Body)
	if pErr != nil {
		slog.Error("proxy to auth failed", "error", pErr)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
