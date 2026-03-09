import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import type { RequestTrend } from "@/types/stats";

const chartConfig = {
  count: {
    label: "Requests",
    color: "var(--chart-1)",
  },
} satisfies ChartConfig;

export function RequestTrendChart({ trend }: { trend: RequestTrend | null }) {
  return (
    <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
      <CardHeader>
        <CardTitle>Request Trend</CardTitle>
        <CardDescription>按时间窗口查看请求量波动。</CardDescription>
      </CardHeader>
      <CardContent>
        {!trend ? (
          <p className="text-sm text-muted-foreground">暂无数据</p>
        ) : (
          <ChartContainer config={chartConfig} className="h-[280px] w-full">
            <BarChart accessibilityLayer data={trend.points} margin={{ left: 8, right: 8, top: 12 }}>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="label"
                tickLine={false}
                axisLine={false}
                tickMargin={10}
                minTickGap={24}
              />
              <ChartTooltip
                cursor={false}
                content={<ChartTooltipContent indicator="dot" />}
              />
              <Bar
                dataKey="count"
                radius={[10, 10, 4, 4]}
                fill="var(--color-count)"
              />
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
