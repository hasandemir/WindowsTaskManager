import type { LucideIcon } from "lucide-react";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
}

export function EmptyState({ icon: Icon, title, description }: EmptyStateProps) {
  return (
    <div className="surface flex min-h-56 flex-col items-center justify-center gap-3 px-6 py-10 text-center">
      <div className="rounded-2xl bg-accent-muted p-4 text-accent">
        <Icon className="h-7 w-7" />
      </div>
      <div className="space-y-1">
        <h3 className="text-lg font-semibold tracking-tight text-foreground">{title}</h3>
        <p className="text-sm leading-relaxed text-secondary">{description}</p>
      </div>
    </div>
  );
}
