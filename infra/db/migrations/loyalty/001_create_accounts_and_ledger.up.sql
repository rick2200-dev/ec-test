-- loyalty_svc.point_accounts: one row per buyer, created lazily on the
-- first earn or reservation. balance and pending_redemption are kept
-- consistent with the append-only ledger. version is incremented on
-- every write for optimistic locking (two concurrent ReservePoints /
-- Commits on the same account).
CREATE TABLE loyalty_svc.point_accounts (
    buyer_auth0_id      VARCHAR(255) PRIMARY KEY,
    balance             BIGINT NOT NULL DEFAULT 0,
    pending_redemption  BIGINT NOT NULL DEFAULT 0,
    lifetime_earned     BIGINT NOT NULL DEFAULT 0,
    lifetime_redeemed   BIGINT NOT NULL DEFAULT 0,
    version             INT    NOT NULL DEFAULT 0,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT point_accounts_balance_nonneg_chk
        CHECK (balance >= 0),
    CONSTRAINT point_accounts_pending_nonneg_chk
        CHECK (pending_redemption >= 0),
    CONSTRAINT point_accounts_pending_le_balance_chk
        CHECK (pending_redemption <= balance)
);

-- point_reservations: pending state for a buyer's "use N points" at
-- checkout, before the order is paid. Reaped on expiry or released on
-- checkout failure.
CREATE TABLE loyalty_svc.point_reservations (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_auth0_id           VARCHAR(255) NOT NULL,
    amount                   BIGINT NOT NULL,
    order_candidate_id       UUID NULL,
    stripe_payment_intent_id VARCHAR(255) NULL,
    status                   VARCHAR(16) NOT NULL DEFAULT 'pending',
    expires_at               TIMESTAMPTZ NOT NULL,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    committed_at             TIMESTAMPTZ NULL,
    released_at              TIMESTAMPTZ NULL,

    CONSTRAINT point_reservations_status_chk
        CHECK (status IN ('pending', 'committed', 'released', 'expired')),
    CONSTRAINT point_reservations_amount_pos_chk
        CHECK (amount > 0)
);

CREATE INDEX point_reservations_reaper_idx
    ON loyalty_svc.point_reservations (status, expires_at)
    WHERE status = 'pending';
CREATE INDEX point_reservations_buyer_idx
    ON loyalty_svc.point_reservations (buyer_auth0_id, status);

-- point_transactions: append-only double-entry ledger. balance_after is
-- captured at write time so history queries don't need to replay
-- every row. The (source_type, source_id, type) UNIQUE is the
-- idempotency key that makes Stripe webhook replay safe.
CREATE TABLE loyalty_svc.point_transactions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_auth0_id   VARCHAR(255) NOT NULL,
    type             VARCHAR(16) NOT NULL,
    amount           BIGINT NOT NULL,
    balance_after    BIGINT NOT NULL,
    source_type      VARCHAR(32) NOT NULL,
    source_id        VARCHAR(255) NOT NULL,
    reservation_id   UUID NULL REFERENCES loyalty_svc.point_reservations(id),
    expires_at       TIMESTAMPTZ NULL,
    note             TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT point_transactions_type_chk
        CHECK (type IN ('earn', 'redeem', 'refund', 'reverse_earn', 'adjust', 'expire')),
    CONSTRAINT point_transactions_idem_unique
        UNIQUE (source_type, source_id, type)
);

CREATE INDEX point_transactions_buyer_time_idx
    ON loyalty_svc.point_transactions (buyer_auth0_id, created_at DESC);
CREATE INDEX point_transactions_reservation_idx
    ON loyalty_svc.point_transactions (reservation_id)
    WHERE reservation_id IS NOT NULL;
