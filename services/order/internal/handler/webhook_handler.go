package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/order/internal/service"
)

// WebhookHandler handles Stripe webhook events.
type WebhookHandler struct {
	svc           *service.OrderService
	webhookSecret string
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(svc *service.OrderService, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		svc:           svc,
		webhookSecret: webhookSecret,
	}
}

// HandleStripeWebhook handles POST /webhooks/stripe.
func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read webhook body", "error", err)
		httputil.JSON(w, http.StatusServiceUnavailable, map[string]string{"error": "failed to read body"})
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sigHeader, h.webhookSecret)
	if err != nil {
		slog.Error("failed to verify webhook signature", "error", err)
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid signature"})
		return
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var pi gostripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			slog.Error("failed to unmarshal payment intent", "error", err)
			httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event data"})
			return
		}

		if err := h.svc.HandlePaymentSuccess(r.Context(), pi.ID); err != nil {
			slog.Error("failed to handle payment success", "error", err, "payment_intent", pi.ID)
			httputil.Error(w, err)
			return
		}

		slog.Info("payment_intent.succeeded processed", "payment_intent", pi.ID)

	default:
		slog.Debug("unhandled webhook event type", "type", event.Type)
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
