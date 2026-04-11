package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Riku-KANO/ec-test/pkg/database"
	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// ============================================================================
// Seller API access token service layer.
//
// Token wire format (plaintext, never persisted after issuance):
//
//     <prefix><lookup12>_<secret43>
//
//   prefix  — env-configured, rotatable (default "sk_live_")
//   lookup  — 12 base62 chars (~71 bits of entropy), unique-indexed in the
//             DB so the gateway hot path can do an O(1) B-tree probe
//   secret  — 43 base62 chars (~256 bits of entropy), SHA-256'd on the
//             server and compared in constant time on every request
//
// Both components are generated directly as base62 characters using
// rejection sampling (see generateBase62) rather than by encoding a
// fixed-length byte slice — this guarantees constant output width and
// sidesteps modulo-bias, which matters for the 256-bit security claim on
// the secret.
//
// Rationale: mirrors the Stripe/GitHub PAT pattern. We never hash inbound
// tokens against every row; the lookup narrows the search to exactly one
// candidate, then subtle.ConstantTimeCompare verifies the secret.
// ============================================================================

const (
	apiTokenLookupChars = 12 // 12 base62 chars → ≈71 bits of lookup entropy
	apiTokenSecretChars = 43 // 43 base62 chars → ≈256 bits of secret entropy
)

// IssueAPIToken creates a new seller API access token. Returns the persisted
// record and the plaintext token (which the caller MUST return to the client
// exactly once — it is never recoverable afterward).
//
// Authorization: caller must be owner on the target seller. Enforced via
// requireSellerRoleAtLeastTx inside the same transaction that performs the
// insert, so RBAC checks and the insert are atomic.
func (s *AuthService) IssueAPIToken(
	ctx context.Context,
	tenantID, sellerID uuid.UUID,
	name string,
	scopes []domain.APITokenScope,
	rateLimitRPS, rateLimitBurst *int,
	expiresAt *time.Time,
	tokenPrefix string,
) (*domain.SellerAPIToken, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", apperrors.BadRequest("name is required")
	}
	if len(name) > 120 {
		return nil, "", apperrors.BadRequest("name must be 120 characters or fewer")
	}
	if len(scopes) == 0 {
		return nil, "", apperrors.BadRequest("at least one scope is required")
	}
	// Deduplicate + validate scopes up-front so the CHECK constraint never
	// has to take the fall for bad client input.
	seen := make(map[domain.APITokenScope]struct{}, len(scopes))
	cleanScopes := make([]domain.APITokenScope, 0, len(scopes))
	for _, sc := range scopes {
		if !sc.Valid() {
			return nil, "", apperrors.BadRequest("invalid scope: " + string(sc))
		}
		if _, dup := seen[sc]; dup {
			continue
		}
		seen[sc] = struct{}{}
		cleanScopes = append(cleanScopes, sc)
	}
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return nil, "", apperrors.BadRequest("expires_at must be in the future")
	}
	if rateLimitRPS != nil && *rateLimitRPS <= 0 {
		return nil, "", apperrors.BadRequest("rate_limit_rps must be positive")
	}
	if rateLimitBurst != nil && *rateLimitBurst <= 0 {
		return nil, "", apperrors.BadRequest("rate_limit_burst must be positive")
	}

	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return nil, "", apperrors.Unauthorized("caller identity required")
	}

	// Generate plaintext pieces before opening the transaction so a crypto
	// failure doesn't leave a half-open tx.
	plaintext, lookup, hash, err := generateAPIToken(tokenPrefix)
	if err != nil {
		return nil, "", apperrors.Internal("failed to generate api token", err)
	}

	token := &domain.SellerAPIToken{
		TenantID:            tenantID,
		SellerID:            sellerID,
		Name:                name,
		TokenPrefix:         tokenPrefix,
		TokenLookup:         lookup,
		TokenHash:           hash,
		Scopes:              cleanScopes,
		RateLimitRPS:        rateLimitRPS,
		RateLimitBurst:      rateLimitBurst,
		IssuedByAuth0UserID: tc.UserID,
		ExpiresAt:           expiresAt,
	}

	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Only owners may issue tokens. ScopesForSellerRole already enforces
		// the same rule structurally — we check the actor's live role here
		// so a member/admin can never issue even via a forged request.
		if err := s.requireSellerRoleAtLeastTx(ctx, tx, tenantID, sellerID, tc.UserID, domain.SellerUserRoleOwner); err != nil {
			return err
		}
		// Double-check the requested scopes are within the set the issuer's
		// role is allowed to grant (today: owner → all six).
		allowed := make(map[domain.APITokenScope]struct{}, 6)
		for _, sc := range domain.ScopesForSellerRole(domain.SellerUserRoleOwner) {
			allowed[sc] = struct{}{}
		}
		for _, sc := range cleanScopes {
			if _, ok := allowed[sc]; !ok {
				return domain.ErrAPITokenScopeNotGranted
			}
		}
		return s.apiTokens.CreateTx(ctx, tx, token)
	})
	if err != nil {
		return nil, "", mapAPITokenError(err, "failed to issue api token")
	}

	slog.Info("seller api token issued",
		"id", token.ID,
		"tenant_id", tenantID,
		"seller_id", sellerID,
		"issuer", tc.UserID,
		"scopes", cleanScopes,
	)
	return token, plaintext, nil
}

// ListAPITokens returns tokens belonging to the given seller, newest first,
// including revoked/expired rows so the dashboard can display history.
func (s *AuthService) ListAPITokens(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error) {
	tokens, total, err := s.apiTokens.ListBySeller(ctx, tenantID, sellerID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list api tokens", err)
	}
	return tokens, total, nil
}

// GetAPIToken retrieves a single token by ID. Returns NotFound if the token
// does not exist or does not belong to the given seller (same shape either
// way to avoid leaking existence across sellers).
func (s *AuthService) GetAPIToken(ctx context.Context, tenantID, sellerID, id uuid.UUID) (*domain.SellerAPIToken, error) {
	t, err := s.apiTokens.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get api token", err)
	}
	if t == nil || t.SellerID != sellerID {
		return nil, apperrors.NotFound("api token not found")
	}
	return t, nil
}

// RevokeAPIToken marks a token as revoked. Only owners may call this. The
// call is idempotent: revoking an already-revoked token returns success.
//
// Returns the (prefix, lookup) pair of the revoked token so upstream
// callers — specifically the gateway — can evict their lookup cache
// synchronously instead of waiting for the short TTL. This is a
// deliberate leak of persistence-layer fields into the service signature:
// the alternative (a second round-trip from the handler) would race the
// eviction against a concurrent Load from another gateway request and
// defeat the purpose.
func (s *AuthService) RevokeAPIToken(ctx context.Context, tenantID, sellerID, id uuid.UUID) (prefix, lookup string, err error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return "", "", apperrors.Unauthorized("caller identity required")
	}

	// Pre-check seller membership so we return 404 (not 403) for tokens
	// belonging to other sellers.
	existing, err := s.apiTokens.GetByID(ctx, tenantID, id)
	if err != nil {
		return "", "", apperrors.Internal("failed to get api token", err)
	}
	if existing == nil || existing.SellerID != sellerID {
		return "", "", apperrors.NotFound("api token not found")
	}

	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.requireSellerRoleAtLeastTx(ctx, tx, tenantID, sellerID, tc.UserID, domain.SellerUserRoleOwner); err != nil {
			return err
		}
		return s.apiTokens.RevokeTx(ctx, tx, tenantID, id, tc.UserID)
	})
	if err != nil {
		return "", "", mapAPITokenError(err, "failed to revoke api token")
	}

	slog.Info("seller api token revoked",
		"id", id,
		"tenant_id", tenantID,
		"seller_id", sellerID,
		"actor", tc.UserID,
	)
	return existing.TokenPrefix, existing.TokenLookup, nil
}

// LookupAPIToken is the gateway hot-path entry point. Given the three pieces
// of a plaintext token it resolves the row, verifies the secret in constant
// time, checks revoke/expiry, and returns the token record so the gateway
// can construct its tenant context.
//
// Callers must NOT persist the returned token struct beyond the lifetime of
// a single request; the gateway keeps its own short-TTL cache.
func (s *AuthService) LookupAPIToken(ctx context.Context, prefix, lookup, secret string) (*domain.SellerAPIToken, error) {
	if prefix == "" || lookup == "" || secret == "" {
		return nil, domain.ErrAPITokenInvalidFormat
	}

	t, err := s.apiTokens.GetByLookup(ctx, prefix, lookup)
	if err != nil {
		return nil, apperrors.Internal("failed to look up api token", err)
	}
	if t == nil {
		return nil, domain.ErrAPITokenNotFound
	}

	// Constant-time secret comparison. We hash the provided secret once and
	// compare byte-wise against the stored hash — timing depends only on the
	// fixed 32-byte length, not on match-vs-miss position.
	provided := sha256.Sum256([]byte(secret))
	if subtle.ConstantTimeCompare(provided[:], t.TokenHash) != 1 {
		return nil, domain.ErrAPITokenNotFound
	}

	now := time.Now()
	if t.RevokedAt != nil {
		return nil, domain.ErrAPITokenRevoked
	}
	if t.ExpiresAt != nil && !now.Before(*t.ExpiresAt) {
		return nil, domain.ErrAPITokenExpired
	}

	// Best-effort last_used_at update. We intentionally fire-and-forget: the
	// gateway will cache this token for up to APITokenCacheTTL, so we only
	// incur this write once per cache miss anyway.
	go func(id uuid.UUID) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := s.apiTokens.TouchLastUsedAt(bgCtx, id); err != nil {
			slog.Warn("api token last_used_at update failed", "id", id, "error", err)
		}
	}(t.ID)

	return t, nil
}

// generateAPIToken produces a fresh plaintext token and its persistence
// pieces: (plaintext, lookup, hash). Returns an error only if crypto/rand
// fails, which should never happen on a healthy host.
//
// The plaintext is returned so the service layer can hand it back to the
// caller exactly once. Everything else (lookup, hash) is what lands in the
// database row.
func generateAPIToken(prefix string) (plaintext, lookup string, hash []byte, err error) {
	lookup, err = generateBase62(apiTokenLookupChars)
	if err != nil {
		return "", "", nil, fmt.Errorf("generate lookup: %w", err)
	}
	secret, err := generateBase62(apiTokenSecretChars)
	if err != nil {
		return "", "", nil, fmt.Errorf("generate secret: %w", err)
	}

	h := sha256.Sum256([]byte(secret))
	hash = h[:]

	plaintext = prefix + lookup + "_" + secret
	return plaintext, lookup, hash, nil
}

// ParseAPIToken splits a plaintext token into (prefix, lookup, secret).
// Returns ErrAPITokenInvalidFormat on any structural issue. Exposed for the
// gateway-side authz path and the internal lookup handler so both use the
// same parser.
func ParseAPIToken(raw, prefix string) (lookup, secret string, err error) {
	if prefix == "" || !strings.HasPrefix(raw, prefix) {
		return "", "", domain.ErrAPITokenInvalidFormat
	}
	rest := raw[len(prefix):]
	sep := strings.IndexByte(rest, '_')
	if sep <= 0 || sep == len(rest)-1 {
		return "", "", domain.ErrAPITokenInvalidFormat
	}
	return rest[:sep], rest[sep+1:], nil
}

// base62Alphabet is the standard alphanumeric alphabet. Order matches
// base62's conventional layout (digits first, then upper, then lower).
const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// base62RejectThreshold is the smallest byte value that would introduce
// modulo bias when mapped to 62. Because 256 = 62*4 + 8, bytes in [248,255]
// would map to digits [0,7] more often than others. Rejecting them gives
// a uniform distribution over the 62-character alphabet at the cost of
// ~3% rejection overhead — irrelevant for issuance, and issuance is the
// only place this is called.
const base62RejectThreshold = 248

// generateBase62 returns n base62 characters drawn uniformly from the
// alphabet using rejection sampling. Each output character is produced
// from a single random byte; biased bytes are rejected and re-drawn.
//
// Why not encode N random bytes into base62 digits? For fixed widths like
// 12 and 43 chars, byte-to-base62 conversion isn't length-stable — e.g.
// 9 bytes of 0xFF overflow 12 digits. Direct per-char generation is
// length-stable by construction and easier to reason about.
func generateBase62(n int) (string, error) {
	out := make([]byte, n)
	// Refill strategy: pull 2x bytes at a time to amortize the syscall
	// cost. 3% rejection means on average we need 1.03n bytes.
	bufLen := n * 2
	if bufLen < 16 {
		bufLen = 16
	}
	buf := make([]byte, bufLen)
	pos := bufLen // force an initial refill
	for i := 0; i < n; {
		if pos >= bufLen {
			if _, err := rand.Read(buf); err != nil {
				return "", err
			}
			pos = 0
		}
		b := buf[pos]
		pos++
		if b >= base62RejectThreshold {
			continue
		}
		out[i] = base62Alphabet[b%62]
		i++
	}
	return string(out), nil
}

// mapAPITokenError translates repository/domain sentinel errors into HTTP
// AppErrors so handlers return correct status codes without each having to
// know the whole error catalogue.
func mapAPITokenError(err error, internalMsg string) error {
	if err == nil {
		return nil
	}
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	switch {
	case errors.Is(err, domain.ErrAPITokenNotFound):
		return apperrors.NotFound("api token not found")
	case errors.Is(err, domain.ErrAPITokenRevoked):
		return apperrors.Forbidden("api token has been revoked")
	case errors.Is(err, domain.ErrAPITokenExpired):
		return apperrors.Forbidden("api token has expired")
	case errors.Is(err, domain.ErrAPITokenInvalidFormat):
		return apperrors.BadRequest("api token malformed")
	case errors.Is(err, domain.ErrAPITokenScopeNotGranted):
		return apperrors.Forbidden("requested scope is not permitted for the issuing role")
	case errors.Is(err, domain.ErrInsufficientRole):
		return apperrors.Forbidden("owner role is required to manage api tokens")
	}
	return apperrors.Internal(internalMsg, err)
}
