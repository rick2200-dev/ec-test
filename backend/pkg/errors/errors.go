package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard application errors.
var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrBadRequest   = errors.New("bad request")
	ErrInternal     = errors.New("internal error")
)

// AppError is a structured application error returned by handlers.
//
// The wire shape is `{"error": "<message>", "code": "<CODE>"}`. `code` is
// omitted when unset, so legacy handlers that never set a code keep the
// exact same response body they always had.
//
// Two distinct concepts live side-by-side:
//
//   - Status: the HTTP status code (400, 404, ...). Drives the response
//     status line and is also what cacheing/monitoring tiers see.
//   - Code:   a stable, application-defined string like "INVALID_EMAIL" or
//     "DUPLICATE_SKU". Clients switch on this to pick a display
//     pattern for errors that share the same HTTP status — e.g.
//     two different 400s that must show two different UI messages.
//
// Services own their own Code values. Keep them SCREAMING_SNAKE_CASE and
// stable across releases: they are part of the public API contract, same
// as field names in a response body. Renaming a Code is a breaking change
// for any client switching on it.
type AppError struct {
	// Status is the HTTP status code. Not serialized in the body; it is
	// written to the response status line by httputil.Error.
	Status int `json:"-"`
	// Code is an optional application-defined error code. Empty when the
	// caller did not attach one. Serialized as `code`.
	Code string `json:"code,omitempty"`
	// Message is the human-readable message. Serialized as `error` for
	// backwards compatibility with the previous `{"error": "..."}` shape.
	Message string `json:"error"`
	// Err is the underlying cause used for errors.Is / Unwrap. Never
	// serialized — underlying errors may contain sensitive details.
	Err error `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// WithCode attaches a semantic error code and returns the receiver so
// constructors can be chained:
//
//	return apperrors.BadRequest("email already registered").
//	    WithCode("DUPLICATE_EMAIL")
//
// It mutates the receiver in place, which is fine because AppError values
// are always freshly constructed by the helpers below (never shared).
func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

// New creates a new AppError with the given HTTP status and message.
func New(status int, message string, err error) *AppError {
	return &AppError{Status: status, Message: message, Err: err}
}

// NotFound returns an AppError with HTTP 404. Use when a requested resource
// does not exist or the caller is not permitted to know it exists.
func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Message: msg, Err: ErrNotFound}
}

// BadRequest returns an AppError with HTTP 400 for invalid client input.
// Pair with WithCode when the client needs to distinguish between several
// kinds of 400s (e.g. "INVALID_EMAIL" vs "PASSWORD_TOO_SHORT").
func BadRequest(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Message: msg, Err: ErrBadRequest}
}

// Unauthorized returns an AppError with HTTP 401. Use when authentication is
// required but missing or invalid. Tells the client to re-authenticate.
func Unauthorized(msg string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Message: msg, Err: ErrUnauthorized}
}

// Forbidden returns an AppError with HTTP 403 when the authenticated user
// lacks permission for the requested operation.
func Forbidden(msg string) *AppError {
	return &AppError{Status: http.StatusForbidden, Message: msg, Err: ErrForbidden}
}

// Conflict returns an AppError with HTTP 409 for duplicate resource creation
// or concurrent modification conflicts.
func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Message: msg, Err: ErrConflict}
}

// Internal wraps an unexpected error as HTTP 500. msg is the user-visible
// message; err is the underlying cause logged for debugging (may be nil).
func Internal(msg string, err error) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Message: msg, Err: err}
}
