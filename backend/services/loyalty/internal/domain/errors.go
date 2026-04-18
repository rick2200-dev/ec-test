package domain

import "errors"

// Domain sentinel errors. Adapters translate these to transport-specific
// failures (HTTP status, gRPC code) — they are intentionally
// transport-agnostic.
var (
	// ErrAccountNotFound is returned when a balance/transaction lookup
	// targets a buyer that has no point_accounts row yet.
	ErrAccountNotFound = errors.New("point account not found")

	// ErrInsufficientPoints is returned from ReservePoints when the
	// requested amount exceeds (balance - pending_redemption).
	ErrInsufficientPoints = errors.New("insufficient point balance")

	// ErrReservationNotFound is returned when Commit or Release cannot
	// locate the referenced reservation.
	ErrReservationNotFound = errors.New("point reservation not found")

	// ErrReservationAlreadyTerminal means the reservation is no longer in
	// pending state (committed / released / expired) and cannot be
	// transitioned further. Callers get idempotent behavior via
	// ReservationID-based lookups elsewhere.
	ErrReservationAlreadyTerminal = errors.New("point reservation no longer pending")

	// ErrInvalidAmount is a catch-all for zero/negative amounts where the
	// API expects a positive value.
	ErrInvalidAmount = errors.New("invalid point amount")
)
