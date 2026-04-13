package middleware

import (
	"context"
	"crypto/subtle"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// InternalTokenMetadataKey is the lowercased gRPC metadata key used to
// carry the shared internal-token secret on in-cluster RPCs. We mirror
// the HTTP header name (X-Internal-Token) so operators only have to
// reason about one credential. gRPC lower-cases all metadata keys.
const InternalTokenMetadataKey = "x-internal-token"

// UnaryInternalTokenInterceptor returns a grpc.UnaryServerInterceptor
// that rejects any RPC whose `x-internal-token` metadata does not match
// the configured shared secret. The health/reflection RPCs are exempt
// (matched by their fully qualified method name prefix) so liveness and
// tooling probes continue to work.
//
// Fails closed: an empty secret causes every call to be rejected with
// codes.FailedPrecondition. This mirrors the HTTP middleware's 503
// behaviour — it's better to break loudly at startup than to silently
// serve unauthenticated traffic.
func UnaryInternalTokenInterceptor(secret string) grpc.UnaryServerInterceptor {
	secretBytes := []byte(secret)
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Bypass for gRPC health + reflection. They're safe to expose
		// in-cluster and need to work without credentials.
		if strings.HasPrefix(info.FullMethod, "/grpc.health.") ||
			strings.HasPrefix(info.FullMethod, "/grpc.reflection.") {
			return handler(ctx, req)
		}
		if len(secretBytes) == 0 {
			return nil, status.Error(codes.FailedPrecondition, "internal token not configured")
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		var got []byte
		if vals := md.Get(InternalTokenMetadataKey); len(vals) > 0 {
			got = []byte(vals[0])
		}
		if subtle.ConstantTimeEq(int32(len(got)), int32(len(secretBytes))) != 1 ||
			subtle.ConstantTimeCompare(got, secretBytes) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid internal token")
		}
		return handler(ctx, req)
	}
}
