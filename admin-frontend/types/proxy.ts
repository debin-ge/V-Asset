export interface ProxySourceStatus {
  healthy: boolean;
  mode: string;
  message: string;
  proxy_url?: string;
  proxy_lease_id?: string;
  proxy_expire_at?: string;
  checked_at: string;
}

export interface ProxySourcePolicy {
  id: number;
  scope_type: string;
  scope_value?: string;
  primary_source: string;
  fallback_source?: string;
  fallback_enabled: boolean;
  dynamic_timeout_ms: number;
  dynamic_retry_count: number;
  dynamic_circuit_breaker_sec: number;
  min_lease_ttl_sec: number;
  manual_selection_strategy: string;
}

export interface UpdateProxySourcePolicyPayload {
  primary_source: string;
  fallback_source: string;
  fallback_enabled: boolean;
  dynamic_timeout_ms: number;
  dynamic_retry_count: number;
  dynamic_circuit_breaker_sec: number;
  min_lease_ttl_sec: number;
  manual_selection_strategy: string;
}

export interface ProxyInfo {
  id: number;
  host: string;
  port: number;
  protocol: string;
  username?: string;
  region?: string;
  priority: number;
  platform_tags?: string;
  remark?: string;
  status: number;
  last_used_at?: string;
  success_count: number;
  fail_count: number;
  created_at: string;
  updated_at: string;
}

export interface ProxyListResponse {
  items: ProxyInfo[];
}

export interface ProxyListParams {
  search?: string;
  protocol?: string;
  region?: string;
  status?: number;
}

export interface ProxyCreatePayload {
  host: string;
  port: number;
  protocol: string;
  username?: string;
  password?: string;
  region?: string;
  priority: number;
  platform_tags?: string;
  remark?: string;
  status: number;
}

export interface ProxyUpdatePayload {
  host: string;
  port: number;
  protocol: string;
  username?: string;
  password?: string;
  region?: string;
  priority: number;
  platform_tags?: string;
  remark?: string;
}
