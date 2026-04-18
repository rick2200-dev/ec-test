// Package grpcserver is the gRPC-side adapter for the loyalty service.
// Only AdjustPoints is live — the buyer/internal RPCs remain on HTTP
// because no in-cluster caller consumes them via gRPC today. Adding a
// single RPC here establishes the listener + interceptor wiring so
// future admin/manager RPCs can land without a second standup cost.
package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	loyaltyv1 "github.com/Riku-KANO/ec-test/services/loyalty/api/gen/go/loyalty/v1"
	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// Server implements LoyaltyServiceServer. Unimplemented methods fall
// through to the embedded UnimplementedLoyaltyServiceServer.
type Server struct {
	loyaltyv1.UnimplementedLoyaltyServiceServer
	svc port.LoyaltyUseCase
}

// NewServer wires the gRPC handler onto the loyalty use case.
func NewServer(svc port.LoyaltyUseCase) *Server {
	return &Server{svc: svc}
}

// AdjustPoints forwards to the app layer. The transport-level
// X-Internal-Token interceptor is applied at server construction in
// main.go — nothing further to check here.
func (s *Server) AdjustPoints(ctx context.Context, req *loyaltyv1.AdjustPointsRequest) (*loyaltyv1.AdjustPointsResponse, error) {
	if req.GetBuyerAuth0Id() == "" {
		return nil, status.Error(codes.InvalidArgument, "buyer_auth0_id is required")
	}
	if req.GetAmount() == 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be non-zero")
	}
	if req.GetReason() == "" {
		return nil, status.Error(codes.InvalidArgument, "reason is required")
	}
	result, err := s.svc.AdjustPoints(ctx, port.AdjustPointsInput{
		BuyerAuth0ID: req.GetBuyerAuth0Id(),
		Amount:       req.GetAmount(),
		Reason:       req.GetReason(),
		AdminUserID:  req.GetAdminUserId(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &loyaltyv1.AdjustPointsResponse{
		Transaction: transactionToProto(result.Transaction),
		NewBalance:  result.NewBalance,
	}, nil
}

func toGRPCError(err error) error {
	if errors.Is(err, domain.ErrInsufficientPoints) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}
	if errors.Is(err, domain.ErrInvalidAmount) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, domain.ErrAccountNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		switch {
		case errors.Is(appErr.Err, apperrors.ErrNotFound):
			return status.Error(codes.NotFound, appErr.Message)
		case errors.Is(appErr.Err, apperrors.ErrBadRequest):
			return status.Error(codes.InvalidArgument, appErr.Message)
		case errors.Is(appErr.Err, apperrors.ErrConflict):
			return status.Error(codes.AlreadyExists, appErr.Message)
		case errors.Is(appErr.Err, apperrors.ErrForbidden):
			return status.Error(codes.PermissionDenied, appErr.Message)
		case errors.Is(appErr.Err, apperrors.ErrUnauthorized):
			return status.Error(codes.Unauthenticated, appErr.Message)
		default:
			return status.Error(codes.Internal, appErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
