package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/notification/internal/email"
	"github.com/Riku-KANO/ec-test/services/notification/internal/templates"
)

const orderSubscription = "order-events-notification"

// OrderSubscriber handles order-related events and sends notifications.
type OrderSubscriber struct {
	subscriber pubsub.Subscriber
	sender     email.Sender
}

// NewOrderSubscriber creates a new OrderSubscriber.
func NewOrderSubscriber(subscriber pubsub.Subscriber, sender email.Sender) *OrderSubscriber {
	return &OrderSubscriber{
		subscriber: subscriber,
		sender:     sender,
	}
}

// Start begins listening for order events. Blocks until context is cancelled.
func (s *OrderSubscriber) Start(ctx context.Context) error {
	slog.Info("starting order event subscriber", "subscription", orderSubscription)
	return s.subscriber.Subscribe(ctx, orderSubscription, s.handleEvent)
}

func (s *OrderSubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	slog.Info("received order event",
		"event_id", event.ID,
		"event_type", event.Type,
		"tenant_id", event.TenantID,
	)

	switch event.Type {
	case "order.created":
		return s.handleOrderCreated(ctx, event)
	case "order.paid":
		return s.handleOrderPaid(ctx, event)
	case "order.shipped":
		return s.handleOrderShipped(ctx, event)
	case "order.cancellation_requested":
		return s.handleCancellationRequested(ctx, event)
	case "order.cancellation_approved":
		return s.handleCancellationApproved(ctx, event)
	case "order.cancellation_rejected":
		return s.handleCancellationRejected(ctx, event)
	case "order.cancelled":
		return s.handleOrderCancelled(ctx, event)
	default:
		slog.Warn("unhandled order event type", "event_type", event.Type)
		return nil
	}
}

type orderCreatedData struct {
	OrderID     string `json:"order_id"`
	BuyerName   string `json:"buyer_name"`
	BuyerEmail  string `json:"buyer_email"`
	SellerName  string `json:"seller_name"`
	TotalAmount int64  `json:"total_amount"`
}

func (s *OrderSubscriber) handleOrderCreated(ctx context.Context, event pubsub.Event) error {
	var data orderCreatedData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.created data: %w", err)
	}

	subject, body := templates.OrderConfirmation(data.OrderID, data.BuyerName, data.TotalAmount, data.SellerName)

	slog.Info("sending order confirmation",
		"order_id", data.OrderID,
		"buyer_email", data.BuyerEmail,
		"tenant_id", event.TenantID,
	)

	if err := s.sender.Send(ctx, data.BuyerEmail, subject, body); err != nil {
		return fmt.Errorf("send order confirmation: %w", err)
	}
	return nil
}

type orderPaidData struct {
	OrderID     string `json:"order_id"`
	SellerName  string `json:"seller_name"`
	SellerEmail string `json:"seller_email"`
	TotalAmount int64  `json:"total_amount"`
}

func (s *OrderSubscriber) handleOrderPaid(ctx context.Context, event pubsub.Event) error {
	var data orderPaidData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.paid data: %w", err)
	}

	subject, body := templates.OrderPaidNotification(data.OrderID, data.SellerName, data.TotalAmount)

	slog.Info("sending payment notification to seller",
		"order_id", data.OrderID,
		"seller_email", data.SellerEmail,
		"tenant_id", event.TenantID,
	)

	if err := s.sender.Send(ctx, data.SellerEmail, subject, body); err != nil {
		return fmt.Errorf("send payment notification: %w", err)
	}
	return nil
}

type orderShippedData struct {
	OrderID    string `json:"order_id"`
	BuyerName  string `json:"buyer_name"`
	BuyerEmail string `json:"buyer_email"`
}

func (s *OrderSubscriber) handleOrderShipped(ctx context.Context, event pubsub.Event) error {
	var data orderShippedData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.shipped data: %w", err)
	}

	subject, body := templates.OrderShippedNotification(data.OrderID, data.BuyerName)

	slog.Info("sending shipping notification",
		"order_id", data.OrderID,
		"buyer_email", data.BuyerEmail,
		"tenant_id", event.TenantID,
	)

	if err := s.sender.Send(ctx, data.BuyerEmail, subject, body); err != nil {
		return fmt.Errorf("send shipping notification: %w", err)
	}
	return nil
}

// ─── Cancellation event handlers ─────────────────────────────
//
// These four handlers consume events published by the order service's
// cancellation package. Field names here must stay in sync with
// backend/services/order/internal/cancellation/events.go — that file is
// the source of truth for the payload contract.
//
// Today the LogSender just logs to stdout, so we use buyer_auth0_id /
// seller_id as the "to" address. When a real SMTP backend is wired in
// and user profiles are resolvable, these handlers will need to
// translate auth0 sub → email, same as the existing order.created
// handler will.

type cancellationRequestedData struct {
	OrderID      string `json:"order_id"`
	SellerID     string `json:"seller_id"`
	BuyerAuth0ID string `json:"buyer_auth0_id"`
	Reason       string `json:"reason"`
}

func (s *OrderSubscriber) handleCancellationRequested(ctx context.Context, event pubsub.Event) error {
	var data cancellationRequestedData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancellation_requested data: %w", err)
	}

	subject, body := templates.OrderCancellationRequested(data.OrderID, data.Reason)

	// The seller is the recipient — they need to act on the request.
	if err := s.sender.Send(ctx, data.SellerID, subject, body); err != nil {
		return fmt.Errorf("send cancellation requested notification: %w", err)
	}
	return nil
}

type cancellationApprovedData struct {
	OrderID      string `json:"order_id"`
	BuyerAuth0ID string `json:"buyer_auth0_id"`
	RefundAmount int64  `json:"refund_amount"`
}

func (s *OrderSubscriber) handleCancellationApproved(ctx context.Context, event pubsub.Event) error {
	var data cancellationApprovedData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancellation_approved data: %w", err)
	}

	subject, body := templates.OrderCancellationApproved(data.OrderID, data.RefundAmount)

	if err := s.sender.Send(ctx, data.BuyerAuth0ID, subject, body); err != nil {
		return fmt.Errorf("send cancellation approved notification: %w", err)
	}
	return nil
}

type cancellationRejectedData struct {
	OrderID       string `json:"order_id"`
	BuyerAuth0ID  string `json:"buyer_auth0_id"`
	SellerComment string `json:"seller_comment"`
}

func (s *OrderSubscriber) handleCancellationRejected(ctx context.Context, event pubsub.Event) error {
	var data cancellationRejectedData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancellation_rejected data: %w", err)
	}

	subject, body := templates.OrderCancellationRejected(data.OrderID, data.SellerComment)

	if err := s.sender.Send(ctx, data.BuyerAuth0ID, subject, body); err != nil {
		return fmt.Errorf("send cancellation rejected notification: %w", err)
	}
	return nil
}

type orderCancelledNotificationData struct {
	OrderID      string `json:"order_id"`
	BuyerAuth0ID string `json:"buyer_auth0_id"`
	Reason       string `json:"reason"`
}

func (s *OrderSubscriber) handleOrderCancelled(ctx context.Context, event pubsub.Event) error {
	var data orderCancelledNotificationData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancelled data: %w", err)
	}

	subject, body := templates.OrderCancelledNotification(data.OrderID, data.Reason)

	if err := s.sender.Send(ctx, data.BuyerAuth0ID, subject, body); err != nil {
		return fmt.Errorf("send order cancelled notification: %w", err)
	}
	return nil
}

// decodeEventData converts event.Data (which may be a map or raw JSON) into the target struct.
func decodeEventData(eventData any, target any) error {
	raw, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal event data: %w", err)
	}
	return nil
}
