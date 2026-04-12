package handler

import (
	"errors"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
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
	case errors.Is(err, domain.ErrInquiryNotFound):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrInquiryClosed),
		errors.Is(err, domain.ErrNotParticipant),
		errors.Is(err, domain.ErrPurchaseRequired):
		return apperrors.Forbidden(err.Error())
	case errors.Is(err, domain.ErrInvalidSenderType),
		errors.Is(err, domain.ErrInvalidReaderType):
		return apperrors.BadRequest(err.Error())
	default:
		return apperrors.Internal("internal error", err)
	}
}
