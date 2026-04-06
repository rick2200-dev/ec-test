-- User events for recommendation engine
CREATE TABLE catalog_svc.user_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,  -- product_viewed, added_to_cart, purchased
    product_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_events_tenant ON catalog_svc.user_events(tenant_id);
CREATE INDEX idx_user_events_user ON catalog_svc.user_events(tenant_id, user_id);
CREATE INDEX idx_user_events_product ON catalog_svc.user_events(tenant_id, product_id);
CREATE INDEX idx_user_events_type ON catalog_svc.user_events(tenant_id, event_type);
CREATE INDEX idx_user_events_created ON catalog_svc.user_events(created_at);

ALTER TABLE catalog_svc.user_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON catalog_svc.user_events
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Materialized view for popular products (refreshed periodically)
CREATE MATERIALIZED VIEW catalog_svc.popular_products AS
SELECT
    p.tenant_id,
    p.id AS product_id,
    p.seller_id,
    p.name,
    p.slug,
    COUNT(DISTINCT ue.id) FILTER (WHERE ue.event_type = 'purchased') AS purchase_count,
    COUNT(DISTINCT ue.id) FILTER (WHERE ue.event_type = 'product_viewed') AS view_count,
    COUNT(DISTINCT ue.id) AS total_events
FROM catalog_svc.products p
LEFT JOIN catalog_svc.user_events ue ON p.id = ue.product_id AND p.tenant_id = ue.tenant_id
WHERE p.status = 'active'
GROUP BY p.tenant_id, p.id, p.seller_id, p.name, p.slug;

CREATE UNIQUE INDEX idx_popular_products ON catalog_svc.popular_products(tenant_id, product_id);
