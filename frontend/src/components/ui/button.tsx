import type { ButtonHTMLAttributes, PropsWithChildren } from "react";
import { cn } from "../../lib/cn";

interface ButtonProps extends PropsWithChildren<ButtonHTMLAttributes<HTMLButtonElement>> {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  size?: "sm" | "md" | "icon";
}

export function Button({ children, className, size = "md", variant = "primary", ...props }: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex min-h-11 items-center justify-center rounded-xl border text-sm font-semibold transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-[var(--ring)] focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50",
        size === "sm" && "px-3 py-2",
        size === "md" && "px-4 py-2.5",
        size === "icon" && "h-11 w-11",
        variant === "primary" && "border-accent bg-accent text-accent-foreground hover:bg-accent-hover",
        variant === "secondary" && "border-border bg-surface text-foreground hover:bg-background-muted",
        variant === "ghost" && "border-transparent bg-transparent text-secondary hover:bg-background-muted hover:text-foreground",
        variant === "danger" && "border-error bg-error text-accent-foreground hover:opacity-90",
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}
