package grpcserver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	orderv1 "github.com/Riku-KANO/ec-test/gen/go/order/v1"
	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/service"
)

// Server implements the orderv1.OrderServiceServer interface.
//
// The cancellation RPCs declared in order.proto are INTENTIONALLY served
// as codes.Unimplemented by this server — see cancellation_server.go for
// the full rationale. The order cancellation workflow is only reachable
// through the REST handler today because the gRPC messages carry no
// authenticated caller identity and there is no gRPC auth interceptor.
type Server struct {
	orderv1.UnimplementedOrderServiceServer
	svc *service.OrderService
}

// NewServer creates a new gRPC server wrapping the existing OrderService.
func NewServer(svc *service.OrderService) *Server {
	return &Server{svc: svc}
}

// CreateOrder creates a new order with Stripe PaymentIntent.
//
// The single-seller CreateOrder RPC is marked deprecated in the proto file;
// this server still implements it for backwards compatibility with existing
// clients. New callers should use CreateCheckout instead.
//
//nolint:staticcheck // SA1019: implementing deprecated RPC for backwards compatibility
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
//
// Note: the gRPC request shape (tenant_id, id, status) does not
// carry an authenticated seller identity, so this adapter cannot
// independently enforce seller ownership before calling the service.
// It resolves the order's seller_id up front and passes it as the
// ownership argument — this effectively makes the check a no-op on
// the gRPC path, matching the pre-existing behavior. The REST
// handler (order_handler.UpdateStatus) IS tightened: it extracts
// seller_id from tenant.Context and passes the authenticated caller
// to the service, which is where the real ownership guarantee lives.
// If the gRPC path ever needs a real auth check, the proto message
// must add a seller_id field and this adapter should pass the
// authenticated caller just like the REST handler does.
func (s *Server) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	orderID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order id: %v", err)
	}

	// Resolve the order's current seller_id so the service's
	// ownership comparison is a no-op rather than a hard 404. See
	// the method comment for why this is intentional on this path.
	existing, err := s.svc.GetOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, serviceErrToGRPC(err)
	}

	if err := s.svc.UpdateOrderStatus(ctx, tenantID, existing.SellerID, orderID, req.GetStatus()); err != nil {
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

// serviceErrToGRPC converts application-level errors to gRPC status
// errors. The mapping is primarily driven by *apperrors.AppError, so any
// handler that returns a typed AppError gets a meaningful gRPC code and
// does not leak the business error as codes.Internal.
//
// Mapping rules:
//
//	400 → InvalidArgument       401 → Unauthenticated
//	403 → PermissionDenied      404 → NotFound
//	409 → FailedPrecondition    (AlreadyExists when Code ends in
//	                             "_ALREADY_EXISTS")
//	429 → ResourceExhausted     502/503/504 → Unavailable
//	5xx → Internal              (other)
//
// When an AppError carries a semantic code (Code field) it is prefixed
// into the gRPC message so clients that log the status can still see
// the stable business code. Non-AppError errors fall back to simple
// substring matching so domain / repository sentinels returned without
// wrapping still land on a sensible code (NotFound for "not found",
// InvalidArgument for "invalid" / "bad request").
func serviceErrToGRPC(err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErrorToGRPC(appErr)
	}

	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "not found"):
		return status.Errorf(codes.NotFound, "%s", msg)
	case strings.Contains(lower, "bad request"), strings.Contains(lower, "invalid"):
		return status.Errorf(codes.InvalidArgument, "%s", msg)
	default:
		return status.Errorf(codes.Internal, "%s", msg)
	}
}

// appErrorToGRPC maps a typed *apperrors.AppError onto the closest gRPC
// status code. See serviceErrToGRPC for the full table.
func appErrorToGRPC(appErr *apperrors.AppError) error {
	code := appErrorGRPCCode(appErr)
	// Include the semantic code (when present) in the message so a
	// client that logs the raw gRPC status line still sees the stable
	// business code — e.g. "ORDER_NOT_CANCELLABLE: order cannot be
	// cancelled in its current status".
	if appErr.Code != "" {
		return status.Errorf(code, "%s: %s", appErr.Code, appErr.Message)
	}
	return status.Errorf(code, "%s", appErr.Message)
}

func appErrorGRPCCode(appErr *apperrors.AppError) codes.Code {
	switch appErr.Status {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		// Duplicate-resource errors map to AlreadyExists; state-machine
		// conflicts (ORDER_NOT_CANCELLABLE, *_ALREADY_PROCESSED) map to
		// FailedPrecondition — AlreadyExists would mislead a client
		// into thinking a retry with a different id would succeed.
		if strings.HasSuffix(appErr.Code, "_ALREADY_EXISTS") {
			return codes.AlreadyExists
		}
		return codes.FailedPrecondition
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		// Upstream failure (e.g. Stripe refund / reversal). Unavailable
		// signals a transient error that clients can surface / retry.
		return codes.Unavailable
	default:
		return codes.Internal
	}
}
