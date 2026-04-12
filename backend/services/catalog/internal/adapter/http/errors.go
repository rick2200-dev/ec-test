package handler

import (
	"errors"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
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
	case errors.Is(err, domain.ErrCategoryNotFound),
		errors.Is(err, domain.ErrProductNotFound),
		errors.Is(err, domain.ErrSKUNotFound):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrCategorySlugConflict),
		errors.Is(err, domain.ErrProductSlugConflict):
		return apperrors.Conflict(err.Error())
	case errors.Is(err, domain.ErrSellerRequired):
		return apperrors.BadRequest(err.Error())
	case errors.Is(err, domain.ErrNotProductOwner):
		return apperrors.Forbidden(err.Error())
	default:
		return apperrors.Internal("internal error", err)
	}
}
