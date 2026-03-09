import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { Overview } from "@/types/stats";
import type { ProxySourcePolicy, ProxySourceStatus } from "@/types/proxy";

export function ResourceSummary({
  overview,
  proxyStatus,
  proxyPolicy,
}: {
  overview: Overview | null;
  proxyStatus: ProxySourceStatus | null;
  proxyPolicy: ProxySourcePolicy | null;
}) {
  return (
    <Card className="overflow-hidden rounded-[32px] border-white/60 bg-white/78 shadow-xl shadow-blue-950/5 backdrop-blur-xl">
      <CardHeader>
        <CardTitle>Operational Snapshot</CardTitle>
        <CardDescription>聚焦成功率、代理健康度和资源利用率。</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-4 md:grid-cols-2">
        <SnapshotTile
          label="Success Rate"
          value={
            overview && overview.total_downloads > 0
              ? `${Math.round((overview.success_downloads / overview.total_downloads) * 100)}%`
              : "N/A"
          }
          accent="from-blue-500/16 to-transparent"
        />
        <SnapshotTile label="Failure Load" value={String(overview?.failed_downloads ?? "-")} accent="from-amber-500/16 to-transparent" />
        <SnapshotTile label="Proxy Health" value={proxyStatus?.healthy ? "Healthy" : "Unavailable"} accent="from-emerald-500/16 to-transparent" />
        <SnapshotTile label="Current Lease" value={proxyStatus?.proxy_lease_id || "N/A"} accent="from-violet-500/16 to-transparent" />
        <SnapshotTile
          label="Proxy Strategy"
          value={proxyPolicy ? `${proxyPolicy.primary_source} -> ${proxyPolicy.fallback_source || "none"}` : "N/A"}
          accent="from-pink-500/16 to-transparent"
        />
        <SnapshotTile
          label="Manual Pool"
          value={overview ? `${overview.active_manual_proxies}/${overview.total_manual_proxies} active` : "N/A"}
          accent="from-cyan-500/16 to-transparent"
        />
      </CardContent>
    </Card>
  );
}

function SnapshotTile({ label, value, accent }: { label: string; value: string; accent: string }) {
  return (
    <div className={`rounded-[24px] border border-white/70 bg-gradient-to-br ${accent} bg-white/80 p-5 shadow-sm`}>
      <p className="text-xs font-medium uppercase tracking-[0.16em] text-slate-500">{label}</p>
      <p className="mt-3 text-lg font-semibold text-slate-950">{value}</p>
    </div>
  );
}
