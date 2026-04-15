package handler

import (
	"errors"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
)

// mapError converts domain sentinel errors to HTTP-aware AppErrors.
// Infrastructure errors wrapped as AppError already pass through unchanged.
// Any unrecognised error becomes a generic 500.
func mapError(err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return apperrors.Internal("internal error", err)
}
