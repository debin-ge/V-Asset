import { Activity, AlertTriangle, Download, Server, Users } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { DashboardStatusBadge } from "@/components/dashboard/DashboardStatusBadge";
import {
  failureStatus,
  formatNumber,
  formatPercent,
  proxyPoolStatus,
  successRateStatus,
} from "@/components/dashboard/dashboard-helpers";
import type { DashboardHealthResponse } from "@/types/stats";

export function DashboardKpiStrip({
  health,
  loading,
}: {
  health: DashboardHealthResponse | null;
  loading: boolean;
}) {
  const successRate = health && health.downloads.total > 0 ? health.downloads.success_rate : null;
  const proxyStatusTone = proxyPoolStatus(health);

  const items = [
    {
      label: "Downloads Today",
      value: formatNumber(health?.downloads.today_total),
      detail: `${formatNumber(health?.downloads.total)} total downloads`,
      status: health ? "healthy" as const : "neutral" as const,
      icon: Download,
    },
    {
      label: "Success Rate",
      value: formatPercent(successRate),
      detail: `${formatNumber(health?.downloads.success_total)} successful`,
      status: successRateStatus(successRate),
      icon: Activity,
    },
    {
      label: "Active Users",
      value: formatNumber(health?.users.daily_active),
      detail: `${formatNumber(health?.users.weekly_active)} weekly active`,
      status: health ? "healthy" as const : "neutral" as const,
      icon: Users,
    },
    {
      label: "Proxy Pool",
      value: health ? `${formatNumber(health.proxies.active)} / ${formatNumber(health.proxies.total)}` : "-",
      detail: `${formatNumber(health?.proxies.available)} currently available`,
      status: proxyStatusTone,
      icon: Server,
    },
    {
      label: "Failures",
      value: formatNumber(health?.downloads.failed_total),
      detail: `${formatPercent(health ? health.downloads.failure_rate : null)} failure rate`,
      status: failureStatus(health),
      icon: AlertTriangle,
    },
  ];

  return (
    <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
      {items.map((item) => (
        <KpiCard
          key={item.label}
          label={item.label}
          value={loading && !health ? "Loading" : item.value}
          detail={item.detail}
          status={item.status}
          icon={item.icon}
        />
      ))}
    </section>
  );
}

function KpiCard({
  label,
  value,
  detail,
  status,
  icon: Icon,
}: {
  label: string;
  value: string;
  detail: string;
  status: "healthy" | "warning" | "critical" | "neutral";
  icon: LucideIcon;
}) {
  return (
    <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardContent className="flex flex-col gap-3 p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="truncate text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">{label}</p>
            <p className="mt-1 truncate text-2xl font-semibold text-foreground">{value}</p>
          </div>
          <div className="flex size-8 shrink-0 items-center justify-center rounded-lg border border-border/70 bg-muted/45 text-muted-foreground">
            <Icon className="size-4" />
          </div>
        </div>
        <div className="flex items-center justify-between gap-2">
          <p className="min-w-0 truncate text-xs text-muted-foreground">{detail}</p>
          <DashboardStatusBadge status={status} />
        </div>
      </CardContent>
    </Card>
  );
}
