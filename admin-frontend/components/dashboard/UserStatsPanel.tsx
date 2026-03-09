import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { UserStats } from "@/types/stats";

export function UserStatsPanel({ stats }: { stats: UserStats | null }) {
  return (
    <Card className="overflow-hidden rounded-[32px] border-white/60 bg-white/78 shadow-xl shadow-blue-950/5 backdrop-blur-xl">
      <CardHeader>
        <CardTitle>User Activity Window</CardTitle>
        <CardDescription>注册、日活和周活统一对照。</CardDescription>
      </CardHeader>
      <CardContent>
        {!stats ? (
          <p className="text-sm text-muted-foreground">加载中...</p>
        ) : (
          <div className="grid gap-4 md:grid-cols-3">
            <StatBlock label="Total Registered Users" value={stats.total_users} accent="from-blue-500 to-cyan-400" />
            <StatBlock label="Daily Active Users" value={stats.daily_active_users} accent="from-violet-500 to-fuchsia-400" />
            <StatBlock label="Weekly Active Users" value={stats.weekly_active_users} accent="from-pink-500 to-orange-400" />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function StatBlock({ label, value, accent }: { label: string; value: number; accent: string }) {
  return (
    <div className="relative overflow-hidden rounded-[26px] border border-white/70 bg-white/85 p-5 shadow-sm">
      <div className={`absolute inset-x-0 top-0 h-1.5 bg-gradient-to-r ${accent}`} />
      <p className="text-xs font-medium uppercase tracking-[0.16em] text-slate-500">{label}</p>
      <p className="mt-4 text-4xl font-semibold tracking-tight text-slate-950">{value}</p>
      <p className="mt-2 text-sm text-slate-500">Live platform activity</p>
    </div>
  );
}
