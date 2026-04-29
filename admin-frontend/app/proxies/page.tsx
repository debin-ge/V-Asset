"use client";

import * as React from "react";
import { Filter, Plus, RefreshCcw, RotateCcw, Search } from "lucide-react";
import { toast } from "sonner";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { AppShell } from "@/components/layout/AppShell";
import { ProxyPolicyCard } from "@/components/proxies/ProxyPolicyCard";
import { ProxyStatusCard } from "@/components/proxies/ProxyStatusCard";
import { ProxyTable } from "@/components/proxies/ProxyTable";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { proxyApi } from "@/lib/api/proxy";
import type {
  ProxyCreatePayload,
  ProxyInfo,
  ProxyListSortBy,
  ProxySourcePolicy,
  ProxySourceStatus,
  ProxyUpdatePayload,
  UpdateProxySourcePolicyPayload,
} from "@/types/proxy";

type FilterState = {
  search: string;
  protocol: string;
  region: string;
  status: string;
};

const defaultFilters: FilterState = {
  search: "",
  protocol: "",
  region: "",
  status: "-1",
};

export default function ProxiesPage() {
  const [status, setStatus] = React.useState<ProxySourceStatus | null>(null);
  const [policy, setPolicy] = React.useState<ProxySourcePolicy | null>(null);
  const [items, setItems] = React.useState<ProxyInfo[]>([]);
  const [filters, setFilters] = React.useState<FilterState>(defaultFilters);
  const [appliedFilters, setAppliedFilters] = React.useState<FilterState>(defaultFilters);
  const [loading, setLoading] = React.useState(true);
  const [createOpen, setCreateOpen] = React.useState(false);
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(20);
  const [total, setTotal] = React.useState(0);
  const [sortBy, setSortBy] = React.useState<ProxyListSortBy | "">("");
  const [sortOrder, setSortOrder] = React.useState<"asc" | "desc">("desc");

  const loadData = React.useCallback(async () => {
    setLoading(true);
    try {
      const [statusResponse, policyResponse, listResponse] = await Promise.all([
        proxyApi.getSourceStatus(),
        proxyApi.getCurrentPolicy(),
        proxyApi.list({
          ...(appliedFilters.search ? { search: appliedFilters.search } : {}),
          ...(appliedFilters.protocol ? { protocol: appliedFilters.protocol } : {}),
          ...(appliedFilters.region ? { region: appliedFilters.region } : {}),
          ...(appliedFilters.status !== "-1" ? { status: Number(appliedFilters.status) } : {}),
          page,
          page_size: pageSize,
          ...(sortBy ? { sort_by: sortBy, sort_order: sortOrder } : {}),
        }),
      ]);
      const pagination = listResponse.pagination ?? {
        page,
        page_size: pageSize,
        total: listResponse.items?.length ?? 0,
      };
      if ((listResponse.items || []).length === 0 && pagination.total > 0 && page > 1) {
        setStatus(statusResponse);
        setPolicy(policyResponse);
        setItems([]);
        setTotal(pagination.total);
        setPage(Math.max(1, Math.ceil(pagination.total / Math.max(pageSize, 1))));
        return;
      }
      setStatus(statusResponse);
      setPolicy(policyResponse);
      setItems(listResponse.items || []);
      setTotal(pagination.total || 0);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load proxies");
    } finally {
      setLoading(false);
    }
  }, [appliedFilters, page, pageSize, sortBy, sortOrder]);

  React.useEffect(() => {
    void loadData();
  }, [loadData]);

  const summary = React.useMemo(() => getProxySummary(items), [items]);
  const hasActiveFilters = React.useMemo(() => {
    return Boolean(appliedFilters.search || appliedFilters.protocol || appliedFilters.region || appliedFilters.status !== "-1");
  }, [appliedFilters]);

  const handleSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setAppliedFilters({
      search: filters.search.trim(),
      protocol: filters.protocol,
      region: filters.region.trim(),
      status: filters.status,
    });
    setPage(1);
  };

  const handleResetFilters = () => {
    setFilters(defaultFilters);
    setAppliedFilters(defaultFilters);
    setPage(1);
  };

  const handlePageSizeChange = (nextPageSize: number) => {
    setPageSize(nextPageSize);
    setPage(1);
  };

  const handleSortChange = (nextSortBy: ProxyListSortBy) => {
    setPage(1);
    if (sortBy === nextSortBy) {
      setSortOrder((currentSortOrder) => currentSortOrder === "asc" ? "desc" : "asc");
      return;
    }
    setSortBy(nextSortBy);
    setSortOrder("desc");
  };

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
      <AppShell
        actions={(
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" onClick={() => void loadData()} disabled={loading}>
              <RefreshCcw data-icon="inline-start" className={loading ? "animate-spin" : ""} />
              Refresh
            </Button>
            <Button onClick={() => setCreateOpen(true)}>
              <Plus data-icon="inline-start" />
              Add Proxy
            </Button>
          </div>
        )}
      >
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <h1 className="text-2xl font-semibold text-foreground">Proxy Pool</h1>
            <p className="text-sm text-muted-foreground">
              管理手动代理池、风控状态和全局代理策略。列表已改为高密度表格，适合持续增长的数据量。
            </p>
          </div>

          <ProxySummaryGrid summary={summary} status={status} total={total} />

          <div className="grid gap-4 xl:grid-cols-[0.85fr_1.15fr]">
            <ProxyStatusCard status={status} />
            <ProxyPolicyCard policy={policy} onSubmit={(payload) => void handlePolicyUpdate(payload)} />
          </div>

          <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
            <CardHeader className="pb-3">
              <CardTitle className="flex items-center gap-2 text-base">
                <Filter className="size-4 text-muted-foreground" />
                Filters
              </CardTitle>
            </CardHeader>
            <CardContent>
              <form className="grid gap-3 lg:grid-cols-[minmax(240px,1.8fr)_minmax(130px,0.8fr)_minmax(140px,0.9fr)_minmax(130px,0.8fr)_auto]" onSubmit={handleSearch}>
                <div className="relative">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    className="pl-9"
                    placeholder="Search host, region, tags, or remark"
                    value={filters.search}
                    onChange={(event) => setFilters((prev) => ({ ...prev, search: event.target.value }))}
                  />
                </div>
                <NativeSelect
                  aria-label="Protocol"
                  value={filters.protocol}
                  onChange={(event) => setFilters((prev) => ({ ...prev, protocol: event.target.value }))}
                >
                  <option value="">All Protocols</option>
                  <option value="http">HTTP</option>
                  <option value="https">HTTPS</option>
                  <option value="socks5">SOCKS5</option>
                </NativeSelect>
                <Input
                  placeholder="Region"
                  value={filters.region}
                  onChange={(event) => setFilters((prev) => ({ ...prev, region: event.target.value }))}
                />
                <NativeSelect
                  aria-label="Status"
                  value={filters.status}
                  onChange={(event) => setFilters((prev) => ({ ...prev, status: event.target.value }))}
                >
                  <option value="-1">All Status</option>
                  <option value="0">Active</option>
                  <option value="1">Inactive</option>
                  <option value="2">Checking</option>
                </NativeSelect>
                <div className="flex gap-2">
                  <Button type="submit" className="flex-1 lg:flex-none">
                    <Search data-icon="inline-start" />
                    Search
                  </Button>
                  <Button type="button" variant="outline" onClick={handleResetFilters} aria-label="Reset filters">
                    <RotateCcw />
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>

          <ProxyTable
            items={items}
            loading={loading}
            hasActiveFilters={hasActiveFilters}
            page={page}
            pageSize={pageSize}
            total={total}
            sortBy={sortBy}
            sortOrder={sortOrder}
            onPageChange={setPage}
            onPageSizeChange={handlePageSizeChange}
            onSortChange={handleSortChange}
            createOpen={createOpen}
            onCreateOpenChange={setCreateOpen}
            onCreate={(payload) => void handleCreate(payload)}
            onUpdate={(id, payload) => void handleUpdate(id, payload)}
            onUpdateStatus={(id, nextStatus) => void handleStatusChange(id, nextStatus)}
            onDelete={(id) => void handleDelete(id)}
          />
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}

function ProxySummaryGrid({
  summary,
  status,
  total,
}: {
  summary: ReturnType<typeof getProxySummary>;
  status: ProxySourceStatus | null;
  total: number;
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
      <SummaryCard label="Total" value={total} detail={`${summary.total} shown on this page`} />
      <SummaryCard label="Active" value={summary.active} detail={`${summary.inactive} inactive`} />
      <SummaryCard label="Cooling" value={summary.coolingDown} detail="Temporarily excluded" />
      <SummaryCard label="Saturated" value={summary.saturated} detail="At max concurrency" />
      <SummaryCard label="Source Health" value={status?.healthy ? "Healthy" : "Check"} detail={`${status?.available_manual_proxy_count ?? 0} manual available`} />
    </div>
  );
}

function SummaryCard({ label, value, detail }: { label: string; value: number | string; detail: string }) {
  return (
    <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardContent className="flex flex-col gap-1 p-4">
        <p className="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">{label}</p>
        <p className="text-2xl font-semibold text-foreground">{value}</p>
        <p className="text-xs text-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  );
}

function getProxySummary(items: ProxyInfo[]) {
  const now = Date.now();
  return items.reduce(
    (summary, item) => {
      const cooldownUntil = parseTime(item.cooldown_until);
      return {
        total: summary.total + 1,
        active: summary.active + (item.status === 0 ? 1 : 0),
        inactive: summary.inactive + (item.status === 1 ? 1 : 0),
        checking: summary.checking + (item.status === 2 ? 1 : 0),
        coolingDown: summary.coolingDown + (cooldownUntil && cooldownUntil > now ? 1 : 0),
        saturated: summary.saturated + (item.max_concurrent > 0 && item.active_task_count >= item.max_concurrent ? 1 : 0),
      };
    },
    { total: 0, active: 0, inactive: 0, checking: 0, coolingDown: 0, saturated: 0 }
  );
}

function parseTime(value?: string) {
  if (!value) {
    return null;
  }
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? null : timestamp;
}

function NativeSelect(props: React.ComponentProps<"select">) {
  return (
    <select
      {...props}
      className="h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
    />
  );
}
