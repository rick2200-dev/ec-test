// Package pubsub contains the coupon service's event subscribers.
// The order.cancelled handler reverses a committed redemption and
// returns the usage_count seat to the coupon's available pool.
package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	pkgpubsub "github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

// OrderCancelledSubscriber subscribes to order-events-coupon and
// triggers a refund via CouponUseCase.RefundRedemption for each event.
// The use case is idempotent so replayed deliveries no-op cleanly.
type OrderCancelledSubscriber struct {
	svc        port.CouponUseCase
	subscriber pkgpubsub.Subscriber
}

func NewOrderCancelledSubscriber(svc port.CouponUseCase, subscriber pkgpubsub.Subscriber) *OrderCancelledSubscriber {
	return &OrderCancelledSubscriber{svc: svc, subscriber: subscriber}
}

// orderCancelledPayload mirrors the subset of order.v1.OrderCancelled
// this subscriber needs. coupon_reservation_id + coupon_discount_amount
// are stamped on every seller order that received a share of the cart-
// wide coupon, so per-seller cancellations can refund their slice of
// the shared redemption without needing to look anything up across
// services.
type orderCancelledPayload struct {
	OrderID              string `json:"order_id"`
	CouponReservationID  string `json:"coupon_reservation_id"`
	CouponDiscountAmount int64  `json:"coupon_discount_amount,string"`
	Reason               string `json:"reason"`
}

// Run blocks until ctx is cancelled. Non-order.cancelled events are
// ignored so the same topic can carry other fan-outs safely.
func (s *OrderCancelledSubscriber) Run(ctx context.Context, subscription string) error {
	slog.Info("coupon order.cancelled subscriber starting", "subscription", subscription)
	return s.subscriber.Subscribe(ctx, subscription, func(ctx context.Context, event pkgpubsub.Event) error {
		if event.Type != "order.cancelled" {
			return nil
		}
		var payload orderCancelledPayload
		if err := decodePayload(event.Data, &payload); err != nil {
			return fmt.Errorf("decode order.cancelled payload: %w", err)
		}
		if payload.OrderID == "" || payload.CouponReservationID == "" {
			// Non-coupon cancellation — nothing for us to do.
			return nil
		}
		reservationID, err := uuid.Parse(payload.CouponReservationID)
		if err != nil {
			return fmt.Errorf("parse coupon_reservation_id: %w", err)
		}
		orderID, err := uuid.Parse(payload.OrderID)
		if err != nil {
			return fmt.Errorf("parse order_id: %w", err)
		}
		// The payload carries this seller order's share of the
		// cart-wide discount; the service accumulates it into the
		// single redemption row and only releases the coupon seat
		// when the row is fully refunded.
		if _, err := s.svc.RefundRedemption(ctx, reservationID, orderID, payload.CouponDiscountAmount, payload.Reason); err != nil {
			return fmt.Errorf("refund redemption: %w", err)
		}
		return nil
	})
}

func decodePayload(data any, target any) error {
	switch v := data.(type) {
	case json.RawMessage:
		return json.Unmarshal(v, target)
	case []byte:
		return json.Unmarshal(v, target)
	case string:
		return json.Unmarshal([]byte(v), target)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, target)
	}
}
