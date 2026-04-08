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

// AppError is a structured application error with an HTTP status code.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
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

// New creates a new AppError.
func New(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// NotFound returns an AppError with HTTP 404. Use when a requested resource
// does not exist or the caller is not permitted to know it exists.
func NotFound(msg string) *AppError {
	return &AppError{Code: http.StatusNotFound, Message: msg, Err: ErrNotFound}
}

// BadRequest returns an AppError with HTTP 400 for invalid client input.
func BadRequest(msg string) *AppError {
	return &AppError{Code: http.StatusBadRequest, Message: msg, Err: ErrBadRequest}
}

// Unauthorized returns an AppError with HTTP 401. Use when authentication is
// required but missing or invalid. Tells the client to re-authenticate.
func Unauthorized(msg string) *AppError {
	return &AppError{Code: http.StatusUnauthorized, Message: msg, Err: ErrUnauthorized}
}

// Forbidden returns an AppError with HTTP 403 when the authenticated user
// lacks permission for the requested operation.
func Forbidden(msg string) *AppError {
	return &AppError{Code: http.StatusForbidden, Message: msg, Err: ErrForbidden}
}

// Conflict returns an AppError with HTTP 409 for duplicate resource creation
// or concurrent modification conflicts.
func Conflict(msg string) *AppError {
	return &AppError{Code: http.StatusConflict, Message: msg, Err: ErrConflict}
}

// Internal wraps an unexpected error as HTTP 500. msg is the user-visible
// message; err is the underlying cause logged for debugging (may be nil).
func Internal(msg string, err error) *AppError {
	return &AppError{Code: http.StatusInternalServerError, Message: msg, Err: err}
}
