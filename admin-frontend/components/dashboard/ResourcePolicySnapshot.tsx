import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { formatNumber } from "@/components/dashboard/dashboard-helpers";
import type { DashboardHealthResponse } from "@/types/stats";

export function ResourcePolicySnapshot({
  health,
}: {
  health: DashboardHealthResponse | null;
}) {
  return (
    <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardHeader>
        <CardTitle>Resource & Policy Snapshot</CardTitle>
        <CardDescription>代理来源、租约和手动池资源摘要。</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-3 sm:grid-cols-2">
        <SnapshotItem label="Proxy strategy" value={health ? `${health.proxy_policy.primary_source || "unknown"} -> ${health.proxy_policy.fallback_source || "none"}` : "Unknown"} />
        <SnapshotItem label="Current lease" value={health?.proxy_source.proxy_lease_id || "N/A"} />
        <SnapshotItem label="Dynamic configured" value={health?.proxy_source.dynamic_configured ? "Enabled" : "Disabled"} />
        <SnapshotItem label="Manual availability" value={`${formatNumber(health?.proxies.available)} available`} />
        <SnapshotItem label="Manual pool" value={`${formatNumber(health?.proxies.active)} / ${formatNumber(health?.proxies.total)} active`} />
        <SnapshotItem label="Lease expires" value={health?.proxy_source.proxy_expire_at ? new Date(health.proxy_source.proxy_expire_at).toLocaleString() : "N/A"} />
        <SnapshotItem label="Cookies" value={`${formatNumber(health?.cookies.active)} active / ${formatNumber(health?.cookies.total)} total`} />
        <SnapshotItem label="Shortfalls" value={formatNumber(health?.billing.shortfall_count)} />
      </CardContent>
    </Card>
  );
}

function SnapshotItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border/70 bg-muted/30 p-3">
      <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">{label}</p>
      <p className="mt-1 truncate text-sm font-medium text-foreground">{value}</p>
    </div>
  );
}
