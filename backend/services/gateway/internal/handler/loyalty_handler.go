package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// LoyaltyHandler proxies buyer-facing and admin-facing point endpoints
// to the loyalty service. The underlying ServiceClient already attaches
// X-Internal-Token, and ServiceClient.do() forwards the authenticated
// X-User-ID header — loyalty uses it as the buyer identifier.
type LoyaltyHandler struct {
	loyalty *proxy.ServiceClient
}

func NewLoyaltyHandler(svc *proxy.Services) *LoyaltyHandler {
	return &LoyaltyHandler{loyalty: svc.Loyalty}
}

// GetBalance proxies to loyalty /points/balance.
// GET /api/v1/buyer/points/balance
func (h *LoyaltyHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.loyalty.Get(r.Context(), "/points/balance", "")
	if err != nil {
		slog.Error("proxy to loyalty failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "loyalty service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListTransactions proxies to loyalty /points/transactions.
// GET /api/v1/buyer/points/transactions
func (h *LoyaltyHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.loyalty.Get(r.Context(), "/points/transactions", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to loyalty failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "loyalty service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// AdminGetBalance proxies admin balance lookup keyed by buyer auth0 id.
// GET /api/v1/admin/loyalty/buyers/{buyer_auth0_id}/balance
func (h *LoyaltyHandler) AdminGetBalance(w http.ResponseWriter, r *http.Request) {
	buyer := chi.URLParam(r, "buyer_auth0_id")
	if buyer == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "buyer_auth0_id is required"})
		return
	}
	body, status, err := h.loyalty.Get(r.Context(), "/points/buyers/"+url.PathEscape(buyer)+"/balance", "")
	if err != nil {
		slog.Error("proxy to loyalty failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "loyalty service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// AdminListTransactions proxies admin transaction history by buyer.
// GET /api/v1/admin/loyalty/buyers/{buyer_auth0_id}/transactions
func (h *LoyaltyHandler) AdminListTransactions(w http.ResponseWriter, r *http.Request) {
	buyer := chi.URLParam(r, "buyer_auth0_id")
	if buyer == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "buyer_auth0_id is required"})
		return
	}
	body, status, err := h.loyalty.Get(r.Context(), "/points/buyers/"+url.PathEscape(buyer)+"/transactions", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to loyalty failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "loyalty service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// AdminAdjust proxies a signed manual adjustment to the buyer's balance.
// POST /api/v1/admin/loyalty/buyers/{buyer_auth0_id}/adjust
func (h *LoyaltyHandler) AdminAdjust(w http.ResponseWriter, r *http.Request) {
	buyer := chi.URLParam(r, "buyer_auth0_id")
	if buyer == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "buyer_auth0_id is required"})
		return
	}
	body, status, err := h.loyalty.Post(r.Context(), "/points/buyers/"+url.PathEscape(buyer)+"/adjust", r.Body)
	if err != nil {
		slog.Error("proxy to loyalty failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "loyalty service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
