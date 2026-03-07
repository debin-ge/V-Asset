import type { ProxySourcePolicy, UpdateProxySourcePolicyPayload } from "@/types/proxy";

export function ProxyPolicyCard({
  policy,
  onSubmit,
}: {
  policy: ProxySourcePolicy | null;
  onSubmit: (payload: UpdateProxySourcePolicyPayload) => void;
}) {
  if (!policy) {
    return (
      <div className="card">
        <p className="muted">Loading policy...</p>
      </div>
    );
  }

  return (
    <div className="card">
      <div style={{ display: "flex", justifyContent: "space-between", gap: 16, marginBottom: 16 }}>
        <div>
          <p className="muted" style={{ marginTop: 0 }}>Source Policy</p>
          <h3 style={{ margin: 0 }}>Global Proxy Strategy</h3>
        </div>
      </div>
      <form
        className="grid proxy-policy-grid"
        onSubmit={(event) => {
          event.preventDefault();
          const form = new FormData(event.currentTarget);
          onSubmit({
            primary_source: String(form.get("primary_source") || "dynamic_api"),
            fallback_source: String(form.get("fallback_source") || "manual_pool"),
            fallback_enabled: form.get("fallback_enabled") === "on",
            dynamic_timeout_ms: Number(form.get("dynamic_timeout_ms") || 3000),
            dynamic_retry_count: Number(form.get("dynamic_retry_count") || 2),
            dynamic_circuit_breaker_sec: Number(form.get("dynamic_circuit_breaker_sec") || 60),
            min_lease_ttl_sec: Number(form.get("min_lease_ttl_sec") || 600),
            manual_selection_strategy: String(form.get("manual_selection_strategy") || "lru"),
          });
        }}
      >
        <label>
          <div className="muted">Primary Source</div>
          <select className="select" name="primary_source" defaultValue={policy.primary_source}>
            <option value="dynamic_api">Dynamic API</option>
            <option value="manual_pool">Manual Pool</option>
          </select>
        </label>
        <label>
          <div className="muted">Fallback Source</div>
          <select className="select" name="fallback_source" defaultValue={policy.fallback_source || "manual_pool"}>
            <option value="manual_pool">Manual Pool</option>
            <option value="dynamic_api">Dynamic API</option>
          </select>
        </label>
        <label style={{ display: "flex", alignItems: "center", gap: 8, paddingTop: 24 }}>
          <input type="checkbox" name="fallback_enabled" defaultChecked={policy.fallback_enabled} />
          <span>Enable Fallback</span>
        </label>
        <label>
          <div className="muted">Dynamic Timeout (ms)</div>
          <input className="field" name="dynamic_timeout_ms" type="number" defaultValue={policy.dynamic_timeout_ms} />
        </label>
        <label>
          <div className="muted">Dynamic Retry Count</div>
          <input className="field" name="dynamic_retry_count" type="number" defaultValue={policy.dynamic_retry_count} />
        </label>
        <label>
          <div className="muted">Circuit Breaker (sec)</div>
          <input className="field" name="dynamic_circuit_breaker_sec" type="number" defaultValue={policy.dynamic_circuit_breaker_sec} />
        </label>
        <label>
          <div className="muted">Min Lease TTL (sec)</div>
          <input className="field" name="min_lease_ttl_sec" type="number" defaultValue={policy.min_lease_ttl_sec} />
        </label>
        <label>
          <div className="muted">Manual Selection Strategy</div>
          <select className="select" name="manual_selection_strategy" defaultValue={policy.manual_selection_strategy}>
            <option value="lru">LRU</option>
          </select>
        </label>
        <div style={{ display: "flex", alignItems: "end" }}>
          <button className="button" type="submit">Save Policy</button>
        </div>
      </form>
    </div>
  );
}
