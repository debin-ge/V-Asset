export function MetricCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="card">
      <p className="muted" style={{ margin: 0 }}>{label}</p>
      <p className="metric-value">{value}</p>
    </div>
  );
}

