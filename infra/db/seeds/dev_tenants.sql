-- Seed development data
-- Tenant
INSERT INTO auth_svc.tenants (id, name, slug, status) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Demo Marketplace', 'demo', 'active')
ON CONFLICT (slug) DO NOTHING;

-- Sellers
INSERT INTO auth_svc.sellers (id, tenant_id, name, slug, status, commission_rate_bps) VALUES
    ('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Tokyo Electronics', 'tokyo-electronics', 'approved', 1000),
    ('c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Osaka Fashion', 'osaka-fashion', 'approved', 1200),
    ('d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Kyoto Crafts', 'kyoto-crafts', 'pending', 1500)
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- Sample categories
INSERT INTO catalog_svc.categories (id, tenant_id, name, slug, sort_order) VALUES
    ('e4eebc99-9c0b-4ef8-bb6d-6bb9bd380a55', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Electronics', 'electronics', 1),
    ('f5eebc99-9c0b-4ef8-bb6d-6bb9bd380a66', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Fashion', 'fashion', 2),
    ('06eebc99-9c0b-4ef8-bb6d-6bb9bd380a77', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Handmade', 'handmade', 3)
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- Sample products
INSERT INTO catalog_svc.products (id, tenant_id, seller_id, name, slug, description, status) VALUES
    ('11eebc99-9c0b-4ef8-bb6d-6bb9bd380b01', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'Wireless Headphones', 'wireless-headphones', 'High-quality wireless headphones with noise cancellation', 'active'),
    ('22eebc99-9c0b-4ef8-bb6d-6bb9bd380b02', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'Cotton T-Shirt', 'cotton-tshirt', 'Premium organic cotton t-shirt', 'active'),
    ('33eebc99-9c0b-4ef8-bb6d-6bb9bd380b03', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'USB-C Hub', 'usb-c-hub', '7-in-1 USB-C hub with 4K HDMI', 'active')
ON CONFLICT (tenant_id, seller_id, slug) DO NOTHING;

-- Sample SKUs
INSERT INTO catalog_svc.skus (id, tenant_id, product_id, seller_id, sku_code, price_amount, price_currency, attributes) VALUES
    ('44eebc99-9c0b-4ef8-bb6d-6bb9bd380c01', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '11eebc99-9c0b-4ef8-bb6d-6bb9bd380b01', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'WH-BLK-001', 12800, 'JPY', '{"color": "black"}'),
    ('55eebc99-9c0b-4ef8-bb6d-6bb9bd380c02', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '11eebc99-9c0b-4ef8-bb6d-6bb9bd380b01', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'WH-WHT-001', 12800, 'JPY', '{"color": "white"}'),
    ('66eebc99-9c0b-4ef8-bb6d-6bb9bd380c03', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '22eebc99-9c0b-4ef8-bb6d-6bb9bd380b02', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'TS-M-BLK', 3500, 'JPY', '{"size": "M", "color": "black"}'),
    ('77eebc99-9c0b-4ef8-bb6d-6bb9bd380c04', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '22eebc99-9c0b-4ef8-bb6d-6bb9bd380b02', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'TS-L-BLK', 3500, 'JPY', '{"size": "L", "color": "black"}'),
    ('88eebc99-9c0b-4ef8-bb6d-6bb9bd380c05', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '33eebc99-9c0b-4ef8-bb6d-6bb9bd380b03', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'HUB-7IN1', 5980, 'JPY', '{}')
ON CONFLICT (tenant_id, seller_id, sku_code) DO NOTHING;

-- Sample inventory
INSERT INTO inventory_svc.inventory (tenant_id, sku_id, seller_id, quantity_available, quantity_reserved) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '44eebc99-9c0b-4ef8-bb6d-6bb9bd380c01', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 50, 0),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '55eebc99-9c0b-4ef8-bb6d-6bb9bd380c02', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 30, 0),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '66eebc99-9c0b-4ef8-bb6d-6bb9bd380c03', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 100, 0),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '77eebc99-9c0b-4ef8-bb6d-6bb9bd380c04', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 80, 0),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '88eebc99-9c0b-4ef8-bb6d-6bb9bd380c05', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 200, 0)
ON CONFLICT (tenant_id, sku_id) DO NOTHING;

-- Default commission rule
INSERT INTO order_svc.commission_rules (tenant_id, rate_bps, priority) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 1000, 0)
ON CONFLICT DO NOTHING;
