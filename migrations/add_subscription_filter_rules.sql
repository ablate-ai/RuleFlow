-- 为订阅源添加节点过滤规则字段
-- 执行方式：psql -d <数据库名> -f migrations/add_subscription_filter_rules.sql

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS filter_rules JSONB DEFAULT '{}';

COMMENT ON COLUMN subscriptions.filter_rules IS '节点过滤规则：exclude_keywords（排除关键词列表）、exclude_regex（排除正则）、include_protocols（协议白名单）';
