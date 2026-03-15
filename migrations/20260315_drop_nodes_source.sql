BEGIN;

DROP INDEX IF EXISTS idx_nodes_source;

ALTER TABLE nodes
    DROP CONSTRAINT IF EXISTS nodes_source_check;

ALTER TABLE nodes
    DROP CONSTRAINT IF EXISTS nodes_source_name_server_port_key;

DROP INDEX IF EXISTS uq_nodes_manual_identity;
DROP INDEX IF EXISTS uq_nodes_subscription_identity;

ALTER TABLE nodes
    DROP COLUMN IF EXISTS source;

CREATE UNIQUE INDEX uq_nodes_manual_identity
    ON nodes(name, server, port)
    WHERE source_id IS NULL;

CREATE UNIQUE INDEX uq_nodes_subscription_identity
    ON nodes(source_id, name, server, port)
    WHERE source_id IS NOT NULL;

COMMENT ON COLUMN nodes.source_id IS '订阅 ID；为空表示手动添加节点';

COMMIT;
