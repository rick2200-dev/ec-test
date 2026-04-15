package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	inquiryv1 "github.com/Riku-KANO/ec-test/services/inquiry/api/gen/go/inquiry/v1"
	"github.com/Riku-KANO/ec-test/services/notification/internal/email"
	"github.com/Riku-KANO/ec-test/services/notification/internal/templates"
)

const inquirySubscription = "inquiry-events-notification"

// InquirySubscriber handles inquiry-related events and sends notifications
// to the opposite party (buyer → seller and vice versa). Emails are routed
// via the log sender in MVP; a real SMTP sender can be swapped in later.
type InquirySubscriber struct {
	subscriber pubsub.Subscriber
	sender     email.Sender
}

// NewInquirySubscriber constructs an InquirySubscriber.
func NewInquirySubscriber(subscriber pubsub.Subscriber, sender email.Sender) *InquirySubscriber {
	return &InquirySubscriber{subscriber: subscriber, sender: sender}
}

// Start begins listening for inquiry events. Blocks until ctx is cancelled.
func (s *InquirySubscriber) Start(ctx context.Context) error {
	slog.Info("starting inquiry event subscriber", "subscription", inquirySubscription)
	return s.subscriber.Subscribe(ctx, inquirySubscription, s.handleEvent)
}

func (s *InquirySubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	slog.Info("received inquiry event",
		"event_id", event.ID,
		"event_type", event.Type,
	)

	switch event.Type {
	case "inquiry.message_created":
		return s.handleMessageCreated(ctx, event)
	default:
		slog.Warn("unhandled inquiry event type", "event_type", event.Type)
		return nil
	}
}

func (s *InquirySubscriber) handleMessageCreated(ctx context.Context, event pubsub.Event) error {
	var data inquiryv1.InquiryMessageCreated
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode inquiry.message_created data: %w", err)
	}

	// The opposite party is the recipient. MVP: label them by role rather
	// than resolve a real mailbox, which would require an auth/order call.
	var recipientLabel, senderLabel, recipientHint string
	switch data.SenderType {
	case "buyer":
		senderLabel = "購入者"
		recipientLabel = "出品者"
		recipientHint = "seller:" + data.SellerId
	case "seller":
		senderLabel = "出品者"
		recipientLabel = "購入者"
		recipientHint = "buyer:" + data.BuyerAuth0Id
	default:
		slog.Warn("unknown sender_type on inquiry event", "sender_type", data.SenderType)
		return nil
	}

	subject, body := templates.InquiryNewMessageNotification(
		recipientLabel, senderLabel, data.Subject, data.ProductName, data.BodyPreview,
	)

	slog.Info("sending inquiry notification",
		"inquiry_id", data.InquiryId,
		"recipient", recipientHint,
	)

	if err := s.sender.Send(ctx, recipientHint, subject, body); err != nil {
		return fmt.Errorf("send inquiry notification: %w", err)
	}
	return nil
}
