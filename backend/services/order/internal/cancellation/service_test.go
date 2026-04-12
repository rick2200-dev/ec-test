package cancellation

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// -----------------------------------------------------------------------------
// Test fixtures and stubs
// -----------------------------------------------------------------------------

// fakeOrderReader is an OrderReader that returns whatever the test
// configured, so every branch of ownership / status / fetch errors is
// exercisable without touching the DB.
type fakeOrderReader struct {
	order *domain.OrderWithLines
	err   error
	calls int
}

func (f *fakeOrderReader) GetByID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error) {
	f.calls++
	return f.order, f.err
}

// fakePayoutReader returns a fixed payout list. Tests that care about
// the reversal loop populate this with completed/non-completed payouts.
type fakePayoutReader struct {
	payouts []domain.Payout
	err     error
}

func (f *fakePayoutReader) ListByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) ([]domain.Payout, error) {
	return f.payouts, f.err
}

// stripeCall captures an individual CreateRefund / ReverseTransfer
// invocation so tests can assert on the idempotency key format.
type stripeCall struct {
	kind           string // "refund" or "reverse"
	target         string // paymentIntentID or transferID
	amount         int64
	idempotencyKey string
}

// fakeStripeClient is a scripted StripeClient. refundResult and
// reversalResults drive what each call returns; calls accumulates the
// invocations in order so the test can check both the key format and
// the call count.
type fakeStripeClient struct {
	refundResult    stripeResult
	reversalResults []stripeResult // consumed in order
	reversalIdx     int

	calls []stripeCall
}

type stripeResult struct {
	id  string
	err error
}

func (s *fakeStripeClient) CreateRefund(paymentIntentID string, amount int64, idempotencyKey string) (string, error) {
	s.calls = append(s.calls, stripeCall{
		kind:           "refund",
		target:         paymentIntentID,
		amount:         amount,
		idempotencyKey: idempotencyKey,
	})
	return s.refundResult.id, s.refundResult.err
}

func (s *fakeStripeClient) ReverseTransfer(transferID string, amount int64, idempotencyKey string) (string, error) {
	s.calls = append(s.calls, stripeCall{
		kind:           "reverse",
		target:         transferID,
		amount:         amount,
		idempotencyKey: idempotencyKey,
	})
	if s.reversalIdx >= len(s.reversalResults) {
		return "", fmt.Errorf("unexpected reversal call #%d", s.reversalIdx+1)
	}
	r := s.reversalResults[s.reversalIdx]
	s.reversalIdx++
	return r.id, r.err
}

// fakeRequestsStore is a scripted RequestsStore covering every method
// used by Service. The individual hook fields let each test supply its
// own behavior for the one or two methods it actually exercises.
type fakeRequestsStore struct {
	createFn       func(ctx context.Context, tenantID uuid.UUID, req *CancellationRequest) error
	getByIDFn      func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error)
	listByStatusFn func(ctx context.Context, tenantID, sellerID uuid.UUID, status Status, limit, offset int) ([]CancellationRequest, int, error)
	rejectFn       func(ctx context.Context, tenantID, requestID, sellerID uuid.UUID, comment string) (*CancellationRequest, error)
	markFailedFn   func(ctx context.Context, tenantID, requestID uuid.UUID, failureReason string) error
	approveTxFn    func(ctx context.Context, tenantID uuid.UUID, in ApprovalTxInput) (*CancellationRequest, *domain.Order, error)

	// Call capture for assertions.
	markFailedCalls     int
	lastFailedReason    string
	approveTxInputs     []ApprovalTxInput
	lastListSellerID    uuid.UUID
	listByStatusCalls   int
}

func (f *fakeRequestsStore) Create(ctx context.Context, tenantID uuid.UUID, req *CancellationRequest) error {
	if f.createFn != nil {
		return f.createFn(ctx, tenantID, req)
	}
	// Default: stamp plausible values so the happy path can assert.
	req.ID = uuid.New()
	req.TenantID = tenantID
	req.Status = StatusPending
	req.CreatedAt = time.Now().UTC()
	req.UpdatedAt = req.CreatedAt
	return nil
}

func (f *fakeRequestsStore) GetByID(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, tenantID, requestID)
	}
	return nil, nil
}

func (f *fakeRequestsStore) GetLatestByOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*CancellationRequest, error) {
	return nil, nil
}

func (f *fakeRequestsStore) ListByStatus(ctx context.Context, tenantID, sellerID uuid.UUID, status Status, limit, offset int) ([]CancellationRequest, int, error) {
	f.listByStatusCalls++
	f.lastListSellerID = sellerID
	if f.listByStatusFn != nil {
		return f.listByStatusFn(ctx, tenantID, sellerID, status, limit, offset)
	}
	return nil, 0, nil
}

func (f *fakeRequestsStore) Reject(ctx context.Context, tenantID, requestID, sellerID uuid.UUID, comment string) (*CancellationRequest, error) {
	if f.rejectFn != nil {
		return f.rejectFn(ctx, tenantID, requestID, sellerID, comment)
	}
	return nil, errors.New("reject not configured")
}

func (f *fakeRequestsStore) MarkFailed(ctx context.Context, tenantID, requestID uuid.UUID, failureReason string) error {
	f.markFailedCalls++
	f.lastFailedReason = failureReason
	if f.markFailedFn != nil {
		return f.markFailedFn(ctx, tenantID, requestID, failureReason)
	}
	return nil
}

func (f *fakeRequestsStore) ApproveTx(ctx context.Context, tenantID uuid.UUID, in ApprovalTxInput) (*CancellationRequest, *domain.Order, error) {
	f.approveTxInputs = append(f.approveTxInputs, in)
	if f.approveTxFn != nil {
		return f.approveTxFn(ctx, tenantID, in)
	}
	return nil, nil, errors.New("approveTx not configured")
}

// -----------------------------------------------------------------------------
// Fixtures
// -----------------------------------------------------------------------------

const (
	testBuyerAuth0ID = "auth0|buyer-123"
	testReason       = "changed my mind"
)

var (
	testTenantID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testOrderID   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testSellerID  = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	testRequestID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
)

func buildTestOrder(status string) *domain.OrderWithLines {
	pi := "pi_test_12345"
	return &domain.OrderWithLines{
		Order: domain.Order{
			ID:                    testOrderID,
			TenantID:              testTenantID,
			SellerID:              testSellerID,
			BuyerAuth0ID:          testBuyerAuth0ID,
			Status:                status,
			TotalAmount:           5000,
			Currency:              "jpy",
			StripePaymentIntentID: &pi,
		},
		Lines: []domain.OrderLine{
			{
				ID:          uuid.New(),
				TenantID:    testTenantID,
				OrderID:     testOrderID,
				SKUID:       uuid.New(),
				ProductID:   uuid.New(),
				ProductName: "Test Product",
				SKUCode:     "SKU-001",
				Quantity:    1,
				UnitPrice:   5000,
				LineTotal:   5000,
			},
		},
	}
}

func buildPendingRequest() *CancellationRequest {
	return &CancellationRequest{
		ID:                 testRequestID,
		TenantID:           testTenantID,
		OrderID:            testOrderID,
		RequestedByAuth0ID: testBuyerAuth0ID,
		Reason:             testReason,
		Status:             StatusPending,
	}
}

// assertAppErrorCode is a small ergonomic helper so tests read cleanly.
func assertAppErrorCode(t *testing.T, err error, wantStatus int, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *AppError, got %T: %v", err, err)
	}
	if appErr.Status != wantStatus {
		t.Errorf("status = %d, want %d", appErr.Status, wantStatus)
	}
	if appErr.Code != wantCode {
		t.Errorf("code = %q, want %q", appErr.Code, wantCode)
	}
}

// -----------------------------------------------------------------------------
// Pure helpers
// -----------------------------------------------------------------------------

func TestCanOrderBeCancelled(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{domain.StatusPending, true},
		{domain.StatusPaid, true},
		{domain.StatusProcessing, true},
		{domain.StatusShipped, false},
		{domain.StatusDelivered, false},
		{domain.StatusCompleted, false},
		{domain.StatusCancelled, false},
		{"bogus", false},
	}
	for _, c := range cases {
		t.Run(c.status, func(t *testing.T) {
			if got := canOrderBeCancelled(c.status); got != c.want {
				t.Errorf("canOrderBeCancelled(%q) = %v, want %v", c.status, got, c.want)
			}
		})
	}
}

func TestStatusIsTerminal(t *testing.T) {
	cases := []struct {
		s    Status
		want bool
	}{
		{StatusPending, false},
		{StatusApproved, true},
		{StatusRejected, true},
		{StatusFailed, true},
	}
	for _, c := range cases {
		t.Run(string(c.s), func(t *testing.T) {
			if got := c.s.IsTerminal(); got != c.want {
				t.Errorf("%s.IsTerminal() = %v, want %v", c.s, got, c.want)
			}
		})
	}
}

func TestCancellableStatusesMatchesCanOrderBeCancelled(t *testing.T) {
	// The SQL WHERE guard in ApproveTx and the Go pre-check must agree,
	// so cancellableStatuses() must list exactly the statuses for which
	// canOrderBeCancelled returns true.
	list := cancellableStatuses()
	set := map[string]bool{}
	for _, s := range list {
		set[s] = true
	}
	for _, status := range []string{
		domain.StatusPending, domain.StatusPaid, domain.StatusProcessing,
	} {
		if !set[status] {
			t.Errorf("cancellableStatuses missing %q", status)
		}
		if !canOrderBeCancelled(status) {
			t.Errorf("canOrderBeCancelled(%q) = false, want true", status)
		}
	}
	for _, status := range []string{
		domain.StatusShipped, domain.StatusDelivered, domain.StatusCompleted, domain.StatusCancelled,
	} {
		if set[status] {
			t.Errorf("cancellableStatuses must not contain %q", status)
		}
		if canOrderBeCancelled(status) {
			t.Errorf("canOrderBeCancelled(%q) = true, want false", status)
		}
	}
}

// -----------------------------------------------------------------------------
// RequestCancellation
// -----------------------------------------------------------------------------

func TestRequestCancellation_EmptyReasonReturnsBadRequest(t *testing.T) {
	svc := NewService(&fakeOrderReader{}, &fakePayoutReader{}, &fakeRequestsStore{}, &fakeStripeClient{}, nil)
	_, err := svc.RequestCancellation(context.Background(), testTenantID, testOrderID, testBuyerAuth0ID, "")
	assertAppErrorCode(t, err, 400, "")
}

func TestRequestCancellation_OrderNotFound(t *testing.T) {
	svc := NewService(&fakeOrderReader{order: nil}, &fakePayoutReader{}, &fakeRequestsStore{}, &fakeStripeClient{}, nil)
	_, err := svc.RequestCancellation(context.Background(), testTenantID, testOrderID, testBuyerAuth0ID, testReason)
	assertAppErrorCode(t, err, 404, "")
}

func TestRequestCancellation_BuyerMismatch404(t *testing.T) {
	order := buildTestOrder(domain.StatusPaid)
	order.BuyerAuth0ID = "auth0|someone-else"
	svc := NewService(&fakeOrderReader{order: order}, &fakePayoutReader{}, &fakeRequestsStore{}, &fakeStripeClient{}, nil)
	_, err := svc.RequestCancellation(context.Background(), testTenantID, testOrderID, testBuyerAuth0ID, testReason)
	// Wrapped in 404 to avoid leaking existence.
	assertAppErrorCode(t, err, 404, CodeNotOrderBuyer)
}

func TestRequestCancellation_NotCancellableStatus(t *testing.T) {
	order := buildTestOrder(domain.StatusShipped)
	svc := NewService(&fakeOrderReader{order: order}, &fakePayoutReader{}, &fakeRequestsStore{}, &fakeStripeClient{}, nil)
	_, err := svc.RequestCancellation(context.Background(), testTenantID, testOrderID, testBuyerAuth0ID, testReason)
	assertAppErrorCode(t, err, 409, CodeOrderNotCancellable)
}

func TestRequestCancellation_DuplicatePendingRequest(t *testing.T) {
	order := buildTestOrder(domain.StatusPaid)
	store := &fakeRequestsStore{
		createFn: func(ctx context.Context, tenantID uuid.UUID, req *CancellationRequest) error {
			return ErrPendingRequestExists
		},
	}
	svc := NewService(&fakeOrderReader{order: order}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)
	_, err := svc.RequestCancellation(context.Background(), testTenantID, testOrderID, testBuyerAuth0ID, testReason)
	assertAppErrorCode(t, err, 409, CodeCancellationRequestAlreadyExists)
}

func TestRequestCancellation_Success(t *testing.T) {
	order := buildTestOrder(domain.StatusPaid)
	store := &fakeRequestsStore{} // default Create stamps pending
	svc := NewService(&fakeOrderReader{order: order}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)

	req, err := svc.RequestCancellation(context.Background(), testTenantID, testOrderID, testBuyerAuth0ID, testReason)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req == nil {
		t.Fatal("expected non-nil request")
	}
	if req.Status != StatusPending {
		t.Errorf("status = %q, want pending", req.Status)
	}
	if req.OrderID != testOrderID {
		t.Errorf("order id = %s, want %s", req.OrderID, testOrderID)
	}
	if req.RequestedByAuth0ID != testBuyerAuth0ID {
		t.Errorf("buyer = %q, want %q", req.RequestedByAuth0ID, testBuyerAuth0ID)
	}
}

// -----------------------------------------------------------------------------
// RejectCancellation
// -----------------------------------------------------------------------------

func TestRejectCancellation_RequestNotFound(t *testing.T) {
	store := &fakeRequestsStore{
		getByIDFn: func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
			return nil, nil
		},
	}
	svc := NewService(&fakeOrderReader{}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)
	_, err := svc.RejectCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "nope")
	assertAppErrorCode(t, err, 404, CodeCancellationRequestNotFound)
}

func TestRejectCancellation_SellerMismatch(t *testing.T) {
	store := &fakeRequestsStore{
		getByIDFn: func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
			return buildPendingRequest(), nil
		},
	}
	order := buildTestOrder(domain.StatusPaid)
	order.SellerID = uuid.MustParse("99999999-9999-9999-9999-999999999999")
	svc := NewService(&fakeOrderReader{order: order}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)

	_, err := svc.RejectCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "nope")
	assertAppErrorCode(t, err, 404, CodeNotOrderSeller)
}

func TestRejectCancellation_AlreadyRejected(t *testing.T) {
	req := buildPendingRequest()
	req.Status = StatusRejected
	store := &fakeRequestsStore{
		getByIDFn: func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
			return req, nil
		},
	}
	svc := NewService(&fakeOrderReader{order: buildTestOrder(domain.StatusPaid)}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)
	_, err := svc.RejectCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "nope")
	assertAppErrorCode(t, err, 409, CodeCancellationRequestAlreadyProcessed)
}

func TestRejectCancellation_RaceAtWrite(t *testing.T) {
	store := &fakeRequestsStore{
		getByIDFn: func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
			return buildPendingRequest(), nil
		},
		rejectFn: func(ctx context.Context, tenantID, requestID, sellerID uuid.UUID, comment string) (*CancellationRequest, error) {
			return nil, ErrAlreadyProcessed
		},
	}
	svc := NewService(&fakeOrderReader{order: buildTestOrder(domain.StatusPaid)}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)
	_, err := svc.RejectCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "nope")
	assertAppErrorCode(t, err, 409, CodeCancellationRequestAlreadyProcessed)
}

func TestRejectCancellation_Success(t *testing.T) {
	updatedReq := buildPendingRequest()
	updatedReq.Status = StatusRejected
	store := &fakeRequestsStore{
		getByIDFn: func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
			return buildPendingRequest(), nil
		},
		rejectFn: func(ctx context.Context, tenantID, requestID, sellerID uuid.UUID, comment string) (*CancellationRequest, error) {
			if comment != "already shipped" {
				t.Errorf("comment = %q, want %q", comment, "already shipped")
			}
			return updatedReq, nil
		},
	}
	svc := NewService(&fakeOrderReader{order: buildTestOrder(domain.StatusPaid)}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)

	result, err := svc.RejectCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "already shipped")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusRejected {
		t.Errorf("status = %q, want rejected", result.Status)
	}
}

// -----------------------------------------------------------------------------
// ApproveCancellation
// -----------------------------------------------------------------------------

// approveHarness bundles the pieces for an approve test. Individual
// tests tweak specific fields before calling .svc() to build the
// Service.
type approveHarness struct {
	order  *domain.OrderWithLines
	req    *CancellationRequest
	stripe *fakeStripeClient
	store  *fakeRequestsStore
	reader *fakeOrderReader
	payout *fakePayoutReader
}

func newApproveHarness() *approveHarness {
	order := buildTestOrder(domain.StatusPaid)
	req := buildPendingRequest()
	stripe := &fakeStripeClient{refundResult: stripeResult{id: "re_test_ok"}}
	store := &fakeRequestsStore{
		getByIDFn: func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
			return req, nil
		},
		approveTxFn: func(ctx context.Context, tenantID uuid.UUID, in ApprovalTxInput) (*CancellationRequest, *domain.Order, error) {
			updatedReq := *req
			updatedReq.Status = StatusApproved
			refundID := in.StripeRefundID
			updatedReq.StripeRefundID = &refundID
			updatedOrder := order.Order
			updatedOrder.Status = domain.StatusCancelled
			return &updatedReq, &updatedOrder, nil
		},
	}
	return &approveHarness{
		order:  order,
		req:    req,
		stripe: stripe,
		store:  store,
		reader: &fakeOrderReader{order: order},
		payout: &fakePayoutReader{},
	}
}

func (h *approveHarness) svc() *Service {
	return NewService(h.reader, h.payout, h.store, h.stripe, nil)
}

func TestApproveCancellation_AlreadyProcessed(t *testing.T) {
	h := newApproveHarness()
	h.req.Status = StatusApproved
	_, err := h.svc().ApproveCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "")
	assertAppErrorCode(t, err, 409, CodeCancellationRequestAlreadyProcessed)
}

func TestApproveCancellation_OrderShippedDuringRequest(t *testing.T) {
	h := newApproveHarness()
	h.order.Status = domain.StatusShipped
	_, err := h.svc().ApproveCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "")
	assertAppErrorCode(t, err, 409, CodeOrderNotCancellable)
	if len(h.stripe.calls) != 0 {
		t.Errorf("expected no Stripe calls, got %d", len(h.stripe.calls))
	}
}

func TestApproveCancellation_MissingPaymentIntent(t *testing.T) {
	h := newApproveHarness()
	h.order.StripePaymentIntentID = nil
	_, err := h.svc().ApproveCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "")
	assertAppErrorCode(t, err, 409, CodeOrderNotCancellable)
}

func TestApproveCancellation_RefundFailsMarksRequestFailed(t *testing.T) {
	h := newApproveHarness()
	h.stripe.refundResult = stripeResult{err: errors.New("network blip")}

	_, err := h.svc().ApproveCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "")
	assertAppErrorCode(t, err, 502, CodeRefundFailed)
	if h.store.markFailedCalls != 1 {
		t.Errorf("MarkFailed calls = %d, want 1", h.store.markFailedCalls)
	}
	if h.store.lastFailedReason == "" {
		t.Error("expected a populated failure reason")
	}
	// Refund should have been attempted exactly once and with the
	// deterministic cancellation:<request>:refund idempotency key.
	if len(h.stripe.calls) != 1 || h.stripe.calls[0].kind != "refund" {
		t.Fatalf("expected single refund call, got %+v", h.stripe.calls)
	}
	wantKey := fmt.Sprintf("cancellation:%s:refund", testRequestID)
	if h.stripe.calls[0].idempotencyKey != wantKey {
		t.Errorf("refund idempotency key = %q, want %q", h.stripe.calls[0].idempotencyKey, wantKey)
	}
}

func TestApproveCancellation_ReversalFailsMarksRequestFailed(t *testing.T) {
	h := newApproveHarness()
	payoutID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	xfer := "tr_test_001"
	h.payout.payouts = []domain.Payout{
		{
			ID:               payoutID,
			TenantID:         testTenantID,
			SellerID:         testSellerID,
			OrderID:          testOrderID,
			Amount:           4000,
			Status:           domain.PayoutStatusCompleted,
			StripeTransferID: &xfer,
		},
	}
	h.stripe.reversalResults = []stripeResult{{err: errors.New("stripe 500")}}

	_, err := h.svc().ApproveCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "")
	assertAppErrorCode(t, err, 502, CodeTransferReversalFailed)
	if h.store.markFailedCalls != 1 {
		t.Errorf("MarkFailed calls = %d, want 1", h.store.markFailedCalls)
	}
	// Refund should have already happened before the reversal loop.
	if len(h.stripe.calls) != 2 {
		t.Fatalf("expected refund + reversal, got %d calls", len(h.stripe.calls))
	}
	wantReverseKey := fmt.Sprintf("cancellation:%s:reverse:%s", testRequestID, payoutID)
	if h.stripe.calls[1].idempotencyKey != wantReverseKey {
		t.Errorf("reverse idempotency key = %q, want %q", h.stripe.calls[1].idempotencyKey, wantReverseKey)
	}
}

func TestApproveCancellation_SkipsIncompletePayouts(t *testing.T) {
	h := newApproveHarness()
	pendingXfer := "tr_should_skip_pending"
	nilXferPayoutID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	completedPayoutID := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	completedXfer := "tr_test_do_reverse"

	h.payout.payouts = []domain.Payout{
		// Pending payout — skipped (transfer not yet confirmed).
		{
			ID:               uuid.MustParse("66666666-6666-6666-6666-666666666666"),
			OrderID:          testOrderID,
			Status:           domain.PayoutStatusPending,
			Amount:           1000,
			StripeTransferID: &pendingXfer,
		},
		// Completed but missing transfer id — also skipped.
		{
			ID:      nilXferPayoutID,
			OrderID: testOrderID,
			Status:  domain.PayoutStatusCompleted,
			Amount:  1500,
		},
		// Completed and has transfer id — the one we actually reverse.
		{
			ID:               completedPayoutID,
			OrderID:          testOrderID,
			Status:           domain.PayoutStatusCompleted,
			Amount:           2500,
			StripeTransferID: &completedXfer,
		},
	}
	h.stripe.reversalResults = []stripeResult{{id: "trr_test_001"}}

	result, err := h.svc().ApproveCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "op comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusApproved {
		t.Errorf("final status = %q, want approved", result.Status)
	}

	// Exactly one reversal should have fired — against the completed payout.
	reversals := 0
	for _, c := range h.stripe.calls {
		if c.kind == "reverse" {
			reversals++
			if c.target != completedXfer {
				t.Errorf("reversed %q, want %q", c.target, completedXfer)
			}
		}
	}
	if reversals != 1 {
		t.Errorf("reversal count = %d, want 1", reversals)
	}

	// ApproveTx should have received exactly the reversed payouts the
	// service confirmed succeeded — not the skipped ones.
	if len(h.store.approveTxInputs) != 1 {
		t.Fatalf("ApproveTx called %d times, want 1", len(h.store.approveTxInputs))
	}
	in := h.store.approveTxInputs[0]
	if len(in.ReversedPayouts) != 1 {
		t.Fatalf("ReversedPayouts len = %d, want 1", len(in.ReversedPayouts))
	}
	if in.ReversedPayouts[0].PayoutID != completedPayoutID {
		t.Errorf("reversed payout id = %s, want %s", in.ReversedPayouts[0].PayoutID, completedPayoutID)
	}
	if in.ReversedPayouts[0].StripeReversalID != "trr_test_001" {
		t.Errorf("stripe reversal id = %q, want %q", in.ReversedPayouts[0].StripeReversalID, "trr_test_001")
	}
	// The Go-side pre-check list must match the SQL guard the tx uses.
	wantStatuses := cancellableStatuses()
	if len(in.CancellableStatus) != len(wantStatuses) {
		t.Errorf("CancellableStatus len = %d, want %d", len(in.CancellableStatus), len(wantStatuses))
	}
}

func TestApproveCancellation_StatusRaceAfterStripeMarksFailed(t *testing.T) {
	h := newApproveHarness()
	// Stripe succeeds, but between phase 2 and phase 3 the seller
	// shipped the order, so ApproveTx reports ErrOrderStatusChanged.
	h.store.approveTxFn = func(ctx context.Context, tenantID uuid.UUID, in ApprovalTxInput) (*CancellationRequest, *domain.Order, error) {
		return nil, nil, ErrOrderStatusChanged
	}

	_, err := h.svc().ApproveCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "")
	assertAppErrorCode(t, err, 409, CodeOrderNotCancellable)
	if h.store.markFailedCalls != 1 {
		t.Errorf("MarkFailed calls = %d, want 1 (refund must be reconciled manually)", h.store.markFailedCalls)
	}
	if h.store.lastFailedReason == "" {
		t.Error("expected failure reason describing the race")
	}
}

func TestApproveCancellation_HappyPathNoPayouts(t *testing.T) {
	h := newApproveHarness() // zero payouts by default
	result, err := h.svc().ApproveCancellation(context.Background(), testTenantID, testRequestID, testSellerID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusApproved {
		t.Errorf("status = %q, want approved", result.Status)
	}
	// Only the refund call should have fired.
	if len(h.stripe.calls) != 1 || h.stripe.calls[0].kind != "refund" {
		t.Errorf("expected single refund call, got %+v", h.stripe.calls)
	}
	if h.store.markFailedCalls != 0 {
		t.Errorf("MarkFailed calls = %d, want 0 on happy path", h.store.markFailedCalls)
	}
	if len(h.store.approveTxInputs) != 1 {
		t.Fatalf("ApproveTx calls = %d, want 1", len(h.store.approveTxInputs))
	}
	// The ApproveTx input must carry the refund id the Stripe call returned.
	if h.store.approveTxInputs[0].StripeRefundID != "re_test_ok" {
		t.Errorf("ApproveTx StripeRefundID = %q, want %q",
			h.store.approveTxInputs[0].StripeRefundID, "re_test_ok")
	}
}

// -----------------------------------------------------------------------------
// GetByIDForSeller — seller-scoped single lookup
// -----------------------------------------------------------------------------

func TestGetByIDForSeller_Success(t *testing.T) {
	store := &fakeRequestsStore{
		getByIDFn: func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
			return buildPendingRequest(), nil
		},
	}
	svc := NewService(&fakeOrderReader{order: buildTestOrder(domain.StatusPaid)}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)

	got, err := svc.GetByIDForSeller(context.Background(), testTenantID, testSellerID, testRequestID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ID != testRequestID {
		t.Errorf("got = %+v, want request id %s", got, testRequestID)
	}
}

func TestGetByIDForSeller_RequestNotFound(t *testing.T) {
	store := &fakeRequestsStore{
		getByIDFn: func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
			return nil, nil
		},
	}
	svc := NewService(&fakeOrderReader{order: buildTestOrder(domain.StatusPaid)}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)
	_, err := svc.GetByIDForSeller(context.Background(), testTenantID, testSellerID, testRequestID)
	assertAppErrorCode(t, err, 404, CodeCancellationRequestNotFound)
}

// A foreign seller must NOT be able to read another seller's cancellation
// request through the id-lookup endpoint. The service wraps the mismatch
// in 404 so it does not leak existence.
func TestGetByIDForSeller_SellerMismatch(t *testing.T) {
	store := &fakeRequestsStore{
		getByIDFn: func(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
			return buildPendingRequest(), nil
		},
	}
	order := buildTestOrder(domain.StatusPaid)
	order.SellerID = uuid.MustParse("99999999-9999-9999-9999-999999999999")
	svc := NewService(&fakeOrderReader{order: order}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)

	_, err := svc.GetByIDForSeller(context.Background(), testTenantID, testSellerID, testRequestID)
	assertAppErrorCode(t, err, 404, CodeNotOrderSeller)
}

// -----------------------------------------------------------------------------
// ListByStatus — seller-scoped list
// -----------------------------------------------------------------------------

// The service MUST propagate the seller id into the repository call so the
// repository's JOIN against order_svc.orders can filter on seller ownership.
// Regression guard for the cross-seller leak fix.
func TestListByStatus_PropagatesSellerIDToStore(t *testing.T) {
	store := &fakeRequestsStore{}
	svc := NewService(&fakeOrderReader{}, &fakePayoutReader{}, store, &fakeStripeClient{}, nil)

	_, _, err := svc.ListByStatus(context.Background(), testTenantID, testSellerID, StatusPending, 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.listByStatusCalls != 1 {
		t.Fatalf("ListByStatus calls = %d, want 1", store.listByStatusCalls)
	}
	if store.lastListSellerID != testSellerID {
		t.Errorf("store received seller id %s, want %s", store.lastListSellerID, testSellerID)
	}
}
