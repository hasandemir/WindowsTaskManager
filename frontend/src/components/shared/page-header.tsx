import type { ComponentType, ReactNode } from "react";
import { Sparkles } from "lucide-react";
import { cn } from "../../lib/cn";

interface PageHeaderProps {
  title: string;
  description: string;
  eyebrow?: string;
  meta?: ReactNode;
  icon?: ComponentType<{ className?: string }>;
  actions?: ReactNode;
}

export function PageHeader({ title, description, eyebrow = "Live workspace", meta, icon: Icon = Sparkles, actions }: PageHeaderProps) {
  return (
    <section className="hero-panel">
      <div className="relative flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
        <div className="min-w-0 max-w-3xl">
          <div className="eyebrow">
            <Icon className="h-3.5 w-3.5 text-accent" />
            {eyebrow}
          </div>
          <h1 className="mt-2 text-[1.5rem] font-semibold tracking-tight text-foreground sm:text-[1.7rem]">{title}</h1>
          <p className="mt-1 max-w-2xl text-sm leading-6 text-secondary">{description}</p>
          {meta ? <div className="mt-2 flex flex-wrap items-center gap-1.5">{meta}</div> : null}
        </div>
        {actions ? (
          <div
            className={cn(
              "relative z-10 flex w-full flex-col gap-2 border-t border-border pt-3 sm:flex-row sm:flex-wrap sm:items-center xl:w-auto xl:max-w-xl xl:justify-end xl:border-t-0 xl:pt-0",
            )}
          >
            {actions}
          </div>
        ) : null}
      </div>
    </section>
  );
}
