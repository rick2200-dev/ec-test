-- coupon_svc.coupons — the catalog of issued coupons. MVP issues
-- platform-scoped coupons only, but issuer_type/issuer_id are kept on
-- the table so a future "seller-issued coupon" phase doesn't require a
-- backfill. issuer_id is NULL for platform coupons.
CREATE TABLE coupon_svc.coupons (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                  VARCHAR(64) NOT NULL,
    issuer_type           VARCHAR(16) NOT NULL,
    issuer_id             UUID NULL,
    discount_type         VARCHAR(16) NOT NULL,
    discount_percent_bps  INT NULL,
    discount_amount       BIGINT NULL,
    currency              VARCHAR(3) NOT NULL DEFAULT 'JPY',
    min_order_amount      BIGINT NOT NULL DEFAULT 0,
    max_discount_amount   BIGINT NULL,
    valid_from            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at            TIMESTAMPTZ NULL,
    usage_limit_total     INT NULL,
    usage_limit_per_user  INT NULL,
    usage_count           INT NOT NULL DEFAULT 0,
    status                VARCHAR(16) NOT NULL DEFAULT 'active',
    title                 VARCHAR(255),
    description           TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT coupons_issuer_type_chk
        CHECK (issuer_type IN ('platform', 'seller')),
    CONSTRAINT coupons_discount_type_chk
        CHECK (discount_type IN ('percent', 'fixed_amount')),
    CONSTRAINT coupons_status_chk
        CHECK (status IN ('active', 'revoked')),
    CONSTRAINT coupons_discount_exactly_one_chk
        CHECK (
            (discount_type = 'percent'      AND discount_percent_bps IS NOT NULL AND discount_amount IS NULL)
         OR (discount_type = 'fixed_amount' AND discount_amount IS NOT NULL      AND discount_percent_bps IS NULL)
        ),
    CONSTRAINT coupons_usage_count_nonneg_chk
        CHECK (usage_count >= 0)
);

-- UNIQUE by (issuer_type, issuer_id, code) with case-insensitive code match.
-- coalesce lets platform coupons (issuer_id IS NULL) still enforce uniqueness.
CREATE UNIQUE INDEX coupons_code_unique
    ON coupon_svc.coupons (
        issuer_type,
        COALESCE(issuer_id, '00000000-0000-0000-0000-000000000000'::uuid),
        UPPER(code)
    );

CREATE INDEX coupons_status_expires_idx ON coupon_svc.coupons (status, expires_at);
CREATE INDEX coupons_issuer_id_idx       ON coupon_svc.coupons (issuer_id)
    WHERE issuer_id IS NOT NULL;
