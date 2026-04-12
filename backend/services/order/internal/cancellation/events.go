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

// Typed event structs for the cancellation lifecycle. Using structs instead
// of map[string]any makes the event schema self-documenting and catches field
// name typos at compile time. The JSON tags are the public contract and must
// not be changed without coordinating downstream subscribers.

type cancellationRequestedEvent struct {
	RequestID    string `json:"request_id"`
	OrderID      string `json:"order_id"`
	TenantID     string `json:"tenant_id"`
	SellerID     string `json:"seller_id"`
	BuyerAuth0ID string `json:"buyer_auth0_id"`
	Reason       string `json:"reason"`
}

type cancellationRejectedEvent struct {
	RequestID     string `json:"request_id"`
	OrderID       string `json:"order_id"`
	TenantID      string `json:"tenant_id"`
	SellerID      string `json:"seller_id"`
	BuyerAuth0ID  string `json:"buyer_auth0_id"`
	SellerComment string `json:"seller_comment"`
}

type cancellationApprovedEvent struct {
	RequestID      string `json:"request_id"`
	OrderID        string `json:"order_id"`
	TenantID       string `json:"tenant_id"`
	SellerID       string `json:"seller_id"`
	BuyerAuth0ID   string `json:"buyer_auth0_id"`
	StripeRefundID string `json:"stripe_refund_id"`
	RefundAmount   int64  `json:"refund_amount"`
}

type orderCancelledEvent struct {
	OrderID      string              `json:"order_id"`
	TenantID     string              `json:"tenant_id"`
	SellerID     string              `json:"seller_id"`
	BuyerAuth0ID string              `json:"buyer_auth0_id"`
	RequestID    string              `json:"request_id"`
	Reason       string              `json:"reason"`
	LineItems    []CancelledLineItem `json:"line_items"`
	CancelledAt  *string             `json:"cancelled_at,omitempty"`
}

// publishRequested fires order.cancellation_requested after a buyer
// successfully opens a new request. Consumers: notification.
func publishRequested(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order) {
	pubsub.PublishEvent(ctx, pub, req.TenantID, EventTypeCancellationRequested, orderEventsTopic, cancellationRequestedEvent{
		RequestID:    req.ID.String(),
		OrderID:      req.OrderID.String(),
		TenantID:     req.TenantID.String(),
		SellerID:     order.SellerID.String(),
		BuyerAuth0ID: req.RequestedByAuth0ID,
		Reason:       req.Reason,
	})
}

// publishRejected fires order.cancellation_rejected after a seller
// rejects a request. Consumers: notification.
func publishRejected(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order) {
	comment := ""
	if req.SellerComment != nil {
		comment = *req.SellerComment
	}
	pubsub.PublishEvent(ctx, pub, req.TenantID, EventTypeCancellationRejected, orderEventsTopic, cancellationRejectedEvent{
		RequestID:     req.ID.String(),
		OrderID:       req.OrderID.String(),
		TenantID:      req.TenantID.String(),
		SellerID:      order.SellerID.String(),
		BuyerAuth0ID:  req.RequestedByAuth0ID,
		SellerComment: comment,
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
	pubsub.PublishEvent(ctx, pub, req.TenantID, EventTypeCancellationApproved, orderEventsTopic, cancellationApprovedEvent{
		RequestID:      req.ID.String(),
		OrderID:        req.OrderID.String(),
		TenantID:       req.TenantID.String(),
		SellerID:       order.SellerID.String(),
		BuyerAuth0ID:   req.RequestedByAuth0ID,
		StripeRefundID: refundID,
		RefundAmount:   refundAmount,
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

	evt := orderCancelledEvent{
		OrderID:      order.ID.String(),
		TenantID:     order.TenantID.String(),
		SellerID:     order.SellerID.String(),
		BuyerAuth0ID: order.BuyerAuth0ID,
		RequestID:    req.ID.String(),
		Reason:       req.Reason,
		LineItems:    cancelledLines,
	}
	if order.CancelledAt != nil {
		s := order.CancelledAt.UTC().String()
		evt.CancelledAt = &s
	}

	pubsub.PublishEvent(ctx, pub, req.TenantID, EventTypeOrderCancelled, orderEventsTopic, evt)
}
