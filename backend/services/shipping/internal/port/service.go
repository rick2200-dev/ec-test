package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/shipping/internal/domain"
)

// RegisterShipmentInput holds the parameters for marking a shipment as shipped.
type RegisterShipmentInput struct {
	ShipmentID     uuid.UUID
	SellerID       uuid.UUID
	Carrier        string
	TrackingNumber string
	Note           string
	ShippedAt      *time.Time // nil → current time
}

// MarkDeliveredInput holds the parameters for marking a shipment as delivered.
type MarkDeliveredInput struct {
	ShipmentID  uuid.UUID
	SellerID    uuid.UUID
	DeliveredAt *time.Time // nil → current time
}

// CreateShipmentInput holds the parameters for auto-creating a shipment on order.paid.
type CreateShipmentInput struct {
	SellerID        uuid.UUID
	OrderID         uuid.UUID
	BuyerAuth0ID    string
	ShippingAddress []byte // raw JSON snapshot from order
}

// ListShipmentsInput holds the pagination and filter params for listing.
type ListShipmentsInput struct {
	SellerID uuid.UUID
	Status   string
	Limit    int
	Offset   int
}

// ListShipmentsResult is the paginated response.
type ListShipmentsResult struct {
	Items  []*domain.Shipment
	Total  int
	Limit  int
	Offset int
}

// ShipmentUseCase is the driving port consumed by HTTP handlers and gRPC server.
type ShipmentUseCase interface {
	// CreateShipment creates a ready_to_ship shipment from an order.paid event.
	// Idempotent — duplicate calls for the same order_id are silently ignored.
	CreateShipment(ctx context.Context, in CreateShipmentInput) error

	// RegisterShipment records carrier + tracking_number and transitions to shipped.
	RegisterShipment(ctx context.Context, in RegisterShipmentInput) (*domain.Shipment, error)

	// MarkDelivered transitions the shipment to delivered.
	MarkDelivered(ctx context.Context, in MarkDeliveredInput) (*domain.Shipment, error)

	// CancelShipment transitions to cancelled (driven by order.cancelled).
	// Idempotent — if the shipment is already shipped/delivered, it is a no-op.
	CancelShipment(ctx context.Context, orderID uuid.UUID) error

	// GetByID returns a shipment by its ID. Verifies seller ownership.
	GetByID(ctx context.Context, sellerID, id uuid.UUID) (*domain.Shipment, error)

	// GetByOrderIDSeller returns a shipment by order ID. Verifies seller ownership.
	GetByOrderIDSeller(ctx context.Context, sellerID, orderID uuid.UUID) (*domain.Shipment, error)

	// GetByOrderIDBuyer returns a shipment by order ID. Verifies buyer identity.
	GetByOrderIDBuyer(ctx context.Context, buyerAuth0ID string, orderID uuid.UUID) (*domain.Shipment, error)

	// ListBySeller returns paginated shipments for a seller.
	ListBySeller(ctx context.Context, in ListShipmentsInput) (*ListShipmentsResult, error)
}
