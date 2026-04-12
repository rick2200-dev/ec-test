-- Product review feature.
-- Buyers who have purchased a product can leave one review per product.
-- Sellers can reply once per review.
-- Aggregate ratings are maintained in a denormalized table.
CREATE SCHEMA IF NOT EXISTS review_svc;

-- reviews: one review per (tenant, buyer, product)
CREATE TABLE review_svc.reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    buyer_auth0_id  VARCHAR(255) NOT NULL,
    product_id      UUID NOT NULL,
    seller_id       UUID NOT NULL,
    product_name    VARCHAR(500) NOT NULL,
    rating          SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title           VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 4000),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_review_per_product UNIQUE (tenant_id, buyer_auth0_id, product_id)
);

-- review_replies: one seller reply per review
CREATE TABLE review_svc.review_replies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    review_id       UUID NOT NULL REFERENCES review_svc.reviews(id) ON DELETE CASCADE,
    seller_auth0_id VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reply_per_review UNIQUE (tenant_id, review_id)
);

-- product_ratings: materialized aggregate per (tenant, product)
CREATE TABLE review_svc.product_ratings (
    tenant_id       UUID NOT NULL,
    product_id      UUID NOT NULL,
    average_rating  NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count    INTEGER NOT NULL DEFAULT 0,
    rating_sum      INTEGER NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, product_id)
);

-- Indexes for reviews
CREATE INDEX idx_reviews_tenant
    ON review_svc.reviews(tenant_id);
CREATE INDEX idx_reviews_product
    ON review_svc.reviews(tenant_id, product_id, created_at DESC);
CREATE INDEX idx_reviews_buyer
    ON review_svc.reviews(tenant_id, buyer_auth0_id, created_at DESC);
CREATE INDEX idx_reviews_seller
    ON review_svc.reviews(tenant_id, seller_id, created_at DESC);

-- Indexes for review_replies
CREATE INDEX idx_review_replies_review
    ON review_svc.review_replies(tenant_id, review_id);

-- Indexes for product_ratings
CREATE INDEX idx_product_ratings_tenant
    ON review_svc.product_ratings(tenant_id);

-- RLS
ALTER TABLE review_svc.reviews           ENABLE ROW LEVEL SECURITY;
ALTER TABLE review_svc.review_replies    ENABLE ROW LEVEL SECURITY;
ALTER TABLE review_svc.product_ratings   ENABLE ROW LEVEL SECURITY;

ALTER TABLE review_svc.reviews           FORCE ROW LEVEL SECURITY;
ALTER TABLE review_svc.review_replies    FORCE ROW LEVEL SECURITY;
ALTER TABLE review_svc.product_ratings   FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON review_svc.reviews
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON review_svc.review_replies
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON review_svc.product_ratings
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
