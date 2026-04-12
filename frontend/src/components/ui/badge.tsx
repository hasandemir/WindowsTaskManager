import type { PropsWithChildren } from "react";
import { cn } from "../../lib/cn";

interface BadgeProps extends PropsWithChildren {
  variant?: "neutral" | "info" | "success" | "warning" | "error";
}

export function Badge({ children, variant = "neutral" }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex min-h-9 items-center gap-2 rounded-full px-3 py-1 text-xs font-semibold",
        variant === "neutral" && "bg-background-muted text-secondary",
        variant === "info" && "bg-[color:var(--info-bg)] text-info",
        variant === "success" && "bg-[color:var(--success-bg)] text-success",
        variant === "warning" && "bg-[color:var(--warning-bg)] text-warning",
        variant === "error" && "bg-[color:var(--error-bg)] text-error",
      )}
    >
      {children}
    </span>
  );
}
