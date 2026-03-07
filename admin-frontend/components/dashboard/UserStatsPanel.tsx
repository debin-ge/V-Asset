import type { UserStats } from "@/types/stats";

export function UserStatsPanel({ stats }: { stats: UserStats | null }) {
  return (
    <div className="card">
      <p className="muted" style={{ marginTop: 0 }}>User Activity Window</p>
      {!stats ? (
        <p className="muted">Loading...</p>
      ) : (
        <div className="grid">
          <div>
            <p className="muted" style={{ margin: 0 }}>Total Registered Users</p>
            <p className="metric-value">{stats.total_users}</p>
          </div>
          <div>
            <p className="muted" style={{ margin: 0 }}>Daily Active Users</p>
            <p className="metric-value">{stats.daily_active_users}</p>
          </div>
          <div>
            <p className="muted" style={{ margin: 0 }}>Weekly Active Users</p>
            <p className="metric-value">{stats.weekly_active_users}</p>
          </div>
        </div>
      )}
    </div>
  );
}

