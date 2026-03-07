import type { CookieInfo } from "@/types/cookie";

export function CookieTable({
  items,
  onDelete,
  onFreeze,
}: {
  items: CookieInfo[];
  onDelete: (id: number) => void;
  onFreeze: (id: number) => void;
}) {
  return (
    <div className="card">
      <table className="table">
        <thead>
          <tr>
            <th>Platform</th>
            <th>Name</th>
            <th>Status</th>
            <th>Expire At</th>
            <th>Frozen Until</th>
            <th>Usage</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id}>
              <td>{item.platform}</td>
              <td>{item.name}</td>
              <td>
                <span className={`status-badge ${statusClassName(item.status)}`}>
                  {statusLabel(item.status)}
                </span>
              </td>
              <td>{item.expire_at || "N/A"}</td>
              <td>{item.frozen_until || "N/A"}</td>
              <td>{item.use_count}</td>
              <td>
                <div className="inline-actions">
                  <button className="button ghost" onClick={() => onFreeze(item.id)}>Freeze</button>
                  <button className="button secondary" onClick={() => onDelete(item.id)}>Delete</button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function statusLabel(status: number) {
  switch (status) {
    case 0:
      return "Active";
    case 1:
      return "Expired";
    case 2:
      return "Frozen";
    default:
      return `Unknown(${status})`;
  }
}

function statusClassName(status: number) {
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
