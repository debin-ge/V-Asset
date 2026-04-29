import Link from "next/link";
import { ExternalLink, Server } from "lucide-react";

import { buttonVariants } from "@/components/ui/button";
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { DashboardStatusBadge } from "@/components/dashboard/DashboardStatusBadge";
import { formatNumber, proxyPoolStatus, ratio } from "@/components/dashboard/dashboard-helpers";
import { cn } from "@/lib/utils";
import type { DashboardHealthResponse } from "@/types/stats";

export function ProxyPoolHealthPanel({
  health,
}: {
  health: DashboardHealthResponse | null;
}) {
  const activeRatio = ratio(health?.proxies.active, health?.proxies.total);
  const availableRatio = ratio(health?.proxies.available, health?.proxies.total);
  const status = proxyPoolStatus(health);

  return (
    <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardHeader>
        <div>
          <CardTitle>Proxy Pool Health</CardTitle>
          <CardDescription>代理池容量、可用性和来源配置。</CardDescription>
        </div>
        <CardAction>
          <Link href="/proxies" className={buttonVariants({ variant: "outline", size: "sm" })}>
            <ExternalLink data-icon="inline-start" />
            Manage Proxies
          </Link>
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border/70 bg-muted/30 p-3">
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-lg border border-border/70 bg-background text-muted-foreground">
              <Server className="size-4" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">
                {formatNumber(health?.proxies.active)} active / {formatNumber(health?.proxies.total)} total
              </p>
              <p className="text-xs text-muted-foreground">
                {formatNumber(health?.proxies.available)} available manual proxies
              </p>
            </div>
          </div>
          <DashboardStatusBadge status={status} />
        </div>

        <div className="grid gap-3">
          <ProgressLine label="Active pool coverage" value={activeRatio} />
          <ProgressLine label="Available manual coverage" value={availableRatio} />
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <InfoTile label="Cooling" value={formatNumber(health?.proxies.cooling)} />
          <InfoTile label="Saturated" value={formatNumber(health?.proxies.saturated)} />
          <InfoTile label="High risk" value={formatNumber(health?.proxies.high_risk)} />
          <InfoTile label="Dynamic configured" value={health?.proxy_source.dynamic_configured ? "Enabled" : "Disabled"} />
          <InfoTile label="Primary source" value={health?.proxy_policy.primary_source || "Unknown"} />
          <InfoTile label="Fallback source" value={health?.proxy_policy.fallback_source || "None"} />
        </div>
      </CardContent>
    </Card>
  );
}

function ProgressLine({ label, value }: { label: string; value: number | null }) {
  const clamped = value === null ? 0 : Math.max(0, Math.min(value, 1));

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">{label}</p>
        <p className="text-sm font-medium text-foreground">{value === null ? "N/A" : `${Math.round(clamped * 100)}%`}</p>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-muted">
        <div
          className={cn("h-full rounded-full bg-chart-4", clamped === 0 && "bg-muted-foreground/35")}
          style={{ width: `${clamped * 100}%` }}
        />
      </div>
    </div>
  );
}

function InfoTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border/70 bg-muted/30 p-3">
      <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">{label}</p>
      <p className="mt-1 truncate text-sm font-medium text-foreground">{value}</p>
    </div>
  );
}
