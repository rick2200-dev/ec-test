package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// ReviewHandler proxies buyer and seller review requests to the review
// micro-service. The review service owns purchase verification and
// authorization.
type ReviewHandler struct {
	review *proxy.ServiceClient
}

// NewReviewHandler creates a new ReviewHandler.
func NewReviewHandler(svc *proxy.Services) *ReviewHandler {
	return &ReviewHandler{review: svc.Review}
}

// --- Buyer routes ---

// BuyerCreate proxies POST /buyer/reviews.
func (h *ReviewHandler) BuyerCreate(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.review.Post(r.Context(), "/reviews", r.Body)
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// BuyerGet proxies GET /buyer/reviews/{id}.
func (h *ReviewHandler) BuyerGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.review.Get(r.Context(), "/reviews/"+url.PathEscape(id), "")
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// BuyerUpdate proxies PUT /buyer/reviews/{id}.
func (h *ReviewHandler) BuyerUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.review.Put(r.Context(), "/reviews/"+url.PathEscape(id), r.Body)
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// BuyerDelete proxies DELETE /buyer/reviews/{id}.
func (h *ReviewHandler) BuyerDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.review.Delete(r.Context(), "/reviews/"+url.PathEscape(id))
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListByProduct proxies GET /buyer/products/{productId}/reviews.
func (h *ReviewHandler) ListByProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	body, status, err := h.review.Get(r.Context(), "/reviews/product/"+url.PathEscape(productID), r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// GetProductRating proxies GET /buyer/products/{productId}/rating.
func (h *ReviewHandler) GetProductRating(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	body, status, err := h.review.Get(r.Context(), "/reviews/product/"+url.PathEscape(productID)+"/rating", "")
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// --- Seller routes ---

// SellerList proxies GET /seller/reviews.
func (h *ReviewHandler) SellerList(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.review.Get(r.Context(), "/seller/reviews", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// SellerGet proxies GET /seller/reviews/{id}.
func (h *ReviewHandler) SellerGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.review.Get(r.Context(), "/seller/reviews/"+url.PathEscape(id), "")
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// SellerCreateReply proxies POST /seller/reviews/{id}/reply.
func (h *ReviewHandler) SellerCreateReply(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.review.Post(r.Context(), "/seller/reviews/"+url.PathEscape(id)+"/reply", r.Body)
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// SellerUpdateReply proxies PUT /seller/reviews/{id}/reply.
func (h *ReviewHandler) SellerUpdateReply(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.review.Put(r.Context(), "/seller/reviews/"+url.PathEscape(id)+"/reply", r.Body)
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// SellerDeleteReply proxies DELETE /seller/reviews/{id}/reply.
func (h *ReviewHandler) SellerDeleteReply(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.review.Delete(r.Context(), "/seller/reviews/"+url.PathEscape(id)+"/reply")
	if err != nil {
		slog.Error("proxy to review failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "review service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
