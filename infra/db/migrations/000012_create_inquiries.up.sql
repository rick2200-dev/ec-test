-- Buyer-to-seller inquiry (お問い合わせ) feature.
-- Buyers can open a thread against a seller for a specific SKU they have
-- already purchased. One thread per (buyer, seller, sku).
CREATE SCHEMA IF NOT EXISTS inquiry_svc;

CREATE TABLE inquiry_svc.inquiries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_auth0_id  VARCHAR(255) NOT NULL,
    seller_id       UUID NOT NULL,
    sku_id          UUID NOT NULL,
    product_name    VARCHAR(500) NOT NULL,
    sku_code        VARCHAR(100) NOT NULL,
    subject         VARCHAR(255) NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'open'
        CHECK (status IN ('open','closed')),
    last_message_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_inquiry_per_sku UNIQUE (buyer_auth0_id, seller_id, sku_id)
);

CREATE TABLE inquiry_svc.inquiry_messages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inquiry_id   UUID NOT NULL REFERENCES inquiry_svc.inquiries(id) ON DELETE CASCADE,
    sender_type  VARCHAR(10) NOT NULL
        CHECK (sender_type IN ('buyer','seller')),
    sender_id    VARCHAR(255) NOT NULL,
    body         TEXT NOT NULL
        CHECK (char_length(body) BETWEEN 1 AND 4000),
    read_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inquiries_buyer
    ON inquiry_svc.inquiries(buyer_auth0_id, last_message_at DESC);
CREATE INDEX idx_inquiries_seller
    ON inquiry_svc.inquiries(seller_id, last_message_at DESC);
CREATE INDEX idx_inquiry_messages_thread
    ON inquiry_svc.inquiry_messages(inquiry_id, created_at ASC);
