package apitoken

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// newTestLimiter spins up a miniredis (in-process fake) and returns the
// limiter. We use miniredis rather than a real Redis because the Lua
// script runs unchanged against it and keeping the test hermetic beats
// adding a docker dependency for `go test`.
func newTestLimiter(t *testing.T, rps, burst int) *Limiter { //nolint:unparam
	t.Helper()
	srv := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	// pkgredis.Client is a type alias for goredis.Client so *goredis.Client
	// satisfies the limiter's *pkgredis.Client parameter directly.
	return NewLimiter(client, rps, burst)
}

// runRequests fires n sequential requests through the limiter and
// returns the status-code histogram.
func runRequests(t *testing.T, l *Limiter, ac Context, n int) map[int]int {
	t.Helper()
	counts := make(map[int]int)
	mw := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for range n {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req = req.WithContext(WithContext(req.Context(), ac))
		mw.ServeHTTP(rec, req)
		counts[rec.Code]++
	}
	return counts
}

// JWT requests (no apitoken.Context) must pass through untouched. This
// is the most important invariant — if it breaks, every UI request
// takes a Redis round-trip.
func TestLimiter_JWTPassesThrough(t *testing.T) {
	l := newTestLimiter(t, 1, 0)

	var reached int
	mw := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached++
		w.WriteHeader(http.StatusOK)
	}))
	// Fire well above the limit; every one should succeed because no
	// apitoken.Context is set.
	for range 20 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		mw.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("JWT request hit rate limit: code=%d", rec.Code)
		}
	}
	if reached != 20 {
		t.Errorf("handler reached %d times, want 20", reached)
	}
}

// With rps=1 burst=0 the very first call should pass and the second
// immediate call should 429. Confirms the basic limiter math.
func TestLimiter_FirstCallAllowedSecondDenied(t *testing.T) {
	l := newTestLimiter(t, 1, 0)
	ac := Context{ID: uuid.New()}
	counts := runRequests(t, l, ac, 2)
	if counts[http.StatusOK] != 1 {
		t.Errorf("200 count = %d, want 1", counts[http.StatusOK])
	}
	if counts[http.StatusTooManyRequests] != 1 {
		t.Errorf("429 count = %d, want 1", counts[http.StatusTooManyRequests])
	}
}

// With rps=5 burst=5, the caller should be able to drive 10 requests
// through in the current second before being denied. This exercises
// the burst headroom calculation (limit + burst = 10).
func TestLimiter_BurstHeadroom(t *testing.T) {
	l := newTestLimiter(t, 5, 5)
	ac := Context{ID: uuid.New()}
	counts := runRequests(t, l, ac, 15)
	if counts[http.StatusOK] < 10 {
		t.Errorf("200 count = %d, want at least 10 (rps+burst)", counts[http.StatusOK])
	}
	if counts[http.StatusTooManyRequests] < 5 {
		t.Errorf("429 count = %d, want at least 5", counts[http.StatusTooManyRequests])
	}
}

// Two tokens with independent IDs must have independent buckets — one
// going into 429 must not affect the other.
func TestLimiter_PerTokenIsolation(t *testing.T) {
	l := newTestLimiter(t, 1, 0)

	idA := Context{ID: uuid.New()}
	idB := Context{ID: uuid.New()}

	// Drain token A.
	if c := runRequests(t, l, idA, 1)[http.StatusOK]; c != 1 {
		t.Fatalf("token A first call 200s: got %d", c)
	}
	if c := runRequests(t, l, idA, 1)[http.StatusTooManyRequests]; c != 1 {
		t.Fatalf("token A second call 429s: got %d", c)
	}

	// Token B should still be fresh.
	if c := runRequests(t, l, idB, 1)[http.StatusOK]; c != 1 {
		t.Errorf("token B should not have been rate-limited by token A: got %d", c)
	}
}

// Per-token overrides must win over the global defaults. If a token
// opts in to a higher rate (say rps=50), the caller should be able to
// issue 50+ requests per second even when the default is 1.
func TestLimiter_PerTokenOverride(t *testing.T) {
	l := newTestLimiter(t, 1, 0)
	ac := Context{
		ID:             uuid.New(),
		RateLimitRPS:   50,
		RateLimitBurst: 0,
	}
	counts := runRequests(t, l, ac, 20)
	if counts[http.StatusOK] != 20 {
		t.Errorf("200 count = %d, want 20 (override should allow)", counts[http.StatusOK])
	}
}

// 429 responses must include Retry-After: 1 so clients back off
// predictably instead of hot-looping.
func TestLimiter_RetryAfterHeader(t *testing.T) {
	l := newTestLimiter(t, 1, 0)
	ac := Context{ID: uuid.New()}

	mw := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Burn the first slot.
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/", nil).WithContext(WithContext(httptest.NewRequest("GET", "/", nil).Context(), ac))
	mw.ServeHTTP(rec1, req1)

	// Second call: should be 429 with Retry-After.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil).WithContext(WithContext(httptest.NewRequest("GET", "/", nil).Context(), ac))
	mw.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second call status = %d, want 429", rec2.Code)
	}
	if got := rec2.Header().Get("Retry-After"); got != "1" {
		t.Errorf("Retry-After = %q, want 1", got)
	}
}
