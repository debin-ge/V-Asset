import type { ReactNode } from "react";
import { Activity, ShieldAlert, ShieldCheck } from "lucide-react";

import { StatusBadge } from "@/components/common/StatusBadge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { ProxySourceStatus } from "@/types/proxy";

export function ProxyStatusCard({ status }: { status: ProxySourceStatus | null }) {
  return (
    <Card className="overflow-hidden rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardHeader>
        <CardTitle>Proxy Source</CardTitle>
        <CardDescription>动态代理源当前健康状况与租约信息。</CardDescription>
      </CardHeader>
      <CardContent>
        {!status ? (
          <p className="text-sm text-muted-foreground">Loading...</p>
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            <ProxyInfoCell
              label="Status"
              value={
                <StatusBadge
                  label={status.healthy ? "Healthy" : "Unavailable"}
                  tone={status.healthy ? "success" : "danger"}
                />
              }
              icon={status.healthy ? ShieldCheck : ShieldAlert}
            />
            <ProxyInfoCell label="Mode" value={status.mode} icon={Activity} />
            <ProxyInfoCell label="Dynamic" value={status.dynamic_configured ? "Configured" : "Not configured"} />
            <ProxyInfoCell label="Manual Capacity" value={String(status.available_manual_proxy_count)} />
            <ProxyInfoCell label="Checked" value={status.checked_at} />
            <ProxyInfoCell label="Message" value={status.message || "N/A"} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ProxyInfoCell({
  label,
  value,
  icon: Icon,
}: {
  label: string;
  value: ReactNode;
  icon?: typeof Activity;
}) {
  return (
    <div className="rounded-lg border border-border/70 bg-muted/35 p-4">
      <div className="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-[0.16em] text-slate-500">
        {Icon ? <Icon className="size-3.5" /> : null}
        {label}
      </div>
      <div className="text-sm font-medium text-slate-950">{value}</div>
    </div>
  );
}
