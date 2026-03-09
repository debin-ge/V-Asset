import type { ReactNode } from "react";
import { Activity, ShieldAlert, ShieldCheck } from "lucide-react";

import { StatusBadge } from "@/components/common/StatusBadge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { ProxySourceStatus } from "@/types/proxy";

export function ProxyStatusCard({ status }: { status: ProxySourceStatus | null }) {
  return (
    <Card className="overflow-hidden rounded-[32px] border-white/60 bg-white/78 shadow-xl shadow-blue-950/5 backdrop-blur-xl">
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
            <ProxyInfoCell label="Lease" value={status.proxy_lease_id || "N/A"} />
            <ProxyInfoCell label="Expires" value={status.proxy_expire_at || "N/A"} />
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
    <div className="rounded-[24px] border border-white/70 bg-white/85 p-4 shadow-sm">
      <div className="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-[0.16em] text-slate-500">
        {Icon ? <Icon className="size-3.5" /> : null}
        {label}
      </div>
      <div className="text-sm font-medium text-slate-950">{value}</div>
    </div>
  );
}
