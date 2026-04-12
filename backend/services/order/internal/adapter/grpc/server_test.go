package grpcserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/Riku-KANO/ec-test/gen/go/order/v1"
	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
)

// TestServiceErrToGRPC_MapsAppErrorStatus verifies that a typed
// *apperrors.AppError is converted to the correct gRPC status code
// regardless of message text. This is the regression guard for the
// "conflict leaked as Internal" review finding.
func TestServiceErrToGRPC_MapsAppErrorStatus(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name:     "nil returns nil",
			err:      nil,
			wantCode: codes.OK, // sentinel; checked separately below
		},
		{
			name:     "400 BadRequest → InvalidArgument",
			err:      apperrors.BadRequest("reason is empty"),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "401 Unauthorized → Unauthenticated",
			err:      apperrors.Unauthorized("missing token"),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "403 Forbidden → PermissionDenied",
			err:      apperrors.Forbidden("not allowed"),
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "404 NotFound → NotFound",
			err:      apperrors.NotFound("order not found"),
			wantCode: codes.NotFound,
		},
		{
			name:     "409 Conflict default → FailedPrecondition",
			err:      apperrors.Conflict("order cannot be cancelled in its current status").WithCode("ORDER_NOT_CANCELLABLE"),
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "409 Conflict with _ALREADY_EXISTS suffix → AlreadyExists",
			err:      apperrors.Conflict("a pending request already exists").WithCode("CANCELLATION_REQUEST_ALREADY_EXISTS"),
			wantCode: codes.AlreadyExists,
		},
		{
			// _ALREADY_PROCESSED is a state-machine conflict, not a
			// duplicate-resource error. AlreadyExists would mislead a
			// retrying client into thinking a new id would succeed.
			name:     "409 Conflict with _ALREADY_PROCESSED → FailedPrecondition",
			err:      apperrors.Conflict("request is no longer pending").WithCode("CANCELLATION_REQUEST_ALREADY_PROCESSED"),
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "502 BadGateway → Unavailable (Stripe refund)",
			err:      apperrors.New(http.StatusBadGateway, "stripe refund failed", errors.New("network")).WithCode("REFUND_FAILED"),
			wantCode: codes.Unavailable,
		},
		{
			name:     "503 ServiceUnavailable → Unavailable",
			err:      apperrors.New(http.StatusServiceUnavailable, "upstream down", nil),
			wantCode: codes.Unavailable,
		},
		{
			name:     "504 GatewayTimeout → Unavailable",
			err:      apperrors.New(http.StatusGatewayTimeout, "upstream timed out", nil),
			wantCode: codes.Unavailable,
		},
		{
			name:     "429 TooManyRequests → ResourceExhausted",
			err:      apperrors.New(http.StatusTooManyRequests, "rate limited", nil),
			wantCode: codes.ResourceExhausted,
		},
		{
			name:     "500 Internal → Internal",
			err:      apperrors.Internal("db down", errors.New("conn reset")),
			wantCode: codes.Internal,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := serviceErrToGRPC(c.err)
			if c.err == nil {
				if got != nil {
					t.Fatalf("nil input must return nil, got %v", got)
				}
				return
			}
			st, ok := status.FromError(got)
			if !ok {
				t.Fatalf("expected gRPC status error, got %T: %v", got, got)
			}
			if st.Code() != c.wantCode {
				t.Errorf("code = %s, want %s", st.Code(), c.wantCode)
			}
		})
	}
}

// TestServiceErrToGRPC_IncludesSemanticCode verifies that when an
// AppError carries a stable business code, that code is prefixed into
// the gRPC message so clients logging the raw status still see the
// semantic hint.
func TestServiceErrToGRPC_IncludesSemanticCode(t *testing.T) {
	err := apperrors.Conflict("order cannot be cancelled").WithCode("ORDER_NOT_CANCELLABLE")
	got := serviceErrToGRPC(err)
	st, ok := status.FromError(got)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", got)
	}
	if !strings.Contains(st.Message(), "ORDER_NOT_CANCELLABLE") {
		t.Errorf("message %q does not contain semantic code", st.Message())
	}
	if !strings.Contains(st.Message(), "order cannot be cancelled") {
		t.Errorf("message %q does not contain human message", st.Message())
	}
}

// TestServiceErrToGRPC_FallbackStringMatching covers the legacy
// non-AppError path — raw sentinel errors returned by domain/repository
// layers should still land on a sensible code.
func TestServiceErrToGRPC_FallbackStringMatching(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"plain not found", errors.New("order not found"), codes.NotFound},
		{"plain bad request", errors.New("bad request: missing field"), codes.InvalidArgument},
		{"plain invalid", errors.New("invalid order id"), codes.InvalidArgument},
		{"unrelated", errors.New("something exploded"), codes.Internal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := serviceErrToGRPC(c.err)
			st, _ := status.FromError(got)
			if st.Code() != c.wantCode {
				t.Errorf("code = %s, want %s", st.Code(), c.wantCode)
			}
		})
	}
}

// TestCancellationRPCs_ReturnUnimplemented is the regression guard for
// the "auth is a no-op on gRPC" review finding. Every cancellation RPC
// must refuse to serve traffic until a real gRPC auth story exists —
// see cancellation_server.go for the detailed rationale.
func TestCancellationRPCs_ReturnUnimplemented(t *testing.T) {
	s := &Server{}
	ctx := context.Background()

	assertUnimplemented := func(t *testing.T, err error) {
		t.Helper()
		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %T: %v", err, err)
		}
		if st.Code() != codes.Unimplemented {
			t.Errorf("code = %s, want Unimplemented", st.Code())
		}
	}

	t.Run("RequestOrderCancellation", func(t *testing.T) {
		_, err := s.RequestOrderCancellation(ctx, &orderv1.RequestOrderCancellationRequest{})
		assertUnimplemented(t, err)
	})
	t.Run("ApproveOrderCancellation", func(t *testing.T) {
		_, err := s.ApproveOrderCancellation(ctx, &orderv1.ApproveOrderCancellationRequest{})
		assertUnimplemented(t, err)
	})
	t.Run("RejectOrderCancellation", func(t *testing.T) {
		_, err := s.RejectOrderCancellation(ctx, &orderv1.RejectOrderCancellationRequest{})
		assertUnimplemented(t, err)
	})
	t.Run("ListOrderCancellationRequests", func(t *testing.T) {
		_, err := s.ListOrderCancellationRequests(ctx, &orderv1.ListOrderCancellationRequestsRequest{})
		assertUnimplemented(t, err)
	})
	t.Run("GetOrderCancellationRequest", func(t *testing.T) {
		_, err := s.GetOrderCancellationRequest(ctx, &orderv1.GetOrderCancellationRequestRequest{})
		assertUnimplemented(t, err)
	})
}
