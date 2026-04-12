import { Info, MonitorCog, Workflow } from "lucide-react";
import { SummaryCard } from "../components/shared/detail-tile";
import { PageHeader } from "../components/shared/page-header";
import { Badge } from "../components/ui/badge";
import { Card } from "../components/ui/card";
import { useAIStatusQuery, useSystemSnapshotQuery } from "../lib/api-client";

export function AboutPage() {
  const { data: system } = useSystemSnapshotQuery();
  const { data: aiStatus } = useAIStatusQuery();

  return (
    <div className="space-y-6">
      <PageHeader title="About" description="WTM is a local Windows monitoring and control console focused on processes, alerts, rules, and operator workflows." />

      <Card className="space-y-4">
        <div className="flex items-center gap-3">
          <div className="rounded-2xl bg-accent-muted p-3 text-accent">
            <Info className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-lg font-semibold tracking-tight text-foreground">Windows Task Manager</h2>
            <p className="text-sm text-secondary">A local-first operator console for watching resource pressure, investigating processes, and taking action without leaving the machine.</p>
          </div>
        </div>
        <p className="text-sm leading-relaxed text-secondary">
          WTM brings system telemetry, process control, automation rules, alerts, AI guidance, and Telegram operations into one local dashboard.
          The goal is simple: make it obvious what is happening on the box, what matters, and what action is safe to take next.
        </p>
      </Card>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <AboutMetric icon={MonitorCog} label="Logical cores" value={system ? String(system.cpu.num_logical) : "--"} />
        <AboutMetric icon={Workflow} label="Processes tracked" value={system ? String(system.processes.length) : "--"} />
        <SummaryCard
          label="AI status"
          value={aiStatus?.enabled ? "Enabled" : "Disabled"}
          valueClassName="text-2xl font-semibold"
          accent={<Badge variant={aiStatus?.enabled ? "success" : "warning"}>AI status</Badge>}
        />
        <SummaryCard
          label="GPU telemetry"
          value={system?.gpu.available ? "Available" : "Unavailable"}
          valueClassName="text-2xl font-semibold"
          accent={<Badge variant={system?.gpu.available ? "info" : "neutral"}>GPU telemetry</Badge>}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold tracking-tight text-foreground">What WTM covers</h2>
            <Badge variant="info">Operator view</Badge>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <StackItem label="Live telemetry" value="CPU, memory, GPU, disks, ports, process tree, and network activity" />
            <StackItem label="Operator actions" value="Inspect, suspend, resume, or kill processes with guardrails" />
            <StackItem label="Automation" value="Rules and alerting for recurring patterns and threshold breaches" />
            <StackItem label="Escalation" value="Telegram notifications and confirm-before-action flows" />
            <StackItem label="AI assistance" value="Local advisor chat, analysis, and approval-based suggestions" />
            <StackItem label="Deployment" value="Single local service with an embedded web UI" />
          </div>
        </Card>

        <Card className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold tracking-tight text-foreground">How it runs</h2>
            <Badge variant="success">Localhost</Badge>
          </div>
          <div className="grid gap-3">
            <StackItem label="Data path" value="Local collector snapshots are streamed into the dashboard and refreshed continuously" />
            <StackItem label="Safety" value="Destructive actions require explicit intent and protected paths remain guarded" />
            <StackItem label="Resilience" value="Streaming is preferred and polling remains available as a fallback" />
            <StackItem label="Scope" value="Designed for local monitoring, triage, automation, and fast operator feedback loops" />
          </div>
        </Card>
      </div>
    </div>
  );
}

interface AboutMetricProps {
  icon: typeof Info;
  label: string;
  value: string;
}

function AboutMetric({ icon: Icon, label, value }: AboutMetricProps) {
  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</div>
          <div className="mt-3 text-2xl font-semibold tracking-tight text-foreground">{value}</div>
        </div>
        <div className="rounded-2xl bg-accent-muted p-3 text-accent">
          <Icon className="h-5 w-5" />
        </div>
      </div>
    </Card>
  );
}

interface StackItemProps {
  label: string;
  value: string;
}

function StackItem({ label, value }: StackItemProps) {
  return (
    <div className="rounded-2xl border border-border bg-background px-4 py-3">
      <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</div>
      <div className="mt-2 text-sm font-semibold text-foreground">{value}</div>
    </div>
  );
}
