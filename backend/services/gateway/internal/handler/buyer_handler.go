package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	body, st, err := h.order.Get(r.Context(), "/orders/buyer", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, st, body)
}

// GetOrder returns a single order with catalog-enriched line items so the
// buyer detail page can render product images and product-page links for
// live products while still showing a "deleted" placeholder for products
// that have been archived or removed from the catalog since checkout.
// GET /orders/{id}
//
// Flow:
//  1. Proxy the raw order body from order-svc (snapshot data: seller_name,
//     product_name, product_id per line).
//  2. Fan out one catalog gRPC GetProduct call per line in parallel via
//     errgroup. Missing / archived products map to is_deleted=true with
//     blank image+slug; live products pick up image_url + slug.
//  3. Product NAME always comes from the order_line snapshot — never the
//     current catalog — so renaming a product after purchase doesn't
//     rewrite the buyer's history.
func (h *BuyerHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing tenant context"})
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "order id required"})
		return
	}

	body, st, err := h.order.Get(r.Context(), "/orders/"+url.PathEscape(id), "")
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	if st != http.StatusOK {
		// Pass through non-200 responses from order-svc (NotFound, etc.)
		// verbatim so error shape matches the upstream handler.
		writeRaw(w, st, body)
		return
	}

	var detail orderDetailJSON
	if err := json.Unmarshal(body, &detail); err != nil {
		slog.Error("decode order body", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "invalid upstream response"})
		return
	}

	// Buyer ownership check: order-svc's GET /orders/{id} is only
	// tenant-scoped, so it would happily return another buyer's order
	// within the same tenant. Reject any mismatch with 404 (rather than
	// 403) to avoid leaking the existence of the id to probes.
	if detail.BuyerAuth0ID != tc.UserID {
		httputil.JSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}

	// Enrich each line. A missing or archived product degrades to
	// is_deleted=true; other errors also degrade (logged) so a single
	// flaky catalog lookup doesn't fail the whole order view.
	enriched := make([]enrichedLineJSON, len(detail.Lines))
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(r.Context())
	for i, line := range detail.Lines {
		g.Go(func() error {
			el := enrichedLineJSON{
				ID:          line.ID,
				SKUID:       line.SKUID,
				ProductID:   line.ProductID,
				ProductName: line.ProductName,
				SKUCode:     line.SKUCode,
				Quantity:    line.Quantity,
				UnitPrice:   line.UnitPrice,
				LineTotal:   line.LineTotal,
			}

			// Historical rows backfilled by migration 000013 may carry
			// the nil UUID sentinel when their original SKU no longer
			// exists in catalog_svc.skus. Short-circuit to deleted
			// without issuing a guaranteed-NotFound gRPC call.
			if line.ProductID == nilUUIDString {
				el.IsDeleted = true
				mu.Lock()
				enriched[i] = el
				mu.Unlock()
				return nil
			}

			resp, err := h.catalogGRPC.GetProduct(gctx, &catalogv1.GetProductRequest{
				TenantId:   tc.TenantID.String(),
				Identifier: &catalogv1.GetProductRequest_Id{Id: line.ProductID},
			})
			if err != nil {
				if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
					slog.Warn("catalog GetProduct enrich failed", "product_id", line.ProductID, "error", err)
				}
				el.IsDeleted = true
			} else if p := resp.GetProduct(); p == nil || p.GetStatus() == "archived" {
				el.IsDeleted = true
			} else {
				el.ImageURL = p.GetImageUrl()
				el.ProductSlug = p.GetSlug()
			}

			mu.Lock()
			enriched[i] = el
			mu.Unlock()
			return nil
		})
	}
	// All goroutines swallow their errors (graceful degradation), so
	// g.Wait() never actually returns a non-nil error here — but we
	// still wait to ensure all writes to `enriched` have completed.
	_ = g.Wait()

	resp := orderDetailResponseJSON{
		ID:                    detail.ID,
		TenantID:              detail.TenantID,
		SellerID:              detail.SellerID,
		SellerName:            detail.SellerName,
		Status:                detail.Status,
		SubtotalAmount:        detail.SubtotalAmount,
		ShippingFee:           detail.ShippingFee,
		CommissionAmount:      detail.CommissionAmount,
		TotalAmount:           detail.TotalAmount,
		Currency:              detail.Currency,
		ShippingAddress:       detail.ShippingAddress,
		StripePaymentIntentID: detail.StripePaymentIntentID,
		PaidAt:                detail.PaidAt,
		CreatedAt:             detail.CreatedAt,
		UpdatedAt:             detail.UpdatedAt,
		Lines:                 enriched,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// RequestOrderCancellation proxies to the order service to open a buyer
// cancellation request for an order. The buyer's identity and ownership
// are re-verified inside order-svc using the forwarded tenant headers.
// POST /orders/{id}/cancellation-request
func (h *BuyerHandler) RequestOrderCancellation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "order id required"})
		return
	}
	body, st, err := h.order.Post(r.Context(), "/orders/"+url.PathEscape(id)+"/cancellation-request", r.Body)
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, st, body)
}

// GetOrderCancellationRequest proxies to the order service to fetch the
// latest cancellation request for the given order. Returns 404 if none
// exists. Ownership is re-checked inside order-svc.
// GET /orders/{id}/cancellation-request
func (h *BuyerHandler) GetOrderCancellationRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "order id required"})
		return
	}
	body, st, err := h.order.Get(r.Context(), "/orders/"+url.PathEscape(id)+"/cancellation-request", "")
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, st, body)
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
