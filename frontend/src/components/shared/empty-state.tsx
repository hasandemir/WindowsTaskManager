import type { LucideIcon } from "lucide-react";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
}

export function EmptyState({ icon: Icon, title, description }: EmptyStateProps) {
  return (
    <div className="surface flex min-h-48 flex-col items-center justify-center gap-3 px-6 py-10 text-center">
      <div className="rounded-lg border border-border bg-background px-3 py-3 text-accent">
        <Icon className="h-5 w-5" />
      </div>
      <div className="space-y-2">
        <div className="eyebrow justify-center">Nothing to show yet</div>
        <h3 className="text-lg font-semibold tracking-tight text-foreground">{title}</h3>
        <p className="max-w-xl text-sm leading-6 text-secondary">{description}</p>
      </div>
    </div>
  );
}
