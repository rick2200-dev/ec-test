package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	shippingv1 "github.com/Riku-KANO/ec-test/services/shipping/api/gen/go/shipping/v1"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/domain"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/port"
)

// ShipmentService is the application-layer implementation of port.ShipmentUseCase.
//
// Event publishing is handled by the outbox relay (adapter/pubsub/outbox_relay.go),
// not directly by this service. Events are written atomically into the outbox table
// inside the same DB transaction as the status update, so they are guaranteed to be
// published at-least-once even if the process crashes after commit.
// Consumers must treat the event envelope's ID (= outbox row id) as a stable dedup key
// to achieve idempotent processing; exactly-once delivery is the consumer's responsibility.
type ShipmentService struct {
	repo     port.ShipmentRepository
	txRunner port.TxRunner
}

// NewShipmentService wires the service with its dependencies.
func NewShipmentService(
	repo port.ShipmentRepository,
	txRunner port.TxRunner,
) *ShipmentService {
	return &ShipmentService{
		repo:     repo,
		txRunner: txRunner,
	}
}

// CreateShipment inserts a ready_to_ship shipment for the given order.
// The operation is idempotent — a duplicate order_id is silently ignored.
func (s *ShipmentService) CreateShipment(ctx context.Context, in port.CreateShipmentInput) error {
	shipment := &domain.Shipment{
		SellerID:        in.SellerID,
		OrderID:         in.OrderID,
		BuyerAuth0ID:    in.BuyerAuth0ID,
		Status:          domain.StatusReadyToShip,
		ShippingAddress: json.RawMessage(in.ShippingAddress),
	}
	return s.repo.Create(ctx, shipment)
}

// RegisterShipment transitions the shipment from ready_to_ship to shipped and
// records the carrier + tracking number. Publishes shipment.shipped on success.
func (s *ShipmentService) RegisterShipment(ctx context.Context, in port.RegisterShipmentInput) (*domain.Shipment, error) {
	if in.Carrier == "" || in.TrackingNumber == "" {
		return nil, domain.ErrTrackingNumberRequired
	}

	shipment, err := s.repo.GetByID(ctx, in.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment.SellerID != in.SellerID {
		return nil, domain.ErrNotOrderSeller
	}
	if shipment.Status == domain.StatusShipped || shipment.Status == domain.StatusDelivered {
		return nil, domain.ErrAlreadyRegistered
	}
	if !shipment.CanRegister() {
		return nil, domain.ErrInvalidTransition
	}

	now := time.Now().UTC()
	shippedAt := &now
	if in.ShippedAt != nil {
		shippedAt = in.ShippedAt
	}

	fromStatus := shipment.Status
	shipment.Status = domain.StatusShipped
	shipment.Carrier = in.Carrier
	shipment.TrackingNumber = in.TrackingNumber
	shipment.ShippedAt = shippedAt
	shipment.Note = in.Note

	shippedEvt := &shippingv1.ShipmentShipped{
		ShipmentId:     shipment.ID.String(),
		OrderId:        shipment.OrderID.String(),
		SellerId:       shipment.SellerID.String(),
		BuyerAuth0Id:   shipment.BuyerAuth0ID,
		Carrier:        shipment.Carrier,
		TrackingNumber: shipment.TrackingNumber,
		ShippedAt:      timestamppb.New(*shipment.ShippedAt),
	}

	err = s.txRunner.RunTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateStatus(txCtx, shipment, fromStatus); err != nil {
			return err
		}
		auditEvt := &domain.ShipmentEvent{
			ShipmentID: shipment.ID,
			FromStatus: fromStatus,
			ToStatus:   domain.StatusShipped,
			ActorType:  "seller",
			ActorID:    in.SellerID.String(),
			Payload: mustJSON(map[string]string{
				"carrier":         in.Carrier,
				"tracking_number": in.TrackingNumber,
			}),
		}
		if err := s.repo.AppendEvent(txCtx, auditEvt); err != nil {
			return err
		}
		// Write the domain event to the outbox in the same TX.
		// The relay worker will publish it to Pub/Sub asynchronously.
		return s.repo.AppendOutboxEvent(txCtx, port.OutboxEvent{
			EventType: domain.EventTypeShipmentShipped,
			Topic:     "shipping-events",
			Payload:   mustProtoJSON(shippedEvt),
		})
	})
	if err != nil {
		return nil, err
	}

	return shipment, nil
}

// MarkDelivered transitions the shipment from shipped to delivered.
// Publishes shipment.delivered on success.
func (s *ShipmentService) MarkDelivered(ctx context.Context, in port.MarkDeliveredInput) (*domain.Shipment, error) {
	shipment, err := s.repo.GetByID(ctx, in.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment.SellerID != in.SellerID {
		return nil, domain.ErrNotOrderSeller
	}
	if !shipment.CanDeliver() {
		return nil, domain.ErrInvalidTransition
	}

	now := time.Now().UTC()
	deliveredAt := &now
	if in.DeliveredAt != nil {
		deliveredAt = in.DeliveredAt
	}

	fromStatus := shipment.Status
	shipment.Status = domain.StatusDelivered
	shipment.DeliveredAt = deliveredAt

	deliveredEvt := &shippingv1.ShipmentDelivered{
		ShipmentId:   shipment.ID.String(),
		OrderId:      shipment.OrderID.String(),
		SellerId:     shipment.SellerID.String(),
		BuyerAuth0Id: shipment.BuyerAuth0ID,
		DeliveredAt:  timestamppb.New(*shipment.DeliveredAt),
	}

	err = s.txRunner.RunTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateStatus(txCtx, shipment, fromStatus); err != nil {
			return err
		}
		auditEvt := &domain.ShipmentEvent{
			ShipmentID: shipment.ID,
			FromStatus: fromStatus,
			ToStatus:   domain.StatusDelivered,
			ActorType:  "seller",
			ActorID:    in.SellerID.String(),
		}
		if err := s.repo.AppendEvent(txCtx, auditEvt); err != nil {
			return err
		}
		return s.repo.AppendOutboxEvent(txCtx, port.OutboxEvent{
			EventType: domain.EventTypeShipmentDelivered,
			Topic:     "shipping-events",
			Payload:   mustProtoJSON(deliveredEvt),
		})
	})
	if err != nil {
		return nil, err
	}

	return shipment, nil
}

// CancelShipment transitions the shipment to cancelled when an order is
// cancelled. If the shipment is already shipped or delivered, this is a no-op.
func (s *ShipmentService) CancelShipment(ctx context.Context, orderID uuid.UUID) error {
	return s.repo.CancelByOrderID(ctx, orderID)
}

// GetByID returns a shipment. Verifies seller ownership.
func (s *ShipmentService) GetByID(ctx context.Context, sellerID, id uuid.UUID) (*domain.Shipment, error) {
	shipment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if shipment.SellerID != sellerID {
		return nil, domain.ErrNotOrderSeller
	}
	return shipment, nil
}

// GetByOrderIDSeller returns a shipment by order ID. Verifies seller ownership.
func (s *ShipmentService) GetByOrderIDSeller(ctx context.Context, sellerID, orderID uuid.UUID) (*domain.Shipment, error) {
	shipment, err := s.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if shipment.SellerID != sellerID {
		return nil, domain.ErrNotOrderSeller
	}
	return shipment, nil
}

// GetByOrderIDBuyer returns a shipment by order ID. Verifies buyer identity.
func (s *ShipmentService) GetByOrderIDBuyer(ctx context.Context, buyerAuth0ID string, orderID uuid.UUID) (*domain.Shipment, error) {
	shipment, err := s.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if shipment.BuyerAuth0ID != buyerAuth0ID {
		return nil, domain.ErrNotOrderBuyer
	}
	return shipment, nil
}

// ListBySeller returns paginated shipments for a seller.
func (s *ShipmentService) ListBySeller(ctx context.Context, in port.ListShipmentsInput) (*port.ListShipmentsResult, error) {
	if in.Limit <= 0 {
		in.Limit = 20
	}
	if in.Limit > 100 {
		in.Limit = 100
	}

	items, total, err := s.repo.ListBySeller(ctx, in.SellerID, in.Status, in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}
	return &port.ListShipmentsResult{
		Items:  items,
		Total:  total,
		Limit:  in.Limit,
		Offset: in.Offset,
	}, nil
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// mustProtoJSON serializes a proto event payload using snake_case field names
// so the envelope wire format stays compatible with snake_case consumers.
func mustProtoJSON(m proto.Message) json.RawMessage {
	b, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(m)
	return b
}
