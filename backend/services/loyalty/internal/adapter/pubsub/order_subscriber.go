// Package pubsub contains the loyalty service's event subscribers.
// order.paid is the trigger for point accrual; the subscriber is a
// safety net behind the synchronous internal HTTP call from order-svc
// so that a missed HTTP call (e.g. network blip between the services)
// still credits the buyer eventually.
package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pkgpubsub "github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// OrderPaidSubscriber subscribes to order-events-loyalty and calls
// LoyaltyUseCase.AwardEarn for every order.paid event. The use case
// deduplicates by (source_type, source_id, type) so it is safe for
// this subscriber to run alongside the synchronous HTTP call from
// order-svc — whichever arrives first credits, the other no-ops.
type OrderPaidSubscriber struct {
	svc        port.LoyaltyUseCase
	subscriber pkgpubsub.Subscriber
}

func NewOrderPaidSubscriber(svc port.LoyaltyUseCase, subscriber pkgpubsub.Subscriber) *OrderPaidSubscriber {
	return &OrderPaidSubscriber{svc: svc, subscriber: subscriber}
}

// orderPaidPayload is the subset of order.v1.OrderPaid we need. The
// order service marshals events with protojson + snake_case field
// names, so json.Unmarshal with matching tags reads them cleanly.
type orderPaidPayload struct {
	OrderID                 string `json:"order_id"`
	BuyerAuth0ID            string `json:"buyer_auth0_id"`
	TotalAmount             int64  `json:"total_amount,string"`
	SubtotalBeforeDiscounts int64  `json:"subtotal_before_discounts,string"`
	CouponDiscountAmount    int64  `json:"coupon_discount_amount,string"`
	PointDiscountAmount     int64  `json:"point_discount_amount,string"`
	Currency                string `json:"currency"`
}

// orderCancelledPayload is the subset of order.v1.OrderCancelled we
// need to reverse the ledger. Fields are 0/empty when the cancelled
// order had no discount / points activity.
type orderCancelledPayload struct {
	OrderID             string `json:"order_id"`
	BuyerAuth0ID        string `json:"buyer_auth0_id"`
	PointDiscountAmount int64  `json:"point_discount_amount,string"`
	PointsEarned        int64  `json:"points_earned,string"`
}

// Run blocks the caller until ctx is cancelled. Handles both
// order.paid (earn fan-in) and order.cancelled (refund + reverse_earn).
// Everything else is ignored so this subscriber can safely share the
// order-events topic with other domains.
func (s *OrderPaidSubscriber) Run(ctx context.Context, subscription string) error {
	slog.Info("loyalty order events subscriber starting", "subscription", subscription)
	return s.subscriber.Subscribe(ctx, subscription, func(ctx context.Context, event pkgpubsub.Event) error {
		switch event.Type {
		case "order.paid":
			return s.handleOrderPaid(ctx, event)
		case "order.cancelled":
			return s.handleOrderCancelled(ctx, event)
		default:
			return nil
		}
	})
}

func (s *OrderPaidSubscriber) handleOrderPaid(ctx context.Context, event pkgpubsub.Event) error {
	var payload orderPaidPayload
	if err := decodePayload(event.Data, &payload); err != nil {
		return fmt.Errorf("decode order.paid payload: %w", err)
	}
	if payload.BuyerAuth0ID == "" || payload.OrderID == "" {
		return nil
	}
	// Earn base is the product subtotal actually paid. Prefer the
	// explicit subtotal_before_discounts field when present; fall
	// back to total_amount for events produced before the
	// discount-columns migration.
	earnBase := payload.SubtotalBeforeDiscounts
	if earnBase == 0 {
		earnBase = payload.TotalAmount
	}
	if _, err := s.svc.AwardEarn(ctx, payload.BuyerAuth0ID, payload.OrderID, earnBase, payload.Currency); err != nil {
		return fmt.Errorf("award earn: %w", err)
	}
	return nil
}

func (s *OrderPaidSubscriber) handleOrderCancelled(ctx context.Context, event pkgpubsub.Event) error {
	var payload orderCancelledPayload
	if err := decodePayload(event.Data, &payload); err != nil {
		return fmt.Errorf("decode order.cancelled payload: %w", err)
	}
	if payload.BuyerAuth0ID == "" || payload.OrderID == "" {
		return nil
	}
	if _, err := s.svc.ApplyCancellation(ctx, port.CancelPointsInput{
		BuyerAuth0ID:        payload.BuyerAuth0ID,
		OrderID:             payload.OrderID,
		PointDiscountAmount: payload.PointDiscountAmount,
		PointsEarned:        payload.PointsEarned,
	}); err != nil {
		return fmt.Errorf("apply cancellation: %w", err)
	}
	return nil
}

// decodePayload handles both `map[string]any` and `json.RawMessage` /
// `[]byte` shapes the envelope's Data field can take depending on
// whether the event was produced by PublishProtoEvent or PublishEvent.
func decodePayload(data any, target any) error {
	switch v := data.(type) {
	case json.RawMessage:
		return json.Unmarshal(v, target)
	case []byte:
		return json.Unmarshal(v, target)
	case string:
		return json.Unmarshal([]byte(v), target)
	default:
		// Re-marshal then unmarshal — slow path but used rarely.
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, target)
	}
}
