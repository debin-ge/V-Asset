import type { Overview } from "@/types/stats";
import type { ProxySourcePolicy, ProxySourceStatus } from "@/types/proxy";

export function ResourceSummary({
  overview,
  proxyStatus,
  proxyPolicy,
}: {
  overview: Overview | null;
  proxyStatus: ProxySourceStatus | null;
  proxyPolicy: ProxySourcePolicy | null;
}) {
  return (
    <div className="card">
      <p className="muted" style={{ marginTop: 0 }}>Operational Snapshot</p>
      <div className="grid">
        <div>
          <strong>Success Rate</strong>
          <p className="muted" style={{ marginBottom: 0 }}>
            {overview && overview.total_downloads > 0
              ? `${Math.round((overview.success_downloads / overview.total_downloads) * 100)}%`
              : "N/A"}
          </p>
        </div>
        <div>
          <strong>Failure Load</strong>
          <p className="muted" style={{ marginBottom: 0 }}>{overview?.failed_downloads ?? "-"}</p>
        </div>
        <div>
          <strong>Proxy Health</strong>
          <p className="muted" style={{ marginBottom: 0 }}>{proxyStatus?.healthy ? "Healthy" : "Unavailable"}</p>
        </div>
        <div>
          <strong>Current Lease</strong>
          <p className="muted" style={{ marginBottom: 0 }}>{proxyStatus?.proxy_lease_id || "N/A"}</p>
        </div>
        <div>
          <strong>Proxy Strategy</strong>
          <p className="muted" style={{ marginBottom: 0 }}>
            {proxyPolicy ? `${proxyPolicy.primary_source} -> ${proxyPolicy.fallback_source || "none"}` : "N/A"}
          </p>
        </div>
        <div>
          <strong>Manual Pool</strong>
          <p className="muted" style={{ marginBottom: 0 }}>
            {overview ? `${overview.active_manual_proxies}/${overview.total_manual_proxies} active` : "N/A"}
          </p>
        </div>
      </div>
    </div>
  );
}
