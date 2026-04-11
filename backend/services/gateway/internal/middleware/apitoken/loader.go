package apitoken

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// Default cache TTL for successful lookups. Matches the role loader's 30s
// window so revocation semantics are consistent across auth surfaces.
const defaultCacheTTL = 30 * time.Second

// TokenInfo is the decoded payload returned by a successful auth service
// lookup. It is what the Load method returns and what the cache stores.
// Keep this in sync with apiTokenSummary in the auth service handler.
type TokenInfo struct {
	ID                  uuid.UUID
	TenantID            uuid.UUID
	SellerID            uuid.UUID
	Scopes              []string
	RateLimitRPS        int // 0 = use gateway default
	RateLimitBurst      int // 0 = use gateway default
	IssuedByAuth0UserID string
}

// LookupStatus mirrors the status enum emitted by the auth service at
// POST /internal/authz/api-token. "active" is the only terminal success;
// every other value means the token exists but should be rejected.
type LookupStatus string

const (
	StatusActive   LookupStatus = "active"
	StatusRevoked  LookupStatus = "revoked"
	StatusExpired  LookupStatus = "expired"
	StatusNotFound LookupStatus = "not_found"
	StatusInvalid  LookupStatus = "invalid"
)

// Loader resolves API tokens by calling the auth service and caches the
// result for defaultCacheTTL. The client it holds must already be wrapped
// with the X-Internal-Token header via proxy.ServiceClient.WithHeader.
//
// The cache stores only "active" hits — negative results (revoked,
// not_found, etc.) are intentionally not cached so a newly-issued token
// works on its very first call. Negative caching can be added later if
// brute-force volume makes it worthwhile.
type Loader struct {
	client *proxy.ServiceClient
	ttl    time.Duration

	mu      sync.RWMutex
	entries map[string]loaderEntry
}

type loaderEntry struct {
	info      TokenInfo
	expiresAt time.Time
}

// NewLoader builds a Loader. The client must already carry the
// X-Internal-Token header. ttl controls the success-cache TTL; pass 0 to
// use the default (30s).
func NewLoader(client *proxy.ServiceClient, ttl time.Duration) *Loader {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &Loader{
		client:  client,
		ttl:     ttl,
		entries: make(map[string]loaderEntry),
	}
}

// lookupRequest matches the auth service request shape.
type lookupRequest struct {
	Prefix string `json:"prefix"`
	Lookup string `json:"lookup"`
	Secret string `json:"secret"`
}

// lookupResponse mirrors apiTokenLookupResponse on the auth service.
type lookupResponse struct {
	Status string            `json:"status"`
	Token  *lookupTokenField `json:"token,omitempty"`
}

type lookupTokenField struct {
	ID                  uuid.UUID `json:"id"`
	TenantID            uuid.UUID `json:"tenant_id"`
	SellerID            uuid.UUID `json:"seller_id"`
	Scopes              []string  `json:"scopes"`
	RateLimitRPS        *int      `json:"rate_limit_rps,omitempty"`
	RateLimitBurst      *int      `json:"rate_limit_burst,omitempty"`
	IssuedByAuth0UserID string    `json:"issued_by_auth0_user_id"`
}

// ErrLoaderTransport is returned when the auth service call itself fails
// (network error, non-200 HTTP, undecodable body). Authentication failures
// that come back as Status != "active" are not errors — they are returned
// as (zero, status, nil) so callers can 401 without logging as a transport
// issue.
var ErrLoaderTransport = errors.New("apitoken: transport error")

// Load resolves a token by its three wire pieces. The boolean return is
// true only when the token is "active"; any non-active status comes back
// as (zero, status, nil) so the caller can distinguish a revoked/expired
// token (401) from a transport failure (503).
func (l *Loader) Load(ctx context.Context, prefix, lookup, secret string) (TokenInfo, LookupStatus, error) {
	key := cacheKey(prefix, lookup)

	// Fast path: cache hit. We still verify the secret hash on the server
	// side for every miss, but we trust the 30s cache for hits — same
	// staleness window as RBAC role lookups.
	if info, ok := l.getCached(key); ok {
		return info, StatusActive, nil
	}

	body, err := json.Marshal(lookupRequest{Prefix: prefix, Lookup: lookup, Secret: secret})
	if err != nil {
		return TokenInfo{}, "", fmt.Errorf("%w: marshal request: %v", ErrLoaderTransport, err)
	}

	raw, status, err := l.client.Post(ctx, "/internal/authz/api-token", bytes.NewReader(body))
	if err != nil {
		return TokenInfo{}, "", fmt.Errorf("%w: %v", ErrLoaderTransport, err)
	}
	if status != 200 {
		return TokenInfo{}, "", fmt.Errorf("%w: auth service status %d: %s", ErrLoaderTransport, status, string(raw))
	}

	var resp lookupResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return TokenInfo{}, "", fmt.Errorf("%w: decode response: %v", ErrLoaderTransport, err)
	}

	lookupStatus := LookupStatus(resp.Status)
	if lookupStatus != StatusActive {
		return TokenInfo{}, lookupStatus, nil
	}
	if resp.Token == nil {
		return TokenInfo{}, "", fmt.Errorf("%w: active response missing token body", ErrLoaderTransport)
	}

	info := TokenInfo{
		ID:                  resp.Token.ID,
		TenantID:            resp.Token.TenantID,
		SellerID:            resp.Token.SellerID,
		Scopes:              resp.Token.Scopes,
		IssuedByAuth0UserID: resp.Token.IssuedByAuth0UserID,
	}
	if resp.Token.RateLimitRPS != nil {
		info.RateLimitRPS = *resp.Token.RateLimitRPS
	}
	if resp.Token.RateLimitBurst != nil {
		info.RateLimitBurst = *resp.Token.RateLimitBurst
	}

	l.setCached(key, info)
	return info, StatusActive, nil
}

// Evict removes a token from the cache. Called by the gateway's revoke
// handler after a successful DELETE so the change is visible immediately
// instead of after the TTL window.
func (l *Loader) Evict(prefix, lookup string) {
	l.mu.Lock()
	delete(l.entries, cacheKey(prefix, lookup))
	l.mu.Unlock()
}

func (l *Loader) getCached(key string) (TokenInfo, bool) {
	l.mu.RLock()
	e, ok := l.entries[key]
	l.mu.RUnlock()
	if !ok {
		return TokenInfo{}, false
	}
	if time.Now().After(e.expiresAt) {
		l.mu.Lock()
		// Re-check under write lock in case another goroutine refreshed it.
		if cur, still := l.entries[key]; still && time.Now().After(cur.expiresAt) {
			delete(l.entries, key)
		}
		l.mu.Unlock()
		return TokenInfo{}, false
	}
	return e.info, true
}

func (l *Loader) setCached(key string, info TokenInfo) {
	l.mu.Lock()
	l.entries[key] = loaderEntry{info: info, expiresAt: time.Now().Add(l.ttl)}
	l.mu.Unlock()
}

// cacheKey is "<prefix>:<lookup>". The prefix is included so rotating the
// wire prefix invalidates caches the same instant it flips the namespace.
func cacheKey(prefix, lookup string) string { return prefix + ":" + lookup }
