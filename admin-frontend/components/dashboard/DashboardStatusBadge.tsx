import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { DashboardSeverity } from "@/components/dashboard/dashboard-helpers";
import { statusLabel } from "@/components/dashboard/dashboard-helpers";

const severityClass: Record<DashboardSeverity, string> = {
  healthy: "border-chart-4/30 bg-chart-4/10 text-chart-4",
  warning: "border-chart-5/30 bg-chart-5/10 text-chart-5",
  critical: "border-destructive/30 bg-destructive/10 text-destructive",
  neutral: "border-border bg-secondary text-secondary-foreground",
};

export function DashboardStatusBadge({
  status,
  label,
  className,
}: {
  status: DashboardSeverity;
  label?: string;
  className?: string;
}) {
  return (
    <Badge variant="outline" className={cn(severityClass[status], className)}>
      {label || statusLabel(status)}
    </Badge>
  );
}
