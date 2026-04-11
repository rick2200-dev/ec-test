ALTER TABLE auth_svc.seller_users           NO FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.platform_admins        NO FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.rbac_audit_log         NO FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.subscription_plans     NO FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.seller_subscriptions   NO FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.buyer_plans            NO FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.buyer_subscriptions    NO FORCE ROW LEVEL SECURITY;

ALTER TABLE inventory_svc.inventory         NO FORCE ROW LEVEL SECURITY;
ALTER TABLE inventory_svc.stock_movements   NO FORCE ROW LEVEL SECURITY;

ALTER TABLE order_svc.order_lines           NO FORCE ROW LEVEL SECURITY;
ALTER TABLE order_svc.commission_rules      NO FORCE ROW LEVEL SECURITY;
ALTER TABLE order_svc.payouts               NO FORCE ROW LEVEL SECURITY;

ALTER TABLE inquiry_svc.inquiries           NO FORCE ROW LEVEL SECURITY;
ALTER TABLE inquiry_svc.inquiry_messages    NO FORCE ROW LEVEL SECURITY;
