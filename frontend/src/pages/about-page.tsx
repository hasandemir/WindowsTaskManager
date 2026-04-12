import { Bot, Info, MonitorCog, ShieldCheck, Workflow } from "lucide-react";
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
      <PageHeader
        title="About"
        description="WTM is a local-first Windows operator console that brings telemetry, guarded controls, automation, alerts, and AI-assisted workflows into one place."
        eyebrow="Product"
        icon={Info}
        meta={
          <>
            <Badge variant="success">Localhost</Badge>
            <Badge variant={aiStatus?.enabled ? "info" : "neutral"}>{aiStatus?.enabled ? "AI available" : "AI optional"}</Badge>
          </>
        }
      />

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

      <Card className="space-y-4">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
          <div>
            <h2 className="section-title">What WTM gives you</h2>
            <p className="mt-1 text-sm leading-6 text-secondary">A local operator console for telemetry, guarded control, automation, alerts, and optional AI-assisted triage.</p>
          </div>
          <Badge variant="success">Local-first</Badge>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <StackItem label="Live telemetry" value="CPU, memory, GPU, disks, ports, process tree, and network activity in one pass." />
          <StackItem label="Operator actions" value="Inspect, suspend, resume, or kill processes with clear guardrails and confirmations." />
          <StackItem label="Automation" value="Rules and alerts help encode recurring operational instincts into repeatable actions." />
          <StackItem label="Escalation" value="Telegram hooks keep high-value critical signals available outside the dashboard." />
          <StackItem label="AI assistance" value="Chat, analyze, and approve suggested mitigations instead of executing blind automation." />
          <StackItem label="Single binary UX" value="The frontend is embedded, so the experience stays local and portable." />
        </div>
      </Card>

      <Card className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          <h2 className="section-title">How it behaves</h2>
          <Badge variant="info">Operator model</Badge>
        </div>
        <div className="grid gap-3 xl:grid-cols-2">
          <BehaviorRow icon={ShieldCheck} title="Safety first" description="Protected flows, confirmations, and backend checks reduce the chance of rash actions." />
          <BehaviorRow icon={Bot} title="AI is advisory" description="Suggestions stay approval-based. Nothing meaningful runs just because the model proposed it." />
          <BehaviorRow icon={MonitorCog} title="Realtime where possible" description="SSE keeps the UI feeling alive, and polling gives you a fallback when the stream drops." />
          <BehaviorRow icon={Workflow} title="Built for triage" description="Each page answers a different operator question: what is hot, where it came from, what ports it owns, and what to do next." />
        </div>
      </Card>
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
          <div className="metric-label">{label}</div>
          <div className="mt-2 text-xl font-semibold tracking-tight text-foreground sm:text-2xl">{value}</div>
        </div>
        <div className="rounded-[0.95rem] border border-border bg-accent-muted p-2.5 text-accent">
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
    <div className="soft-panel">
      <div className="metric-label">{label}</div>
      <div className="mt-1.5 text-sm font-semibold leading-6 text-foreground">{value}</div>
    </div>
  );
}

interface BehaviorRowProps {
  icon: typeof Info;
  title: string;
  description: string;
}

function BehaviorRow({ icon: Icon, title, description }: BehaviorRowProps) {
  return (
    <div className="soft-panel">
      <div className="flex items-start gap-3">
        <div className="rounded-lg border border-border bg-accent-muted p-2 text-accent">
          <Icon className="h-4 w-4" />
        </div>
        <div>
          <div className="text-sm font-semibold text-foreground">{title}</div>
          <div className="mt-1 text-sm leading-6 text-secondary">{description}</div>
        </div>
      </div>
    </div>
  );
}
