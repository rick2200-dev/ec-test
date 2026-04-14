-- Dev data seed (single-tenant deployment)

-- Sellers
INSERT INTO auth_svc.sellers (id, name, slug, status, commission_rate_bps) VALUES
    ('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'Tokyo Electronics', 'tokyo-electronics', 'approved', 1000),
    ('c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'Osaka Fashion', 'osaka-fashion', 'approved', 1200),
    ('d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44', 'Kyoto Crafts', 'kyoto-crafts', 'pending', 1500)
ON CONFLICT (slug) DO NOTHING;

-- Sample categories
INSERT INTO catalog_svc.categories (id, name, slug, sort_order) VALUES
    ('e4eebc99-9c0b-4ef8-bb6d-6bb9bd380a55', 'Electronics', 'electronics', 1),
    ('f5eebc99-9c0b-4ef8-bb6d-6bb9bd380a66', 'Fashion', 'fashion', 2),
    ('06eebc99-9c0b-4ef8-bb6d-6bb9bd380a77', 'Handmade', 'handmade', 3)
ON CONFLICT (slug) DO NOTHING;

-- Sample products
INSERT INTO catalog_svc.products (id, seller_id, name, slug, description, status) VALUES
    ('11eebc99-9c0b-4ef8-bb6d-6bb9bd380b01', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'Wireless Headphones', 'wireless-headphones', 'High-quality wireless headphones with noise cancellation', 'active'),
    ('22eebc99-9c0b-4ef8-bb6d-6bb9bd380b02', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'Cotton T-Shirt', 'cotton-tshirt', 'Premium organic cotton t-shirt', 'active'),
    ('33eebc99-9c0b-4ef8-bb6d-6bb9bd380b03', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'USB-C Hub', 'usb-c-hub', '7-in-1 USB-C hub with 4K HDMI', 'active')
ON CONFLICT (seller_id, slug) DO NOTHING;

-- Sample SKUs
INSERT INTO catalog_svc.skus (id, product_id, seller_id, sku_code, price_amount, price_currency, attributes) VALUES
    ('44eebc99-9c0b-4ef8-bb6d-6bb9bd380c01', '11eebc99-9c0b-4ef8-bb6d-6bb9bd380b01', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'WH-BLK-001', 12800, 'JPY', '{"color": "black"}'),
    ('55eebc99-9c0b-4ef8-bb6d-6bb9bd380c02', '11eebc99-9c0b-4ef8-bb6d-6bb9bd380b01', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'WH-WHT-001', 12800, 'JPY', '{"color": "white"}'),
    ('66eebc99-9c0b-4ef8-bb6d-6bb9bd380c03', '22eebc99-9c0b-4ef8-bb6d-6bb9bd380b02', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'TS-M-BLK', 3500, 'JPY', '{"size": "M", "color": "black"}'),
    ('77eebc99-9c0b-4ef8-bb6d-6bb9bd380c04', '22eebc99-9c0b-4ef8-bb6d-6bb9bd380b02', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'TS-L-BLK', 3500, 'JPY', '{"size": "L", "color": "black"}'),
    ('88eebc99-9c0b-4ef8-bb6d-6bb9bd380c05', '33eebc99-9c0b-4ef8-bb6d-6bb9bd380b03', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'HUB-7IN1', 5980, 'JPY', '{}')
ON CONFLICT (seller_id, sku_code) DO NOTHING;

-- Sample inventory
INSERT INTO inventory_svc.inventory (sku_id, seller_id, quantity_available, quantity_reserved) VALUES
    ('44eebc99-9c0b-4ef8-bb6d-6bb9bd380c01', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 50, 0),
    ('55eebc99-9c0b-4ef8-bb6d-6bb9bd380c02', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 30, 0),
    ('66eebc99-9c0b-4ef8-bb6d-6bb9bd380c03', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 100, 0),
    ('77eebc99-9c0b-4ef8-bb6d-6bb9bd380c04', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 80, 0),
    ('88eebc99-9c0b-4ef8-bb6d-6bb9bd380c05', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 200, 0)
ON CONFLICT (sku_id) DO NOTHING;

-- Default commission rule
INSERT INTO order_svc.commission_rules (rate_bps, priority) VALUES
    (1000, 0)
ON CONFLICT DO NOTHING;
