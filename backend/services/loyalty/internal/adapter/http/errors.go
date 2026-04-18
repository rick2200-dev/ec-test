package handler

import (
	"errors"
	"net/http"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
)

// mapError translates domain / app-layer errors to AppError with the
// appropriate HTTP status. App-layer code already wraps infrastructure
// failures as *apperrors.AppError, so those pass through.
func mapError(err error) *apperrors.AppError {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrInsufficientPoints):
		return apperrors.BadRequest(err.Error()).WithCode("INSUFFICIENT_POINTS")
	case errors.Is(err, domain.ErrReservationNotFound):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrReservationAlreadyTerminal):
		return apperrors.Conflict(err.Error()).WithCode("RESERVATION_TERMINAL")
	case errors.Is(err, domain.ErrInvalidAmount):
		return apperrors.BadRequest(err.Error())
	}
	return &apperrors.AppError{Status: http.StatusInternalServerError, Message: "internal error", Err: err}
}
