ALTER TABLE config_policies
    ADD COLUMN IF NOT EXISTS last_accessed_at TIMESTAMP;

COMMENT ON COLUMN config_policies.last_accessed_at IS '用户最近一次成功请求订阅配置的时间';

CREATE TABLE IF NOT EXISTS config_access_logs (
    id BIGSERIAL PRIMARY KEY,
    policy_id INTEGER REFERENCES config_policies(id) ON DELETE CASCADE,
    token VARCHAR(64),
    client_ip INET,
    user_agent TEXT,
    status_code INTEGER NOT NULL,
    success BOOLEAN NOT NULL DEFAULT false,
    cache_hit BOOLEAN NOT NULL DEFAULT false,
    node_count INTEGER,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_config_access_logs_policy_id_created_at
    ON config_access_logs(policy_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_config_access_logs_created_at
    ON config_access_logs(created_at DESC);

COMMENT ON TABLE config_access_logs IS '订阅配置访问日志';
COMMENT ON COLUMN config_access_logs.policy_id IS '被访问的配置策略 ID，token 无效时可为空';
COMMENT ON COLUMN config_access_logs.token IS '请求中携带的 token，便于排查无效访问';
COMMENT ON COLUMN config_access_logs.client_ip IS '客户端 IP，优先取反向代理透传头';
COMMENT ON COLUMN config_access_logs.user_agent IS '客户端 User-Agent';
COMMENT ON COLUMN config_access_logs.status_code IS 'HTTP 返回状态码';
COMMENT ON COLUMN config_access_logs.success IS '是否成功生成或返回配置';
COMMENT ON COLUMN config_access_logs.cache_hit IS '是否命中配置缓存';
COMMENT ON COLUMN config_access_logs.node_count IS '本次返回的节点数，失败时可为空';
COMMENT ON COLUMN config_access_logs.error_message IS '失败原因';
