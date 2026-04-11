-- Seller API access tokens: opaque bearer tokens that let sellers call the
-- public /api/v1/seller/* surface from external systems (ERPs, scripts,
-- mobile backends, etc). Issued by owners via the seller dashboard.
--
-- Wire format of the plaintext token (never stored):
--     <token_prefix><token_lookup>_<secret>
-- where:
--     token_prefix   — env-configured (default "sk_live_"); rotatable
--     token_lookup   — 12-char base62 (9 random bytes); unique-indexed to
--                      give the gateway an O(1) DB probe per request
--     secret         — ~43-char base62 (32 random bytes); SHA-256'd on the
--                      server and compared in constant time
--
-- NOTE on RLS: the gateway hot-path `GetByLookup` resolves the tenant_id
-- *from* this row, so it cannot set `app.current_tenant_id` before the
-- lookup. In the current single-role deployment (service and migrations
-- both run as `ecmarket`) the auth service owns the table and therefore
-- bypasses RLS policies. Migration 000015 deliberately does NOT apply
-- FORCE ROW LEVEL SECURITY to this table for the same reason. If the
-- deployment ever moves to separate DB roles, `GetByLookup` must be
-- reimplemented as a `SECURITY DEFINER` SQL function owned by a role that
-- can bypass RLS for this one query.
CREATE TABLE auth_svc.seller_api_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES auth_svc.tenants(id) ON DELETE CASCADE,
    seller_id       UUID NOT NULL REFERENCES auth_svc.sellers(id) ON DELETE CASCADE,

    -- Human-readable label chosen by the issuer, e.g. "ERP sync (staging)".
    name            VARCHAR(120) NOT NULL,

    token_prefix    VARCHAR(16) NOT NULL,
    token_lookup    VARCHAR(16) NOT NULL,
    token_hash      BYTEA       NOT NULL, -- 32-byte SHA-256 of the secret

    -- Scopes use TEXT[] (not JSONB) so the CHECK can enforce the closed
    -- vocabulary with a single `<@` operator, and so GIN indexing by
    -- scope is available cheaply if we ever need it.
    scopes          TEXT[] NOT NULL CHECK (cardinality(scopes) > 0),

    -- Per-token rate-limit overrides; NULL → use gateway default.
    rate_limit_rps   INT,
    rate_limit_burst INT,

    issued_by_auth0_user_id  VARCHAR(255) NOT NULL,

    expires_at               TIMESTAMPTZ,
    revoked_at               TIMESTAMPTZ,
    revoked_by_auth0_user_id VARCHAR(255),
    last_used_at             TIMESTAMPTZ,

    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT seller_api_tokens_scopes_valid CHECK (
        scopes <@ ARRAY[
            'products:read','products:write',
            'orders:read','orders:write',
            'inventory:read','inventory:write'
        ]::text[]
    )
);

-- Hot-path lookup: one B-tree probe, then constant-time hash compare.
CREATE UNIQUE INDEX idx_seller_api_tokens_lookup
    ON auth_svc.seller_api_tokens(token_prefix, token_lookup);

-- Listing for the dashboard (newest first).
CREATE INDEX idx_seller_api_tokens_seller
    ON auth_svc.seller_api_tokens(tenant_id, seller_id, created_at DESC);

ALTER TABLE auth_svc.seller_api_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON auth_svc.seller_api_tokens
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
