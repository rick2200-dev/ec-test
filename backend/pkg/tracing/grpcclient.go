package tracing

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// DialOptions prepends the OTel gRPC client stats handler to whatever dial
// options the caller already uses. Keeping this in one helper lets every
// grpcclient wrapper stay a one-line change.
func DialOptions(extra ...grpc.DialOption) []grpc.DialOption {
	opts := make([]grpc.DialOption, 0, len(extra)+1)
	opts = append(opts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	opts = append(opts, extra...)
	return opts
}
