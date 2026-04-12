package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	gostripe "github.com/stripe/stripe-go/v82"
)

// capturedRequest holds the values the test httptest.Server observed
// for a single Stripe API call. The Stripe client posts form-encoded
// bodies, so we snapshot both the URL and the parsed form values plus
// the Idempotency-Key header which is the whole point of these tests.
type capturedRequest struct {
	method         string
	path           string
	form           url.Values
	idempotencyKey string
}

// stripeTestServer spins up an httptest.Server, installs it as the
// active Stripe API backend, and returns the captured requests + a
// cleanup func. The returned slice is populated in order as the client
// makes calls.
func stripeTestServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, captured *capturedRequest)) (*[]capturedRequest, func()) {
	t.Helper()

	var mu sync.Mutex
	captured := make([]capturedRequest, 0, 2)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form: %v", err)
		}
		c := capturedRequest{
			method:         r.Method,
			path:           r.URL.Path,
			form:           r.Form,
			idempotencyKey: r.Header.Get("Idempotency-Key"),
		}
		mu.Lock()
		captured = append(captured, c)
		mu.Unlock()

		handler(w, r, &c)
	}))

	// Override the Stripe SDK to talk to our test server. Stripe keeps
	// backends in a package-level map, so restore the previous
	// implementation in cleanup to avoid leaking into other tests in the
	// same package.
	prev := gostripe.GetBackend(gostripe.APIBackend)
	backend := gostripe.GetBackendWithConfig(gostripe.APIBackend, &gostripe.BackendConfig{
		URL: gostripe.String(srv.URL),
	})
	gostripe.SetBackend(gostripe.APIBackend, backend)

	cleanup := func() {
		gostripe.SetBackend(gostripe.APIBackend, prev)
		srv.Close()
	}
	return &captured, cleanup
}

// TestCreateRefund_SendsIdempotencyKey is the load-bearing assertion
// for the order-cancellation flow: cancel approvals build their
// idempotency keys from the request id so safe retries never produce
// double refunds, which only works if the Stripe client actually
// forwards the key as the Idempotency-Key header.
func TestCreateRefund_SendsIdempotencyKey(t *testing.T) {
	captured, cleanup := stripeTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *capturedRequest) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":             "re_test_returned",
			"object":         "refund",
			"amount":         5000,
			"currency":       "jpy",
			"payment_intent": "pi_test_xyz",
			"status":         "succeeded",
		})
	})
	defer cleanup()

	// Construct a client — the secret key value does not matter for
	// tests because the override backend doesn't validate it. Using
	// "sk_test_dummy" keeps it obvious in logs.
	c := NewClient("sk_test_dummy")

	refundID, err := c.CreateRefund("pi_test_xyz", 5000, "cancellation:req-abc:refund")
	if err != nil {
		t.Fatalf("CreateRefund returned error: %v", err)
	}
	if refundID != "re_test_returned" {
		t.Errorf("refund id = %q, want %q", refundID, "re_test_returned")
	}

	if len(*captured) != 1 {
		t.Fatalf("expected 1 captured request, got %d", len(*captured))
	}
	got := (*captured)[0]
	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if got.path != "/v1/refunds" {
		t.Errorf("path = %q, want /v1/refunds", got.path)
	}
	if got.idempotencyKey != "cancellation:req-abc:refund" {
		t.Errorf("Idempotency-Key header = %q, want %q",
			got.idempotencyKey, "cancellation:req-abc:refund")
	}
	// The client also sends the refund amount and payment intent as
	// form fields — verify both so a future parameter rename shows up
	// in a failing test rather than in a Stripe 400.
	if got.form.Get("amount") != "5000" {
		t.Errorf("form amount = %q, want 5000", got.form.Get("amount"))
	}
	if got.form.Get("payment_intent") != "pi_test_xyz" {
		t.Errorf("form payment_intent = %q, want pi_test_xyz", got.form.Get("payment_intent"))
	}
}

func TestCreateRefund_PropagatesStripeError(t *testing.T) {
	captured, cleanup := stripeTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *capturedRequest) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "payment_intent cannot be refunded",
			},
		})
	})
	defer cleanup()

	c := NewClient("sk_test_dummy")

	_, err := c.CreateRefund("pi_test_bad", 5000, "cancellation:req-bad:refund")
	if err == nil {
		t.Fatal("expected error from Stripe 400, got nil")
	}
	if len(*captured) != 1 {
		t.Errorf("expected 1 captured request, got %d", len(*captured))
	}
}

func TestReverseTransfer_SendsIdempotencyKey(t *testing.T) {
	captured, cleanup := stripeTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *capturedRequest) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "trr_test_returned",
			"object":   "transfer_reversal",
			"amount":   2500,
			"currency": "jpy",
			"transfer": "tr_test_abc",
		})
	})
	defer cleanup()

	c := NewClient("sk_test_dummy")

	reversalID, err := c.ReverseTransfer("tr_test_abc", 2500, "cancellation:req-xyz:reverse:payout-1")
	if err != nil {
		t.Fatalf("ReverseTransfer returned error: %v", err)
	}
	if reversalID != "trr_test_returned" {
		t.Errorf("reversal id = %q, want %q", reversalID, "trr_test_returned")
	}

	if len(*captured) != 1 {
		t.Fatalf("expected 1 captured request, got %d", len(*captured))
	}
	got := (*captured)[0]
	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	// The transfer reversal endpoint is /v1/transfers/{id}/reversals.
	wantPath := "/v1/transfers/tr_test_abc/reversals"
	if got.path != wantPath {
		t.Errorf("path = %q, want %q", got.path, wantPath)
	}
	wantKey := "cancellation:req-xyz:reverse:payout-1"
	if got.idempotencyKey != wantKey {
		t.Errorf("Idempotency-Key header = %q, want %q", got.idempotencyKey, wantKey)
	}
	if got.form.Get("amount") != "2500" {
		t.Errorf("form amount = %q, want 2500", got.form.Get("amount"))
	}
}

func TestReverseTransfer_PropagatesStripeError(t *testing.T) {
	_, cleanup := stripeTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *capturedRequest) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "no such transfer",
			},
		})
	})
	defer cleanup()

	c := NewClient("sk_test_dummy")

	_, err := c.ReverseTransfer("tr_does_not_exist", 100, "cancellation:req-abc:reverse:payout-1")
	if err == nil {
		t.Fatal("expected error from Stripe 404, got nil")
	}
}
