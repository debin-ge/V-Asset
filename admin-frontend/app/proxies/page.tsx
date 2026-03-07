"use client";

import * as React from "react";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { Header } from "@/components/layout/Header";
import { Sidebar } from "@/components/layout/Sidebar";
import { ProxyPolicyCard } from "@/components/proxies/ProxyPolicyCard";
import { ProxyStatusCard } from "@/components/proxies/ProxyStatusCard";
import { ProxyTable } from "@/components/proxies/ProxyTable";
import { proxyApi } from "@/lib/api/proxy";
import type { ProxyCreatePayload, ProxySourcePolicy, ProxySourceStatus, UpdateProxySourcePolicyPayload, ProxyInfo, ProxyUpdatePayload } from "@/types/proxy";

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
    await proxyApi.updatePolicy(policy.id, payload);
    await loadData();
  };

  const handleCreate = async (payload: ProxyCreatePayload) => {
    await proxyApi.create(payload);
    await loadData();
  };

  const handleUpdate = async (id: number, payload: ProxyUpdatePayload) => {
    await proxyApi.update(id, payload);
    await loadData();
  };

  const handleStatusChange = async (id: number, nextStatus: number) => {
    await proxyApi.updateStatus(id, nextStatus);
    await loadData();
  };

  const handleDelete = async (id: number) => {
    await proxyApi.delete(id);
    await loadData();
  };

  return (
    <ProtectedRoute>
      <div className="layout shell">
        <Sidebar />
        <main className="content">
          <Header />
          <div className="toolbar">
            <div>
              <h1 className="page-title">Proxies</h1>
              <p className="muted">管理 Proxy 主备策略和手动代理池。</p>
            </div>
            <button className="button" onClick={() => void loadData()}>Refresh</button>
          </div>
          <div className="card" style={{ marginBottom: 16 }}>
            <div className="grid proxy-form-grid">
              <input className="field" placeholder="Search host / region / tags / remark" value={search} onChange={(e) => setSearch(e.target.value)} />
              <select className="select" value={protocol} onChange={(e) => setProtocol(e.target.value)}>
                <option value="">All Protocols</option>
                <option value="http">HTTP</option>
                <option value="https">HTTPS</option>
                <option value="socks5">SOCKS5</option>
              </select>
              <input className="field" placeholder="Region" value={region} onChange={(e) => setRegion(e.target.value)} />
              <select className="select" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
                <option value="-1">All Status</option>
                <option value="0">Active</option>
                <option value="1">Inactive</option>
                <option value="2">Checking</option>
              </select>
              <button className="button ghost" type="button" onClick={() => { setSearch(""); setProtocol(""); setRegion(""); setStatusFilter("-1"); }}>Reset Filters</button>
            </div>
          </div>
          <div className="grid">
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
        </main>
      </div>
    </ProtectedRoute>
  );
}
