import { Users } from "lucide-react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { formatNumber, formatPercent } from "@/components/dashboard/dashboard-helpers";
import type { DashboardHealthResponse } from "@/types/stats";

export function UserActivityPanel({
  health,
  loading,
}: {
  health: DashboardHealthResponse | null;
  loading: boolean;
}) {
  return (
    <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardHeader>
        <CardTitle>User Activity</CardTitle>
        <CardDescription>注册用户、日活、周活和活跃粘性。</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {!health ? (
          <div className="flex h-40 items-center justify-center rounded-lg border border-dashed border-border/80 bg-muted/35 text-sm text-muted-foreground">
            {loading ? "Loading user activity..." : "No user activity data available."}
          </div>
        ) : (
          <>
            <div className="grid gap-3 sm:grid-cols-3">
              <UserMetric label="Total Users" value={formatNumber(health.users.total)} />
              <UserMetric label="DAU" value={formatNumber(health.users.daily_active)} />
              <UserMetric label="WAU" value={formatNumber(health.users.weekly_active)} />
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              <StickinessMetric label="DAU / WAU" value={formatPercent(health.users.weekly_active > 0 ? health.users.dau_wau_rate : null)} />
              <StickinessMetric label="WAU / Total" value={formatPercent(health.users.total > 0 ? health.users.wau_total_rate : null)} />
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function UserMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border/70 bg-muted/30 p-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">{label}</p>
        <Users className="size-3.5 text-muted-foreground" />
      </div>
      <p className="mt-2 text-2xl font-semibold text-foreground">{value}</p>
    </div>
  );
}

function StickinessMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border/70 bg-background p-3">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="mt-1 text-xl font-semibold text-foreground">{value}</p>
    </div>
  );
}
