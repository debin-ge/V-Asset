import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { CookieInfo } from "@/types/cookie";

export function CookieTable({
  items,
  onDelete,
  onFreeze,
}: {
  items: CookieInfo[];
  onDelete: (id: number) => void;
  onFreeze: (id: number) => void;
}) {
  return (
    <Card className="overflow-hidden rounded-[32px] border-white/60 bg-white/78 shadow-xl shadow-blue-950/5 backdrop-blur-xl">
      <CardHeader>
        <CardTitle>Cookie Inventory</CardTitle>
        <CardDescription>查看状态、过期时间和调用情况。</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {items.map((item) => (
            <div key={item.id} className="rounded-[28px] border border-white/70 bg-white/90 p-5 shadow-sm transition-transform hover:-translate-y-0.5">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-4">
                  <div className="flex flex-wrap items-center gap-3">
                    <div className="rounded-2xl bg-gradient-to-br from-fuchsia-500 to-pink-500 px-4 py-3 text-white shadow-lg shadow-pink-500/20">
                      <p className="text-xs uppercase tracking-[0.18em] text-white/70">{item.platform}</p>
                      <p className="text-lg font-semibold">{item.name}</p>
                    </div>
                    <StatusBadge label={statusLabel(item.status)} tone={statusTone(item.status)} />
                    <MetaPill label="Usage" value={String(item.use_count)} />
                    <MetaPill label="Success" value={String(item.success_count)} />
                    <MetaPill label="Fail" value={String(item.fail_count)} />
                  </div>
                  <div className="grid gap-3 md:grid-cols-3">
                    <InfoStat label="Expire At" value={item.expire_at || "N/A"} />
                    <InfoStat label="Frozen Until" value={item.frozen_until || "N/A"} />
                    <InfoStat label="Updated" value={item.updated_at} />
                  </div>
                </div>
                <div className="flex flex-wrap gap-2 lg:justify-end">
                  <div className="flex flex-wrap gap-2">
                    <Button variant="ghost" size="sm" onClick={() => onFreeze(item.id)}>Freeze</Button>
                    <ConfirmDialog
                      trigger={<Button variant="outline" size="sm">Delete</Button>}
                      title="Delete cookie?"
                      description={`This will permanently remove cookie "${item.name}" from the platform asset pool.`}
                      actionLabel="Delete"
                      onConfirm={() => onDelete(item.id)}
                    />
                  </div>
                </div>
              </div>
            </div>
          ))}
          {items.length === 0 ? (
            <div className="rounded-[28px] border border-dashed border-slate-200 bg-white/70 px-6 py-12 text-center text-sm text-slate-500">
              No cookies available for this filter.
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}

function statusLabel(status: number) {
  switch (status) {
    case 0:
      return "Active";
    case 1:
      return "Expired";
    case 2:
      return "Frozen";
    default:
      return `Unknown(${status})`;
  }
}

function statusTone(status: number): "success" | "warning" | "info" | "neutral" {
  switch (status) {
    case 0:
      return "success";
    case 1:
      return "warning";
    case 2:
      return "info";
    default:
      return "neutral";
  }
}

function MetaPill({ label, value }: { label: string; value: string }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs text-slate-600">
      <span className="font-medium text-slate-400">{label}</span>
      {value}
    </span>
  );
}

function InfoStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-slate-50/80 px-4 py-3">
      <p className="text-[11px] uppercase tracking-[0.14em] text-slate-400">{label}</p>
      <p className="mt-1 text-sm font-medium text-slate-900">{value}</p>
    </div>
  );
}
