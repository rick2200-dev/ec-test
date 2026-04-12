package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
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
		"tenant_id", event.TenantID,
	)

	switch event.Type {
	case "inquiry.message_created":
		return s.handleMessageCreated(ctx, event)
	default:
		slog.Warn("unhandled inquiry event type", "event_type", event.Type)
		return nil
	}
}

// inquiryMessageCreatedData is the expected shape of the event payload
// emitted by the inquiry service. Recipient email is not carried on the
// event — the subscriber logs the send without a real SMTP endpoint, so
// this MVP just records who we would have emailed.
type inquiryMessageCreatedData struct {
	InquiryID    string `json:"inquiry_id"`
	SellerID     string `json:"seller_id"`
	BuyerAuth0ID string `json:"buyer_auth0_id"`
	SenderType   string `json:"sender_type"`
	Subject      string `json:"subject"`
	ProductName  string `json:"product_name"`
	BodyPreview  string `json:"body_preview"`
}

func (s *InquirySubscriber) handleMessageCreated(ctx context.Context, event pubsub.Event) error {
	var data inquiryMessageCreatedData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode inquiry.message_created data: %w", err)
	}

	// The opposite party is the recipient. MVP: label them by role rather
	// than resolve a real mailbox, which would require an auth/order call.
	var recipientLabel, senderLabel, recipientHint string
	switch data.SenderType {
	case "buyer":
		senderLabel = "購入者"
		recipientLabel = "出品者"
		recipientHint = "seller:" + data.SellerID
	case "seller":
		senderLabel = "出品者"
		recipientLabel = "購入者"
		recipientHint = "buyer:" + data.BuyerAuth0ID
	default:
		slog.Warn("unknown sender_type on inquiry event", "sender_type", data.SenderType)
		return nil
	}

	subject, body := templates.InquiryNewMessageNotification(
		recipientLabel, senderLabel, data.Subject, data.ProductName, data.BodyPreview,
	)

	slog.Info("sending inquiry notification",
		"inquiry_id", data.InquiryID,
		"recipient", recipientHint,
		"tenant_id", event.TenantID,
	)

	if err := s.sender.Send(ctx, recipientHint, subject, body); err != nil {
		return fmt.Errorf("send inquiry notification: %w", err)
	}
	return nil
}
