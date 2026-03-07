import type { RequestTrend } from "@/types/stats";

export function RequestTrendChart({ trend }: { trend: RequestTrend | null }) {
  return (
    <div className="card">
      <p className="muted" style={{ marginTop: 0 }}>Request Trend</p>
      {!trend ? (
        <p className="muted">No data</p>
      ) : (
        <div className="grid">
          {trend.points.map((point) => (
            <div key={point.label} style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
              <span>{point.label}</span>
              <strong>{point.count}</strong>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

