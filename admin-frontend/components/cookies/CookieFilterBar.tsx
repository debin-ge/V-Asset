export function CookieFilterBar({
  platform,
  onPlatformChange,
  onRefresh,
  onCreateToggle,
  creating,
}: {
  platform: string;
  onPlatformChange: (value: string) => void;
  onRefresh: () => void;
  onCreateToggle: () => void;
  creating: boolean;
}) {
  return (
    <div className="toolbar">
      <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
        <select className="select" value={platform} onChange={(e) => onPlatformChange(e.target.value)} style={{ minWidth: 180 }}>
          <option value="">All Platforms</option>
          <option value="youtube">YouTube</option>
          <option value="bilibili">Bilibili</option>
          <option value="tiktok">TikTok</option>
          <option value="twitter">Twitter</option>
          <option value="instagram">Instagram</option>
        </select>
        <button className="button secondary" onClick={onRefresh}>Refresh</button>
      </div>
      <button className="button" onClick={onCreateToggle}>
        {creating ? "Close Form" : "Add Cookie"}
      </button>
    </div>
  );
}

