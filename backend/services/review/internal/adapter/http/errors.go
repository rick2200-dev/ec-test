package handler

import (
	"errors"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/review/internal/domain"
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
	case errors.Is(err, domain.ErrReviewNotFound),
		errors.Is(err, domain.ErrReplyNotFound),
		errors.Is(err, domain.ErrProductNotFound):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrAlreadyReviewed):
		return apperrors.Conflict(err.Error()).WithCode("ALREADY_REVIEWED")
	case errors.Is(err, domain.ErrAlreadyReplied):
		return apperrors.Conflict(err.Error()).WithCode("ALREADY_REPLIED")
	case errors.Is(err, domain.ErrPurchaseRequired):
		return apperrors.Forbidden(err.Error()).WithCode("PURCHASE_REQUIRED")
	case errors.Is(err, domain.ErrNotReviewOwner),
		errors.Is(err, domain.ErrNotSellerOfProduct):
		return apperrors.Forbidden(err.Error())
	case errors.Is(err, domain.ErrInvalidRating):
		return apperrors.BadRequest(err.Error())
	default:
		return apperrors.Internal("internal error", err)
	}
}
