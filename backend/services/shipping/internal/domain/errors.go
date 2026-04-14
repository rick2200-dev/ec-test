package domain

import "errors"

var (
	ErrShipmentNotFound       = errors.New("shipment not found")
	ErrInvalidTransition      = errors.New("invalid shipment status transition")
	ErrAlreadyRegistered      = errors.New("shipment already registered")
	ErrTrackingNumberRequired = errors.New("carrier and tracking_number are required")
	ErrNotOrderSeller         = errors.New("caller is not the order seller")
	ErrNotOrderBuyer          = errors.New("caller is not the order buyer")
)
