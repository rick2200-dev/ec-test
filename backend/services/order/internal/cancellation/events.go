package cancellation

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// Pub/Sub topic that carries every order lifecycle event.
// Kept in one place so the constant cannot drift between call sites.
const orderEventsTopic = "order-events"

// Event type names for the order-cancellation lifecycle. These strings
// are the switch-on values for every downstream subscriber
// (notification, inventory) and are therefore part of the public
// event contract — renaming is a breaking change.
const (
	EventTypeCancellationRequested = "order.cancellation_requested"
	EventTypeCancellationApproved  = "order.cancellation_approved"
	EventTypeCancellationRejected  = "order.cancellation_rejected"
	EventTypeOrderCancelled        = "order.cancelled"
)

// CancelledLineItem is the minimal per-line snapshot embedded in the
// order.cancelled event so the inventory subscriber can release stock
// without an RPC callback to the order service.
type CancelledLineItem struct {
	SKUID       uuid.UUID `json:"sku_id"`
	ProductName string    `json:"product_name"`
	SKUCode     string    `json:"sku_code"`
	Quantity    int       `json:"quantity"`
}

// publishRequested fires order.cancellation_requested after a buyer
// successfully opens a new request. Consumers: notification.
func publishRequested(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order) {
	pubsub.PublishEvent(ctx, pub, req.TenantID, EventTypeCancellationRequested, orderEventsTopic, map[string]any{
		"request_id":        req.ID.String(),
		"order_id":          req.OrderID.String(),
		"tenant_id":         req.TenantID.String(),
		"seller_id":         order.SellerID.String(),
		"buyer_auth0_id":    req.RequestedByAuth0ID,
		"reason":            req.Reason,
	})
}

// publishRejected fires order.cancellation_rejected after a seller
// rejects a request. Consumers: notification.
func publishRejected(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order) {
	comment := ""
	if req.SellerComment != nil {
		comment = *req.SellerComment
	}
	pubsub.PublishEvent(ctx, pub, req.TenantID, EventTypeCancellationRejected, orderEventsTopic, map[string]any{
		"request_id":     req.ID.String(),
		"order_id":       req.OrderID.String(),
		"tenant_id":      req.TenantID.String(),
		"seller_id":      order.SellerID.String(),
		"buyer_auth0_id": req.RequestedByAuth0ID,
		"seller_comment": comment,
	})
}

// publishApproved fires order.cancellation_approved after the
// approval orchestration (Stripe + DB) finishes successfully.
// Consumers: notification.
func publishApproved(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order, refundAmount int64) {
	refundID := ""
	if req.StripeRefundID != nil {
		refundID = *req.StripeRefundID
	}
	pubsub.PublishEvent(ctx, pub, req.TenantID, EventTypeCancellationApproved, orderEventsTopic, map[string]any{
		"request_id":       req.ID.String(),
		"order_id":         req.OrderID.String(),
		"tenant_id":        req.TenantID.String(),
		"seller_id":        order.SellerID.String(),
		"buyer_auth0_id":   req.RequestedByAuth0ID,
		"stripe_refund_id": refundID,
		"refund_amount":    refundAmount,
	})
}

// publishOrderCancelled fires order.cancelled with per-line snapshots
// so the inventory subscriber can release stock without a callback.
// Consumers: notification, inventory.
//
// The payload intentionally embeds product_name / sku_code alongside
// sku_id / quantity so the notification email does not need to look
// up the catalog (orders are immutable for line_items after creation,
// so these snapshots are already authoritative).
func publishOrderCancelled(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order, lines []domain.OrderLine) {
	cancelledLines := make([]CancelledLineItem, 0, len(lines))
	for _, l := range lines {
		cancelledLines = append(cancelledLines, CancelledLineItem{
			SKUID:       l.SKUID,
			ProductName: l.ProductName,
			SKUCode:     l.SKUCode,
			Quantity:    l.Quantity,
		})
	}

	var cancelledAt any
	if order.CancelledAt != nil {
		cancelledAt = order.CancelledAt.UTC()
	}

	pubsub.PublishEvent(ctx, pub, req.TenantID, EventTypeOrderCancelled, orderEventsTopic, map[string]any{
		"order_id":       order.ID.String(),
		"tenant_id":      order.TenantID.String(),
		"seller_id":      order.SellerID.String(),
		"buyer_auth0_id": order.BuyerAuth0ID,
		"request_id":     req.ID.String(),
		"reason":         req.Reason,
		"line_items":     cancelledLines,
		"cancelled_at":   cancelledAt,
	})
}
