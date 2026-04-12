export function PageSkeleton() {
  return (
    <div className="space-y-6">
      <div className="h-8 w-48 animate-pulse rounded-xl bg-background-muted" />
      <div className="stat-grid">
        {Array.from({ length: 4 }, (_, index) => (
          <div key={index} className="h-32 animate-pulse rounded-2xl border border-border bg-background-subtle" />
        ))}
      </div>
      <div className="h-96 animate-pulse rounded-2xl border border-border bg-background-subtle" />
    </div>
  );
}
