"use client";

import * as React from "react";
import { usePathname, useRouter, useSearchParams, type ReadonlyURLSearchParams } from "next/navigation";
import { Copy, Eye, Filter, RefreshCcw, RotateCcw, Search } from "lucide-react";
import { toast } from "sonner";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { StatusBadge } from "@/components/common/StatusBadge";
import { AppShell } from "@/components/layout/AppShell";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { proxyApi } from "@/lib/api/proxy";
import type { ListProxyUsageEventsParams, ProxyUsageEvent, ProxyUsageEventSummary } from "@/types/proxy";

type FilterState = {
  task_id: string;
  proxy_id: string;
  proxy_lease_id: string;
  source_type: string;
  stage: string;
  platform: string;
  success: "all" | "success" | "failed";
  error_category: string;
  start_time: string;
  end_time: string;
  page: number;
  page_size: number;
  sort_order: "asc" | "desc";
};

const errorCategories = [
  "network_timeout",
  "proxy_auth",
  "proxy_unreachable",
  "rate_limited",
  "bot_detected",
  "cookie_invalid",
  "terminal_video",
  "unknown",
];

const filterControlClassName = "h-9 w-full rounded-md border border-input bg-background px-3 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50";

const emptySummary: ProxyUsageEventSummary = {
  success_count: 0,
  failure_count: 0,
  failure_rate: 0,
  category_counts: [],
  stage_counts: [],
  platform_counts: [],
};

export default function ProxyEventsPage() {
  return (
    <ProtectedRoute>
      <AppShell>
        <React.Suspense fallback={<div className="rounded-lg border border-border bg-white p-6 text-sm text-slate-500">Loading proxy events...</div>}>
          <ProxyEventsView />
        </React.Suspense>
      </AppShell>
    </ProtectedRoute>
  );
}

function ProxyEventsView() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const queryFilters = React.useMemo(() => filtersFromSearchParams(searchParams), [searchParams]);

  const [filters, setFilters] = React.useState<FilterState>(queryFilters);
  const [events, setEvents] = React.useState<ProxyUsageEvent[]>([]);
  const [summary, setSummary] = React.useState<ProxyUsageEventSummary>(emptySummary);
  const [pagination, setPagination] = React.useState({ page: queryFilters.page, page_size: queryFilters.page_size, total: 0 });
  const [loading, setLoading] = React.useState(false);
  const [selectedEvent, setSelectedEvent] = React.useState<ProxyUsageEvent | null>(null);

  React.useEffect(() => {
    setFilters(queryFilters);
  }, [queryFilters]);

  const loadEvents = React.useCallback(async () => {
    setLoading(true);
    try {
      const response = await proxyApi.listUsageEvents(toApiParams(queryFilters));
      setEvents(response.events || []);
      setSummary(response.summary || emptySummary);
      setPagination(response.pagination || { page: queryFilters.page, page_size: queryFilters.page_size, total: 0 });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load proxy events");
    } finally {
      setLoading(false);
    }
  }, [queryFilters]);

  React.useEffect(() => {
    void loadEvents();
  }, [loadEvents]);

  const pushFilters = React.useCallback(
    (next: FilterState) => {
      const params = filtersToSearchParams(next);
      router.push(params.size > 0 ? `${pathname}?${params.toString()}` : pathname);
    },
    [pathname, router]
  );

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    pushFilters({ ...filters, page: 1 });
  };

  const handleReset = () => {
    router.push(pathname);
  };

  const pageCount = Math.max(1, Math.ceil(pagination.total / Math.max(pagination.page_size, 1)));

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-slate-950">Proxy Usage Events</h1>
          <p className="mt-1 text-sm text-slate-500">Filter proxy usage records by task, platform, stage, error category, and time range.</p>
        </div>
        <Button variant="outline" onClick={() => void loadEvents()} disabled={loading}>
          <RefreshCcw data-icon="inline-start" className={loading ? "animate-spin" : ""} />
          Refresh
        </Button>
      </div>

      <SummaryGrid summary={summary} total={pagination.total} />

      <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <Filter className="size-4 text-slate-500" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form className="grid gap-3 lg:grid-cols-6" onSubmit={handleSubmit}>
            <Field label="Start Time">
              <Input type="datetime-local" value={filters.start_time} onChange={(event) => setFilters((prev) => ({ ...prev, start_time: event.target.value }))} />
            </Field>
            <Field label="End Time">
              <Input type="datetime-local" value={filters.end_time} onChange={(event) => setFilters((prev) => ({ ...prev, end_time: event.target.value }))} />
            </Field>
            <Field label="Platform">
              <Input value={filters.platform} placeholder="youtube" onChange={(event) => setFilters((prev) => ({ ...prev, platform: event.target.value }))} />
            </Field>
            <Field label="Stage">
              <select className={filterControlClassName} value={filters.stage} onChange={(event) => setFilters((prev) => ({ ...prev, stage: event.target.value }))}>
                <option value="">All</option>
                <option value="parse">Parse</option>
                <option value="download">Download</option>
              </select>
            </Field>
            <Field label="Result">
              <select className={filterControlClassName} value={filters.success} onChange={(event) => setFilters((prev) => ({ ...prev, success: event.target.value as FilterState["success"] }))}>
                <option value="all">All</option>
                <option value="success">Success</option>
                <option value="failed">Failed</option>
              </select>
            </Field>
            <Field label="Error Category">
              <select className={filterControlClassName} value={filters.error_category} onChange={(event) => setFilters((prev) => ({ ...prev, error_category: event.target.value }))}>
                <option value="">All</option>
                {errorCategories.map((category) => (
                  <option key={category} value={category}>{category}</option>
                ))}
              </select>
            </Field>
            <Field label="Source">
              <select className={filterControlClassName} value={filters.source_type} onChange={(event) => setFilters((prev) => ({ ...prev, source_type: event.target.value }))}>
                <option value="">All</option>
                <option value="manual">Manual</option>
                <option value="dynamic">Dynamic</option>
              </select>
            </Field>
            <Field label="Proxy ID">
              <Input inputMode="numeric" value={filters.proxy_id} onChange={(event) => setFilters((prev) => ({ ...prev, proxy_id: event.target.value }))} />
            </Field>
            <Field label="Task ID">
              <Input value={filters.task_id} onChange={(event) => setFilters((prev) => ({ ...prev, task_id: event.target.value }))} />
            </Field>
            <Field label="Lease ID">
              <Input value={filters.proxy_lease_id} onChange={(event) => setFilters((prev) => ({ ...prev, proxy_lease_id: event.target.value }))} />
            </Field>
            <Field label="Page Size">
              <select className={filterControlClassName} value={filters.page_size} onChange={(event) => setFilters((prev) => ({ ...prev, page_size: Number(event.target.value) }))}>
                <option value={20}>20</option>
                <option value={50}>50</option>
                <option value={100}>100</option>
              </select>
            </Field>
            <div className="flex items-end gap-2">
              <Button type="submit" className="w-full">
                <Search data-icon="inline-start" />
                Search
              </Button>
              <Button type="button" variant="outline" onClick={handleReset} aria-label="Reset filters">
                <RotateCcw />
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
        <CardHeader className="flex flex-row items-center justify-between gap-3 pb-2">
          <CardTitle className="text-base">Events</CardTitle>
          <span className="text-sm text-slate-500">{loading ? "Loading..." : `${pagination.total} records`}</span>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Time</TableHead>
                <TableHead>Platform</TableHead>
                <TableHead>Stage</TableHead>
                <TableHead>Result</TableHead>
                <TableHead>Category</TableHead>
                <TableHead>Proxy</TableHead>
                <TableHead>Source</TableHead>
                <TableHead>Task</TableHead>
                <TableHead>Lease</TableHead>
                <TableHead>Error</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {events.map((event) => (
                <TableRow key={event.id}>
                  <TableCell className="text-slate-600">{formatDateTime(event.created_at)}</TableCell>
                  <TableCell>{event.platform || "N/A"}</TableCell>
                  <TableCell><StatusBadge label={event.stage || "N/A"} tone="info" /></TableCell>
                  <TableCell><StatusBadge label={event.success ? "Success" : "Failed"} tone={event.success ? "success" : "danger"} /></TableCell>
                  <TableCell><StatusBadge label={event.error_category || "none"} tone={categoryTone(event.error_category)} /></TableCell>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left font-mono text-xs text-slate-700 hover:text-blue-700"
                      onClick={() => pushFilters({ ...filters, proxy_id: event.proxy_id ? String(event.proxy_id) : "", page: 1 })}
                    >
                      {proxyLabel(event)}
                    </button>
                  </TableCell>
                  <TableCell>{sourceLabel(event.source_type)}</TableCell>
                  <TableCell><CopyableText value={event.task_id} /></TableCell>
                  <TableCell><CopyableText value={event.proxy_lease_id || ""} /></TableCell>
                  <TableCell className="max-w-[260px] truncate text-slate-600">{event.error_message || "N/A"}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button variant="ghost" size="sm" onClick={() => setSelectedEvent(event)} aria-label="View error details">
                        <Eye />
                      </Button>
                      {event.proxy_id ? (
                        <Button variant="ghost" size="sm" onClick={() => pushFilters({ ...filters, proxy_id: String(event.proxy_id), page: 1 })}>
                          Filter
                        </Button>
                      ) : null}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {events.length === 0 ? (
            <div className="rounded-lg border border-dashed border-slate-200 bg-slate-50 px-6 py-10 text-center text-sm text-slate-500">
              {loading ? "Loading proxy usage events..." : "No proxy usage events match the current filters."}
            </div>
          ) : null}
          <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm text-slate-500">Page {pagination.page} of {pageCount}</p>
            <div className="flex gap-2">
              <Button variant="outline" disabled={pagination.page <= 1 || loading} onClick={() => pushFilters({ ...filters, page: Math.max(1, pagination.page - 1) })}>
                Previous
              </Button>
              <Button variant="outline" disabled={pagination.page >= pageCount || loading} onClick={() => pushFilters({ ...filters, page: pagination.page + 1 })}>
                Next
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog open={selectedEvent !== null} onOpenChange={(open) => !open && setSelectedEvent(null)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>Proxy Event Details</DialogTitle>
          </DialogHeader>
          {selectedEvent ? (
            <div className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <Detail label="Task ID" value={selectedEvent.task_id || "N/A"} />
                <Detail label="Proxy Lease ID" value={selectedEvent.proxy_lease_id || "N/A"} />
                <Detail label="Proxy" value={proxyLabel(selectedEvent)} />
                <Detail label="Created At" value={formatDateTime(selectedEvent.created_at)} />
              </div>
              <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
                <p className="mb-2 text-sm font-medium text-slate-700">Error Message</p>
                <pre className="max-h-[360px] overflow-auto whitespace-pre-wrap break-words text-xs text-slate-700">{selectedEvent.error_message || "N/A"}</pre>
              </div>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function SummaryGrid({ summary, total }: { summary: ProxyUsageEventSummary; total: number }) {
  const topCategory = summary.category_counts[0];
  const topPlatform = summary.platform_counts[0];
  return (
    <div className="grid gap-3 md:grid-cols-3 xl:grid-cols-6">
      <SummaryTile label="Total" value={String(total)} />
      <SummaryTile label="Success" value={String(summary.success_count)} />
      <SummaryTile label="Failed" value={String(summary.failure_count)} />
      <SummaryTile label="Failure Rate" value={`${Math.round(summary.failure_rate * 1000) / 10}%`} />
      <SummaryTile label="Top Category" value={topCategory ? `${topCategory.key} (${topCategory.count})` : "N/A"} />
      <SummaryTile label="Top Platform" value={topPlatform ? `${topPlatform.key} (${topPlatform.count})` : "N/A"} />
    </div>
  );
}

function SummaryTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border/70 bg-white/90 p-4 shadow-sm">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 truncate text-xl font-semibold text-slate-950" title={value}>{value}</p>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="space-y-1.5 text-sm font-medium text-slate-700">
      <span>{label}</span>
      {children}
    </label>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white px-3 py-2">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-all font-mono text-xs text-slate-800">{value}</p>
    </div>
  );
}

function CopyableText({ value }: { value: string }) {
  if (!value) {
    return <span className="text-slate-400">N/A</span>;
  }
  const shortValue = value.length > 14 ? `${value.slice(0, 6)}...${value.slice(-6)}` : value;
  return (
    <button
      type="button"
      className="inline-flex items-center gap-1 font-mono text-xs text-slate-700 hover:text-blue-700"
      title={value}
      onClick={() => void copyText(value)}
    >
      {shortValue}
      <Copy className="size-3" />
    </button>
  );
}

function filtersFromSearchParams(params: URLSearchParams | ReadonlyURLSearchParams): FilterState {
  const now = new Date();
  const end = params.get("end_time") || now.toISOString();
  const start = params.get("start_time") || new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
  const page = parsePositiveNumber(params.get("page"), 1);
  const pageSize = parsePositiveNumber(params.get("page_size"), 20);
  const success = normalizeSuccess(params.get("success"));
  const sortOrder = params.get("sort_order") === "asc" ? "asc" : "desc";

  return {
    task_id: params.get("task_id") || "",
    proxy_id: params.get("proxy_id") || "",
    proxy_lease_id: params.get("proxy_lease_id") || "",
    source_type: params.get("source_type") || "",
    stage: params.get("stage") || "",
    platform: params.get("platform") || "",
    success,
    error_category: params.get("error_category") || "",
    start_time: toDateTimeLocal(start),
    end_time: toDateTimeLocal(end),
    page,
    page_size: Math.min(pageSize, 100),
    sort_order: sortOrder,
  };
}

function filtersToSearchParams(filters: FilterState) {
  const params = new URLSearchParams();
  appendParam(params, "task_id", filters.task_id);
  appendParam(params, "proxy_id", filters.proxy_id);
  appendParam(params, "proxy_lease_id", filters.proxy_lease_id);
  appendParam(params, "source_type", filters.source_type);
  appendParam(params, "stage", filters.stage);
  appendParam(params, "platform", filters.platform);
  if (filters.success !== "all") {
    params.set("success", filters.success);
  }
  appendParam(params, "error_category", filters.error_category);
  appendParam(params, "start_time", dateTimeLocalToISO(filters.start_time));
  appendParam(params, "end_time", dateTimeLocalToISO(filters.end_time));
  if (filters.page > 1) {
    params.set("page", String(filters.page));
  }
  if (filters.page_size !== 20) {
    params.set("page_size", String(filters.page_size));
  }
  if (filters.sort_order !== "desc") {
    params.set("sort_order", filters.sort_order);
  }
  return params;
}

function toApiParams(filters: FilterState): ListProxyUsageEventsParams {
  return {
    task_id: filters.task_id || undefined,
    proxy_id: filters.proxy_id ? Number(filters.proxy_id) : undefined,
    proxy_lease_id: filters.proxy_lease_id || undefined,
    source_type: filters.source_type || undefined,
    stage: filters.stage || undefined,
    platform: filters.platform || undefined,
    success: filters.success,
    error_category: filters.error_category || undefined,
    start_time: dateTimeLocalToISO(filters.start_time),
    end_time: dateTimeLocalToISO(filters.end_time),
    page: filters.page,
    page_size: filters.page_size,
    sort_order: filters.sort_order,
  };
}

function appendParam(params: URLSearchParams, key: string, value: string | undefined) {
  if (value) {
    params.set(key, value);
  }
}

function parsePositiveNumber(value: string | null, fallback: number) {
  if (!value) {
    return fallback;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function normalizeSuccess(value: string | null): FilterState["success"] {
  return value === "success" || value === "failed" ? value : "all";
}

function toDateTimeLocal(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}

function dateTimeLocalToISO(value: string) {
  if (!value) {
    return undefined;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }
  return date.toISOString();
}

function formatDateTime(value: string) {
  if (!value) {
    return "N/A";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function proxyLabel(event: ProxyUsageEvent) {
  if (!event.proxy_id && !event.proxy_host) {
    return "N/A";
  }
  const endpoint = event.proxy_host ? `${event.proxy_host}${event.proxy_port ? `:${event.proxy_port}` : ""}` : "unknown";
  return `#${event.proxy_id || "N/A"} ${endpoint}`;
}

function sourceLabel(value: string) {
  switch (value) {
    case "manual":
    case "manual_pool":
      return "Manual";
    case "dynamic":
    case "dynamic_api":
      return "Dynamic";
    default:
      return value || "N/A";
  }
}

function categoryTone(category?: string): "success" | "warning" | "danger" | "info" | "neutral" {
  switch (category) {
    case "bot_detected":
      return "danger";
    case "rate_limited":
    case "network_timeout":
    case "proxy_unreachable":
      return "warning";
    case "proxy_auth":
    case "cookie_invalid":
      return "info";
    default:
      return "neutral";
  }
}

async function copyText(value: string) {
  try {
    await navigator.clipboard.writeText(value);
    toast.success("Copied");
  } catch {
    toast.error("Failed to copy");
  }
}
