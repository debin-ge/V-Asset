export interface ProxySourceStatus {
  healthy: boolean;
  mode: string;
  message: string;
  proxy_url?: string;
  proxy_lease_id?: string;
  proxy_expire_at?: string;
  checked_at: string;
  available_manual_proxy_count: number;
  dynamic_configured: boolean;
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
  cooldown_until?: string;
  consecutive_fail_count: number;
  risk_score: number;
  last_error_category?: string;
  last_fail_at?: string;
  max_concurrent: number;
  active_task_count: number;
}

export type ProxyListSortBy = "risk_score" | "priority" | "fail_count" | "active_task_count" | "updated_at" | "last_used_at";

export interface ProxyListResponse {
  items: ProxyInfo[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
}

export interface ProxyListParams {
  search?: string;
  protocol?: string;
  region?: string;
  status?: number;
  page?: number;
  page_size?: number;
  sort_by?: ProxyListSortBy;
  sort_order?: "asc" | "desc";
}

export interface ProxyUsageEvent {
  id: number;
  task_id: string;
  proxy_id?: number;
  proxy_lease_id?: string;
  source_type: string;
  stage: string;
  platform?: string;
  success: boolean;
  error_category?: string;
  error_message?: string;
  created_at: string;
  proxy_host?: string;
  proxy_port?: number;
  proxy_protocol?: string;
  proxy_region?: string;
  proxy_risk_score: number;
  proxy_cooldown_until?: string;
  proxy_active_task_count: number;
  proxy_max_concurrent: number;
}

export interface ProxyUsageEventCount {
  key: string;
  count: number;
}

export interface ProxyUsageEventSummary {
  success_count: number;
  failure_count: number;
  failure_rate: number;
  category_counts: ProxyUsageEventCount[];
  stage_counts: ProxyUsageEventCount[];
  platform_counts: ProxyUsageEventCount[];
}

export interface ListProxyUsageEventsParams {
  task_id?: string;
  proxy_id?: number;
  proxy_lease_id?: string;
  source_type?: string;
  stage?: string;
  platform?: string;
  success?: "all" | "success" | "failed";
  error_category?: string;
  start_time?: string;
  end_time?: string;
  page?: number;
  page_size?: number;
  sort_order?: "asc" | "desc";
}

export interface ListProxyUsageEventsResponse {
  events: ProxyUsageEvent[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
  summary: ProxyUsageEventSummary;
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
