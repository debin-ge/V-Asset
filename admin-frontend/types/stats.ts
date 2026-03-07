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
