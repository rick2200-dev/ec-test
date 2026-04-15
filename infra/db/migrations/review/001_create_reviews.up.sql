-- Product review feature.
-- Buyers who have purchased a product can leave one review per product.
-- Sellers can reply once per review.
-- Aggregate ratings are maintained in a denormalized table.
CREATE SCHEMA IF NOT EXISTS review_svc;

-- reviews: one review per (buyer, product)
CREATE TABLE review_svc.reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_auth0_id  VARCHAR(255) NOT NULL,
    product_id      UUID NOT NULL,
    seller_id       UUID NOT NULL,
    product_name    VARCHAR(500) NOT NULL,
    rating          SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title           VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 4000),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_review_per_product UNIQUE (buyer_auth0_id, product_id)
);

-- review_replies: one seller reply per review
CREATE TABLE review_svc.review_replies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id       UUID NOT NULL REFERENCES review_svc.reviews(id) ON DELETE CASCADE,
    seller_auth0_id VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reply_per_review UNIQUE (review_id)
);

-- product_ratings: materialized aggregate per product
CREATE TABLE review_svc.product_ratings (
    product_id      UUID NOT NULL PRIMARY KEY,
    average_rating  NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count    INTEGER NOT NULL DEFAULT 0,
    rating_sum      INTEGER NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for reviews
CREATE INDEX idx_reviews_product
    ON review_svc.reviews(product_id, created_at DESC);
CREATE INDEX idx_reviews_buyer
    ON review_svc.reviews(buyer_auth0_id, created_at DESC);
CREATE INDEX idx_reviews_seller
    ON review_svc.reviews(seller_id, created_at DESC);

-- Indexes for review_replies
CREATE INDEX idx_review_replies_review
    ON review_svc.review_replies(review_id);
