import { GitBranch } from "lucide-react";
import { EmptyState } from "../components/shared/empty-state";
import { PageHeader } from "../components/shared/page-header";
import { PageSkeleton } from "../components/shared/page-skeleton";
import { Badge } from "../components/ui/badge";
import { Card } from "../components/ui/card";
import { useSystemSnapshotQuery } from "../lib/api-client";
import type { ProcessNode } from "../types/api";

export function TreePage() {
  const { data, isLoading } = useSystemSnapshotQuery();
  const nodes = data?.process_tree ?? [];

  if (isLoading) {
    return <PageSkeleton />;
  }

  if (nodes.length === 0) {
    return <EmptyState icon={GitBranch} title="No process tree yet" description="WTM has not produced a tree snapshot yet. This usually fills in after the next collector cycle." />;
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Process Tree" description="Hierarchical process view from the embedded API snapshot." />
      <div className="grid gap-4 sm:grid-cols-3">
        <SummaryCard label="Root nodes" value={String(nodes.length)} />
        <SummaryCard label="Processes in tree" value={String(countNodes(nodes))} />
        <SummaryCard label="Orphans" value={String(countOrphans(nodes))} />
      </div>
      <Card className="hidden md:block">
        <ul className="space-y-2 font-mono text-sm text-secondary">
          {nodes.map((node) => (
            <TreeNode key={`${node.process.pid}-${node.process.name}`} depth={0} node={node} />
          ))}
        </ul>
      </Card>
      <div className="grid gap-4 md:hidden">
        {nodes.map((node) => (
          <TreeCard key={`${node.process.pid}-${node.process.name}`} node={node} />
        ))}
      </div>
    </div>
  );
}

interface TreeNodeProps {
  depth: number;
  node: ProcessNode;
}

function TreeNode({ depth, node }: TreeNodeProps) {
  return (
    <li className={depth > 0 ? "border-l border-border pl-4" : ""}>
      <div className="rounded-lg px-3 py-2 text-foreground">
        {node.process.name} ({node.process.pid}) {node.is_orphan ? "[orphan]" : ""}
      </div>
      {node.children?.map((child) => (
        <TreeNode key={`${child.process.pid}-${child.process.name}`} depth={depth + 1} node={child} />
      ))}
    </li>
  );
}

interface SummaryCardProps {
  label: string;
  value: string;
}

function SummaryCard({ label, value }: SummaryCardProps) {
  return (
    <Card>
      <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</div>
      <div className="mt-3 text-3xl font-bold tracking-tight text-foreground">{value}</div>
    </Card>
  );
}

interface TreeCardProps {
  node: ProcessNode;
}

function TreeCard({ node }: TreeCardProps) {
  return (
    <Card className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-lg font-semibold tracking-tight text-foreground">{node.process.name}</div>
          <div className="mt-1 font-mono text-sm text-secondary">PID {node.process.pid}</div>
        </div>
        <Badge variant={node.is_orphan ? "warning" : "info"}>{node.is_orphan ? "Orphan" : `${node.children?.length ?? 0} child`}</Badge>
      </div>
      {node.children && node.children.length > 0 ? (
        <div className="space-y-2">
          <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">Children</div>
          <div className="grid gap-2">
            {node.children.map((child) => (
              <div key={`${child.process.pid}-${child.process.name}`} className="rounded-2xl border border-border bg-background px-3 py-3">
                <div className="text-sm font-semibold text-foreground">{child.process.name}</div>
                <div className="mt-1 font-mono text-xs text-secondary">PID {child.process.pid}</div>
              </div>
            ))}
          </div>
        </div>
      ) : (
        <div className="rounded-2xl border border-border bg-background px-3 py-3 text-sm text-secondary">No child processes under this root.</div>
      )}
    </Card>
  );
}

function countNodes(nodes: ProcessNode[]) {
  let total = 0;
  for (const node of nodes) {
    total += 1;
    if (node.children && node.children.length > 0) {
      total += countNodes(node.children);
    }
  }
  return total;
}

function countOrphans(nodes: ProcessNode[]) {
  let total = 0;
  for (const node of nodes) {
    if (node.is_orphan) {
      total += 1;
    }
    if (node.children && node.children.length > 0) {
      total += countOrphans(node.children);
    }
  }
  return total;
}
