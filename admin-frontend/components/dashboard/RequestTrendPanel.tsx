import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, ChartTooltip, type ChartConfig } from "@/components/ui/chart";
import { formatNumber, formatPercent, trendFailed, trendSuccess, trendSuccessRate, trendTotal } from "@/components/dashboard/dashboard-helpers";
import type { TimeRange } from "@/components/dashboard/dashboard-helpers";
import type { RequestTrend } from "@/types/stats";

const chartConfig = {
  success_count: {
    label: "Successful",
    color: "var(--chart-4)",
  },
  failed_count: {
    label: "Failed",
    color: "var(--chart-5)",
  },
} satisfies ChartConfig;

type TrendChartDatum = {
  label: string;
  total_count: number;
  success_count: number;
  failed_count: number;
  success_rate: number;
};

export function RequestTrendPanel({
  trend,
  timeRange,
  loading,
}: {
  trend: RequestTrend | null;
  timeRange: TimeRange;
  loading: boolean;
}) {
  const points: TrendChartDatum[] = (trend?.points ?? []).map((point) => ({
    label: point.label,
    total_count: trendTotal(point),
    success_count: trendSuccess(point),
    failed_count: trendFailed(point),
    success_rate: trendSuccessRate(point),
  }));

  return (
    <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardHeader>
        <div>
          <CardTitle>Request Trend</CardTitle>
          <CardDescription>
            {timeRange === "24h" ? "Hourly request volume over the last 24 hours." : `Daily request volume over the last ${timeRange}.`}
          </CardDescription>
        </div>
        <CardAction className="text-xs text-muted-foreground">
          {trend ? `${points.length} points` : "No points"}
        </CardAction>
      </CardHeader>
      <CardContent>
        {points.length === 0 ? (
          <div className="flex h-[300px] items-center justify-center rounded-lg border border-dashed border-border/80 bg-muted/35 text-sm text-muted-foreground">
            {loading ? "Loading trend..." : "No request trend data available."}
          </div>
        ) : (
          <ChartContainer config={chartConfig} className="h-[300px] w-full">
            <BarChart accessibilityLayer data={points} margin={{ left: 8, right: 8, top: 12 }}>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="label"
                tickLine={false}
                axisLine={false}
                tickMargin={10}
                minTickGap={24}
              />
              <YAxis
                width={42}
                tickLine={false}
                axisLine={false}
                tickFormatter={(value) => formatNumber(Number(value))}
              />
              <ChartTooltip
                cursor={false}
                content={<TrendTooltip />}
              />
              <Bar
                dataKey="success_count"
                stackId="requests"
                radius={[2, 2, 6, 6]}
                fill="var(--color-success_count)"
              />
              <Bar
                dataKey="failed_count"
                stackId="requests"
                radius={[6, 6, 2, 2]}
                fill="var(--color-failed_count)"
              />
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}

function TrendTooltip({
  active,
  payload,
  label,
}: {
  active?: boolean;
  payload?: Array<{ payload?: TrendChartDatum }>;
  label?: string;
}) {
  if (!active || !payload?.length) {
    return null;
  }

  const point = payload[0]?.payload;
  if (!point) {
    return null;
  }

  return (
    <div className="grid min-w-40 gap-2 rounded-lg border border-border/50 bg-background px-3 py-2 text-xs shadow-xl">
      <div className="font-medium text-foreground">{label}</div>
      <TooltipRow label="Total" value={formatNumber(point.total_count)} />
      <TooltipRow label="Successful" value={formatNumber(point.success_count)} />
      <TooltipRow label="Failed" value={formatNumber(point.failed_count)} />
      <TooltipRow label="Success rate" value={formatPercent(point.total_count > 0 ? point.success_rate : null)} />
    </div>
  );
}

function TooltipRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono font-medium text-foreground tabular-nums">{value}</span>
    </div>
  );
}
