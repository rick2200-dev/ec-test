package grpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	orderv1 "github.com/Riku-KANO/ec-test/gen/go/order/v1"
	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/port"
)

// ---------------------------------------------------------------------------
// Mock OrderUseCase
// ---------------------------------------------------------------------------

type mockOrderUseCase struct {
	CreateOrderFn         func(ctx context.Context, tenantID uuid.UUID, input domain.CreateOrderInput) (*domain.OrderWithLines, string, error)
	CreateCheckoutFn      func(ctx context.Context, tenantID uuid.UUID, input domain.CheckoutInput) (*domain.CheckoutResult, error)
	HandlePaymentSuccessFn func(ctx context.Context, stripePaymentIntentID string) error
	CheckPurchaseFn       func(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*port.PurchaseCheckResult, error)
	GetOrderFn            func(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error)
	ListBuyerOrdersFn     func(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error)
	ListSellerOrdersFn    func(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error)
	UpdateOrderStatusFn   func(ctx context.Context, tenantID, sellerID, orderID uuid.UUID, status string) error
	ListPayoutsFn         func(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error)
	ListCommissionRulesFn func(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.CommissionRule, int, error)
	CreateCommissionRuleFn func(ctx context.Context, tenantID uuid.UUID, rule *domain.CommissionRule) error
}

func (m *mockOrderUseCase) CreateOrder(ctx context.Context, tenantID uuid.UUID, input domain.CreateOrderInput) (*domain.OrderWithLines, string, error) {
	if m.CreateOrderFn != nil {
		return m.CreateOrderFn(ctx, tenantID, input)
	}
	return nil, "", errors.New("not implemented")
}

func (m *mockOrderUseCase) CreateCheckout(ctx context.Context, tenantID uuid.UUID, input domain.CheckoutInput) (*domain.CheckoutResult, error) {
	if m.CreateCheckoutFn != nil {
		return m.CreateCheckoutFn(ctx, tenantID, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockOrderUseCase) HandlePaymentSuccess(ctx context.Context, stripePaymentIntentID string) error {
	if m.HandlePaymentSuccessFn != nil {
		return m.HandlePaymentSuccessFn(ctx, stripePaymentIntentID)
	}
	return errors.New("not implemented")
}

func (m *mockOrderUseCase) CheckPurchase(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*port.PurchaseCheckResult, error) {
	if m.CheckPurchaseFn != nil {
		return m.CheckPurchaseFn(ctx, tenantID, buyerAuth0ID, sellerID, skuID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockOrderUseCase) GetOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error) {
	if m.GetOrderFn != nil {
		return m.GetOrderFn(ctx, tenantID, orderID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockOrderUseCase) ListBuyerOrders(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error) {
	if m.ListBuyerOrdersFn != nil {
		return m.ListBuyerOrdersFn(ctx, tenantID, buyerAuth0ID, limit, offset)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *mockOrderUseCase) ListSellerOrders(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error) {
	if m.ListSellerOrdersFn != nil {
		return m.ListSellerOrdersFn(ctx, tenantID, sellerID, status, limit, offset)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *mockOrderUseCase) UpdateOrderStatus(ctx context.Context, tenantID, sellerID, orderID uuid.UUID, status string) error {
	if m.UpdateOrderStatusFn != nil {
		return m.UpdateOrderStatusFn(ctx, tenantID, sellerID, orderID, status)
	}
	return errors.New("not implemented")
}

func (m *mockOrderUseCase) ListPayouts(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error) {
	if m.ListPayoutsFn != nil {
		return m.ListPayoutsFn(ctx, tenantID, sellerID, limit, offset)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *mockOrderUseCase) ListCommissionRules(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.CommissionRule, int, error) {
	if m.ListCommissionRulesFn != nil {
		return m.ListCommissionRulesFn(ctx, tenantID, limit, offset)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *mockOrderUseCase) CreateCommissionRule(ctx context.Context, tenantID uuid.UUID, rule *domain.CommissionRule) error {
	if m.CreateCommissionRuleFn != nil {
		return m.CreateCommissionRuleFn(ctx, tenantID, rule)
	}
	return errors.New("not implemented")
}

var _ port.OrderUseCase = (*mockOrderUseCase)(nil)

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

// ---------------------------------------------------------------------------
// Conversion / helper function tests
// ---------------------------------------------------------------------------

// TestPaginationDefaults verifies the defaults and override behaviour of the
// paginationDefaults helper.
func TestPaginationDefaults(t *testing.T) {
	t.Run("nil pagination returns 20,0", func(t *testing.T) {
		limit, offset := paginationDefaults(nil)
		if limit != 20 {
			t.Errorf("limit = %d, want 20", limit)
		}
		if offset != 0 {
			t.Errorf("offset = %d, want 0", offset)
		}
	})

	t.Run("custom values work", func(t *testing.T) {
		p := &commonv1.PaginationRequest{Limit: 50, Offset: 10}
		limit, offset := paginationDefaults(p)
		if limit != 50 {
			t.Errorf("limit = %d, want 50", limit)
		}
		if offset != 10 {
			t.Errorf("offset = %d, want 10", offset)
		}
	})

	t.Run("zero values default", func(t *testing.T) {
		p := &commonv1.PaginationRequest{Limit: 0, Offset: 0}
		limit, offset := paginationDefaults(p)
		if limit != 20 {
			t.Errorf("limit = %d, want 20 (zero should default)", limit)
		}
		if offset != 0 {
			t.Errorf("offset = %d, want 0", offset)
		}
	})
}

// TestParseShippingAddress verifies that the parseShippingAddress helper
// converts an empty string to "{}" and passes valid JSON through.
func TestParseShippingAddress(t *testing.T) {
	t.Run("empty string returns {}", func(t *testing.T) {
		got := parseShippingAddress("")
		if string(got) != "{}" {
			t.Errorf("got %q, want %q", string(got), "{}")
		}
	})

	t.Run("valid JSON passes through", func(t *testing.T) {
		input := `{"city":"Tokyo","zip":"100-0001"}`
		got := parseShippingAddress(input)
		if string(got) != input {
			t.Errorf("got %q, want %q", string(got), input)
		}
	})
}

// TestProtoLinesToDomain verifies proto → domain line conversion.
//
//nolint:staticcheck // SA1019: testing deprecated OrderLineInput
func TestProtoLinesToDomain(t *testing.T) {
	skuID := uuid.New()

	t.Run("converts proto lines to domain", func(t *testing.T) {
		lines := []*orderv1.OrderLineInput{
			{SkuId: skuID.String(), Quantity: 3},
		}
		got := protoLinesToDomain(lines)
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].SKUID != skuID {
			t.Errorf("SKUID = %s, want %s", got[0].SKUID, skuID)
		}
		if got[0].Quantity != 3 {
			t.Errorf("Quantity = %d, want 3", got[0].Quantity)
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		got := protoLinesToDomain(nil)
		if len(got) != 0 {
			t.Fatalf("len = %d, want 0", len(got))
		}
	})

	t.Run("handles invalid UUID", func(t *testing.T) {
		lines := []*orderv1.OrderLineInput{
			{SkuId: "not-a-uuid", Quantity: 1},
		}
		got := protoLinesToDomain(lines)
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].SKUID != uuid.Nil {
			t.Errorf("invalid UUID should parse to uuid.Nil, got %s", got[0].SKUID)
		}
	})
}

// TestOrderToProto verifies domain → proto order conversion.
func TestOrderToProto(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	orderID := uuid.New()
	tenantID := uuid.New()
	sellerID := uuid.New()
	lineID := uuid.New()
	skuID := uuid.New()
	productID := uuid.New()

	t.Run("converts domain order with lines to proto", func(t *testing.T) {
		piID := "pi_test_123"
		paidAt := now.Add(-time.Hour)
		o := &domain.Order{
			ID:                    orderID,
			TenantID:              tenantID,
			SellerID:              sellerID,
			SellerName:            "Test Seller",
			BuyerAuth0ID:          "auth0|buyer1",
			Status:                domain.StatusPaid,
			SubtotalAmount:        1000,
			ShippingFee:           500,
			CommissionAmount:      100,
			TotalAmount:           1500,
			Currency:              "jpy",
			ShippingAddress:       json.RawMessage(`{"city":"Tokyo"}`),
			StripePaymentIntentID: &piID,
			PaidAt:                &paidAt,
			CreatedAt:             now,
			UpdatedAt:             now,
		}
		lines := []domain.OrderLine{
			{
				ID:          lineID,
				TenantID:    tenantID,
				OrderID:     orderID,
				SKUID:       skuID,
				ProductID:   productID,
				ProductName: "Widget",
				SKUCode:     "WDG-001",
				Quantity:    2,
				UnitPrice:   500,
				LineTotal:   1000,
				CreatedAt:   now,
			},
		}

		pb := orderToProto(o, lines)

		if pb.Id != orderID.String() {
			t.Errorf("Id = %s, want %s", pb.Id, orderID)
		}
		if pb.TenantId != tenantID.String() {
			t.Errorf("TenantId = %s, want %s", pb.TenantId, tenantID)
		}
		if pb.SellerId != sellerID.String() {
			t.Errorf("SellerId = %s, want %s", pb.SellerId, sellerID)
		}
		if pb.SellerName != "Test Seller" {
			t.Errorf("SellerName = %s, want Test Seller", pb.SellerName)
		}
		if pb.Status != domain.StatusPaid {
			t.Errorf("Status = %s, want %s", pb.Status, domain.StatusPaid)
		}
		if pb.Total.Amount != 1500 {
			t.Errorf("Total.Amount = %d, want 1500", pb.Total.Amount)
		}
		if pb.Total.Currency != "jpy" {
			t.Errorf("Total.Currency = %s, want jpy", pb.Total.Currency)
		}
		if pb.StripePaymentIntentId != "pi_test_123" {
			t.Errorf("StripePaymentIntentId = %s, want pi_test_123", pb.StripePaymentIntentId)
		}
		if pb.PaidAt == nil {
			t.Fatal("PaidAt should not be nil")
		}
		if pb.PaidAt.AsTime().Unix() != paidAt.Unix() {
			t.Errorf("PaidAt = %v, want %v", pb.PaidAt.AsTime(), paidAt)
		}
		if len(pb.Lines) != 1 {
			t.Fatalf("len(Lines) = %d, want 1", len(pb.Lines))
		}
		if pb.Lines[0].ProductName != "Widget" {
			t.Errorf("Lines[0].ProductName = %s, want Widget", pb.Lines[0].ProductName)
		}
		if pb.Lines[0].Quantity != 2 {
			t.Errorf("Lines[0].Quantity = %d, want 2", pb.Lines[0].Quantity)
		}
	})

	t.Run("handles nil lines", func(t *testing.T) {
		o := &domain.Order{
			ID:              orderID,
			TenantID:        tenantID,
			SellerID:        sellerID,
			Status:          domain.StatusPending,
			Currency:        "jpy",
			ShippingAddress: json.RawMessage("{}"),
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		pb := orderToProto(o, nil)
		if len(pb.Lines) != 0 {
			t.Errorf("len(Lines) = %d, want 0", len(pb.Lines))
		}
	})

	t.Run("handles nil optional fields", func(t *testing.T) {
		o := &domain.Order{
			ID:              orderID,
			TenantID:        tenantID,
			SellerID:        sellerID,
			Status:          domain.StatusPending,
			Currency:        "jpy",
			ShippingAddress: json.RawMessage("{}"),
			CreatedAt:       now,
			UpdatedAt:       now,
			// StripePaymentIntentID and PaidAt are nil
		}
		pb := orderToProto(o, nil)
		if pb.StripePaymentIntentId != "" {
			t.Errorf("StripePaymentIntentId = %q, want empty", pb.StripePaymentIntentId)
		}
		if pb.PaidAt != nil {
			t.Errorf("PaidAt should be nil, got %v", pb.PaidAt)
		}
	})
}

// TestPayoutToProto verifies domain → proto payout conversion.
func TestPayoutToProto(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	payoutID := uuid.New()
	tenantID := uuid.New()
	sellerID := uuid.New()
	orderID := uuid.New()

	t.Run("converts domain payout with all fields", func(t *testing.T) {
		transferID := "tr_abc123"
		completedAt := now.Add(-30 * time.Minute)
		p := &domain.Payout{
			ID:               payoutID,
			TenantID:         tenantID,
			SellerID:         sellerID,
			OrderID:          orderID,
			Amount:           9000,
			Currency:         "jpy",
			StripeTransferID: &transferID,
			Status:           domain.PayoutStatusCompleted,
			CreatedAt:        now,
			CompletedAt:      &completedAt,
		}

		pb := payoutToProto(p)

		if pb.Id != payoutID.String() {
			t.Errorf("Id = %s, want %s", pb.Id, payoutID)
		}
		if pb.SellerId != sellerID.String() {
			t.Errorf("SellerId = %s, want %s", pb.SellerId, sellerID)
		}
		if pb.OrderId != orderID.String() {
			t.Errorf("OrderId = %s, want %s", pb.OrderId, orderID)
		}
		if pb.Amount.Amount != 9000 {
			t.Errorf("Amount = %d, want 9000", pb.Amount.Amount)
		}
		if pb.Amount.Currency != "jpy" {
			t.Errorf("Currency = %s, want jpy", pb.Amount.Currency)
		}
		if pb.Status != domain.PayoutStatusCompleted {
			t.Errorf("Status = %s, want %s", pb.Status, domain.PayoutStatusCompleted)
		}
		if pb.StripeTransferId != "tr_abc123" {
			t.Errorf("StripeTransferId = %s, want tr_abc123", pb.StripeTransferId)
		}
		if pb.CompletedAt == nil {
			t.Fatal("CompletedAt should not be nil")
		}
		if pb.CompletedAt.AsTime().Unix() != completedAt.Unix() {
			t.Errorf("CompletedAt = %v, want %v", pb.CompletedAt.AsTime(), completedAt)
		}
	})

	t.Run("handles nil optional fields", func(t *testing.T) {
		p := &domain.Payout{
			ID:        payoutID,
			TenantID:  tenantID,
			SellerID:  sellerID,
			OrderID:   orderID,
			Amount:    5000,
			Currency:  "jpy",
			Status:    domain.PayoutStatusPending,
			CreatedAt: now,
			// StripeTransferID and CompletedAt are nil
		}

		pb := payoutToProto(p)

		if pb.StripeTransferId != "" {
			t.Errorf("StripeTransferId = %q, want empty", pb.StripeTransferId)
		}
		if pb.CompletedAt != nil {
			t.Errorf("CompletedAt should be nil, got %v", pb.CompletedAt)
		}
	})
}

// ---------------------------------------------------------------------------
// gRPC RPC handler tests
// ---------------------------------------------------------------------------

// TestGetOrder_Success verifies that a successful GetOrder call returns the
// correct proto response.
func TestGetOrder_Success(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	tenantID := uuid.New()
	orderID := uuid.New()
	sellerID := uuid.New()
	lineID := uuid.New()
	skuID := uuid.New()
	productID := uuid.New()

	mock := &mockOrderUseCase{
		GetOrderFn: func(_ context.Context, tid, oid uuid.UUID) (*domain.OrderWithLines, error) {
			if tid != tenantID {
				t.Errorf("tenantID = %s, want %s", tid, tenantID)
			}
			if oid != orderID {
				t.Errorf("orderID = %s, want %s", oid, orderID)
			}
			return &domain.OrderWithLines{
				Order: domain.Order{
					ID:              orderID,
					TenantID:        tenantID,
					SellerID:        sellerID,
					BuyerAuth0ID:    "auth0|buyer",
					Status:          domain.StatusPaid,
					TotalAmount:     2000,
					Currency:        "jpy",
					ShippingAddress: json.RawMessage("{}"),
					CreatedAt:       now,
					UpdatedAt:       now,
				},
				Lines: []domain.OrderLine{
					{
						ID:          lineID,
						TenantID:    tenantID,
						OrderID:     orderID,
						SKUID:       skuID,
						ProductID:   productID,
						ProductName: "Test Product",
						SKUCode:     "TP-001",
						Quantity:    1,
						UnitPrice:   2000,
						LineTotal:   2000,
						CreatedAt:   now,
					},
				},
			}, nil
		},
	}

	srv := NewServer(mock)
	resp, err := srv.GetOrder(context.Background(), &orderv1.GetOrderRequest{
		TenantId: tenantID.String(),
		Id:       orderID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Order == nil {
		t.Fatal("response order is nil")
	}
	if resp.Order.Id != orderID.String() {
		t.Errorf("Order.Id = %s, want %s", resp.Order.Id, orderID)
	}
	if resp.Order.Status != domain.StatusPaid {
		t.Errorf("Order.Status = %s, want %s", resp.Order.Status, domain.StatusPaid)
	}
	if len(resp.Order.Lines) != 1 {
		t.Fatalf("len(Lines) = %d, want 1", len(resp.Order.Lines))
	}
	if resp.Order.Lines[0].ProductName != "Test Product" {
		t.Errorf("Lines[0].ProductName = %s, want Test Product", resp.Order.Lines[0].ProductName)
	}
}

// TestGetOrder_InvalidTenantID verifies that an invalid tenant_id returns
// InvalidArgument.
func TestGetOrder_InvalidTenantID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	_, err := srv.GetOrder(context.Background(), &orderv1.GetOrderRequest{
		TenantId: "not-a-uuid",
		Id:       uuid.New().String(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("code = %s, want InvalidArgument", st.Code())
	}
}

// TestGetOrder_InvalidOrderID verifies that an invalid order id returns
// InvalidArgument.
func TestGetOrder_InvalidOrderID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	_, err := srv.GetOrder(context.Background(), &orderv1.GetOrderRequest{
		TenantId: uuid.New().String(),
		Id:       "bad-order-id",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("code = %s, want InvalidArgument", st.Code())
	}
}

// TestListBuyerOrders_Success verifies pagination and order list conversion.
func TestListBuyerOrders_Success(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	tenantID := uuid.New()
	order1ID := uuid.New()
	order2ID := uuid.New()
	sellerID := uuid.New()
	buyerAuth0ID := "auth0|buyer123"

	mock := &mockOrderUseCase{
		ListBuyerOrdersFn: func(_ context.Context, tid uuid.UUID, buyer string, limit, offset int) ([]domain.Order, int, error) {
			if tid != tenantID {
				t.Errorf("tenantID = %s, want %s", tid, tenantID)
			}
			if buyer != buyerAuth0ID {
				t.Errorf("buyerAuth0ID = %s, want %s", buyer, buyerAuth0ID)
			}
			if limit != 10 {
				t.Errorf("limit = %d, want 10", limit)
			}
			if offset != 5 {
				t.Errorf("offset = %d, want 5", offset)
			}
			return []domain.Order{
				{
					ID:              order1ID,
					TenantID:        tenantID,
					SellerID:        sellerID,
					BuyerAuth0ID:    buyerAuth0ID,
					Status:          domain.StatusPaid,
					TotalAmount:     1000,
					Currency:        "jpy",
					ShippingAddress: json.RawMessage("{}"),
					CreatedAt:       now,
					UpdatedAt:       now,
				},
				{
					ID:              order2ID,
					TenantID:        tenantID,
					SellerID:        sellerID,
					BuyerAuth0ID:    buyerAuth0ID,
					Status:          domain.StatusShipped,
					TotalAmount:     2000,
					Currency:        "jpy",
					ShippingAddress: json.RawMessage("{}"),
					CreatedAt:       now,
					UpdatedAt:       now,
				},
			}, 42, nil
		},
	}

	srv := NewServer(mock)
	resp, err := srv.ListBuyerOrders(context.Background(), &orderv1.ListBuyerOrdersRequest{
		TenantId:     tenantID.String(),
		BuyerAuth0Id: buyerAuth0ID,
		Pagination:   &commonv1.PaginationRequest{Limit: 10, Offset: 5},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Orders) != 2 {
		t.Fatalf("len(Orders) = %d, want 2", len(resp.Orders))
	}
	if resp.Orders[0].Id != order1ID.String() {
		t.Errorf("Orders[0].Id = %s, want %s", resp.Orders[0].Id, order1ID)
	}
	if resp.Orders[1].Id != order2ID.String() {
		t.Errorf("Orders[1].Id = %s, want %s", resp.Orders[1].Id, order2ID)
	}
	if resp.Pagination == nil {
		t.Fatal("Pagination is nil")
	}
	if resp.Pagination.Total != 42 {
		t.Errorf("Pagination.Total = %d, want 42", resp.Pagination.Total)
	}
	if resp.Pagination.Limit != 10 {
		t.Errorf("Pagination.Limit = %d, want 10", resp.Pagination.Limit)
	}
	if resp.Pagination.Offset != 5 {
		t.Errorf("Pagination.Offset = %d, want 5", resp.Pagination.Offset)
	}
}

// TestListPayouts_Success verifies pagination and payout list conversion.
func TestListPayouts_Success(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	tenantID := uuid.New()
	sellerID := uuid.New()
	payout1ID := uuid.New()
	payout2ID := uuid.New()
	orderID1 := uuid.New()
	orderID2 := uuid.New()
	transferID := "tr_xyz789"
	completedAt := now.Add(-10 * time.Minute)

	mock := &mockOrderUseCase{
		ListPayoutsFn: func(_ context.Context, tid, sid uuid.UUID, limit, offset int) ([]domain.Payout, int, error) {
			if tid != tenantID {
				t.Errorf("tenantID = %s, want %s", tid, tenantID)
			}
			if sid != sellerID {
				t.Errorf("sellerID = %s, want %s", sid, sellerID)
			}
			return []domain.Payout{
				{
					ID:               payout1ID,
					TenantID:         tenantID,
					SellerID:         sellerID,
					OrderID:          orderID1,
					Amount:           8000,
					Currency:         "jpy",
					StripeTransferID: &transferID,
					Status:           domain.PayoutStatusCompleted,
					CreatedAt:        now,
					CompletedAt:      &completedAt,
				},
				{
					ID:        payout2ID,
					TenantID:  tenantID,
					SellerID:  sellerID,
					OrderID:   orderID2,
					Amount:    3000,
					Currency:  "jpy",
					Status:    domain.PayoutStatusPending,
					CreatedAt: now,
				},
			}, 15, nil
		},
	}

	srv := NewServer(mock)
	resp, err := srv.ListPayouts(context.Background(), &orderv1.ListPayoutsRequest{
		TenantId:   tenantID.String(),
		SellerId:   sellerID.String(),
		Pagination: &commonv1.PaginationRequest{Limit: 20, Offset: 0},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Payouts) != 2 {
		t.Fatalf("len(Payouts) = %d, want 2", len(resp.Payouts))
	}

	// First payout: completed with transfer ID.
	p1 := resp.Payouts[0]
	if p1.Id != payout1ID.String() {
		t.Errorf("Payouts[0].Id = %s, want %s", p1.Id, payout1ID)
	}
	if p1.Amount.Amount != 8000 {
		t.Errorf("Payouts[0].Amount = %d, want 8000", p1.Amount.Amount)
	}
	if p1.StripeTransferId != "tr_xyz789" {
		t.Errorf("Payouts[0].StripeTransferId = %s, want tr_xyz789", p1.StripeTransferId)
	}
	if p1.CompletedAt == nil {
		t.Error("Payouts[0].CompletedAt should not be nil")
	}

	// Second payout: pending, no transfer ID or completed_at.
	p2 := resp.Payouts[1]
	if p2.Id != payout2ID.String() {
		t.Errorf("Payouts[1].Id = %s, want %s", p2.Id, payout2ID)
	}
	if p2.StripeTransferId != "" {
		t.Errorf("Payouts[1].StripeTransferId = %q, want empty", p2.StripeTransferId)
	}
	if p2.CompletedAt != nil {
		t.Errorf("Payouts[1].CompletedAt should be nil, got %v", p2.CompletedAt)
	}

	// Pagination.
	if resp.Pagination == nil {
		t.Fatal("Pagination is nil")
	}
	if resp.Pagination.Total != 15 {
		t.Errorf("Pagination.Total = %d, want 15", resp.Pagination.Total)
	}
}

// ---------------------------------------------------------------------------
// CreateOrder
// ---------------------------------------------------------------------------

func TestCreateOrder_Success(t *testing.T) {
	tid, sid, orderID := uuid.New(), uuid.New(), uuid.New()
	skuID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &mockOrderUseCase{
		CreateOrderFn: func(_ context.Context, tenantID uuid.UUID, input domain.CreateOrderInput) (*domain.OrderWithLines, string, error) {
			if tenantID != tid || input.SellerID != sid {
				t.Errorf("tenantID=%s sellerID=%s", tenantID, input.SellerID)
			}
			if len(input.Lines) != 1 || input.Lines[0].SKUID != skuID || input.Lines[0].Quantity != 2 {
				t.Errorf("lines = %+v", input.Lines)
			}
			return &domain.OrderWithLines{
				Order: domain.Order{
					ID: orderID, TenantID: tid, SellerID: sid,
					Status: "pending", TotalAmount: 1500, Currency: "jpy",
					ShippingAddress: []byte(`{"city":"Tokyo"}`),
					CreatedAt: now, UpdatedAt: now,
				},
				Lines: []domain.OrderLine{
					{ID: uuid.New(), OrderID: orderID, SKUID: skuID, ProductID: uuid.New(), Quantity: 2, UnitPrice: 500, LineTotal: 1000},
				},
			}, "pi_test_secret", nil
		},
	}
	srv := NewServer(mock)

	//nolint:staticcheck // SA1019: deprecated single-seller CreateOrder RPC
	resp, err := srv.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{
		TenantId:     tid.String(),
		SellerId:     sid.String(),
		BuyerAuth0Id: "auth0|buyer",
		Lines: []*orderv1.OrderLineInput{ //nolint:staticcheck
			{SkuId: skuID.String(), Quantity: 2},
		},
		ShippingAddressJson: `{"city":"Tokyo"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StripeClientSecret != "pi_test_secret" {
		t.Errorf("client secret = %q, want pi_test_secret", resp.StripeClientSecret)
	}
	if resp.Order.Id != orderID.String() {
		t.Errorf("order id = %s, want %s", resp.Order.Id, orderID)
	}
}

func TestCreateOrder_InvalidTenantID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	//nolint:staticcheck
	_, err := srv.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{
		TenantId: "not-a-uuid",
		SellerId: uuid.New().String(),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestCreateOrder_InvalidSellerID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	//nolint:staticcheck
	_, err := srv.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{
		TenantId: uuid.New().String(),
		SellerId: "not-a-uuid",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestCreateOrder_ServiceError(t *testing.T) {
	mock := &mockOrderUseCase{
		CreateOrderFn: func(_ context.Context, _ uuid.UUID, _ domain.CreateOrderInput) (*domain.OrderWithLines, string, error) {
			return nil, "", apperrors.NotFound("seller not found")
		},
	}
	srv := NewServer(mock)
	//nolint:staticcheck
	_, err := srv.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{
		TenantId: uuid.New().String(),
		SellerId: uuid.New().String(),
	})
	if status.Code(err) != codes.NotFound {
		t.Errorf("code = %v, want NotFound", status.Code(err))
	}
}

// ---------------------------------------------------------------------------
// ListSellerOrders
// ---------------------------------------------------------------------------

func TestListSellerOrders_Success(t *testing.T) {
	tid, sid := uuid.New(), uuid.New()
	orderID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &mockOrderUseCase{
		ListSellerOrdersFn: func(_ context.Context, tenantID, sellerID uuid.UUID, statusFilter string, limit, offset int) ([]domain.Order, int, error) {
			if tenantID != tid || sellerID != sid {
				t.Errorf("ids mismatch")
			}
			if statusFilter != "paid" {
				t.Errorf("status filter = %q, want paid", statusFilter)
			}
			if limit != 50 || offset != 10 {
				t.Errorf("limit=%d offset=%d, want 50,10", limit, offset)
			}
			return []domain.Order{
				{ID: orderID, TenantID: tid, SellerID: sid, Status: "paid", Currency: "jpy",
					ShippingAddress: []byte("{}"), CreatedAt: now, UpdatedAt: now},
			}, 3, nil
		},
	}
	srv := NewServer(mock)

	resp, err := srv.ListSellerOrders(context.Background(), &orderv1.ListSellerOrdersRequest{
		TenantId: tid.String(), SellerId: sid.String(), Status: "paid",
		Pagination: &commonv1.PaginationRequest{Limit: 50, Offset: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(resp.Orders))
	}
	if resp.Orders[0].Id != orderID.String() {
		t.Errorf("order id = %s, want %s", resp.Orders[0].Id, orderID)
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("total = %d, want 3", resp.Pagination.Total)
	}
}

func TestListSellerOrders_InvalidTenantID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	_, err := srv.ListSellerOrders(context.Background(), &orderv1.ListSellerOrdersRequest{
		TenantId: "bogus", SellerId: uuid.New().String(),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestListSellerOrders_InvalidSellerID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	_, err := srv.ListSellerOrders(context.Background(), &orderv1.ListSellerOrdersRequest{
		TenantId: uuid.New().String(), SellerId: "bogus",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestListSellerOrders_ServiceError(t *testing.T) {
	mock := &mockOrderUseCase{
		ListSellerOrdersFn: func(_ context.Context, _, _ uuid.UUID, _ string, _, _ int) ([]domain.Order, int, error) {
			return nil, 0, errors.New("db down")
		},
	}
	srv := NewServer(mock)
	_, err := srv.ListSellerOrders(context.Background(), &orderv1.ListSellerOrdersRequest{
		TenantId: uuid.New().String(), SellerId: uuid.New().String(),
	})
	if status.Code(err) != codes.Internal {
		t.Errorf("code = %v, want Internal", status.Code(err))
	}
}

// ---------------------------------------------------------------------------
// UpdateOrderStatus
// ---------------------------------------------------------------------------

func TestUpdateOrderStatus_Success(t *testing.T) {
	tid, sid, oid := uuid.New(), uuid.New(), uuid.New()
	now := time.Now().Truncate(time.Second)

	var updateCalled bool
	mock := &mockOrderUseCase{
		GetOrderFn: func(_ context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error) {
			if tenantID != tid || orderID != oid {
				t.Errorf("ids mismatch")
			}
			return &domain.OrderWithLines{
				Order: domain.Order{
					ID: oid, TenantID: tid, SellerID: sid, Status: "paid",
					ShippingAddress: []byte("{}"), CreatedAt: now, UpdatedAt: now,
				},
			}, nil
		},
		UpdateOrderStatusFn: func(_ context.Context, tenantID, sellerID, orderID uuid.UUID, newStatus string) error {
			updateCalled = true
			if sellerID != sid {
				t.Errorf("sellerID = %s, want %s (resolved from GetOrder)", sellerID, sid)
			}
			if newStatus != "shipped" {
				t.Errorf("status = %q, want shipped", newStatus)
			}
			_ = tenantID
			_ = orderID
			return nil
		},
	}
	srv := NewServer(mock)

	resp, err := srv.UpdateOrderStatus(context.Background(), &orderv1.UpdateOrderStatusRequest{
		TenantId: tid.String(), Id: oid.String(), Status: "shipped",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Fatal("UpdateOrderStatus was not called")
	}
	if resp.Order.Id != oid.String() {
		t.Errorf("order id = %s, want %s", resp.Order.Id, oid)
	}
}

func TestUpdateOrderStatus_InvalidTenantID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	_, err := srv.UpdateOrderStatus(context.Background(), &orderv1.UpdateOrderStatusRequest{
		TenantId: "bogus", Id: uuid.New().String(), Status: "shipped",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestUpdateOrderStatus_InvalidOrderID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	_, err := srv.UpdateOrderStatus(context.Background(), &orderv1.UpdateOrderStatusRequest{
		TenantId: uuid.New().String(), Id: "bogus", Status: "shipped",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestUpdateOrderStatus_GetOrderFails(t *testing.T) {
	mock := &mockOrderUseCase{
		GetOrderFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrderWithLines, error) {
			return nil, apperrors.NotFound("order not found")
		},
	}
	srv := NewServer(mock)
	_, err := srv.UpdateOrderStatus(context.Background(), &orderv1.UpdateOrderStatusRequest{
		TenantId: uuid.New().String(), Id: uuid.New().String(), Status: "shipped",
	})
	if status.Code(err) != codes.NotFound {
		t.Errorf("code = %v, want NotFound", status.Code(err))
	}
}

func TestUpdateOrderStatus_UpdateFails(t *testing.T) {
	sid := uuid.New()
	mock := &mockOrderUseCase{
		GetOrderFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrderWithLines, error) {
			return &domain.OrderWithLines{
				Order: domain.Order{SellerID: sid, ShippingAddress: []byte("{}")},
			}, nil
		},
		UpdateOrderStatusFn: func(_ context.Context, _, _, _ uuid.UUID, _ string) error {
			return apperrors.BadRequest("invalid status transition")
		},
	}
	srv := NewServer(mock)
	_, err := srv.UpdateOrderStatus(context.Background(), &orderv1.UpdateOrderStatusRequest{
		TenantId: uuid.New().String(), Id: uuid.New().String(), Status: "bogus",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

// ---------------------------------------------------------------------------
// Misc: ListPayouts invalid IDs, GetOrder path coverage
// ---------------------------------------------------------------------------

func TestListPayouts_InvalidTenantID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	_, err := srv.ListPayouts(context.Background(), &orderv1.ListPayoutsRequest{
		TenantId: "bogus", SellerId: uuid.New().String(),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestListPayouts_InvalidSellerID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	_, err := srv.ListPayouts(context.Background(), &orderv1.ListPayoutsRequest{
		TenantId: uuid.New().String(), SellerId: "bogus",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestListBuyerOrders_InvalidTenantID(t *testing.T) {
	srv := NewServer(&mockOrderUseCase{})
	_, err := srv.ListBuyerOrders(context.Background(), &orderv1.ListBuyerOrdersRequest{
		TenantId: "bogus",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

