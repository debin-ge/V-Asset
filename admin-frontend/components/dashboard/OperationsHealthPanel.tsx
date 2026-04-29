import { CheckCircle2, Clock3, Database, GitBranch, Server } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { DashboardStatusBadge } from "@/components/dashboard/DashboardStatusBadge";
import { formatNumber, proxyPoolStatus } from "@/components/dashboard/dashboard-helpers";
import type { DashboardSeverity } from "@/components/dashboard/dashboard-helpers";
import type { DashboardHealthResponse } from "@/types/stats";

export function OperationsHealthPanel({
  health,
  loading,
  error,
  lastRefreshAt,
}: {
  health: DashboardHealthResponse | null;
  loading: boolean;
  error: string | null;
  lastRefreshAt: Date | null;
}) {
  const rows = [
    {
      label: "Data API",
      value: loading ? "Loading" : error ? "Partial data" : "Connected",
      detail: error || "All dashboard sources responded.",
      status: loading ? "neutral" as const : error ? "warning" as const : "healthy" as const,
      icon: Database,
    },
    {
      label: "Proxy Source",
      value: health ? health.proxy_source.mode || "Configured" : "Unknown",
      detail: health?.proxy_source.message || "Waiting for proxy source status.",
      status: health ? health.proxy_source.healthy ? "healthy" as const : "critical" as const : "neutral" as const,
      icon: Server,
    },
    {
      label: "Manual Pool",
      value: `${formatNumber(health?.proxies.active)} / ${formatNumber(health?.proxies.total)} active`,
      detail: `${formatNumber(health?.proxies.available)} currently selectable.`,
      status: proxyPoolStatus(health),
      icon: CheckCircle2,
    },
    {
      label: "Proxy Strategy",
      value: health ? `${health.proxy_policy.primary_source || "unknown"} -> ${health.proxy_policy.fallback_source || "none"}` : "Unknown",
      detail: health?.proxy_policy.fallback_enabled ? "Fallback enabled." : "Fallback disabled or unavailable.",
      status: health ? "healthy" as const : "neutral" as const,
      icon: GitBranch,
    },
    {
      label: "Last Refresh",
      value: lastRefreshAt ? lastRefreshAt.toLocaleTimeString() : "Not refreshed",
      detail: lastRefreshAt ? lastRefreshAt.toLocaleDateString() : "Dashboard has not loaded yet.",
      status: lastRefreshAt ? "healthy" as const : "neutral" as const,
      icon: Clock3,
    },
  ];

  return (
    <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardHeader>
        <CardTitle>Operations Health</CardTitle>
        <CardDescription>关键服务、代理来源和刷新状态。</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {rows.map((row) => (
          <HealthRow key={row.label} {...row} />
        ))}
      </CardContent>
    </Card>
  );
}

function HealthRow({
  label,
  value,
  detail,
  status,
  icon: Icon,
}: {
  label: string;
  value: string;
  detail: string;
  status: DashboardSeverity;
  icon: LucideIcon;
}) {
  return (
    <div className="grid gap-3 rounded-lg border border-border/70 bg-muted/30 p-3 sm:grid-cols-[auto_1fr_auto] sm:items-center">
      <div className="flex size-8 items-center justify-center rounded-lg border border-border/70 bg-background text-muted-foreground">
        <Icon className="size-4" />
      </div>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-sm font-medium text-foreground">{label}</p>
          <p className="truncate text-sm text-muted-foreground">{value}</p>
        </div>
        <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">{detail}</p>
      </div>
      <DashboardStatusBadge status={status} />
    </div>
  );
}
