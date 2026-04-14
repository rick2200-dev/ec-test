package domain

import "time"

// Event type constants for the shipping-events topic.
const (
	EventTypeShipmentShipped   = "shipment.shipped"
	EventTypeShipmentDelivered = "shipment.delivered"
	EventTypeShipmentCancelled = "shipment.cancelled"
)

// ShipmentShippedEvent is published when a seller registers a tracking number.
// Subscribers: notification (buyer email), order (status update).
type ShipmentShippedEvent struct {
	ShipmentID     string    `json:"shipment_id"`
	OrderID        string    `json:"order_id"`
	TenantID       string    `json:"tenant_id"`
	SellerID       string    `json:"seller_id"`
	BuyerAuth0ID   string    `json:"buyer_auth0_id"`
	Carrier        string    `json:"carrier"`
	TrackingNumber string    `json:"tracking_number"`
	ShippedAt      time.Time `json:"shipped_at"`
}

// ShipmentDeliveredEvent is published when a seller (or future carrier webhook)
// marks a shipment as delivered.
// Subscribers: notification (buyer email + review prompt), order (status update).
type ShipmentDeliveredEvent struct {
	ShipmentID   string    `json:"shipment_id"`
	OrderID      string    `json:"order_id"`
	TenantID     string    `json:"tenant_id"`
	SellerID     string    `json:"seller_id"`
	BuyerAuth0ID string    `json:"buyer_auth0_id"`
	DeliveredAt  time.Time `json:"delivered_at"`
}

// ShipmentCancelledEvent is published when a shipment is cancelled
// (driven by order.cancelled). v1: audit log only.
type ShipmentCancelledEvent struct {
	ShipmentID string `json:"shipment_id"`
	OrderID    string `json:"order_id"`
	TenantID   string `json:"tenant_id"`
	Reason     string `json:"reason"`
}
