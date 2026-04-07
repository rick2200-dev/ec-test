package grpcserver

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	orderv1 "github.com/Riku-KANO/ec-test/gen/go/order/v1"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/service"
)

// Server implements the orderv1.OrderServiceServer interface.
type Server struct {
	orderv1.UnimplementedOrderServiceServer
	svc *service.OrderService
}

// NewServer creates a new gRPC server wrapping the existing OrderService.
func NewServer(svc *service.OrderService) *Server {
	return &Server{svc: svc}
}

// CreateOrder creates a new order with Stripe PaymentIntent.
func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	sellerID, err := uuid.Parse(req.GetSellerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller_id: %v", err)
	}

	input := domain.CreateOrderInput{
		SellerID:        sellerID,
		BuyerAuth0ID:    req.GetBuyerAuth0Id(),
		Lines:           protoLinesToDomain(req.GetLines()),
		ShippingAddress: parseShippingAddress(req.GetShippingAddressJson()),
	}

	orderWithLines, clientSecret, err := s.svc.CreateOrder(ctx, tenantID, input)
	if err != nil {
		return nil, serviceErrToGRPC(err)
	}

	return &orderv1.CreateOrderResponse{
		Order:              orderToProto(&orderWithLines.Order, orderWithLines.Lines),
		StripeClientSecret: clientSecret,
	}, nil
}

// GetOrder retrieves an order by ID.
func (s *Server) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	orderID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order id: %v", err)
	}

	orderWithLines, err := s.svc.GetOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, serviceErrToGRPC(err)
	}

	return &orderv1.GetOrderResponse{
		Order: orderToProto(&orderWithLines.Order, orderWithLines.Lines),
	}, nil
}

// ListBuyerOrders returns paginated orders for a buyer.
func (s *Server) ListBuyerOrders(ctx context.Context, req *orderv1.ListBuyerOrdersRequest) (*orderv1.ListBuyerOrdersResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	limit, offset := paginationDefaults(req.GetPagination())

	orders, total, err := s.svc.ListBuyerOrders(ctx, tenantID, req.GetBuyerAuth0Id(), limit, offset)
	if err != nil {
		return nil, serviceErrToGRPC(err)
	}

	pbOrders := make([]*orderv1.Order, 0, len(orders))
	for i := range orders {
		pbOrders = append(pbOrders, orderSummaryToProto(&orders[i]))
	}

	return &orderv1.ListBuyerOrdersResponse{
		Orders: pbOrders,
		Pagination: &commonv1.PaginationResponse{
			Total:  int32(total),
			Limit:  int32(limit),
			Offset: int32(offset),
		},
	}, nil
}

// ListSellerOrders returns paginated orders for a seller.
func (s *Server) ListSellerOrders(ctx context.Context, req *orderv1.ListSellerOrdersRequest) (*orderv1.ListSellerOrdersResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	sellerID, err := uuid.Parse(req.GetSellerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller_id: %v", err)
	}

	limit, offset := paginationDefaults(req.GetPagination())

	orders, total, err := s.svc.ListSellerOrders(ctx, tenantID, sellerID, req.GetStatus(), limit, offset)
	if err != nil {
		return nil, serviceErrToGRPC(err)
	}

	pbOrders := make([]*orderv1.Order, 0, len(orders))
	for i := range orders {
		pbOrders = append(pbOrders, orderSummaryToProto(&orders[i]))
	}

	return &orderv1.ListSellerOrdersResponse{
		Orders: pbOrders,
		Pagination: &commonv1.PaginationResponse{
			Total:  int32(total),
			Limit:  int32(limit),
			Offset: int32(offset),
		},
	}, nil
}

// UpdateOrderStatus updates the status of an order.
func (s *Server) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	orderID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order id: %v", err)
	}

	if err := s.svc.UpdateOrderStatus(ctx, tenantID, orderID, req.GetStatus()); err != nil {
		return nil, serviceErrToGRPC(err)
	}

	// Fetch the updated order to return it.
	orderWithLines, err := s.svc.GetOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, serviceErrToGRPC(err)
	}

	return &orderv1.UpdateOrderStatusResponse{
		Order: orderToProto(&orderWithLines.Order, orderWithLines.Lines),
	}, nil
}

// ListPayouts returns paginated payouts for a seller.
func (s *Server) ListPayouts(ctx context.Context, req *orderv1.ListPayoutsRequest) (*orderv1.ListPayoutsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	sellerID, err := uuid.Parse(req.GetSellerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller_id: %v", err)
	}

	limit, offset := paginationDefaults(req.GetPagination())

	payouts, total, err := s.svc.ListPayouts(ctx, tenantID, sellerID, limit, offset)
	if err != nil {
		return nil, serviceErrToGRPC(err)
	}

	pbPayouts := make([]*orderv1.Payout, 0, len(payouts))
	for i := range payouts {
		pbPayouts = append(pbPayouts, payoutToProto(&payouts[i]))
	}

	return &orderv1.ListPayoutsResponse{
		Payouts: pbPayouts,
		Pagination: &commonv1.PaginationResponse{
			Total:  int32(total),
			Limit:  int32(limit),
			Offset: int32(offset),
		},
	}, nil
}

// paginationDefaults extracts limit and offset from a PaginationRequest, applying defaults.
func paginationDefaults(p *commonv1.PaginationRequest) (int, int) {
	limit := 20
	offset := 0
	if p != nil {
		if p.Limit > 0 {
			limit = int(p.Limit)
		}
		if p.Offset > 0 {
			offset = int(p.Offset)
		}
	}
	return limit, offset
}

// serviceErrToGRPC converts application-level errors to gRPC status errors.
func serviceErrToGRPC(err error) error {
	if err == nil {
		return nil
	}
	// Check for common error patterns from the apperrors package.
	msg := err.Error()

	// The apperrors package wraps errors with specific types.
	// We use simple string matching as a pragmatic approach.
	switch {
	case contains(msg, "not found"):
		return status.Errorf(codes.NotFound, "%s", msg)
	case contains(msg, "bad request"), contains(msg, "invalid"):
		return status.Errorf(codes.InvalidArgument, "%s", msg)
	default:
		return status.Errorf(codes.Internal, "%s", msg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
