import * as React from "react";

import type { ProxyCreatePayload, ProxyInfo, ProxyUpdatePayload } from "@/types/proxy";

export function ProxyTable({
  items,
  onCreate,
  onUpdate,
  onUpdateStatus,
  onDelete,
}: {
  items: ProxyInfo[];
  onCreate: (payload: ProxyCreatePayload) => void;
  onUpdate: (id: number, payload: ProxyUpdatePayload) => void;
  onUpdateStatus: (id: number, status: number) => void;
  onDelete: (id: number) => void;
}) {
  const [editingId, setEditingId] = React.useState<number | null>(null);

  return (
    <div className="card">
      <div style={{ display: "flex", justifyContent: "space-between", gap: 16, marginBottom: 16 }}>
        <div>
          <p className="muted" style={{ marginTop: 0 }}>Manual Pool</p>
          <h3 style={{ margin: 0 }}>Configured Proxies</h3>
        </div>
      </div>
      <ProxyCreateForm onSubmit={onCreate} />
      <table className="table">
        <thead>
          <tr>
            <th>Host</th>
            <th>Protocol</th>
            <th>Region</th>
            <th>Priority</th>
            <th>Status</th>
            <th>Usage</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <React.Fragment key={item.id}>
              <tr>
                <td>{item.host}:{item.port}</td>
                <td>{item.protocol}</td>
                <td>{item.region || "N/A"}</td>
                <td>{item.priority}</td>
                <td>
                  <span className={`status-badge ${proxyStatusClassName(item.status)}`}>
                    {proxyStatusLabel(item.status)}
                  </span>
                </td>
                <td>{item.success_count}/{item.fail_count}</td>
                <td>
                  <div className="inline-actions">
                    <button
                      className="button ghost"
                      onClick={() => setEditingId((current) => current === item.id ? null : item.id)}
                    >
                      {editingId === item.id ? "Cancel" : "Edit"}
                    </button>
                    {item.status === 0 ? (
                      <button className="button secondary" onClick={() => onUpdateStatus(item.id, 1)}>Disable</button>
                    ) : (
                      <button className="button ghost" onClick={() => onUpdateStatus(item.id, 0)}>Enable</button>
                    )}
                    <button className="button secondary" onClick={() => onDelete(item.id)}>Delete</button>
                  </div>
                </td>
              </tr>
              {editingId === item.id ? (
                <tr>
                  <td colSpan={7}>
                    <ProxyEditForm
                      item={item}
                      onCancel={() => setEditingId(null)}
                      onSubmit={(payload) => {
                        onUpdate(item.id, payload);
                        setEditingId(null);
                      }}
                    />
                  </td>
                </tr>
              ) : null}
            </React.Fragment>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ProxyEditForm({
  item,
  onSubmit,
  onCancel,
}: {
  item: ProxyInfo;
  onSubmit: (payload: ProxyUpdatePayload) => void;
  onCancel: () => void;
}) {
  return (
    <form
      className="grid proxy-form-grid"
      style={{ padding: "8px 0" }}
      onSubmit={(event) => {
        event.preventDefault();
        const form = new FormData(event.currentTarget);
        const password = String(form.get("password") || "");
        onSubmit({
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
      <input className="field" name="host" defaultValue={item.host} placeholder="Host / IP / Domain" />
      <input className="field" name="port" type="number" defaultValue={item.port} placeholder="Port" />
      <select className="select" name="protocol" defaultValue={item.protocol}>
        <option value="http">HTTP</option>
        <option value="https">HTTPS</option>
        <option value="socks5">SOCKS5</option>
      </select>
      <input className="field" name="username" defaultValue={item.username || ""} placeholder="Username" />
      <input className="field" name="password" placeholder="Password (leave empty to keep current)" />
      <input className="field" name="region" defaultValue={item.region || ""} placeholder="Region" />
      <input className="field" name="priority" type="number" defaultValue={item.priority} placeholder="Priority" />
      <input className="field" name="platform_tags" defaultValue={item.platform_tags || ""} placeholder="Platform Tags" />
      <input className="field" name="remark" defaultValue={item.remark || ""} placeholder="Remark" />
      <div className="inline-actions">
        <button className="button" type="submit">Save</button>
        <button className="button secondary" type="button" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}

function ProxyCreateForm({ onSubmit }: { onSubmit: (payload: ProxyCreatePayload) => void }) {
  return (
    <form
      className="grid proxy-form-grid"
      style={{ marginBottom: 20 }}
      onSubmit={(event) => {
        event.preventDefault();
        const form = new FormData(event.currentTarget);
        onSubmit({
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
      <input className="field" name="host" placeholder="Host / IP / Domain" />
      <input className="field" name="port" type="number" placeholder="Port" />
      <select className="select" name="protocol" defaultValue="http">
        <option value="http">HTTP</option>
        <option value="https">HTTPS</option>
        <option value="socks5">SOCKS5</option>
      </select>
      <input className="field" name="username" placeholder="Username" />
      <input className="field" name="password" placeholder="Password" />
      <input className="field" name="region" placeholder="Region" />
      <input className="field" name="priority" type="number" placeholder="Priority" />
      <input className="field" name="platform_tags" placeholder="Platform Tags" />
      <input className="field" name="remark" placeholder="Remark" />
      <button className="button" type="submit">Add Proxy</button>
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

function proxyStatusClassName(status: number) {
  switch (status) {
    case 0:
      return "status-active";
    case 1:
      return "status-expired";
    case 2:
      return "status-frozen";
    default:
      return "";
  }
}
