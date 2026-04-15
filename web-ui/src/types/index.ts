export interface Subscription {
  id: number;
  name: string;
  url: string;
  enabled: boolean;
  auto_refresh: boolean;
  refresh_interval: number;
  description: string;
  tags: string[];
  last_fetched_at: string | null;
  last_fetch_error: string | null;
  node_count: number;
  filter_rules: {
    exclude_keywords: string[];
    exclude_regex: string;
    include_protocols: string[];
  };
  disable_name_prefix: boolean;
  userinfo: {
    upload: number;
    download: number;
    total: number;
    expire: number | null;
  } | null;
  created_at: string;
  updated_at: string;
}

export interface Template {
  id: number;
  name: string;
  description: string;
  content: string;
  target: string;
  tags: string[];
  is_public: boolean;
  created_at: string;
  updated_at: string;
}

export interface Node {
  id: number;
  name: string;
  protocol: string;
  server: string;
  port: number;
  config: Record<string, unknown>;
  source_id: number | null;
  source_name: string;
  enabled: boolean;
  tags: string[];
  created_at: string;
  updated_at: string;
  last_synced_at: string | null;
}

export interface NodeStats {
  total: number;
  enabled: number;
  disabled: number;
  by_protocol: Record<string, number>;
  by_source: Record<string, number>;
}

export interface ConfigPolicy {
  id: number;
  name: string;
  token: string;
  description: string;
  subscription_ids: number[];
  node_ids: number[];
  template_name: string;
  target: string;
  node_filters: Record<string, unknown>;
  enabled: boolean;
  tags: string[];
  last_accessed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface RuleSource {
  id: number;
  name: string;
  description: string;
  url: string;
  source_format: string;
  enabled: boolean;
  auto_refresh: boolean;
  refresh_interval: number;
  tags: string[];
  raw_content: string | null;
  parsed_rules: unknown;
  rule_count: number;
  last_synced_at: string | null;
  last_sync_error: string | null;
  created_at: string;
  updated_at: string;
}

export interface ConfigAccessLog {
  id: number;
  policy_id: number | null;
  token: string;
  client_ip: string | null;
  user_agent: string;
  status_code: number;
  success: boolean;
  cache_hit: boolean;
  node_count: number | null;
  error_message: string;
  created_at: string;
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

export interface ImportResult {
  subscriptions: { created: number; updated: number; skipped: number; errors: string[] };
  manual_nodes: { created: number; updated: number; skipped: number; errors: string[] };
  templates: { created: number; updated: number; skipped: number; errors: string[] };
  config_policies: { created: number; updated: number; skipped: number; errors: string[] };
  rule_sources: { created: number; updated: number; skipped: number; errors: string[] };
}

export interface SqlResult {
  type: "select" | "exec";
  columns: string[];
  rows: Record<string, unknown>[];
  rows_affected: number;
}
