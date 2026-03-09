import { Activity, ArrowDownToLine, CircleCheckBig, TriangleAlert, Users } from "lucide-react";

import { Card, CardContent, CardHeader } from "@/components/ui/card";

const iconMap: Record<string, typeof Users> = {
  "Total Users": Users,
  DAU: Activity,
  WAU: Activity,
  "Total Downloads": ArrowDownToLine,
  Today: ArrowDownToLine,
  Success: CircleCheckBig,
  Failed: TriangleAlert,
};

const accentMap: Record<string, string> = {
  "Total Users": "from-blue-500/20 via-blue-500/8 to-transparent text-blue-600",
  DAU: "from-violet-500/20 via-violet-500/8 to-transparent text-violet-600",
  WAU: "from-fuchsia-500/20 via-fuchsia-500/8 to-transparent text-fuchsia-600",
  "Total Downloads": "from-cyan-500/20 via-cyan-500/8 to-transparent text-cyan-600",
  Today: "from-sky-500/20 via-sky-500/8 to-transparent text-sky-600",
  Success: "from-emerald-500/20 via-emerald-500/8 to-transparent text-emerald-600",
  Failed: "from-amber-500/20 via-amber-500/8 to-transparent text-amber-600",
};

export function MetricCard({ label, value }: { label: string; value: string | number }) {
  const Icon = iconMap[label] || Activity;
  const accent = accentMap[label] || "from-slate-500/20 via-slate-500/8 to-transparent text-slate-700";

  return (
    <Card className="group relative overflow-hidden rounded-[28px] border-white/60 bg-white/80 shadow-lg shadow-blue-950/5 backdrop-blur-xl transition-transform hover:-translate-y-0.5">
      <div className={`absolute inset-0 bg-gradient-to-br ${accent}`} />
      <CardHeader className="relative flex-row items-start justify-between gap-4 pb-2">
        <div className="space-y-1">
          <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">{label}</p>
          <p className="text-3xl font-semibold tracking-tight text-slate-950 md:text-4xl">{value}</p>
        </div>
        <div className="flex size-11 items-center justify-center rounded-2xl bg-white/70 shadow-sm">
          <Icon className="size-5" />
        </div>
      </CardHeader>
      <CardContent className="relative pt-0 text-sm text-slate-600">
        平台实时运营指标
      </CardContent>
    </Card>
  );
}
