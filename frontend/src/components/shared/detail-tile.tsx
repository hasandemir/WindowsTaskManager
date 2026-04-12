import type { ReactNode } from "react";
import { Card } from "../ui/card";

interface DetailTileProps {
  label: string;
  value: string;
  className?: string;
  valueClassName?: string;
}

export function DetailTile({ label, value, className, valueClassName }: DetailTileProps) {
  return (
    <div className={`rounded-2xl border border-border bg-background px-3 py-3 ${className ?? ""}`.trim()}>
      <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</div>
      <div className={`mt-2 text-sm font-semibold text-foreground ${valueClassName ?? ""}`.trim()}>{value}</div>
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
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</div>
          <div className={`mt-3 text-3xl font-bold tracking-tight text-foreground ${valueClassName ?? ""}`.trim()}>{value}</div>
        </div>
        {accent}
      </div>
    </Card>
  );
}
