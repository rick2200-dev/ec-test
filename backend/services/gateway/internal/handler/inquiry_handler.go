package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// InquiryHandler proxies buyer↔seller inquiry requests to the inquiry
// micro-service. The inquiry service owns purchase verification and
// participant access control.
type InquiryHandler struct {
	inquiry *proxy.ServiceClient
}

// NewInquiryHandler creates a new InquiryHandler.
func NewInquiryHandler(svc *proxy.Services) *InquiryHandler {
	return &InquiryHandler{inquiry: svc.Inquiry}
}

// --- Buyer routes ---

// BuyerList proxies GET /buyer/inquiries.
func (h *InquiryHandler) BuyerList(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.inquiry.Get(r.Context(), "/inquiries", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// BuyerCreate proxies POST /buyer/inquiries.
func (h *InquiryHandler) BuyerCreate(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.inquiry.Post(r.Context(), "/inquiries", r.Body)
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// BuyerGet proxies GET /buyer/inquiries/{id}.
func (h *InquiryHandler) BuyerGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.inquiry.Get(r.Context(), "/inquiries/"+url.PathEscape(id), "")
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// BuyerPostMessage proxies POST /buyer/inquiries/{id}/messages.
func (h *InquiryHandler) BuyerPostMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.inquiry.Post(r.Context(), "/inquiries/"+url.PathEscape(id)+"/messages", r.Body)
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// BuyerMarkRead proxies POST /buyer/inquiries/{id}/read.
func (h *InquiryHandler) BuyerMarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.inquiry.Post(r.Context(), "/inquiries/"+url.PathEscape(id)+"/read", nil)
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// --- Seller routes ---

// SellerList proxies GET /seller/inquiries.
func (h *InquiryHandler) SellerList(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.inquiry.Get(r.Context(), "/seller/inquiries", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// SellerGet proxies GET /seller/inquiries/{id}.
func (h *InquiryHandler) SellerGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.inquiry.Get(r.Context(), "/seller/inquiries/"+url.PathEscape(id), "")
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// SellerPostMessage proxies POST /seller/inquiries/{id}/messages.
func (h *InquiryHandler) SellerPostMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.inquiry.Post(r.Context(), "/seller/inquiries/"+url.PathEscape(id)+"/messages", r.Body)
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// SellerMarkRead proxies POST /seller/inquiries/{id}/read.
func (h *InquiryHandler) SellerMarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.inquiry.Post(r.Context(), "/seller/inquiries/"+url.PathEscape(id)+"/read", nil)
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// SellerClose proxies POST /seller/inquiries/{id}/close.
func (h *InquiryHandler) SellerClose(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.inquiry.Post(r.Context(), "/seller/inquiries/"+url.PathEscape(id)+"/close", nil)
	if err != nil {
		slog.Error("proxy to inquiry failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inquiry service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
