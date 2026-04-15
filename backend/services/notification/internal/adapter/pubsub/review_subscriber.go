package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/notification/internal/email"
	"github.com/Riku-KANO/ec-test/services/notification/internal/templates"
	reviewv1 "github.com/Riku-KANO/ec-test/services/review/api/gen/go/review/v1"
)

const reviewSubscription = "review-events-notification"

// ReviewSubscriber handles review-related events and sends notifications.
// review.created → notify seller; review.replied → notify buyer.
type ReviewSubscriber struct {
	subscriber pubsub.Subscriber
	sender     email.Sender
}

// NewReviewSubscriber constructs a ReviewSubscriber.
func NewReviewSubscriber(subscriber pubsub.Subscriber, sender email.Sender) *ReviewSubscriber {
	return &ReviewSubscriber{subscriber: subscriber, sender: sender}
}

// Start begins listening for review events. Blocks until ctx is cancelled.
func (s *ReviewSubscriber) Start(ctx context.Context) error {
	slog.Info("starting review event subscriber", "subscription", reviewSubscription)
	return s.subscriber.Subscribe(ctx, reviewSubscription, s.handleEvent)
}

func (s *ReviewSubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	slog.Info("received review event",
		"event_id", event.ID,
		"event_type", event.Type,
	)

	switch event.Type {
	case "review.created":
		return s.handleReviewCreated(ctx, event)
	case "review.replied":
		return s.handleReviewReplied(ctx, event)
	default:
		slog.Debug("ignoring review event type", "event_type", event.Type)
		return nil
	}
}

func (s *ReviewSubscriber) handleReviewCreated(ctx context.Context, event pubsub.Event) error {
	var data reviewv1.ReviewCreated
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode review.created data: %w", err)
	}

	subject, body := templates.ReviewCreatedNotification(data.ProductName, int(data.Rating), data.Title)
	recipientHint := "seller:" + data.SellerId

	slog.Info("sending review notification",
		"review_id", data.ReviewId,
		"recipient", recipientHint,
	)

	if err := s.sender.Send(ctx, recipientHint, subject, body); err != nil {
		return fmt.Errorf("send review created notification: %w", err)
	}
	return nil
}

func (s *ReviewSubscriber) handleReviewReplied(ctx context.Context, event pubsub.Event) error {
	var data reviewv1.ReviewReplied
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode review.replied data: %w", err)
	}

	subject, body := templates.ReviewRepliedNotification(data.ProductName, data.ReplyPreview)
	recipientHint := "buyer:" + data.BuyerAuth0Id

	slog.Info("sending review reply notification",
		"review_id", data.ReviewId,
		"recipient", recipientHint,
	)

	if err := s.sender.Send(ctx, recipientHint, subject, body); err != nil {
		return fmt.Errorf("send review replied notification: %w", err)
	}
	return nil
}
