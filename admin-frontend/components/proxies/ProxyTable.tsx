import * as React from "react";

import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Button } from "@/components/ui/button";
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
import type { ProxyCreatePayload, ProxyInfo, ProxyUpdatePayload } from "@/types/proxy";

export function ProxyTable({
  items,
  onCreate,
  onUpdate,
  onUpdateStatus,
  onDelete,
}: {
  items: ProxyInfo[];
  onCreate: (payload: ProxyCreatePayload) => Promise<void> | void;
  onUpdate: (id: number, payload: ProxyUpdatePayload) => Promise<void> | void;
  onUpdateStatus: (id: number, status: number) => void;
  onDelete: (id: number) => void;
}) {
  const [editingItem, setEditingItem] = React.useState<ProxyInfo | null>(null);
  const [createOpen, setCreateOpen] = React.useState(false);

  return (
    <Card className="overflow-hidden rounded-[32px] border-white/60 bg-white/78 shadow-xl shadow-blue-950/5 backdrop-blur-xl">
      <CardHeader>
        <CardTitle>Configured Proxies</CardTitle>
        <CardDescription>维护手动代理池，支持在线编辑与状态切换。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex justify-end">
          <Button className="rounded-full" onClick={() => setCreateOpen(true)}>Add Proxy</Button>
        </div>
        <div className="space-y-4">
          {items.map((item) => (
            <div
              key={item.id}
              className="rounded-[28px] border border-white/70 bg-white/90 p-5 shadow-sm transition-transform hover:-translate-y-0.5"
            >
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-4">
                  <div className="flex flex-wrap items-center gap-3">
                    <div className="rounded-2xl bg-gradient-to-br from-blue-500 to-violet-600 px-4 py-3 text-white shadow-lg shadow-blue-500/20">
                      <p className="text-xs uppercase tracking-[0.18em] text-white/70">{item.protocol}</p>
                      <p className="text-lg font-semibold">{item.host}:{item.port}</p>
                    </div>
                    <StatusBadge label={proxyStatusLabel(item.status)} tone={proxyStatusTone(item.status)} />
                    <MetaPill label="Region" value={item.region || "N/A"} />
                    <MetaPill label="Priority" value={String(item.priority)} />
                  </div>
                  <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                    <InfoStat label="Usage" value={`${item.success_count}/${item.fail_count}`} />
                    <InfoStat label="Platforms" value={item.platform_tags || "Any"} />
                    <InfoStat label="Remark" value={item.remark || "No remark"} />
                    <InfoStat label="Updated" value={item.updated_at} />
                  </div>
                </div>
                <div className="flex flex-wrap gap-2 lg:justify-end">
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
              </div>
            </div>
          ))}
          {items.length === 0 ? (
            <div className="rounded-[28px] border border-dashed border-slate-200 bg-white/70 px-6 py-12 text-center text-sm text-slate-500">
              No proxies configured yet.
            </div>
          ) : null}
        </div>
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
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
      </CardContent>
    </Card>
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

function NativeSelect(props: React.ComponentProps<"select">) {
  return (
    <select
      {...props}
      className="h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
    />
  );
}

function MetaPill({ label, value }: { label: string; value: string }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs text-slate-600">
      <span className="font-medium text-slate-400">{label}</span>
      {value}
    </span>
  );
}

function InfoStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-slate-50/80 px-4 py-3">
      <p className="text-[11px] uppercase tracking-[0.14em] text-slate-400">{label}</p>
      <p className="mt-1 text-sm font-medium text-slate-900">{value}</p>
    </div>
  );
}
