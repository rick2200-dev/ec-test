package handler

import (
	"errors"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// mapError converts domain sentinel errors to HTTP-aware AppErrors.
// Infrastructure errors already wrapped as AppError pass through unchanged.
// Any unrecognised error becomes a generic 500.
func mapError(err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrEmptyOrder),
		errors.Is(err, domain.ErrBuyerRequired),
		errors.Is(err, domain.ErrInvalidQuantity),
		errors.Is(err, domain.ErrInvalidOrderStatus):
		return apperrors.BadRequest(err.Error())
	case errors.Is(err, domain.ErrOrderNotPending):
		return apperrors.Conflict(err.Error())
	default:
		return apperrors.Internal("internal error", err)
	}
}
