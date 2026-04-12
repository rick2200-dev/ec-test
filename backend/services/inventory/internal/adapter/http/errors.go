package handler

import (
	"errors"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
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
	case errors.Is(err, domain.ErrInventoryNotFound):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrInvalidQuantity):
		return apperrors.BadRequest(err.Error())
	default:
		return apperrors.Internal("internal error", err)
	}
}
