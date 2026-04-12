import type { ReactNode } from "react";
import { cn } from "../../lib/cn";
import { Card } from "../ui/card";

interface DetailTileProps {
  label: string;
  value: string;
  className?: string;
  valueClassName?: string;
  hint?: string;
}

export function DetailTile({ label, value, className, valueClassName, hint }: DetailTileProps) {
  return (
    <div className={cn("soft-panel min-w-0", className)}>
      <div className="metric-label">{label}</div>
      <div className={cn("mt-1 overflow-hidden text-ellipsis whitespace-nowrap text-sm font-semibold text-foreground", valueClassName)}>{value}</div>
      {hint ? <div className="mt-1 text-[0.78rem] leading-5 text-secondary">{hint}</div> : null}
    </div>
  );
}

interface SummaryCardProps {
  label: string;
  value: string;
  accent?: ReactNode;
  valueClassName?: string;
}

export function SummaryCard({ label, value, accent, valueClassName }: SummaryCardProps) {
  return (
    <Card>
      <div className="flex min-w-0 items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="metric-label">{label}</div>
          <div className={cn("mt-1.5 overflow-hidden text-ellipsis whitespace-nowrap text-[1.7rem] font-semibold tracking-tight text-foreground sm:text-[1.9rem]", valueClassName)}>{value}</div>
        </div>
        {accent ? <div className="shrink-0">{accent}</div> : null}
      </div>
    </Card>
  );
}
