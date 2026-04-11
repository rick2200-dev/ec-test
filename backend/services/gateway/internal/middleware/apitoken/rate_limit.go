package apitoken

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	pkgredis "github.com/Riku-KANO/ec-test/pkg/redis"
)

// rateLimitScript implements a sliding-window counter keyed on a token id.
// Two adjacent 1-second buckets are kept live; the weighted sum of the
// previous bucket (by the fraction of the current second elapsed) plus
// the current bucket's live count gives a smooth window that does not
// reset abruptly on the second boundary (the classic failure mode of a
// fixed-window counter).
//
// KEYS:
//
//	KEYS[1] — current-second bucket, e.g. "apitok_rl:<id>:1712000000"
//	KEYS[2] — previous-second bucket, e.g. "apitok_rl:<id>:1711999999"
//
// ARGV:
//
//	ARGV[1] — rate limit (requests per second)
//	ARGV[2] — burst (additive headroom on top of the steady-state rate)
//	ARGV[3] — elapsed fraction of the current second in [0, 1000] (milliseconds)
//
// Returns a 2-tuple: {allowed, weighted_count}.
//
//	allowed = 1 → request allowed, current bucket already incremented.
//	allowed = 0 → request denied, current bucket NOT incremented (so a
//	              thrashing client can't inflate the counter further).
//
// Both keys get a 2-second TTL on every hit so they expire naturally after
// the sliding window no longer references them.
const rateLimitScript = `
local limit = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local elapsed_ms = tonumber(ARGV[3])

local prev = tonumber(redis.call('GET', KEYS[2]) or '0')
local curr = tonumber(redis.call('GET', KEYS[1]) or '0')

-- Weight of the previous bucket is (1000 - elapsed_ms) / 1000.
local weighted = math.floor((prev * (1000 - elapsed_ms)) / 1000) + curr

if weighted + 1 > limit + burst then
  return {0, weighted}
end

local new_curr = redis.call('INCR', KEYS[1])
redis.call('EXPIRE', KEYS[1], 2)
-- Refresh the previous bucket's TTL too so a slow client doesn't lose it
-- mid-window; cheap because it's a single EXPIRE.
if prev > 0 then
  redis.call('EXPIRE', KEYS[2], 2)
end
return {1, weighted + new_curr}
`

// Limiter is a Redis-backed token-based rate limiter. It is only applied
// to API-token requests; JWT requests pass through untouched. Per-token
// RPS/burst overrides come from apitoken.Context (0 = use the defaults
// configured on the limiter).
type Limiter struct {
	rdb     *pkgredis.Client
	script  *goredis.Script
	defRPS  int
	defBurst int
}

// NewLimiter builds a Limiter. defRPS and defBurst are the fallbacks used
// when a token does not override either value. The caller is expected to
// pass positive values — the gateway's config.getEnvInt already enforces
// this on the env-var path, so no additional clamping is done here
// (0/negative would effectively disable token traffic, which is a
// configuration bug we want loudly visible rather than silently masked).
func NewLimiter(rdb *pkgredis.Client, defRPS, defBurst int) *Limiter {
	return &Limiter{
		rdb:      rdb,
		script:   goredis.NewScript(rateLimitScript),
		defRPS:   defRPS,
		defBurst: defBurst,
	}
}

// Middleware returns the http.Handler middleware. It returns 429 +
// Retry-After: 1 when the token exceeds its window; transport/script
// failures fail-open (request allowed, warning logged) because
// blackholing requests on a Redis outage is worse than temporarily
// uncapped throughput on an already-authenticated caller.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ac, ok := FromContext(r.Context())
		if !ok {
			// No API token → JWT path; rate limiting is handled elsewhere.
			next.ServeHTTP(w, r)
			return
		}

		rps := ac.RateLimitRPS
		if rps <= 0 {
			rps = l.defRPS
		}
		burst := ac.RateLimitBurst
		if burst <= 0 {
			burst = l.defBurst
		}

		allowed, err := l.check(r.Context(), ac.ID.String(), rps, burst)
		if err != nil {
			// Fail-open: Redis outage should not take down a
			// successfully-authenticated API-token caller. We still log
			// loudly so ops can act on it.
			slog.Warn("api token rate limit check failed, allowing",
				"token_id", ac.ID,
				"error", err,
			)
			next.ServeHTTP(w, r)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", "1")
			httputil.JSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "rate limit exceeded; retry in 1 second",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// check runs the Lua script and returns whether the request is allowed.
// tokenID is used as the bucket namespace so independent tokens get
// independent windows; aborting an over-limit call on one token does not
// affect another token from the same seller.
func (l *Limiter) check(ctx context.Context, tokenID string, rps, burst int) (bool, error) {
	now := time.Now()
	sec := now.Unix()
	elapsedMs := now.UnixMilli() - (sec * 1000)

	currKey := "apitok_rl:" + tokenID + ":" + strconv.FormatInt(sec, 10)
	prevKey := "apitok_rl:" + tokenID + ":" + strconv.FormatInt(sec-1, 10)

	res, err := l.script.Run(ctx, l.rdb, []string{currKey, prevKey}, rps, burst, elapsedMs).Result()
	if err != nil {
		return false, err
	}
	arr, ok := res.([]any)
	if !ok || len(arr) < 1 {
		return false, errUnexpectedScriptResult
	}
	// Real Redis returns Lua numbers as int64; miniredis's in-process
	// Lua runtime returns plain int. Accept both so tests and prod run
	// the same code path.
	switch v := arr[0].(type) {
	case int64:
		return v == 1, nil
	case int:
		return v == 1, nil
	default:
		return false, errUnexpectedScriptResult
	}
}

// errUnexpectedScriptResult is raised when the rate-limit Lua script
// returns something other than the expected 2-tuple. It should never fire
// in production; the type assertion is defensive against a future script
// edit drifting from the caller.
var errUnexpectedScriptResult = &scriptResultError{}

type scriptResultError struct{}

func (*scriptResultError) Error() string {
	return "apitoken: unexpected rate-limit script result shape"
}
