export interface Overview {
  total_users: number;
  daily_active_users: number;
  weekly_active_users: number;
  total_downloads: number;
  downloads_today: number;
  success_downloads: number;
  failed_downloads: number;
  active_manual_proxies: number;
  total_manual_proxies: number;
}

export interface TrendPoint {
  label: string;
  count: number;
  total_count?: number;
  success_count?: number;
  failed_count?: number;
  success_rate?: number;
}

export interface RequestTrend {
  granularity: string;
  points: TrendPoint[];
}

export interface UserStats {
  total_users: number;
  daily_active_users: number;
  weekly_active_users: number;
}

export interface DashboardDownloads {
  total: number;
  today_total: number;
  success_total: number;
  failed_total: number;
  success_rate: number;
  failure_rate: number;
}

export interface DashboardUsers {
  total: number;
  daily_active: number;
  weekly_active: number;
  dau_wau_rate: number;
  wau_total_rate: number;
}

export interface DashboardCount {
  key: string;
  count: number;
}

export interface DashboardProxies {
  total: number;
  active: number;
  available: number;
  cooling: number;
  saturated: number;
  high_risk: number;
  recent_success: number;
  recent_failure: number;
  recent_failure_rate: number;
  top_error_categories: DashboardCount[];
}

export interface DashboardProxySource {
  healthy: boolean;
  mode: string;
  message: string;
  dynamic_configured: boolean;
  proxy_lease_id?: string;
  proxy_expire_at?: string;
}

export interface DashboardProxyPolicy {
  primary_source: string;
  fallback_source?: string;
  fallback_enabled: boolean;
}

export interface DashboardCookies {
  total: number;
  active: number;
  expired: number;
  frozen: number;
}

export interface DashboardBilling {
  shortfall_count: number;
}

export interface DashboardException {
  area: string;
  severity: "warning" | "critical";
  message: string;
  action_label: string;
  action_href?: string;
}

export interface DashboardHealthResponse {
  generated_at: string;
  downloads: DashboardDownloads;
  users: DashboardUsers;
  proxies: DashboardProxies;
  proxy_source: DashboardProxySource;
  proxy_policy: DashboardProxyPolicy;
  cookies: DashboardCookies;
  billing: DashboardBilling;
  exceptions: DashboardException[];
}
