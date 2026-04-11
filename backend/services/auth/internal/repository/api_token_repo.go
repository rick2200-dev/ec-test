package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// APITokenRepository handles persistence of seller API access tokens.
type APITokenRepository struct {
	pool *pgxpool.Pool
}

// NewAPITokenRepository creates a new APITokenRepository.
func NewAPITokenRepository(pool *pgxpool.Pool) *APITokenRepository {
	return &APITokenRepository{pool: pool}
}

// Create inserts a new seller_api_tokens row. The caller is responsible for
// generating the plaintext token and hashing its secret portion; this repo
// only persists the already-hashed value.
func (r *APITokenRepository) Create(ctx context.Context, t *domain.SellerAPIToken) error {
	return database.TenantTx(ctx, r.pool, t.TenantID, func(tx pgx.Tx) error {
		return r.CreateTx(ctx, tx, t)
	})
}

// CreateTx inserts a new seller_api_tokens row inside an existing tenant
// transaction. Use this variant when the caller already holds a tx (for
// example, to perform an RBAC check and the insert atomically); use Create
// when no outer tx is needed.
func (r *APITokenRepository) CreateTx(ctx context.Context, tx pgx.Tx, t *domain.SellerAPIToken) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	scopes := scopesToStrings(t.Scopes)
	return tx.QueryRow(ctx,
		`INSERT INTO auth_svc.seller_api_tokens (
		    id, tenant_id, seller_id, name,
		    token_prefix, token_lookup, token_hash,
		    scopes, rate_limit_rps, rate_limit_burst,
		    issued_by_auth0_user_id, expires_at
		 )
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING created_at, updated_at`,
		t.ID, t.TenantID, t.SellerID, t.Name,
		t.TokenPrefix, t.TokenLookup, t.TokenHash,
		scopes, t.RateLimitRPS, t.RateLimitBurst,
		t.IssuedByAuth0UserID, t.ExpiresAt,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
}

// GetByID retrieves a token by its primary key within a tenant scope.
// Returns (nil, nil) if no row is found.
func (r *APITokenRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerAPIToken, error) {
	var out *domain.SellerAPIToken
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		t, err := scanTokenRow(tx.QueryRow(ctx, selectTokenColumns+`
		     FROM auth_svc.seller_api_tokens
		     WHERE id = $1 AND tenant_id = $2`, id, tenantID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		out = t
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get api token by id: %w", err)
	}
	return out, nil
}

// ListBySeller returns tokens for the given seller ordered by created_at
// (newest first), along with the total row count for pagination. Includes
// revoked and expired rows so the dashboard can display history.
func (r *APITokenRepository) ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	var tokens []domain.SellerAPIToken
	var total int
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM auth_svc.seller_api_tokens
			 WHERE tenant_id = $1 AND seller_id = $2`,
			tenantID, sellerID,
		).Scan(&total); err != nil {
			return err
		}

		rows, err := tx.Query(ctx, selectTokenColumns+`
		     FROM auth_svc.seller_api_tokens
		     WHERE tenant_id = $1 AND seller_id = $2
		     ORDER BY created_at DESC
		     LIMIT $3 OFFSET $4`,
			tenantID, sellerID, limit, offset,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			t, scanErr := scanTokenRow(rows)
			if scanErr != nil {
				return scanErr
			}
			tokens = append(tokens, *t)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list api tokens: %w", err)
	}
	return tokens, total, nil
}

// Revoke marks a token as revoked. Idempotent: calling it on an already
// revoked token is a no-op that still returns nil.
func (r *APITokenRepository) Revoke(ctx context.Context, tenantID, id uuid.UUID, actorAuth0UserID string) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return r.RevokeTx(ctx, tx, tenantID, id, actorAuth0UserID)
	})
}

// RevokeTx is the transactional variant of Revoke. Use when the caller
// already holds a tenant-scoped tx (typically to pair an RBAC check with
// the revoke atomically).
func (r *APITokenRepository) RevokeTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, actorAuth0UserID string) error {
	tag, err := tx.Exec(ctx,
		`UPDATE auth_svc.seller_api_tokens
		 SET revoked_at = NOW(),
		     revoked_by_auth0_user_id = $3,
		     updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL`,
		id, tenantID, actorAuth0UserID,
	)
	if err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	// RowsAffected() == 0 means either the row doesn't exist OR it was
	// already revoked. We resolve the ambiguity with an existence check so
	// callers get NotFound vs silent idempotent success.
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(
			    SELECT 1 FROM auth_svc.seller_api_tokens
			    WHERE id = $1 AND tenant_id = $2
			 )`,
			id, tenantID,
		).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return domain.ErrAPITokenNotFound
		}
		// Already revoked — idempotent success.
	}
	return nil
}

// GetByLookup retrieves a token by its (prefix, lookup) pair. This is the
// gateway hot path: called on every API-token request before the tenant
// context is known. It intentionally does NOT use TenantTx — in the
// current single-DB-role deployment the auth service owns the table and
// bypasses RLS (no FORCE ROW LEVEL SECURITY). See migration 000012 for
// the hardening note if/when roles split.
//
// Returns (nil, nil) if no row is found.
func (r *APITokenRepository) GetByLookup(ctx context.Context, prefix, lookup string) (*domain.SellerAPIToken, error) {
	t, err := scanTokenRow(r.pool.QueryRow(ctx, selectTokenColumns+`
	     FROM auth_svc.seller_api_tokens
	     WHERE token_prefix = $1 AND token_lookup = $2`, prefix, lookup))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get api token by lookup: %w", err)
	}
	return t, nil
}

// TouchLastUsedAt updates the last_used_at column for an active token.
// Called as a best-effort goroutine after a successful gateway lookup;
// errors are logged but not returned because the debounce ultimately
// lives in the gateway cache (30 s TTL).
//
// Like GetByLookup, this does not set app.current_tenant_id because it
// runs outside a tenant context and relies on table-owner RLS bypass.
func (r *APITokenRepository) TouchLastUsedAt(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE auth_svc.seller_api_tokens
		 SET last_used_at = NOW()
		 WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("touch api token last_used_at: %w", err)
	}
	return nil
}

// selectTokenColumns is shared by GetByID/ListBySeller/GetByLookup so the
// column order stays in sync with scanTokenRow.
const selectTokenColumns = `SELECT
    id, tenant_id, seller_id, name,
    token_prefix, token_lookup, token_hash,
    scopes, rate_limit_rps, rate_limit_burst,
    issued_by_auth0_user_id, expires_at, revoked_at,
    revoked_by_auth0_user_id, last_used_at,
    created_at, updated_at`

// rowScanner is the minimal interface shared by pgx.Row and pgx.Rows so
// scanTokenRow can work with both.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTokenRow(row rowScanner) (*domain.SellerAPIToken, error) {
	var t domain.SellerAPIToken
	var scopes []string
	var revokedBy *string
	if err := row.Scan(
		&t.ID, &t.TenantID, &t.SellerID, &t.Name,
		&t.TokenPrefix, &t.TokenLookup, &t.TokenHash,
		&scopes, &t.RateLimitRPS, &t.RateLimitBurst,
		&t.IssuedByAuth0UserID, &t.ExpiresAt, &t.RevokedAt,
		&revokedBy, &t.LastUsedAt,
		&t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	t.Scopes = stringsToScopes(scopes)
	if revokedBy != nil {
		t.RevokedByAuth0UserID = *revokedBy
	}
	return &t, nil
}

func scopesToStrings(scopes []domain.APITokenScope) []string {
	out := make([]string, len(scopes))
	for i, s := range scopes {
		out[i] = string(s)
	}
	return out
}

func stringsToScopes(in []string) []domain.APITokenScope {
	out := make([]domain.APITokenScope, len(in))
	for i, s := range in {
		out[i] = domain.APITokenScope(s)
	}
	return out
}
