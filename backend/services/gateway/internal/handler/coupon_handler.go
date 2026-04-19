package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// CouponHandler proxies platform-admin CRUD and buyer preview/list
// endpoints to the coupon service. The underlying ServiceClient
// attaches the shared X-Internal-Token and forwards the authenticated
// X-User-ID (buyer auth0 sub) so the coupon service can enforce the
// per-user usage limit.
type CouponHandler struct {
	coupon *proxy.ServiceClient
}

func NewCouponHandler(svc *proxy.Services) *CouponHandler {
	return &CouponHandler{coupon: svc.Coupon}
}

// --- Buyer ---

// Preview proxies POST /buyer/coupons/preview.
func (h *CouponHandler) Preview(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.coupon.Post(r.Context(), "/buyer/coupons/preview", r.Body)
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListMyRedemptions proxies GET /buyer/coupons/redemptions.
func (h *CouponHandler) ListMyRedemptions(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.coupon.Get(r.Context(), "/buyer/coupons/redemptions", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// --- Admin ---

func (h *CouponHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.coupon.Post(r.Context(), "/admin/coupons/", r.Body)
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

func (h *CouponHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.coupon.Get(r.Context(), "/admin/coupons/", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

func (h *CouponHandler) AdminGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.coupon.Get(r.Context(), "/admin/coupons/"+url.PathEscape(id), "")
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

func (h *CouponHandler) AdminRevoke(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.coupon.Post(r.Context(), "/admin/coupons/"+url.PathEscape(id)+"/revoke", r.Body)
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

func (h *CouponHandler) AdminStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.coupon.Get(r.Context(), "/admin/coupons/"+url.PathEscape(id)+"/stats", "")
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// --- Seller ---
//
// The ServiceClient forwards X-Seller-ID (populated from the caller's
// tenant context) automatically; the coupon service uses it to bind
// every write to the authenticated seller and to scope list queries.

func (h *CouponHandler) SellerCreate(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.coupon.Post(r.Context(), "/seller/coupons/", r.Body)
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

func (h *CouponHandler) SellerList(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.coupon.Get(r.Context(), "/seller/coupons/", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

func (h *CouponHandler) SellerGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.coupon.Get(r.Context(), "/seller/coupons/"+url.PathEscape(id), "")
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

func (h *CouponHandler) SellerRevoke(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.coupon.Post(r.Context(), "/seller/coupons/"+url.PathEscape(id)+"/revoke", r.Body)
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

func (h *CouponHandler) SellerStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.coupon.Get(r.Context(), "/seller/coupons/"+url.PathEscape(id)+"/stats", "")
	if err != nil {
		slog.Error("proxy to coupon failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "coupon service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
