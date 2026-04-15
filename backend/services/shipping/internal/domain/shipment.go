package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Status constants for shipment lifecycle.
const (
	StatusPending     = "pending"
	StatusReadyToShip = "ready_to_ship"
	StatusShipped     = "shipped"
	StatusDelivered   = "delivered"
	StatusCancelled   = "cancelled"
)

// Shipment is the root aggregate for a single delivery.
// In v1 there is a 1:1 relationship between an order and a shipment.
type Shipment struct {
	ID              uuid.UUID
	SellerID        uuid.UUID
	OrderID         uuid.UUID
	BuyerAuth0ID    string
	Status          string
	ShippingAddress json.RawMessage
	Carrier         string
	TrackingNumber  string
	ShippedAt       *time.Time
	DeliveredAt     *time.Time
	Note            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ShipmentEvent records a single status transition on a shipment.
type ShipmentEvent struct {
	ID         uuid.UUID
	ShipmentID uuid.UUID
	FromStatus string
	ToStatus   string
	ActorType  string // "seller" | "system"
	ActorID    string
	Payload    json.RawMessage
	CreatedAt  time.Time
}

// CanRegister returns true if the shipment is in a state where tracking
// info can be recorded (i.e. ready_to_ship).
func (s *Shipment) CanRegister() bool {
	return s.Status == StatusReadyToShip
}

// CanDeliver returns true if the shipment is in a state where delivery
// can be confirmed (i.e. shipped).
func (s *Shipment) CanDeliver() bool {
	return s.Status == StatusShipped
}

// CanCancel returns true if the shipment can be cancelled (before shipment).
func (s *Shipment) CanCancel() bool {
	return s.Status == StatusPending || s.Status == StatusReadyToShip
}
