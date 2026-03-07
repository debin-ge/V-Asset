"use client";

import * as React from "react";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { Header } from "@/components/layout/Header";
import { Sidebar } from "@/components/layout/Sidebar";
import { MetricCard } from "@/components/dashboard/MetricCard";
import { RequestTrendChart } from "@/components/dashboard/RequestTrendChart";
import { ResourceSummary } from "@/components/dashboard/ResourceSummary";
import { UserStatsPanel } from "@/components/dashboard/UserStatsPanel";
import { ProxyStatusCard } from "@/components/proxies/ProxyStatusCard";
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
      <div className="layout shell">
        <Sidebar />
        <main className="content">
          <Header />
          <div className="toolbar">
            <div>
              <h1 className="page-title">Platform Dashboard</h1>
              <p className="muted">平台级用户、请求与运行资源概览。</p>
            </div>
            <div className="inline-actions">
              <button
                className={`button ${granularity === "day" ? "" : "secondary"}`}
                onClick={() => setGranularity("day")}
              >
                7 Days
              </button>
              <button
                className={`button ${granularity === "hour" ? "" : "secondary"}`}
                onClick={() => setGranularity("hour")}
              >
                24 Hours
              </button>
            </div>
          </div>
          <section className="grid metrics" style={{ marginTop: 20 }}>
            <MetricCard label="Total Users" value={overview?.total_users ?? "-"} />
            <MetricCard label="DAU" value={overview?.daily_active_users ?? "-"} />
            <MetricCard label="WAU" value={overview?.weekly_active_users ?? "-"} />
            <MetricCard label="Total Downloads" value={overview?.total_downloads ?? "-"} />
            <MetricCard label="Today" value={overview?.downloads_today ?? "-"} />
            <MetricCard label="Success" value={overview?.success_downloads ?? "-"} />
            <MetricCard label="Failed" value={overview?.failed_downloads ?? "-"} />
          </section>
          <section className="split" style={{ gridTemplateColumns: "1.4fr 1fr", marginTop: 20 }}>
            <RequestTrendChart trend={trend} />
            <ProxyStatusCard status={proxyStatus} />
          </section>
          <section className="split" style={{ gridTemplateColumns: "1fr 1fr", marginTop: 20 }}>
            <UserStatsPanel stats={userStats} />
            <ResourceSummary overview={overview} proxyStatus={proxyStatus} proxyPolicy={proxyPolicy} />
          </section>
        </main>
      </div>
    </ProtectedRoute>
  );
}
