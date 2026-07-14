-- 154_drop_sales_tables.sql
-- 移除销售侧子系统（支付/优惠码/公告）后清理对应数据表。
-- redeem_codes 保留：余额/并发调整记录仍复用该表作为审计存储。
DROP TABLE IF EXISTS promo_code_usages;
DROP TABLE IF EXISTS promo_codes;
DROP TABLE IF EXISTS payment_audit_logs;
DROP TABLE IF EXISTS payment_orders;
DROP TABLE IF EXISTS payment_provider_instances;
DROP TABLE IF EXISTS announcement_reads;
DROP TABLE IF EXISTS announcements;
