"use client";

import * as React from "react";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { AppShell } from "@/components/layout/AppShell";
import { PageHeader } from "@/components/layout/PageHeader";
import { ProxyPolicyCard } from "@/components/proxies/ProxyPolicyCard";
import { ProxyStatusCard } from "@/components/proxies/ProxyStatusCard";
import { ProxyTable } from "@/components/proxies/ProxyTable";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { proxyApi } from "@/lib/api/proxy";
import type { ProxyCreatePayload, ProxySourcePolicy, ProxySourceStatus, UpdateProxySourcePolicyPayload, ProxyInfo, ProxyUpdatePayload } from "@/types/proxy";
import { toast } from "sonner";

export default function ProxiesPage() {
  const [status, setStatus] = React.useState<ProxySourceStatus | null>(null);
  const [policy, setPolicy] = React.useState<ProxySourcePolicy | null>(null);
  const [items, setItems] = React.useState<ProxyInfo[]>([]);
  const [search, setSearch] = React.useState("");
  const [protocol, setProtocol] = React.useState("");
  const [region, setRegion] = React.useState("");
  const [statusFilter, setStatusFilter] = React.useState("-1");

  const loadData = React.useCallback(async () => {
    const [statusResponse, policyResponse, listResponse] = await Promise.all([
      proxyApi.getSourceStatus(),
      proxyApi.getCurrentPolicy(),
      proxyApi.list({
        ...(search ? { search } : {}),
        ...(protocol ? { protocol } : {}),
        ...(region ? { region } : {}),
        ...(statusFilter !== "-1" ? { status: Number(statusFilter) } : {}),
      }),
    ]);
    setStatus(statusResponse);
    setPolicy(policyResponse);
    setItems(listResponse.items || []);
  }, [protocol, region, search, statusFilter]);

  React.useEffect(() => {
    void loadData();
  }, [loadData]);

  const handlePolicyUpdate = async (payload: UpdateProxySourcePolicyPayload) => {
    if (!policy) return;
    try {
      await proxyApi.updatePolicy(policy.id, payload);
      await loadData();
      toast.success("Policy updated");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update policy");
    }
  };

  const handleCreate = async (payload: ProxyCreatePayload) => {
    try {
      await proxyApi.create(payload);
      await loadData();
      toast.success("Proxy added");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to add proxy");
      throw error;
    }
  };

  const handleUpdate = async (id: number, payload: ProxyUpdatePayload) => {
    try {
      await proxyApi.update(id, payload);
      await loadData();
      toast.success("Proxy updated");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update proxy");
      throw error;
    }
  };

  const handleStatusChange = async (id: number, nextStatus: number) => {
    try {
      await proxyApi.updateStatus(id, nextStatus);
      await loadData();
      toast.success(nextStatus === 0 ? "Proxy enabled" : "Proxy disabled");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to change proxy status");
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await proxyApi.delete(id);
      await loadData();
      toast.success("Proxy deleted");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete proxy");
    }
  };

  return (
    <ProtectedRoute>
      <AppShell>
        <div className="space-y-4">
          <PageHeader
            eyebrow="Network Pool"
            title="Proxies"
            description="管理 Proxy 主备策略、健康状态与手动代理池。"
            actions={
              <Button onClick={() => void loadData()}>
                Refresh
              </Button>
            }
          />
          <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
            <CardContent className="grid gap-3 py-6 md:grid-cols-2 xl:grid-cols-5">
              <Input
                placeholder="Search host / region / tags / remark"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
              <select
                className="h-8 rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                value={protocol}
                onChange={(e) => setProtocol(e.target.value)}
              >
                <option value="">All Protocols</option>
                <option value="http">HTTP</option>
                <option value="https">HTTPS</option>
                <option value="socks5">SOCKS5</option>
              </select>
              <Input placeholder="Region" value={region} onChange={(e) => setRegion(e.target.value)} />
              <select
                className="h-8 rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <option value="-1">All Status</option>
                <option value="0">Active</option>
                <option value="1">Inactive</option>
                <option value="2">Checking</option>
              </select>
              <Button
                variant="outline"
                type="button"
                onClick={() => { setSearch(""); setProtocol(""); setRegion(""); setStatusFilter("-1"); }}
              >
                Reset Filters
              </Button>
            </CardContent>
          </Card>
          <div className="grid gap-4">
            <ProxyStatusCard status={status} />
            <ProxyPolicyCard policy={policy} onSubmit={(payload) => void handlePolicyUpdate(payload)} />
            <ProxyTable
              items={items}
              onCreate={(payload) => void handleCreate(payload)}
              onUpdate={(id, payload) => void handleUpdate(id, payload)}
              onUpdateStatus={(id, nextStatus) => void handleStatusChange(id, nextStatus)}
              onDelete={(id) => void handleDelete(id)}
            />
          </div>
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}
