import * as React from "react";
import Link from "next/link";
import { ArrowDown, ArrowUp, ChevronsUpDown, Eye } from "lucide-react";

import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import type { ProxyCreatePayload, ProxyInfo, ProxyListSortBy, ProxyUpdatePayload } from "@/types/proxy";

const pageSizeOptions = [20, 50, 100] as const;

export function ProxyTable({
  items,
  loading = false,
  hasActiveFilters = false,
  page,
  pageSize,
  total,
  sortBy,
  sortOrder,
  onPageChange,
  onPageSizeChange,
  onSortChange,
  createOpen,
  onCreateOpenChange,
  onCreate,
  onUpdate,
  onUpdateStatus,
  onDelete,
}: {
  items: ProxyInfo[];
  loading?: boolean;
  hasActiveFilters?: boolean;
  page: number;
  pageSize: number;
  total: number;
  sortBy: ProxyListSortBy | "";
  sortOrder: "asc" | "desc";
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  onSortChange: (sortBy: ProxyListSortBy) => void;
  createOpen?: boolean;
  onCreateOpenChange?: (open: boolean) => void;
  onCreate: (payload: ProxyCreatePayload) => Promise<void> | void;
  onUpdate: (id: number, payload: ProxyUpdatePayload) => Promise<void> | void;
  onUpdateStatus: (id: number, status: number) => void;
  onDelete: (id: number) => void;
}) {
  const [editingItem, setEditingItem] = React.useState<ProxyInfo | null>(null);
  const [detailItem, setDetailItem] = React.useState<ProxyInfo | null>(null);
  const [internalCreateOpen, setInternalCreateOpen] = React.useState(false);

  const isCreateOpen = createOpen ?? internalCreateOpen;
  const setCreateOpen = onCreateOpenChange ?? setInternalCreateOpen;

  const pageCount = React.useMemo(() => {
    if (total === 0) {
      return 1;
    }
    return Math.max(1, Math.ceil(total / pageSize));
  }, [pageSize, total]);

  return (
    <Card className="overflow-hidden rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardHeader className="flex flex-col gap-2 pb-3 md:flex-row md:items-center md:justify-between">
        <div>
          <CardTitle>Configured Proxies</CardTitle>
          <CardDescription>维护手动代理池，支持在线编辑、状态切换、事件追踪和分页查看。</CardDescription>
        </div>
        <div className="text-sm text-muted-foreground">
          {loading ? "Loading..." : `${total} proxies`}
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="overflow-hidden rounded-lg border border-border/70">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/40 hover:bg-muted/40">
                <TableHead>Proxy</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Region</TableHead>
                <SortHeader label="Priority" sortKey="priority" sortBy={sortBy} sortOrder={sortOrder} onSortChange={onSortChange} />
                <SortHeader label="Risk" sortKey="risk_score" sortBy={sortBy} sortOrder={sortOrder} onSortChange={onSortChange} />
                <SortHeader label="Success / Fail" sortKey="fail_count" sortBy={sortBy} sortOrder={sortOrder} onSortChange={onSortChange} />
                <SortHeader label="Concurrency" sortKey="active_task_count" sortBy={sortBy} sortOrder={sortOrder} onSortChange={onSortChange} />
                <TableHead>Cooldown</TableHead>
                <TableHead>Last Error</TableHead>
                <SortHeader label="Last Used" sortKey="last_used_at" sortBy={sortBy} sortOrder={sortOrder} onSortChange={onSortChange} />
                <SortHeader label="Updated" sortKey="updated_at" sortBy={sortBy} sortOrder={sortOrder} onSortChange={onSortChange} />
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item) => (
                <TableRow key={item.id} className={cn(item.risk_score >= 70 && "bg-destructive/5")}>
                  <TableCell className="min-w-[220px]">
                    <div className="flex flex-col gap-1">
                      <span className="font-mono text-xs font-medium text-foreground">
                        {item.protocol}://{item.host}:{item.port}
                      </span>
                      <span className="max-w-[240px] truncate text-xs text-muted-foreground">
                        {item.platform_tags || item.remark || "Any platform"}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <StatusBadge label={proxyStatusLabel(item.status)} tone={proxyStatusTone(item.status)} />
                  </TableCell>
                  <TableCell>{item.region || "N/A"}</TableCell>
                  <TableCell>{item.priority}</TableCell>
                  <TableCell>
                    <StatusBadge label={`${item.risk_score}/100`} tone={riskTone(item.risk_score)} />
                  </TableCell>
                  <TableCell>
                    <span className="font-medium text-foreground">{item.success_count}</span>
                    <span className="text-muted-foreground"> / {item.fail_count}</span>
                  </TableCell>
                  <TableCell>
                    <StatusBadge
                      label={`${item.active_task_count}/${item.max_concurrent}`}
                      tone={concurrencyTone(item)}
                    />
                  </TableCell>
                  <TableCell>
                    <StatusBadge
                      label={isCoolingDown(item) ? "Cooling" : "None"}
                      tone={isCoolingDown(item) ? "warning" : "neutral"}
                    />
                  </TableCell>
                  <TableCell className="max-w-[180px] truncate text-muted-foreground">
                    {item.last_error_category || "None"}
                  </TableCell>
                  <TableCell className="text-muted-foreground">{formatDateTime(item.last_used_at)}</TableCell>
                  <TableCell className="text-muted-foreground">{formatDateTime(item.updated_at)}</TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button variant="ghost" size="icon-sm" onClick={() => setDetailItem(item)} aria-label="View proxy details">
                        <Eye />
                      </Button>
                      <Link className={buttonVariants({ variant: "ghost", size: "sm" })} href={`/proxies/events?proxy_id=${item.id}`}>
                        Events
                      </Link>
                      <Button variant="ghost" size="sm" onClick={() => setEditingItem(item)}>
                        Edit
                      </Button>
                      {item.status === 0 ? (
                        <Button variant="outline" size="sm" onClick={() => onUpdateStatus(item.id, 1)}>
                          Disable
                        </Button>
                      ) : (
                        <Button variant="ghost" size="sm" onClick={() => onUpdateStatus(item.id, 0)}>
                          Enable
                        </Button>
                      )}
                      <ConfirmDialog
                        trigger={<Button variant="outline" size="sm">Delete</Button>}
                        title="Delete proxy?"
                        description={`This will permanently remove proxy ${item.host}:${item.port} from the manual pool.`}
                        actionLabel="Delete"
                        onConfirm={() => onDelete(item.id)}
                      />
                    </div>
                  </TableCell>
                </TableRow>
              ))}

              {items.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={12} className="py-10 text-center text-sm text-muted-foreground">
                    {loading
                      ? "Loading proxies..."
                      : hasActiveFilters
                        ? "No proxies match the current filters."
                        : "No proxies configured yet."}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>

        <div className="flex flex-col gap-3 rounded-lg border border-border/70 bg-muted/35 p-3 md:flex-row md:items-center md:justify-between">
          <p className="text-sm text-muted-foreground">
            Page {page} / {pageCount} · {total} proxies
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <label htmlFor="proxy-page-size" className="text-sm text-muted-foreground">Per page</label>
            <NativeSelect
              id="proxy-page-size"
              className="w-auto min-w-20"
              value={String(pageSize)}
              onChange={(event) => onPageSizeChange(Number(event.target.value))}
            >
              {pageSizeOptions.map((size) => (
                <option key={size} value={size}>{size}</option>
              ))}
            </NativeSelect>
            <Button
              variant="outline"
              disabled={page <= 1 || loading}
              onClick={() => onPageChange(Math.max(1, page - 1))}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              disabled={page >= pageCount || loading}
              onClick={() => onPageChange(Math.min(pageCount, page + 1))}
            >
              Next
            </Button>
          </div>
        </div>

        <Dialog open={isCreateOpen} onOpenChange={setCreateOpen}>
          <DialogContent className="max-w-4xl">
            <ProxyCreateForm
              onCancel={() => setCreateOpen(false)}
              onSubmit={async (payload) => {
                await onCreate(payload);
                setCreateOpen(false);
              }}
            />
          </DialogContent>
        </Dialog>
        <Dialog open={Boolean(editingItem)} onOpenChange={(open) => { if (!open) setEditingItem(null); }}>
          <DialogContent className="max-w-4xl">
            {editingItem ? (
              <ProxyEditForm
                item={editingItem}
                onCancel={() => setEditingItem(null)}
                onSubmit={async (payload) => {
                  await onUpdate(editingItem.id, payload);
                  setEditingItem(null);
                }}
              />
            ) : null}
          </DialogContent>
        </Dialog>
        <Dialog open={Boolean(detailItem)} onOpenChange={(open) => { if (!open) setDetailItem(null); }}>
          <DialogContent className="max-w-3xl">
            {detailItem ? <ProxyDetail item={detailItem} /> : null}
          </DialogContent>
        </Dialog>
      </CardContent>
    </Card>
  );
}

function SortHeader({
  label,
  sortKey,
  sortBy,
  sortOrder,
  onSortChange,
}: {
  label: string;
  sortKey: ProxyListSortBy;
  sortBy: ProxyListSortBy | "";
  sortOrder: "asc" | "desc";
  onSortChange: (sortBy: ProxyListSortBy) => void;
}) {
  const active = sortBy === sortKey;
  const Icon = active ? (sortOrder === "asc" ? ArrowUp : ArrowDown) : ChevronsUpDown;

  return (
    <TableHead>
      <Button variant="ghost" size="sm" className="-ml-2" onClick={() => onSortChange(sortKey)}>
        {label}
        <Icon data-icon="inline-end" />
      </Button>
    </TableHead>
  );
}

function ProxyEditForm({
  item,
  onSubmit,
  onCancel,
}: {
  item: ProxyInfo;
  onSubmit: (payload: ProxyUpdatePayload) => Promise<void> | void;
  onCancel: () => void;
}) {
  return (
    <form
      className="grid gap-4"
      onSubmit={async (event) => {
        event.preventDefault();
        const form = new FormData(event.currentTarget);
        const password = String(form.get("password") || "");
        await onSubmit({
          host: String(form.get("host") || ""),
          port: Number(form.get("port") || 0),
          protocol: String(form.get("protocol") || "http"),
          username: String(form.get("username") || ""),
          region: String(form.get("region") || ""),
          priority: Number(form.get("priority") || 0),
          platform_tags: String(form.get("platform_tags") || ""),
          remark: String(form.get("remark") || ""),
          ...(password ? { password } : {}),
        });
      }}
    >
      <DialogHeader>
        <DialogTitle>Edit Proxy</DialogTitle>
        <DialogDescription>更新代理主机、认证信息和优先级。</DialogDescription>
      </DialogHeader>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Input name="host" defaultValue={item.host} placeholder="Host / IP / Domain" />
        <Input name="port" type="number" defaultValue={item.port} placeholder="Port" />
        <NativeSelect name="protocol" defaultValue={item.protocol}>
          <option value="http">HTTP</option>
          <option value="https">HTTPS</option>
          <option value="socks5">SOCKS5</option>
        </NativeSelect>
        <Input name="username" defaultValue={item.username || ""} placeholder="Username" />
        <Input name="password" placeholder="Password (leave empty to keep current)" />
        <Input name="region" defaultValue={item.region || ""} placeholder="Region" />
        <Input name="priority" type="number" defaultValue={item.priority} placeholder="Priority" />
        <Input name="platform_tags" defaultValue={item.platform_tags || ""} placeholder="Platform Tags" />
        <Input name="remark" defaultValue={item.remark || ""} placeholder="Remark" />
      </div>
      <DialogFooter>
        <Button variant="outline" type="button" onClick={onCancel}>Cancel</Button>
        <Button type="submit">Save</Button>
      </DialogFooter>
    </form>
  );
}

function ProxyCreateForm({
  onSubmit,
  onCancel,
}: {
  onSubmit: (payload: ProxyCreatePayload) => Promise<void> | void;
  onCancel: () => void;
}) {
  return (
    <form
      className="grid gap-4"
      onSubmit={async (event) => {
        event.preventDefault();
        const form = new FormData(event.currentTarget);
        await onSubmit({
          host: String(form.get("host") || ""),
          port: Number(form.get("port") || 0),
          protocol: String(form.get("protocol") || "http"),
          username: String(form.get("username") || ""),
          password: String(form.get("password") || ""),
          region: String(form.get("region") || ""),
          priority: Number(form.get("priority") || 0),
          platform_tags: String(form.get("platform_tags") || ""),
          remark: String(form.get("remark") || ""),
          status: 0,
        } as ProxyCreatePayload);
        event.currentTarget.reset();
      }}
    >
      <DialogHeader>
        <DialogTitle>Add Proxy</DialogTitle>
        <DialogDescription>录入新的代理节点，默认会加入手动池并启用。</DialogDescription>
      </DialogHeader>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Input name="host" placeholder="Host / IP / Domain" />
        <Input name="port" type="number" placeholder="Port" />
        <NativeSelect name="protocol" defaultValue="http">
          <option value="http">HTTP</option>
          <option value="https">HTTPS</option>
          <option value="socks5">SOCKS5</option>
        </NativeSelect>
        <Input name="username" placeholder="Username" />
        <Input name="password" placeholder="Password" />
        <Input name="region" placeholder="Region" />
        <Input name="priority" type="number" placeholder="Priority" />
        <Input name="platform_tags" placeholder="Platform Tags" />
        <Input name="remark" placeholder="Remark" />
      </div>
      <DialogFooter>
        <Button variant="outline" type="button" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Add Proxy</Button>
      </DialogFooter>
    </form>
  );
}

function ProxyDetail({ item }: { item: ProxyInfo }) {
  return (
    <div className="flex flex-col gap-4">
      <DialogHeader>
        <DialogTitle>Proxy Details</DialogTitle>
        <DialogDescription>{item.protocol}://{item.host}:{item.port}</DialogDescription>
      </DialogHeader>
      <div className="grid gap-3 sm:grid-cols-2">
        <Detail label="Status" value={proxyStatusLabel(item.status)} />
        <Detail label="Region" value={item.region || "N/A"} />
        <Detail label="Priority" value={String(item.priority)} />
        <Detail label="Risk Score" value={`${item.risk_score}/100`} />
        <Detail label="Failure Streak" value={String(item.consecutive_fail_count)} />
        <Detail label="Last Fail At" value={formatDateTime(item.last_fail_at)} />
        <Detail label="Cooldown Until" value={formatDateTime(item.cooldown_until)} />
        <Detail label="Last Used At" value={formatDateTime(item.last_used_at)} />
        <Detail label="Created At" value={formatDateTime(item.created_at)} />
        <Detail label="Updated At" value={formatDateTime(item.updated_at)} />
      </div>
      <div className="grid gap-3">
        <Detail label="Platform Tags" value={item.platform_tags || "Any"} multiline />
        <Detail label="Last Error" value={item.last_error_category || "None"} multiline />
        <Detail label="Remark" value={item.remark || "No remark"} multiline />
      </div>
    </div>
  );
}

function Detail({ label, value, multiline = false }: { label: string; value: string; multiline?: boolean }) {
  return (
    <div className="rounded-lg border border-border/70 bg-muted/35 p-3">
      <p className="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">{label}</p>
      <p className={cn("mt-1 text-sm font-medium text-foreground", multiline && "whitespace-pre-wrap break-words")}>{value}</p>
    </div>
  );
}

function proxyStatusLabel(status: number) {
  switch (status) {
    case 0:
      return "Active";
    case 1:
      return "Inactive";
    case 2:
      return "Checking";
    default:
      return `Unknown(${status})`;
  }
}

function proxyStatusTone(status: number): "success" | "danger" | "info" | "neutral" {
  switch (status) {
    case 0:
      return "success";
    case 1:
      return "danger";
    case 2:
      return "info";
    default:
      return "neutral";
  }
}

function riskTone(score: number): "success" | "warning" | "danger" {
  if (score >= 70) {
    return "danger";
  }
  if (score >= 40) {
    return "warning";
  }
  return "success";
}

function concurrencyTone(item: ProxyInfo): "warning" | "info" | "neutral" {
  if (item.max_concurrent > 0 && item.active_task_count >= item.max_concurrent) {
    return "warning";
  }
  if (item.active_task_count > 0) {
    return "info";
  }
  return "neutral";
}

function isCoolingDown(item: ProxyInfo) {
  const cooldownUntil = parseTime(item.cooldown_until);
  return cooldownUntil !== null && cooldownUntil > Date.now();
}

function formatDateTime(value?: string) {
  const timestamp = parseTime(value);
  if (timestamp === null) {
    return "N/A";
  }
  return new Date(timestamp).toLocaleString();
}

function parseTime(value?: string) {
  if (!value) {
    return null;
  }
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? null : timestamp;
}

function NativeSelect({ className, ...props }: React.ComponentProps<"select">) {
  return (
    <select
      {...props}
      className={cn(
        "h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50",
        className
      )}
    />
  );
}
