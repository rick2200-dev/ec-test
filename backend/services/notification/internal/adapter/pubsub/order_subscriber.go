package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/notification/internal/email"
	"github.com/Riku-KANO/ec-test/services/notification/internal/templates"
	orderv1 "github.com/Riku-KANO/ec-test/services/order/api/gen/go/order/v1"
)

const orderSubscription = "order-events-notification"

// OrderSubscriber handles order-related events and sends notifications.
// Payload schemas are defined in services/order/api/proto/order/v1/events.proto —
// the order service is the single source of truth for field names.
//
// TODO(recipient-resolution): handlers currently pass synthetic "buyer:<auth0_id>",
// "seller:<uuid>", "order:<order_id>" strings to email.Sender. This works for the
// MVP LogSender but will break when an SMTPSender is wired in, since `to` must be
// a real email address. Before enabling SMTP, introduce an auth-service
// BatchResolveEmail RPC (or equivalent) and resolve recipients here. Grep this
// file for "buyer:" / "seller:" / "order:" to find the call sites.
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

func (s *OrderSubscriber) handleOrderCreated(ctx context.Context, event pubsub.Event) error {
	var data orderv1.OrderCreated
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.created data: %w", err)
	}

	subject, body := templates.OrderConfirmation(data.OrderId, "buyer:"+data.BuyerAuth0Id, data.TotalAmount, "seller:"+data.SellerId)
	recipient := "buyer:" + data.BuyerAuth0Id

	slog.Info("sending order confirmation",
		"order_id", data.OrderId,
		"recipient", recipient,
	)

	if err := s.sender.Send(ctx, recipient, subject, body); err != nil {
		return fmt.Errorf("send order confirmation: %w", err)
	}
	return nil
}

func (s *OrderSubscriber) handleOrderPaid(ctx context.Context, event pubsub.Event) error {
	var data orderv1.OrderPaid
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.paid data: %w", err)
	}

	subject, body := templates.OrderPaidNotification(data.OrderId, "seller:"+data.SellerId, data.TotalAmount)
	recipient := "seller:" + data.SellerId

	slog.Info("sending payment notification to seller",
		"order_id", data.OrderId,
		"recipient", recipient,
	)

	if err := s.sender.Send(ctx, recipient, subject, body); err != nil {
		return fmt.Errorf("send payment notification: %w", err)
	}
	return nil
}

func (s *OrderSubscriber) handleOrderShipped(ctx context.Context, event pubsub.Event) error {
	var data orderv1.OrderShipped
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.shipped data: %w", err)
	}

	subject, body := templates.OrderShippedNotification(data.OrderId, "")

	slog.Info("sending shipping notification", "order_id", data.OrderId)

	if err := s.sender.Send(ctx, "order:"+data.OrderId, subject, body); err != nil {
		return fmt.Errorf("send shipping notification: %w", err)
	}
	return nil
}

func (s *OrderSubscriber) handleCancellationRequested(ctx context.Context, event pubsub.Event) error {
	var data orderv1.CancellationRequested
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancellation_requested data: %w", err)
	}

	subject, body := templates.OrderCancellationRequested(data.OrderId, data.Reason)

	if err := s.sender.Send(ctx, "seller:"+data.SellerId, subject, body); err != nil {
		return fmt.Errorf("send cancellation requested notification: %w", err)
	}
	return nil
}

func (s *OrderSubscriber) handleCancellationApproved(ctx context.Context, event pubsub.Event) error {
	var data orderv1.CancellationApproved
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancellation_approved data: %w", err)
	}

	subject, body := templates.OrderCancellationApproved(data.OrderId, data.RefundAmount)

	if err := s.sender.Send(ctx, "buyer:"+data.BuyerAuth0Id, subject, body); err != nil {
		return fmt.Errorf("send cancellation approved notification: %w", err)
	}
	return nil
}

func (s *OrderSubscriber) handleCancellationRejected(ctx context.Context, event pubsub.Event) error {
	var data orderv1.CancellationRejected
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancellation_rejected data: %w", err)
	}

	subject, body := templates.OrderCancellationRejected(data.OrderId, data.SellerComment)

	if err := s.sender.Send(ctx, "buyer:"+data.BuyerAuth0Id, subject, body); err != nil {
		return fmt.Errorf("send cancellation rejected notification: %w", err)
	}
	return nil
}

func (s *OrderSubscriber) handleOrderCancelled(ctx context.Context, event pubsub.Event) error {
	var data orderv1.OrderCancelled
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancelled data: %w", err)
	}

	subject, body := templates.OrderCancelledNotification(data.OrderId, data.Reason)

	if err := s.sender.Send(ctx, "buyer:"+data.BuyerAuth0Id, subject, body); err != nil {
		return fmt.Errorf("send order cancelled notification: %w", err)
	}
	return nil
}

// decodeProtoEventData decodes the event payload into a proto message. Use for
// events whose producer marshals the payload via protojson (producer-owned
// api/proto/.../events.proto contracts).
func decodeProtoEventData(eventData any, target protoreflect.ProtoMessage) error {
	raw, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal proto event data: %w", err)
	}
	return nil
}
