package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

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
		TenantID:        in.TenantID,
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

	shipment, err := s.repo.GetByID(ctx, in.TenantID, in.ShipmentID)
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

	shippedEvt := domain.ShipmentShippedEvent{
		ShipmentID:     shipment.ID.String(),
		OrderID:        shipment.OrderID.String(),
		TenantID:       shipment.TenantID.String(),
		SellerID:       shipment.SellerID.String(),
		BuyerAuth0ID:   shipment.BuyerAuth0ID,
		Carrier:        shipment.Carrier,
		TrackingNumber: shipment.TrackingNumber,
		ShippedAt:      *shipment.ShippedAt,
	}

	err = s.txRunner.RunTenantTx(ctx, in.TenantID, func(txCtx context.Context) error {
		if err := s.repo.UpdateStatus(txCtx, shipment, fromStatus); err != nil {
			return err
		}
		auditEvt := &domain.ShipmentEvent{
			TenantID:   in.TenantID,
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
			TenantID:  in.TenantID,
			Payload:   mustJSON(shippedEvt),
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
	shipment, err := s.repo.GetByID(ctx, in.TenantID, in.ShipmentID)
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

	deliveredEvt := domain.ShipmentDeliveredEvent{
		ShipmentID:   shipment.ID.String(),
		OrderID:      shipment.OrderID.String(),
		TenantID:     shipment.TenantID.String(),
		SellerID:     shipment.SellerID.String(),
		BuyerAuth0ID: shipment.BuyerAuth0ID,
		DeliveredAt:  *shipment.DeliveredAt,
	}

	err = s.txRunner.RunTenantTx(ctx, in.TenantID, func(txCtx context.Context) error {
		if err := s.repo.UpdateStatus(txCtx, shipment, fromStatus); err != nil {
			return err
		}
		auditEvt := &domain.ShipmentEvent{
			TenantID:   in.TenantID,
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
			TenantID:  in.TenantID,
			Payload:   mustJSON(deliveredEvt),
		})
	})
	if err != nil {
		return nil, err
	}

	return shipment, nil
}

// CancelShipment transitions the shipment to cancelled when an order is
// cancelled. If the shipment is already shipped or delivered, this is a no-op.
func (s *ShipmentService) CancelShipment(ctx context.Context, tenantID, orderID uuid.UUID) error {
	return s.repo.CancelByOrderID(ctx, tenantID, orderID)
}

// GetByID returns a shipment. Verifies seller ownership.
func (s *ShipmentService) GetByID(ctx context.Context, tenantID, sellerID, id uuid.UUID) (*domain.Shipment, error) {
	shipment, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if shipment.SellerID != sellerID {
		return nil, domain.ErrNotOrderSeller
	}
	return shipment, nil
}

// GetByOrderIDSeller returns a shipment by order ID. Verifies seller ownership.
func (s *ShipmentService) GetByOrderIDSeller(ctx context.Context, tenantID, sellerID, orderID uuid.UUID) (*domain.Shipment, error) {
	shipment, err := s.repo.GetByOrderID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if shipment.SellerID != sellerID {
		return nil, domain.ErrNotOrderSeller
	}
	return shipment, nil
}

// GetByOrderIDBuyer returns a shipment by order ID. Verifies buyer identity.
func (s *ShipmentService) GetByOrderIDBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, orderID uuid.UUID) (*domain.Shipment, error) {
	shipment, err := s.repo.GetByOrderID(ctx, tenantID, orderID)
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

	items, total, err := s.repo.ListBySeller(ctx, in.TenantID, in.SellerID, in.Status, in.Limit, in.Offset)
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
