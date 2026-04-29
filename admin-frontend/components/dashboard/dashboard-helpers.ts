import type { DashboardHealthResponse, TrendPoint } from "@/types/stats";

export type DashboardSeverity = "healthy" | "warning" | "critical" | "neutral";
export type TimeRange = "24h" | "7d" | "30d";

export function trendRequestForRange(timeRange: TimeRange): { granularity: "day" | "hour"; limit: number } {
  if (timeRange === "24h") {
    return { granularity: "hour", limit: 24 };
  }
  return { granularity: "day", limit: timeRange === "7d" ? 7 : 30 };
}

export function formatNumber(value?: number | null) {
  if (value === null || value === undefined) {
    return "-";
  }
  return new Intl.NumberFormat("en-US").format(value);
}

export function formatPercent(value: number | null) {
  if (value === null || Number.isNaN(value)) {
    return "N/A";
  }
  return `${Math.round(value * 100)}%`;
}

export function ratio(numerator?: number, denominator?: number) {
  if (!denominator || denominator <= 0 || numerator === undefined) {
    return null;
  }
  return numerator / denominator;
}

export function successRateStatus(value: number | null): DashboardSeverity {
  if (value === null) {
    return "neutral";
  }
  if (value >= 0.95) {
    return "healthy";
  }
  if (value >= 0.9) {
    return "warning";
  }
  return "critical";
}

export function proxyPoolStatus(health: DashboardHealthResponse | null): DashboardSeverity {
  if (!health) {
    return "neutral";
  }
  if (health.proxies.total <= 0) {
    return "critical";
  }
  if (health.proxies.available <= 0 || !health.proxy_source.healthy) {
    return "warning";
  }
  return "healthy";
}

export function failureStatus(health: DashboardHealthResponse | null): DashboardSeverity {
  if (!health) {
    return "neutral";
  }
  if (health.downloads.failed_total <= 0) {
    return "healthy";
  }
  return successRateStatus(health.downloads.success_rate) === "critical" ? "critical" : "warning";
}

export function statusLabel(status: DashboardSeverity) {
  switch (status) {
    case "healthy":
      return "Healthy";
    case "warning":
      return "Warning";
    case "critical":
      return "Critical";
    default:
      return "Check";
  }
}

export function trendTotal(point: TrendPoint) {
  return point.total_count ?? point.count ?? 0;
}

export function trendSuccess(point: TrendPoint) {
  return point.success_count ?? Math.max(trendTotal(point) - trendFailed(point), 0);
}

export function trendFailed(point: TrendPoint) {
  return point.failed_count ?? 0;
}

export function trendSuccessRate(point: TrendPoint) {
  if (point.success_rate !== undefined) {
    return point.success_rate;
  }
  return ratio(trendSuccess(point), trendTotal(point)) ?? 0;
}
