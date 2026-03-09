import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
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
      <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
        <CardContent className="py-10 text-sm text-muted-foreground">Loading policy...</CardContent>
      </Card>
    );
  }

  return (
    <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
      <CardHeader>
        <CardTitle>Global Proxy Strategy</CardTitle>
        <CardDescription>配置主代理源、回退策略和租约阈值。</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
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
          <Field label="Primary Source">
            <NativeSelect name="primary_source" defaultValue={policy.primary_source}>
              <option value="dynamic_api">Dynamic API</option>
              <option value="manual_pool">Manual Pool</option>
            </NativeSelect>
          </Field>
          <Field label="Fallback Source">
            <NativeSelect name="fallback_source" defaultValue={policy.fallback_source || "manual_pool"}>
              <option value="manual_pool">Manual Pool</option>
              <option value="dynamic_api">Dynamic API</option>
            </NativeSelect>
          </Field>
          <label className="flex min-h-20 items-center gap-3 rounded-2xl border border-border/60 bg-muted/35 px-4">
            <input type="checkbox" name="fallback_enabled" defaultChecked={policy.fallback_enabled} />
            <span className="text-sm font-medium text-foreground">Enable Fallback</span>
          </label>
          <Field label="Dynamic Timeout (ms)">
            <Input name="dynamic_timeout_ms" type="number" defaultValue={policy.dynamic_timeout_ms} />
          </Field>
          <Field label="Dynamic Retry Count">
            <Input name="dynamic_retry_count" type="number" defaultValue={policy.dynamic_retry_count} />
          </Field>
          <Field label="Circuit Breaker (sec)">
            <Input name="dynamic_circuit_breaker_sec" type="number" defaultValue={policy.dynamic_circuit_breaker_sec} />
          </Field>
          <Field label="Min Lease TTL (sec)">
            <Input name="min_lease_ttl_sec" type="number" defaultValue={policy.min_lease_ttl_sec} />
          </Field>
          <Field label="Manual Selection Strategy">
            <NativeSelect name="manual_selection_strategy" defaultValue={policy.manual_selection_strategy}>
              <option value="lru">LRU</option>
            </NativeSelect>
          </Field>
          <div className="flex items-end">
            <Button className="w-full md:w-auto" type="submit">
              Save Policy
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="grid gap-2 rounded-2xl border border-border/60 bg-muted/35 p-4">
      <span className="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{label}</span>
      {children}
    </label>
  );
}

function NativeSelect(props: React.ComponentProps<"select">) {
  return (
    <select
      {...props}
      className="h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
    />
  );
}
