package handler

import (
	"errors"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/domain"
)

// mapError converts domain sentinel errors to HTTP-aware AppErrors.
// Infrastructure errors already wrapped as AppError pass through unchanged.
func mapError(err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	switch {
	case errors.Is(err, domain.ErrShipmentNotFound),
		errors.Is(err, domain.ErrNotOrderSeller),
		errors.Is(err, domain.ErrNotOrderBuyer):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrTrackingNumberRequired):
		return apperrors.BadRequest(err.Error()).WithCode("TRACKING_NUMBER_REQUIRED")
	case errors.Is(err, domain.ErrAlreadyRegistered):
		return apperrors.Conflict(err.Error()).WithCode("SHIPMENT_ALREADY_REGISTERED")
	case errors.Is(err, domain.ErrInvalidTransition):
		return apperrors.Conflict(err.Error()).WithCode("SHIPMENT_INVALID_TRANSITION")
	default:
		return apperrors.Internal("internal error", err)
	}
}
