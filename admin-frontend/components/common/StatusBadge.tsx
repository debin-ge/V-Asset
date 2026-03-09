import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type StatusTone = "success" | "warning" | "danger" | "info" | "neutral";

const toneClassName: Record<StatusTone, string> = {
  success: "border-emerald-500/20 bg-emerald-500/10 text-emerald-700",
  warning: "border-amber-500/20 bg-amber-500/10 text-amber-700",
  danger: "border-rose-500/20 bg-rose-500/10 text-rose-700",
  info: "border-sky-500/20 bg-sky-500/10 text-sky-700",
  neutral: "border-border bg-muted text-muted-foreground",
};

export function StatusBadge({
  label,
  tone = "neutral",
  className,
}: {
  label: string;
  tone?: StatusTone;
  className?: string;
}) {
  return (
    <Badge variant="outline" className={cn("font-medium", toneClassName[tone], className)}>
      {label}
    </Badge>
  );
}
