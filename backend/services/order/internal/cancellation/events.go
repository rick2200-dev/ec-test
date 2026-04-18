package cancellation

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	orderv1 "github.com/Riku-KANO/ec-test/services/order/api/gen/go/order/v1"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// Pub/Sub topic that carries every order lifecycle event.
const orderEventsTopic = "order-events"

// Event type names for the order-cancellation lifecycle. These strings are
// the switch-on values for every downstream subscriber (notification,
// inventory). The payload schemas are defined in
// services/order/api/proto/order/v1/events.proto.
const (
	EventTypeCancellationRequested = "order.cancellation_requested"
	EventTypeCancellationApproved  = "order.cancellation_approved"
	EventTypeCancellationRejected  = "order.cancellation_rejected"
	EventTypeOrderCancelled        = "order.cancelled"
)

func publishRequested(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order) {
	pubsub.PublishProtoEvent(ctx, pub, EventTypeCancellationRequested, orderEventsTopic, &orderv1.CancellationRequested{
		RequestId:    req.ID.String(),
		OrderId:      req.OrderID.String(),
		SellerId:     order.SellerID.String(),
		BuyerAuth0Id: req.RequestedByAuth0ID,
		Reason:       req.Reason,
	})
}

func publishRejected(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order) {
	comment := ""
	if req.SellerComment != nil {
		comment = *req.SellerComment
	}
	pubsub.PublishProtoEvent(ctx, pub, EventTypeCancellationRejected, orderEventsTopic, &orderv1.CancellationRejected{
		RequestId:     req.ID.String(),
		OrderId:       req.OrderID.String(),
		SellerId:      order.SellerID.String(),
		BuyerAuth0Id:  req.RequestedByAuth0ID,
		SellerComment: comment,
	})
}

func publishApproved(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order, refundAmount int64) {
	refundID := ""
	if req.StripeRefundID != nil {
		refundID = *req.StripeRefundID
	}
	pubsub.PublishProtoEvent(ctx, pub, EventTypeCancellationApproved, orderEventsTopic, &orderv1.CancellationApproved{
		RequestId:      req.ID.String(),
		OrderId:        req.OrderID.String(),
		SellerId:       order.SellerID.String(),
		BuyerAuth0Id:   req.RequestedByAuth0ID,
		StripeRefundId: refundID,
		RefundAmount:   refundAmount,
	})
}

func publishOrderCancelled(ctx context.Context, pub pubsub.Publisher, req *CancellationRequest, order *domain.Order, lines []domain.OrderLine) {
	cancelledLines := make([]*orderv1.CancelledLineItem, 0, len(lines))
	for _, l := range lines {
		cancelledLines = append(cancelledLines, &orderv1.CancelledLineItem{
			SkuId:       l.SKUID.String(),
			ProductName: l.ProductName,
			SkuCode:     l.SKUCode,
			Quantity:    int32(l.Quantity),
		})
	}

	evt := &orderv1.OrderCancelled{
		OrderId:              order.ID.String(),
		SellerId:             order.SellerID.String(),
		BuyerAuth0Id:         order.BuyerAuth0ID,
		RequestId:            req.ID.String(),
		Reason:               req.Reason,
		LineItems:            cancelledLines,
		CouponDiscountAmount: order.CouponDiscountAmount,
		PointDiscountAmount:  order.PointDiscountAmount,
		PointsEarned:         order.PointsEarned,
	}
	// Carry reservation + coupon ids so the coupon and loyalty
	// subscribers can reverse their own ledger entries without
	// reverse-looking up order-svc. Empty when the cancelled order
	// had no discount.
	if order.CouponID != nil {
		evt.CouponId = order.CouponID.String()
	}
	if order.CouponReservationID != nil {
		evt.CouponReservationId = order.CouponReservationID.String()
	}
	if order.PointReservationID != nil {
		evt.PointReservationId = order.PointReservationID.String()
	}
	if order.CancelledAt != nil {
		evt.CancelledAt = timestamppb.New(order.CancelledAt.UTC())
	}

	pubsub.PublishProtoEvent(ctx, pub, EventTypeOrderCancelled, orderEventsTopic, evt)
}
