package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/Riku-KANO/ec-test/gen/go/order/v1"
)

// Cancellation RPCs are INTENTIONALLY unimplemented on the gRPC path.
//
// The proto file declares five cancellation RPCs (RequestOrderCancellation,
// ApproveOrderCancellation, RejectOrderCancellation,
// ListOrderCancellationRequests, GetOrderCancellationRequest) so the wire
// contract exists for future use, but this adapter refuses to serve them
// today because the gRPC request messages do not carry an authenticated
// buyer / seller identity AND there is no gRPC auth interceptor in front
// of the order service. Any adapter that resolved the caller from the
// order itself (e.g. "look up the order's buyer and pass it as the
// ownership argument") would make the service-layer ownership check a
// no-op and let any intra-cluster caller approve arbitrary cancellations —
// approval triggers Stripe refund + per-payout transfer reversals, so the
// blast radius is real money, not just a data read.
//
// The REST handler (cancellation/http_handler.go) remains the ONLY path
// buyers and sellers can open/approve/reject requests through; it extracts
// the authenticated identity from tenant.Context (populated by the HTTP
// auth middleware) and passes the real caller to the service.
//
// To turn these RPCs on, do one of:
//
//  1. Add authenticated actor fields to the proto messages and have the
//     adapter pass them straight through to the service (so the ownership
//     check runs against the caller, not the order), OR
//  2. Install a gRPC auth interceptor that parses the caller's JWT and
//     puts the resolved identity onto context, and have these handlers
//     pull from context just like the REST handlers pull from
//     tenant.Context.
//
// Until one of those lands, every mutation and every seller-scoped read
// returns codes.Unimplemented with a clear message so callers fail loudly
// instead of silently bypassing auth.

// unimplementedCancellationMsg is the single message string used by every
// stub below. Centralised so the guidance stays consistent and easy to
// grep for when wiring real auth.
const unimplementedCancellationMsg = "cancellation RPCs are unimplemented on the gRPC path; use the REST endpoints (see cancellation/http_handler.go). The gRPC messages do not carry an authenticated caller identity and there is no gRPC auth interceptor, so serving them would bypass the ownership check that guards Stripe refund and transfer-reversal side effects."

func (s *Server) RequestOrderCancellation(_ context.Context, _ *orderv1.RequestOrderCancellationRequest) (*orderv1.RequestOrderCancellationResponse, error) {
	return nil, status.Error(codes.Unimplemented, unimplementedCancellationMsg)
}

func (s *Server) ApproveOrderCancellation(_ context.Context, _ *orderv1.ApproveOrderCancellationRequest) (*orderv1.ApproveOrderCancellationResponse, error) {
	return nil, status.Error(codes.Unimplemented, unimplementedCancellationMsg)
}

func (s *Server) RejectOrderCancellation(_ context.Context, _ *orderv1.RejectOrderCancellationRequest) (*orderv1.RejectOrderCancellationResponse, error) {
	return nil, status.Error(codes.Unimplemented, unimplementedCancellationMsg)
}

func (s *Server) ListOrderCancellationRequests(_ context.Context, _ *orderv1.ListOrderCancellationRequestsRequest) (*orderv1.ListOrderCancellationRequestsResponse, error) {
	return nil, status.Error(codes.Unimplemented, unimplementedCancellationMsg)
}

func (s *Server) GetOrderCancellationRequest(_ context.Context, _ *orderv1.GetOrderCancellationRequestRequest) (*orderv1.GetOrderCancellationRequestResponse, error) {
	return nil, status.Error(codes.Unimplemented, unimplementedCancellationMsg)
}
