"use client";

import * as React from "react";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { MetricCard } from "@/components/dashboard/MetricCard";
import { RequestTrendChart } from "@/components/dashboard/RequestTrendChart";
import { ResourceSummary } from "@/components/dashboard/ResourceSummary";
import { UserStatsPanel } from "@/components/dashboard/UserStatsPanel";
import { AppShell } from "@/components/layout/AppShell";
import { PageHeader } from "@/components/layout/PageHeader";
import { ProxyStatusCard } from "@/components/proxies/ProxyStatusCard";
import { Button } from "@/components/ui/button";
import { proxyApi } from "@/lib/api/proxy";
import { statsApi } from "@/lib/api/stats";
import type { Overview, RequestTrend, UserStats } from "@/types/stats";
import type { ProxySourcePolicy, ProxySourceStatus } from "@/types/proxy";

export default function DashboardPage() {
  const [overview, setOverview] = React.useState<Overview | null>(null);
  const [trend, setTrend] = React.useState<RequestTrend | null>(null);
  const [userStats, setUserStats] = React.useState<UserStats | null>(null);
  const [proxyStatus, setProxyStatus] = React.useState<ProxySourceStatus | null>(null);
  const [proxyPolicy, setProxyPolicy] = React.useState<ProxySourcePolicy | null>(null);
  const [granularity, setGranularity] = React.useState<"day" | "hour">("day");

  const loadDashboard = React.useCallback(async (targetGranularity: "day" | "hour") => {
    const [overviewData, trendData, userStatsData, proxyData, policyData] = await Promise.all([
      statsApi.getOverview(),
      statsApi.getRequestTrend(targetGranularity, targetGranularity === "day" ? 7 : 24),
      statsApi.getUsers(),
      proxyApi.getSourceStatus(),
      proxyApi.getCurrentPolicy(),
    ]);
    setOverview(overviewData);
    setTrend(trendData);
    setUserStats(userStatsData);
    setProxyStatus(proxyData);
    setProxyPolicy(policyData);
  }, []);

  React.useEffect(() => {
    void loadDashboard(granularity);
  }, [granularity, loadDashboard]);

  return (
    <ProtectedRoute>
      <AppShell>
        <div className="space-y-4">
          <PageHeader
            eyebrow="Operations"
            title="Platform Dashboard"
            description="平台级用户、请求流量、代理资源和失败面板统一汇总。"
            actions={
              <>
                <Button
                  variant={granularity === "day" ? "default" : "outline"}
                  onClick={() => setGranularity("day")}
                >
                7 Days
                </Button>
                <Button
                  variant={granularity === "hour" ? "default" : "outline"}
                  onClick={() => setGranularity("hour")}
                >
                24 Hours
                </Button>
              </>
            }
          />
          <section className="grid gap-4 md:grid-cols-2 2xl:grid-cols-4">
            <MetricCard label="Total Users" value={overview?.total_users ?? "-"} />
            <MetricCard label="DAU" value={overview?.daily_active_users ?? "-"} />
            <MetricCard label="WAU" value={overview?.weekly_active_users ?? "-"} />
            <MetricCard label="Total Downloads" value={overview?.total_downloads ?? "-"} />
            <MetricCard label="Today" value={overview?.downloads_today ?? "-"} />
            <MetricCard label="Success" value={overview?.success_downloads ?? "-"} />
            <MetricCard label="Failed" value={overview?.failed_downloads ?? "-"} />
          </section>
          <section className="grid gap-4 xl:grid-cols-[1.4fr_1fr]">
            <RequestTrendChart trend={trend} />
            <ProxyStatusCard status={proxyStatus} />
          </section>
          <section className="grid gap-4 xl:grid-cols-2">
            <UserStatsPanel stats={userStats} />
            <ResourceSummary overview={overview} proxyStatus={proxyStatus} proxyPolicy={proxyPolicy} />
          </section>
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}
