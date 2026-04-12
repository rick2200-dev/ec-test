package domain

import "errors"

// Domain-level sentinel errors for the inquiry service.
// These are transport-agnostic; the handler layer maps them to HTTP status codes.
var (
	// ErrInquiryNotFound is returned when an inquiry does not exist or does
	// not belong to the requesting participant.
	ErrInquiryNotFound = errors.New("inquiry not found")

	// ErrInquiryClosed is returned when a message is posted to a closed inquiry.
	ErrInquiryClosed = errors.New("cannot post to a closed inquiry")

	// ErrNotParticipant is returned when the caller is not a participant of
	// the inquiry they are trying to interact with.
	ErrNotParticipant = errors.New("not a participant of this inquiry")

	// ErrPurchaseRequired is returned when the buyer has not purchased the SKU
	// they are trying to inquire about.
	ErrPurchaseRequired = errors.New("you can only contact the seller of items you have purchased")

	// ErrInvalidSenderType is returned when the sender_type field is not a
	// known value.
	ErrInvalidSenderType = errors.New("invalid sender_type")

	// ErrInvalidReaderType is returned when the reader_type field is not a
	// known value.
	ErrInvalidReaderType = errors.New("invalid reader type")
)
