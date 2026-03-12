-- 将现有数据库的自增 ID 列改为应用侧生成的雪花 ID（BIGINT）

ALTER TABLE config_access_logs DROP CONSTRAINT IF EXISTS config_access_logs_policy_id_fkey;

ALTER TABLE subscriptions
    ALTER COLUMN id TYPE BIGINT;

ALTER TABLE templates
    ALTER COLUMN id TYPE BIGINT;

ALTER TABLE rule_sources
    ALTER COLUMN id TYPE BIGINT;

ALTER TABLE config_policies
    ALTER COLUMN id TYPE BIGINT,
    ALTER COLUMN subscription_ids TYPE BIGINT[] USING subscription_ids::BIGINT[],
    ALTER COLUMN node_ids TYPE BIGINT[] USING node_ids::BIGINT[];

ALTER TABLE nodes
    ALTER COLUMN id TYPE BIGINT,
    ALTER COLUMN source_id TYPE BIGINT;

ALTER TABLE config_access_logs
    ALTER COLUMN id TYPE BIGINT,
    ALTER COLUMN policy_id TYPE BIGINT;

ALTER TABLE config_access_logs
    ADD CONSTRAINT config_access_logs_policy_id_fkey
    FOREIGN KEY (policy_id) REFERENCES config_policies(id) ON DELETE CASCADE;

DROP SEQUENCE IF EXISTS subscriptions_id_seq CASCADE;
DROP SEQUENCE IF EXISTS templates_id_seq CASCADE;
DROP SEQUENCE IF EXISTS rule_sources_id_seq CASCADE;
DROP SEQUENCE IF EXISTS config_policies_id_seq CASCADE;
DROP SEQUENCE IF EXISTS config_access_logs_id_seq CASCADE;
DROP SEQUENCE IF EXISTS nodes_id_seq CASCADE;
