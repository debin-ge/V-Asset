import type { ProxySourceStatus } from "@/types/proxy";

export function ProxyStatusCard({ status }: { status: ProxySourceStatus | null }) {
  return (
    <div className="card">
      <p className="muted" style={{ marginTop: 0 }}>Proxy Source</p>
      {!status ? (
        <p className="muted">Loading...</p>
      ) : (
        <div className="grid">
          <div><strong>Status:</strong> {status.healthy ? "Healthy" : "Unavailable"}</div>
          <div><strong>Mode:</strong> {status.mode}</div>
          <div><strong>Lease:</strong> {status.proxy_lease_id || "N/A"}</div>
          <div><strong>Expires:</strong> {status.proxy_expire_at || "N/A"}</div>
          <div><strong>Checked:</strong> {status.checked_at}</div>
          <div><strong>Message:</strong> {status.message}</div>
        </div>
      )}
    </div>
  );
}

