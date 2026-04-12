package handler

import (
	"errors"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
)

// mapError converts domain sentinel errors to HTTP-aware AppErrors.
// Infrastructure errors wrapped as AppError already pass through unchanged.
// Any unrecognised error becomes a generic 500.
func mapError(err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	switch {
	case errors.Is(err, domain.ErrMissingTenantID),
		errors.Is(err, domain.ErrMissingUserID),
		errors.Is(err, domain.ErrMissingProductID),
		errors.Is(err, domain.ErrInvalidRecommendationType),
		errors.Is(err, domain.ErrInvalidEventType):
		return apperrors.BadRequest(err.Error())
	default:
		return apperrors.Internal("internal error", err)
	}
}
