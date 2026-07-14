-- 154_drop_sales_tables.sql
-- 移除销售侧子系统（支付/分销/优惠码/公告）后清理对应数据表。
-- redeem_codes 保留：余额/并发调整记录仍复用该表作为审计存储。
-- user_affiliate_ledger 先删：其 source_order_id 外键依赖 payment_orders。
DROP TABLE IF EXISTS user_affiliate_ledger;
DROP TABLE IF EXISTS user_affiliates;
DROP TABLE IF EXISTS promo_code_usages;
DROP TABLE IF EXISTS promo_codes;
DROP TABLE IF EXISTS payment_audit_logs;
DROP TABLE IF EXISTS payment_orders;
DROP TABLE IF EXISTS payment_provider_instances;
DROP TABLE IF EXISTS announcement_reads;
DROP TABLE IF EXISTS announcements;
