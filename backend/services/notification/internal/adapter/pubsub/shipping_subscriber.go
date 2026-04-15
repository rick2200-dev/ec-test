package subscriber

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pkgpubsub "github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/notification/internal/email"
	"github.com/Riku-KANO/ec-test/services/notification/internal/port"
	"github.com/Riku-KANO/ec-test/services/notification/internal/templates"
)

const shippingSubscription = "shipping-events-notification"

// activeProcessingRetryDelay is how long the subscriber sleeps before nacking
// when TryClaim returns ClaimActiveProcessing. GCPSubscriber automatically
// extends the ack deadline while the handler is running, so the message is not
// redelivered during the sleep. The delay should be less than the 10-minute
// processing_at expiry so the retries are meaningful, but large enough to avoid
// hammering Pub/Sub.
const activeProcessingRetryDelay = 30 * time.Second

// noopDeduplicator is used when no DB is configured (local dev without DB).
// Every delivery is processed; no deduplication is performed.
type noopDeduplicator struct{}

func (noopDeduplicator) TryClaim(_ context.Context, _, _ string) (port.ClaimOutcome, error) {
	return port.ClaimAcquired, nil
}
func (noopDeduplicator) MarkSent(_ context.Context, _ string) error  { return nil }
func (noopDeduplicator) Release(_ context.Context, _ string) error   { return nil }

// ShippingSubscriber handles shipping events and sends buyer notifications.
type ShippingSubscriber struct {
	subscriber pkgpubsub.Subscriber
	sender     email.Sender
	dedup      port.Deduplicator
}

// NewShippingSubscriber creates a new ShippingSubscriber.
// If dedup is nil, a no-op deduplicator is used (every delivery is processed).
func NewShippingSubscriber(subscriber pkgpubsub.Subscriber, sender email.Sender, dedup port.Deduplicator) *ShippingSubscriber {
	if dedup == nil {
		dedup = noopDeduplicator{}
	}
	return &ShippingSubscriber{
		subscriber: subscriber,
		sender:     sender,
		dedup:      dedup,
	}
}

// Start begins listening for shipping events. Blocks until context is cancelled.
func (s *ShippingSubscriber) Start(ctx context.Context) error {
	slog.Info("starting shipping event subscriber", "subscription", shippingSubscription)
	return s.subscriber.Subscribe(ctx, shippingSubscription, s.handleEvent)
}

func (s *ShippingSubscriber) handleEvent(ctx context.Context, event pkgpubsub.Event) error {
	slog.Info("received shipping event",
		"event_id", event.ID,
		"event_type", event.Type,
	)

	outcome, claimErr := s.dedup.TryClaim(ctx, event.ID, event.Type)
	if claimErr != nil {
		// DB error: fail open — proceed as if acquired so the notification is
		// not silently dropped. A duplicate send on retry is acceptable.
		slog.Warn("shipping subscriber: dedup TryClaim failed, processing anyway",
			"event_id", event.ID, "error", claimErr)
	} else {
		switch outcome {
		case port.ClaimAlreadySent:
			// Event was previously processed successfully. Ack and skip.
			slog.Info("shipping subscriber: already sent, skipping",
				"event_id", event.ID, "event_type", event.Type)
			return nil

		case port.ClaimActiveProcessing:
			// Another worker holds an active (non-expired) claim. Return a
			// RetryAfterError so GCPSubscriber sleeps before nacking — this
			// prevents a tight redeliver loop during the 10-minute window.
			// The ack deadline is extended automatically by the GCP client
			// while the sleep runs. Returning nil here would ack the message
			// and could permanently lose the notification if the active worker
			// crashes before MarkSent.
			slog.Info("shipping subscriber: active processing claim, delaying nack",
				"event_id", event.ID, "event_type", event.Type,
				"retry_delay", activeProcessingRetryDelay)
			return pkgpubsub.RetryAfter(activeProcessingRetryDelay,
				fmt.Errorf("event %s is being processed by another worker", event.ID))

		case port.ClaimAcquired:
			// This worker owns the claim. Fall through to send.
		}
	}

	// Route to the appropriate handler (sends the notification).
	var handlerErr error
	switch event.Type {
	case "shipment.shipped":
		handlerErr = s.handleShipmentShipped(ctx, event)
	case "shipment.delivered":
		handlerErr = s.handleShipmentDelivered(ctx, event)
	default:
		// Unknown type: release the claim so it does not permanently block a
		// future handler. Return nil — no retry needed for unknown types.
		if releaseErr := s.dedup.Release(ctx, event.ID); releaseErr != nil {
			slog.Warn("shipping subscriber: release after unknown event type failed",
				"event_id", event.ID, "error", releaseErr)
		}
		slog.Debug("ignoring unhandled shipping event", "event_type", event.Type)
		return nil
	}

	if handlerErr != nil {
		// Send failed: release the 'processing' row so Pub/Sub can redeliver.
		if releaseErr := s.dedup.Release(ctx, event.ID); releaseErr != nil {
			slog.Warn("shipping subscriber: release after send failure failed; crash recovery via 10-min timeout",
				"event_id", event.ID, "error", releaseErr)
		}
		return handlerErr
	}

	// Send succeeded: transition to 'sent' so future redeliveries see
	// ClaimAlreadySent and are acked without sending again.
	// If MarkSent fails, log a warning but return nil — the notification was
	// already delivered. The 10-min processing_at timeout will expire and
	// allow at most one duplicate, which is preferable to nacking here.
	if err := s.dedup.MarkSent(ctx, event.ID); err != nil {
		slog.Warn("shipping subscriber: MarkSent failed; one duplicate possible after processing_at expiry",
			"event_id", event.ID, "error", err)
	}
	return nil
}

// shipmentShippedData mirrors domain.ShipmentShippedEvent from the shipping service.
// Field names must stay in sync with shipping/internal/domain/events.go.
type shipmentShippedData struct {
	ShipmentID     string `json:"shipment_id"`
	OrderID        string `json:"order_id"`
	BuyerAuth0ID   string `json:"buyer_auth0_id"`
	Carrier        string `json:"carrier"`
	TrackingNumber string `json:"tracking_number"`
}

func (s *ShippingSubscriber) handleShipmentShipped(ctx context.Context, event pkgpubsub.Event) error {
	var data shipmentShippedData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode shipment.shipped data: %w", err)
	}

	subject, body := templates.ShipmentShippedNotification(data.OrderID, data.Carrier, data.TrackingNumber)

	slog.Info("sending shipment shipped notification",
		"order_id", data.OrderID,
		"buyer_auth0_id", data.BuyerAuth0ID,
		"carrier", data.Carrier,
	)

	if err := s.sender.Send(ctx, data.BuyerAuth0ID, subject, body); err != nil {
		return fmt.Errorf("send shipment shipped notification: %w", err)
	}
	return nil
}

// shipmentDeliveredData mirrors domain.ShipmentDeliveredEvent.
type shipmentDeliveredData struct {
	ShipmentID   string `json:"shipment_id"`
	OrderID      string `json:"order_id"`
	BuyerAuth0ID string `json:"buyer_auth0_id"`
}

func (s *ShippingSubscriber) handleShipmentDelivered(ctx context.Context, event pkgpubsub.Event) error {
	var data shipmentDeliveredData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode shipment.delivered data: %w", err)
	}

	subject, body := templates.ShipmentDeliveredNotification(data.OrderID)

	slog.Info("sending shipment delivered notification",
		"order_id", data.OrderID,
		"buyer_auth0_id", data.BuyerAuth0ID,
	)

	if err := s.sender.Send(ctx, data.BuyerAuth0ID, subject, body); err != nil {
		return fmt.Errorf("send shipment delivered notification: %w", err)
	}
	return nil
}
