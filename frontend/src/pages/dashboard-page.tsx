import type { ComponentType } from "react";
import { Activity, Cpu, MemoryStick, Network } from "lucide-react";
import { EmptyState } from "../components/shared/empty-state";
import { PageHeader } from "../components/shared/page-header";
import { PageSkeleton } from "../components/shared/page-skeleton";
import { Card } from "../components/ui/card";
import { useSystemSnapshotQuery } from "../lib/api-client";
import { formatBytes, formatPercent, formatRate } from "../lib/format";
import type { ProcessInfo } from "../types/api";

export function DashboardPage() {
  const { data, isLoading } = useSystemSnapshotQuery();

  if (isLoading) {
    return <PageSkeleton />;
  }

  if (!data) {
    return <EmptyState icon={Activity} title="Waiting for live metrics" description="WTM is still warming up the local collector and snapshot stream." />;
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Dashboard" description="Overview of live system status from the localhost API." />
      <div className="stat-grid">
        <MetricCard icon={Cpu} label="CPU" value={formatPercent(data.cpu.total_percent)} detail={data.cpu.name || "Processor"} />
        <MetricCard icon={MemoryStick} label="Memory" value={formatPercent(data.memory.used_percent)} detail={`${formatBytes(data.memory.used_phys)} / ${formatBytes(data.memory.total_phys)}`} />
        <MetricCard icon={Activity} label="GPU" value={data.gpu.available ? `${data.gpu.utilization.toFixed(0)}%` : "n/a"} detail={data.gpu.name || "GPU unavailable"} />
        <MetricCard icon={Network} label="Network" value={formatRate(data.network.total_down_bps)} detail={`Up ${formatRate(data.network.total_up_bps)}`} />
      </div>
      <Card className="space-y-4">
        <h2 className="text-lg font-semibold tracking-tight text-foreground">Top processes</h2>

        <div className="grid gap-3 sm:hidden">
          {data.processes.slice(0, 5).map((process) => (
            <TopProcessCard key={process.pid} process={process} />
          ))}
        </div>

        <div className="hidden overflow-x-auto sm:block">
          <table className="min-w-full text-left text-sm">
            <thead className="text-secondary">
              <tr>
                <th className="pb-3 pr-4">PID</th>
                <th className="pb-3 pr-4">Name</th>
                <th className="pb-3 pr-4">CPU</th>
                <th className="pb-3 pr-4">Memory</th>
                <th className="pb-3">Threads</th>
              </tr>
            </thead>
            <tbody>
              {data.processes.slice(0, 10).map((process) => (
                <tr key={process.pid} className="border-t border-border">
                  <td className="py-3 pr-4 font-mono text-secondary">{process.pid}</td>
                  <td className="py-3 pr-4 text-foreground">{process.name}</td>
                  <td className="py-3 pr-4 text-secondary">{process.cpu_percent.toFixed(1)}%</td>
                  <td className="py-3 pr-4 text-secondary">{formatBytes(process.working_set)}</td>
                  <td className="py-3 text-secondary">{process.thread_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
}

interface MetricCardProps {
  icon: ComponentType<{ className?: string }>;
  label: string;
  value: string;
  detail: string;
}

function MetricCard({ icon: Icon, label, value, detail }: MetricCardProps) {
  return (
    <Card>
      <div className="flex items-start justify-between">
        <div>
          <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</div>
          <div className="mt-3 text-3xl font-bold tracking-tight text-foreground">{value}</div>
          <div className="mt-2 text-sm text-secondary">{detail}</div>
        </div>
        <div className="rounded-2xl bg-accent-muted p-3 text-accent">
          <Icon className="h-5 w-5" />
        </div>
      </div>
    </Card>
  );
}

interface TopProcessCardProps {
  process: ProcessInfo;
}

function TopProcessCard({ process }: TopProcessCardProps) {
  return (
    <div className="rounded-2xl border border-border bg-background px-4 py-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-semibold text-foreground">{process.name}</div>
          <div className="mt-1 font-mono text-xs text-secondary">PID {process.pid}</div>
        </div>
        <div className="text-sm font-semibold text-foreground">{process.cpu_percent.toFixed(1)}%</div>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
        <Metric label="Memory" value={formatBytes(process.working_set)} />
        <Metric label="Threads" value={String(process.thread_count)} />
      </div>
    </div>
  );
}

interface MetricProps {
  label: string;
  value: string;
}

function Metric({ label, value }: MetricProps) {
  return (
    <div className="rounded-2xl border border-border bg-background-subtle px-3 py-3">
      <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</div>
      <div className="mt-2 text-sm font-semibold text-foreground">{value}</div>
    </div>
  );
}
