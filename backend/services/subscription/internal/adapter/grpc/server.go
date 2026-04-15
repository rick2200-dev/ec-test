package grpcserver

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	subscriptionv1 "github.com/Riku-KANO/ec-test/gen/go/subscription/v1"
	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/domain"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/port"
)

// Server implements the SubscriptionServiceServer gRPC interface.
type Server struct {
	subscriptionv1.UnimplementedSubscriptionServiceServer
	svc port.SubscriptionUseCase
}

// NewServer creates a new gRPC Server wrapping the subscription use case.
func NewServer(svc port.SubscriptionUseCase) *Server {
	return &Server{svc: svc}
}

// --- Seller Plans ---

func (s *Server) ListSellerPlans(ctx context.Context, _ *subscriptionv1.ListSellerPlansRequest) (*subscriptionv1.ListSellerPlansResponse, error) {
	plans, err := s.svc.ListPlans(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*subscriptionv1.SellerPlan, 0, len(plans))
	for i := range plans {
		out = append(out, sellerPlanToProto(&plans[i]))
	}
	return &subscriptionv1.ListSellerPlansResponse{Plans: out}, nil
}

func (s *Server) GetSellerPlan(ctx context.Context, req *subscriptionv1.GetSellerPlanRequest) (*subscriptionv1.GetSellerPlanResponse, error) {
	pid, err := uuid.Parse(req.GetPlanId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid plan_id")
	}
	p, err := s.svc.GetPlan(ctx, pid)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.GetSellerPlanResponse{Plan: sellerPlanToProto(p)}, nil
}

func (s *Server) CreateSellerPlan(ctx context.Context, req *subscriptionv1.CreateSellerPlanRequest) (*subscriptionv1.CreateSellerPlanResponse, error) {
	plan := &domain.SubscriptionPlan{
		Name:          req.GetName(),
		Slug:          req.GetSlug(),
		Tier:          int(req.GetTier()),
		PriceAmount:   req.GetPrice().GetAmount(),
		PriceCurrency: req.GetPrice().GetCurrency(),
		Features:      planFeaturesFromProto(req.GetFeatures()),
		StripePriceID: req.GetStripePriceId(),
		Status:        req.GetStatus(),
	}
	if err := s.svc.CreatePlan(ctx, plan); err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.CreateSellerPlanResponse{Plan: sellerPlanToProto(plan)}, nil
}

func (s *Server) UpdateSellerPlan(ctx context.Context, req *subscriptionv1.UpdateSellerPlanRequest) (*subscriptionv1.UpdateSellerPlanResponse, error) {
	pid, err := uuid.Parse(req.GetPlanId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid plan_id")
	}
	plan := &domain.SubscriptionPlan{
		ID:            pid,
		Name:          req.GetName(),
		Slug:          req.GetSlug(),
		Tier:          int(req.GetTier()),
		PriceAmount:   req.GetPrice().GetAmount(),
		PriceCurrency: req.GetPrice().GetCurrency(),
		Features:      planFeaturesFromProto(req.GetFeatures()),
		StripePriceID: req.GetStripePriceId(),
		Status:        req.GetStatus(),
	}
	if err := s.svc.UpdatePlan(ctx, plan); err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.UpdateSellerPlanResponse{Plan: sellerPlanToProto(plan)}, nil
}

// --- Seller Subscriptions ---

func (s *Server) GetSellerSubscription(ctx context.Context, req *subscriptionv1.GetSellerSubscriptionRequest) (*subscriptionv1.GetSellerSubscriptionResponse, error) {
	sid, err := uuid.Parse(req.GetSellerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid seller_id")
	}
	sub, err := s.svc.GetSellerSubscription(ctx, sid)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.GetSellerSubscriptionResponse{Subscription: sellerSubscriptionToProto(sub)}, nil
}

func (s *Server) SubscribeSeller(ctx context.Context, req *subscriptionv1.SubscribeSellerRequest) (*subscriptionv1.SubscribeSellerResponse, error) {
	sid, err := uuid.Parse(req.GetSellerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid seller_id")
	}
	pid, err := uuid.Parse(req.GetPlanId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid plan_id")
	}
	sub, err := s.svc.SubscribeSeller(ctx, sid, pid)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.SubscribeSellerResponse{Subscription: sellerSubscriptionBareToProto(sub)}, nil
}

// --- Buyer Plans ---

func (s *Server) ListBuyerPlans(ctx context.Context, _ *subscriptionv1.ListBuyerPlansRequest) (*subscriptionv1.ListBuyerPlansResponse, error) {
	plans, err := s.svc.ListBuyerPlans(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*subscriptionv1.BuyerPlan, 0, len(plans))
	for i := range plans {
		out = append(out, buyerPlanToProto(&plans[i]))
	}
	return &subscriptionv1.ListBuyerPlansResponse{Plans: out}, nil
}

func (s *Server) GetBuyerPlan(ctx context.Context, req *subscriptionv1.GetBuyerPlanRequest) (*subscriptionv1.GetBuyerPlanResponse, error) {
	pid, err := uuid.Parse(req.GetPlanId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid plan_id")
	}
	p, err := s.svc.GetBuyerPlan(ctx, pid)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.GetBuyerPlanResponse{Plan: buyerPlanToProto(p)}, nil
}

func (s *Server) CreateBuyerPlan(ctx context.Context, req *subscriptionv1.CreateBuyerPlanRequest) (*subscriptionv1.CreateBuyerPlanResponse, error) {
	plan := &domain.BuyerPlan{
		Name:          req.GetName(),
		Slug:          req.GetSlug(),
		PriceAmount:   req.GetPrice().GetAmount(),
		PriceCurrency: req.GetPrice().GetCurrency(),
		Features:      buyerPlanFeaturesFromProto(req.GetFeatures()),
		StripePriceID: req.GetStripePriceId(),
		Status:        req.GetStatus(),
	}
	if err := s.svc.CreateBuyerPlan(ctx, plan); err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.CreateBuyerPlanResponse{Plan: buyerPlanToProto(plan)}, nil
}

func (s *Server) UpdateBuyerPlan(ctx context.Context, req *subscriptionv1.UpdateBuyerPlanRequest) (*subscriptionv1.UpdateBuyerPlanResponse, error) {
	pid, err := uuid.Parse(req.GetPlanId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid plan_id")
	}
	plan := &domain.BuyerPlan{
		ID:            pid,
		Name:          req.GetName(),
		Slug:          req.GetSlug(),
		PriceAmount:   req.GetPrice().GetAmount(),
		PriceCurrency: req.GetPrice().GetCurrency(),
		Features:      buyerPlanFeaturesFromProto(req.GetFeatures()),
		StripePriceID: req.GetStripePriceId(),
		Status:        req.GetStatus(),
	}
	if err := s.svc.UpdateBuyerPlan(ctx, plan); err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.UpdateBuyerPlanResponse{Plan: buyerPlanToProto(plan)}, nil
}

// --- Buyer Subscriptions ---

func (s *Server) GetBuyerSubscription(ctx context.Context, req *subscriptionv1.GetBuyerSubscriptionRequest) (*subscriptionv1.GetBuyerSubscriptionResponse, error) {
	sub, err := s.svc.GetBuyerSubscription(ctx, req.GetBuyerAuth0Id())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.GetBuyerSubscriptionResponse{Subscription: buyerSubscriptionToProto(sub)}, nil
}

func (s *Server) SubscribeBuyer(ctx context.Context, req *subscriptionv1.SubscribeBuyerRequest) (*subscriptionv1.SubscribeBuyerResponse, error) {
	pid, err := uuid.Parse(req.GetPlanId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid plan_id")
	}
	sub, err := s.svc.SubscribeBuyer(ctx, req.GetBuyerAuth0Id(), pid)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &subscriptionv1.SubscribeBuyerResponse{Subscription: buyerSubscriptionBareToProto(sub)}, nil
}

func toGRPCError(err error) error {
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
