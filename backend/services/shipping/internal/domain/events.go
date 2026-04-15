package domain

// Event type constants for the shipping-events topic. The payload schema is
// defined in services/shipping/api/proto/shipping/v1/events.proto and consumers
// should import shippingv1 to decode the typed payload.
const (
	EventTypeShipmentShipped   = "shipment.shipped"
	EventTypeShipmentDelivered = "shipment.delivered"
	EventTypeShipmentCancelled = "shipment.cancelled"
)
