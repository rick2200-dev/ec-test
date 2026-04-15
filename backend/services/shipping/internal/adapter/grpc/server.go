package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	shippingv1 "github.com/Riku-KANO/ec-test/services/shipping/api/gen/go/shipping/v1"
)

// Server implements shippingv1.ShippingServiceServer.
//
// NOTE: All RPCs currently return codes.Unimplemented. The primary call path
// for v1 is HTTP proxied through the gateway. gRPC will be enabled once a
// gRPC auth interceptor is in place so caller identity can be propagated
// securely in metadata (see docs/shipping.md § future work).
type Server struct {
	shippingv1.UnimplementedShippingServiceServer
}

// NewServer creates a new gRPC Server.
func NewServer() *Server {
	return &Server{}
}

// GetShipmentByOrder is currently unimplemented.
func (s *Server) GetShipmentByOrder(_ context.Context, _ *shippingv1.GetShipmentByOrderRequest) (*shippingv1.GetShipmentByOrderResponse, error) {
	return nil, status.Error(codes.Unimplemented, "use HTTP API")
}

// RegisterShipment is currently unimplemented.
func (s *Server) RegisterShipment(_ context.Context, _ *shippingv1.RegisterShipmentRequest) (*shippingv1.RegisterShipmentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "use HTTP API")
}

// MarkDelivered is currently unimplemented.
func (s *Server) MarkDelivered(_ context.Context, _ *shippingv1.MarkDeliveredRequest) (*shippingv1.MarkDeliveredResponse, error) {
	return nil, status.Error(codes.Unimplemented, "use HTTP API")
}
