-- 允许 rule_sources.url 为 NULL（支持手动维护规则源）
ALTER TABLE rule_sources ALTER COLUMN url DROP NOT NULL;
