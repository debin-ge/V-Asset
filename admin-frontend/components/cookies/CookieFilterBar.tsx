import { Filter, Plus, RefreshCcw } from "lucide-react";

import { Button } from "@/components/ui/button";

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
    <div className="flex flex-col gap-3 rounded-[24px] border border-border/60 bg-white/80 p-4 shadow-sm lg:flex-row lg:items-center lg:justify-between">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
          <Filter className="size-4" />
          Filter
        </div>
        <select
          className="h-8 min-w-[180px] rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
          value={platform}
          onChange={(e) => onPlatformChange(e.target.value)}
        >
          <option value="">All Platforms</option>
          <option value="youtube">YouTube</option>
          <option value="bilibili">Bilibili</option>
          <option value="tiktok">TikTok</option>
          <option value="twitter">Twitter</option>
          <option value="instagram">Instagram</option>
        </select>
        <Button variant="outline" onClick={onRefresh}>
          <RefreshCcw data-icon="inline-start" />
          Refresh
        </Button>
      </div>
      <Button onClick={onCreateToggle}>
        <Plus data-icon="inline-start" />
        {creating ? "Close Form" : "Add Cookie"}
      </Button>
    </div>
  );
}
