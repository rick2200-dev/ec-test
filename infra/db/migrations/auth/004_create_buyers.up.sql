-- Buyers: minimal profile record created on first Auth0 login via the
-- POST /internal/buyers/upsert endpoint. Downstream services still key off
-- the Auth0 `sub` claim directly; this table exists so profile data
-- (email / display name) is queryable without hitting Auth0 every time.
CREATE TABLE auth_svc.buyers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth0_sub VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_buyers_auth0_sub ON auth_svc.buyers(auth0_sub);
