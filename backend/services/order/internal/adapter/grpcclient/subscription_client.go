// Package grpcclient contains outbound gRPC adapters for the order service.
package grpcclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	subscriptionv1 "github.com/Riku-KANO/ec-test/gen/go/subscription/v1"
)

// internalTokenCreds attaches the shared SUBSCRIPTION_INTERNAL_TOKEN to
// every outgoing RPC as `x-internal-token` metadata. RequireTransportSecurity
// is false because in-cluster calls use plaintext today; revisit if the
// deployment ever traverses an untrusted hop.
type internalTokenCreds struct {
	token string
}

func (c internalTokenCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"x-internal-token": c.token}, nil
}

func (c internalTokenCreds) RequireTransportSecurity() bool { return false }

// hasFreeShippingTimeout caps the per-call deadline on the buyer
// subscription lookup. The call sits on the checkout hot path; if the
// subscription service is slow or unreachable we would rather charge
// standard shipping than block the order.
const hasFreeShippingTimeout = 500 * time.Millisecond

// BuyerSubscriptionClient is a gRPC client for the subscription service.
// It replaces the earlier HTTP client that talked to the auth service; since
// Phase 2 of the auth refactor, subscription lookups live in a dedicated
// service and the order service uses its gRPC surface for lower latency and
// stricter typing.
//
// Implements port.BuyerSubscriptionChecker.
type BuyerSubscriptionClient struct {
	conn   *grpc.ClientConn
	client subscriptionv1.SubscriptionServiceClient
}

// NewBuyerSubscriptionClient dials the subscription service. The connection
// uses insecure transport on the assumption that this is an in-cluster call.
// The internalToken is attached to every outgoing RPC as `x-internal-token`
// metadata; the subscription service's unary interceptor rejects calls
// without it.
func NewBuyerSubscriptionClient(addr, internalToken string) (*BuyerSubscriptionClient, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(internalTokenCreds{token: internalToken}),
	)
	if err != nil {
		return nil, fmt.Errorf("dial subscription service: %w", err)
	}
	return &BuyerSubscriptionClient{
		conn:   conn,
		client: subscriptionv1.NewSubscriptionServiceClient(conn),
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *BuyerSubscriptionClient) Close() error {
	return c.conn.Close()
}

// HasFreeShipping returns true iff the given buyer has an active subscription
// whose plan grants free shipping. A missing subscription (NotFound) returns
// (false, nil); other errors are surfaced.
func (c *BuyerSubscriptionClient) HasFreeShipping(ctx context.Context, buyerAuth0ID string) (bool, error) {
	callCtx, cancel := context.WithTimeout(ctx, hasFreeShippingTimeout)
	defer cancel()

	start := time.Now()
	resp, err := c.client.GetBuyerSubscription(callCtx, &subscriptionv1.GetBuyerSubscriptionRequest{
		BuyerAuth0Id: buyerAuth0ID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return false, nil
		}
		// Hot-path failure: log with duration so operators can tell a
		// deadline exceeded apart from a hard RPC error. Caller treats
		// the error as "unknown — charge standard shipping".
		slog.Warn("buyer subscription lookup failed",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return false, fmt.Errorf("get buyer subscription: %w", err)
	}
	sub := resp.GetSubscription()
	if sub == nil {
		return false, errors.New("empty subscription in response")
	}
	return sub.GetStatus() == "active" && sub.GetFeatures().GetFreeShipping(), nil
}
