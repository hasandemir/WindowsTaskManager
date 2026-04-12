import type { PropsWithChildren } from "react";
import { cn } from "../../lib/cn";

interface BadgeProps extends PropsWithChildren {
  variant?: "neutral" | "info" | "success" | "warning" | "error";
}

export function Badge({ children, variant = "neutral" }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex min-h-6 max-w-full items-center gap-1 rounded-md border px-2 py-0.5 text-[0.67rem] font-semibold whitespace-nowrap",
        variant === "neutral" && "border-border bg-background-muted/75 text-secondary",
        variant === "info" && "border-[color:var(--info-bg)] bg-[color:var(--info-bg)] text-info",
        variant === "success" && "border-[color:var(--success-bg)] bg-[color:var(--success-bg)] text-success",
        variant === "warning" && "border-[color:var(--warning-bg)] bg-[color:var(--warning-bg)] text-warning",
        variant === "error" && "border-[color:var(--error-bg)] bg-[color:var(--error-bg)] text-error",
      )}
    >
      {children}
    </span>
  );
}
