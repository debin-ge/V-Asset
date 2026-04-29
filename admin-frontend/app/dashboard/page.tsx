"use client";

import * as React from "react";
import { RefreshCcw } from "lucide-react";
import { toast } from "sonner";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { DashboardKpiStrip } from "@/components/dashboard/DashboardKpiStrip";
import { OperationsHealthPanel } from "@/components/dashboard/OperationsHealthPanel";
import { ProxyPoolHealthPanel } from "@/components/dashboard/ProxyPoolHealthPanel";
import { RecentExceptionsPanel } from "@/components/dashboard/RecentExceptionsPanel";
import { RequestTrendPanel } from "@/components/dashboard/RequestTrendPanel";
import { ResourcePolicySnapshot } from "@/components/dashboard/ResourcePolicySnapshot";
import { UserActivityPanel } from "@/components/dashboard/UserActivityPanel";
import { trendRequestForRange } from "@/components/dashboard/dashboard-helpers";
import type { TimeRange } from "@/components/dashboard/dashboard-helpers";
import { AppShell } from "@/components/layout/AppShell";
import { Button } from "@/components/ui/button";
import { statsApi } from "@/lib/api/stats";
import type { DashboardHealthResponse, RequestTrend } from "@/types/stats";

const timeRangeOptions: Array<{ value: TimeRange; label: string }> = [
  { value: "24h", label: "24h" },
  { value: "7d", label: "7d" },
  { value: "30d", label: "30d" },
];

export default function DashboardPage() {
  const [health, setHealth] = React.useState<DashboardHealthResponse | null>(null);
  const [trend, setTrend] = React.useState<RequestTrend | null>(null);
  const [timeRange, setTimeRange] = React.useState<TimeRange>("24h");
  const [loading, setLoading] = React.useState(true);
  const [autoRefresh, setAutoRefresh] = React.useState(false);
  const [lastRefreshAt, setLastRefreshAt] = React.useState<Date | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  const loadDashboard = React.useCallback(async (silent = false) => {
    setLoading(true);
    const trendRequest = trendRequestForRange(timeRange);
    const results = await Promise.allSettled([
      statsApi.getDashboardHealth(),
      statsApi.getRequestTrend(trendRequest.granularity, trendRequest.limit),
    ]);

    const [healthResult, trendResult] = results;
    const failures = results.filter((result) => result.status === "rejected");

    if (healthResult.status === "fulfilled") {
      setHealth(healthResult.value);
    }
    if (trendResult.status === "fulfilled") {
      setTrend(trendResult.value);
    }

    if (failures.length > 0) {
      const message = failures.length === results.length ? "Failed to load dashboard data." : "Some dashboard data failed to load.";
      setError(message);
      if (!silent) {
        toast.error(message);
      }
    } else {
      setError(null);
    }

    if (failures.length < results.length) {
      const generatedAt = healthResult.status === "fulfilled" ? new Date(healthResult.value.generated_at) : new Date();
      setLastRefreshAt(Number.isNaN(generatedAt.getTime()) ? new Date() : generatedAt);
    }
    setLoading(false);
  }, [timeRange]);

  React.useEffect(() => {
    void loadDashboard();
  }, [loadDashboard]);

  React.useEffect(() => {
    if (!autoRefresh) {
      return;
    }
    const timer = window.setInterval(() => {
      void loadDashboard(true);
    }, 60_000);
    return () => window.clearInterval(timer);
  }, [autoRefresh, loadDashboard]);

  const exceptions = health?.exceptions ?? [];

  return (
    <ProtectedRoute>
      <AppShell
        actions={(
          <div className="flex flex-wrap items-center gap-2">
            <label className="flex items-center gap-2 text-sm text-muted-foreground">
              Range
              <select
                className="h-8 rounded-lg border border-input bg-background px-2 text-sm text-foreground outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                value={timeRange}
                onChange={(event) => setTimeRange(event.target.value as TimeRange)}
              >
                {timeRangeOptions.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
            </label>
            <label className="flex h-8 items-center gap-2 rounded-lg border border-border bg-background px-2.5 text-sm text-muted-foreground">
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(event) => setAutoRefresh(event.target.checked)}
              />
              Auto refresh
            </label>
            <Button variant="outline" onClick={() => void loadDashboard()} disabled={loading}>
              <RefreshCcw data-icon="inline-start" className={loading ? "animate-spin" : ""} />
              Refresh
            </Button>
          </div>
        )}
      >
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <h1 className="text-2xl font-semibold text-foreground">Dashboard</h1>
            <p className="text-sm text-muted-foreground">
              System health, download activity, proxy capacity, and user activity for daily operations.
            </p>
          </div>

          <DashboardKpiStrip health={health} loading={loading} />

          <section className="grid gap-4 xl:grid-cols-[1.45fr_0.95fr]">
            <RequestTrendPanel trend={trend} timeRange={timeRange} loading={loading} />
            <OperationsHealthPanel
              health={health}
              loading={loading}
              error={error}
              lastRefreshAt={lastRefreshAt}
            />
          </section>

          <section className="grid gap-4 xl:grid-cols-2">
            <ProxyPoolHealthPanel health={health} />
            <UserActivityPanel health={health} loading={loading} />
          </section>

          <section className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
            <RecentExceptionsPanel exceptions={exceptions} loading={loading} />
            <ResourcePolicySnapshot health={health} />
          </section>
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}
