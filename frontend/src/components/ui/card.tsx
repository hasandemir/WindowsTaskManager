import type { PropsWithChildren } from "react";
import { cn } from "../../lib/cn";

interface CardProps extends PropsWithChildren {
  className?: string;
}

export function Card({ children, className }: CardProps) {
  return <section className={cn("surface", className)}>{children}</section>;
}
