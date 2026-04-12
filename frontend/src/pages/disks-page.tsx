import { HardDrive } from "lucide-react";
import { DetailTile } from "../components/shared/detail-tile";
import { EmptyState } from "../components/shared/empty-state";
import { PageHeader } from "../components/shared/page-header";
import { PageSkeleton } from "../components/shared/page-skeleton";
import { Badge } from "../components/ui/badge";
import { Card } from "../components/ui/card";
import { useSystemSnapshotQuery } from "../lib/api-client";
import { formatBytes, formatPercent, formatRate } from "../lib/format";

export function DisksPage() {
  const { data, isLoading } = useSystemSnapshotQuery();
  const drives = data?.disk.drives ?? [];

  if (isLoading) {
    return <PageSkeleton />;
  }

  if (drives.length === 0) {
    return <EmptyState icon={HardDrive} title="No drives detected" description="Disk telemetry will populate this page once the collector responds." />;
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Disks" description="Capacity, throughput, and activity for visible drives." />
      <div className="grid gap-4 lg:grid-cols-2">
        {drives.map((drive) => (
          <Card key={drive.letter}>
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold tracking-tight text-foreground">{drive.letter}</h2>
                <p className="text-sm text-secondary">{drive.label || drive.fs_type}</p>
              </div>
              <Badge variant={drive.used_pct >= 90 ? "warning" : "info"}>
                {formatPercent(drive.used_pct)}
              </Badge>
            </div>
            <div className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
              <DetailTile label="Used" value={`${formatBytes(drive.used_bytes)} / ${formatBytes(drive.total_bytes)}`} />
              <DetailTile label="Filesystem" value={drive.fs_type || "Unknown"} />
              <DetailTile label="Read" value={formatRate(drive.read_bps)} />
              <DetailTile label="Write" value={formatRate(drive.write_bps)} />
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
