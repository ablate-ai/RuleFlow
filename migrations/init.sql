-- RuleFlow 完整初始化脚本
-- 适用于每次重新建库后一次性执行

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    url TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    auto_refresh BOOLEAN NOT NULL DEFAULT false,
    refresh_interval INTEGER NOT NULL DEFAULT 3600,
    description TEXT,
    tags TEXT[] DEFAULT '{}',
    last_fetched_at TIMESTAMP,
    last_fetch_error TEXT,
    node_count INTEGER DEFAULT 0,
    filter_rules JSONB DEFAULT '{}',
    userinfo JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_subscriptions_name ON subscriptions(name);
CREATE INDEX idx_subscriptions_enabled ON subscriptions(enabled) WHERE enabled = true;

CREATE TRIGGER update_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON COLUMN subscriptions.auto_refresh IS '是否自动刷新订阅';
COMMENT ON COLUMN subscriptions.refresh_interval IS '自动刷新间隔（秒）';
COMMENT ON COLUMN subscriptions.filter_rules IS '节点过滤规则：exclude_keywords（排除关键词列表）、exclude_regex（排除正则）、include_protocols（协议白名单）';
COMMENT ON COLUMN subscriptions.userinfo IS '订阅流量信息，来自响应头 Subscription-Userinfo';

CREATE TABLE templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    content TEXT NOT NULL,
    target VARCHAR(20) NOT NULL DEFAULT 'clash',
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT templates_target_check
        CHECK (target IN ('clash', 'clash_meta', 'stash', 'surge', 'loon', 'shadowrocket'))
);

CREATE INDEX idx_templates_name ON templates(name);
CREATE INDEX idx_templates_target ON templates(target);

CREATE TRIGGER update_templates_updated_at
    BEFORE UPDATE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE config_policies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    subscription_ids INTEGER[] NOT NULL DEFAULT '{}',
    node_ids INTEGER[] NOT NULL DEFAULT '{}',
    template_name VARCHAR(255),
    target VARCHAR(20) NOT NULL DEFAULT 'clash',
    node_filters JSONB DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT config_policies_target_check
        CHECK (target IN ('clash', 'stash', 'surge'))
);

CREATE INDEX idx_config_policies_name ON config_policies(name);
CREATE INDEX idx_config_policies_enabled ON config_policies(enabled) WHERE enabled = true;
CREATE INDEX idx_config_policies_tags ON config_policies USING GIN(tags);

CREATE TRIGGER update_config_policies_updated_at
    BEFORE UPDATE ON config_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE config_policies IS '配置生成策略表';
COMMENT ON COLUMN config_policies.subscription_ids IS '关联的订阅源 ID，可以从多个订阅源合并节点';
COMMENT ON COLUMN config_policies.node_ids IS '直接指定的手动节点 ID 列表';
COMMENT ON COLUMN config_policies.template_name IS '使用的规则模板，为空则使用内置模板';
COMMENT ON COLUMN config_policies.node_filters IS '节点过滤条件，JSON 格式存储复杂的过滤逻辑';

CREATE TABLE nodes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    protocol VARCHAR(20) NOT NULL,
    server VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    config JSONB NOT NULL,
    source VARCHAR(50) NOT NULL,
    source_id INTEGER,
    enabled BOOLEAN NOT NULL DEFAULT true,
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_synced_at TIMESTAMP,
    CONSTRAINT nodes_protocol_check
        CHECK (protocol IN ('trojan', 'vmess', 'vless', 'ss', 'hysteria2', 'tuic')),
    CONSTRAINT nodes_source_check
        CHECK (source ~ '^subscription:|^manual$'),
    UNIQUE(source, name, server, port)
);

CREATE INDEX idx_nodes_source ON nodes(source);
CREATE INDEX idx_nodes_source_id ON nodes(source_id) WHERE source_id IS NOT NULL;
CREATE INDEX idx_nodes_protocol ON nodes(protocol);
CREATE INDEX idx_nodes_enabled ON nodes(enabled) WHERE enabled = true;
CREATE INDEX idx_nodes_tags ON nodes USING GIN(tags);

CREATE TRIGGER update_nodes_updated_at
    BEFORE UPDATE ON nodes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE nodes IS '节点表，存储订阅和手动添加的代理节点';
COMMENT ON COLUMN nodes.protocol IS '协议类型：trojan, vmess, vless, ss, hysteria2, tuic';
COMMENT ON COLUMN nodes.config IS '协议特定配置（密码、UUID 等），JSON 格式';
COMMENT ON COLUMN nodes.source IS '节点来源：subscription:{name} 或 manual';
COMMENT ON COLUMN nodes.source_id IS '订阅 ID（用于订阅节点）';
COMMENT ON COLUMN nodes.last_synced_at IS '最后同步时间（订阅节点）';
