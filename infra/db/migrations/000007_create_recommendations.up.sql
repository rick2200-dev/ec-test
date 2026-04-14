-- User events for recommendation engine
CREATE TABLE catalog_svc.user_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,  -- product_viewed, added_to_cart, purchased
    product_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_events_user ON catalog_svc.user_events(user_id);
CREATE INDEX idx_user_events_product ON catalog_svc.user_events(product_id);
CREATE INDEX idx_user_events_type ON catalog_svc.user_events(event_type);
CREATE INDEX idx_user_events_created ON catalog_svc.user_events(created_at);

-- Materialized view for popular products (refreshed periodically)
CREATE MATERIALIZED VIEW catalog_svc.popular_products AS
SELECT
    p.id AS product_id,
    p.seller_id,
    p.name,
    p.slug,
    COUNT(DISTINCT ue.id) FILTER (WHERE ue.event_type = 'purchased') AS purchase_count,
    COUNT(DISTINCT ue.id) FILTER (WHERE ue.event_type = 'product_viewed') AS view_count,
    COUNT(DISTINCT ue.id) AS total_events
FROM catalog_svc.products p
LEFT JOIN catalog_svc.user_events ue ON p.id = ue.product_id
WHERE p.status = 'active'
GROUP BY p.id, p.seller_id, p.name, p.slug;

CREATE UNIQUE INDEX idx_popular_products ON catalog_svc.popular_products(product_id);
