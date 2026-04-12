import { useMemo, useState } from "react";
import { ArrowDownAZ, ArrowUpAZ, Network } from "lucide-react";
import { DetailTile, SummaryCard } from "../components/shared/detail-tile";
import { EmptyState } from "../components/shared/empty-state";
import { FilterChip } from "../components/shared/filter-chip";
import { PageHeader } from "../components/shared/page-header";
import { PageSkeleton } from "../components/shared/page-skeleton";
import { SearchInput } from "../components/shared/search-input";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { useDebouncedValue } from "../hooks/use-debounced-value";
import { useSystemSnapshotQuery } from "../lib/api-client";
import type { PortBinding } from "../types/api";

type PortSortKey = "process" | "pid" | "protocol" | "local" | "remote" | "state";
type SortDirection = "desc" | "asc";
type PortFilter = "all" | "tcp" | "udp" | "listening";

const sortOptions: Array<{ label: string; value: PortSortKey }> = [
  { label: "Process", value: "process" },
  { label: "PID", value: "pid" },
  { label: "Protocol", value: "protocol" },
  { label: "Local port", value: "local" },
  { label: "Remote peer", value: "remote" },
  { label: "State", value: "state" },
];

export function PortsPage() {
  const { data, isLoading } = useSystemSnapshotQuery();
  const [searchValue, setSearchValue] = useState("");
  const [selectedBinding, setSelectedBinding] = useState<PortBinding | null>(null);
  const [sortKey, setSortKey] = useState<PortSortKey>("local");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");
  const [filter, setFilter] = useState<PortFilter>("all");
  const debouncedSearch = useDebouncedValue(searchValue, 300);
  const bindings = data?.port_bindings ?? [];

  const filteredBindings = useMemo(() => {
    const needle = debouncedSearch.trim().toLowerCase();
    const base = bindings.filter((binding) => {
      if (filter === "tcp" && binding.protocol.toLowerCase() !== "tcp") {
        return false;
      }
      if (filter === "udp" && binding.protocol.toLowerCase() !== "udp") {
        return false;
      }
      if (filter === "listening" && binding.state !== "LISTEN") {
        return false;
      }
      if (!needle) {
        return true;
      }
      return (
        binding.protocol.toLowerCase().includes(needle) ||
        binding.process.toLowerCase().includes(needle) ||
        binding.label.toLowerCase().includes(needle) ||
        binding.local_addr.toLowerCase().includes(needle) ||
        binding.remote_addr.toLowerCase().includes(needle) ||
        String(binding.pid).includes(needle) ||
        String(binding.local_port).includes(needle) ||
        String(binding.remote_port).includes(needle)
      );
    });

    const sorted = [...base];
    sorted.sort((left, right) => compareBindings(left, right, sortKey, sortDirection));
    return sorted;
  }, [bindings, debouncedSearch, filter, sortDirection, sortKey]);

  if (isLoading) {
    return <PageSkeleton />;
  }

  if (bindings.length === 0) {
    return <EmptyState icon={Network} title="No port bindings yet" description="Port monitor data will appear here when the collector publishes it." />;
  }

  const listeningCount = bindings.filter((binding) => binding.state === "LISTEN").length;
  const remotePeerCount = bindings.filter((binding) => hasRemotePeer(binding)).length;
  const uniquePIDCount = new Set(bindings.map((binding) => binding.pid)).size;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Ports"
        description="Inspect which process owns which endpoint, whether it is only listening, and where active connections are reaching."
        actions={
          <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
            <SearchInput
              ariaLabel="Search ports by process, PID, address, or port"
              placeholder="Search process, PID, address, or port"
              value={searchValue}
              widthClassName="sm:w-80"
              onChange={setSearchValue}
            />
            <label className="flex min-h-11 items-center gap-2 rounded-2xl border border-border bg-surface px-3 text-sm text-secondary">
              <span>Sort</span>
              <select
                aria-label="Sort port bindings"
                className="bg-transparent text-foreground"
                value={sortKey}
                onChange={(event) => setSortKey(event.target.value as PortSortKey)}
              >
                {sortOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
            <Button
              type="button"
              variant="secondary"
              onClick={() => setSortDirection((current) => (current === "desc" ? "asc" : "desc"))}
            >
              {sortDirection === "desc" ? <ArrowDownAZ className="mr-2 h-4 w-4" /> : <ArrowUpAZ className="mr-2 h-4 w-4" />}
              {sortDirection === "desc" ? "High first" : "Low first"}
            </Button>
          </div>
        }
      />

      <div className="grid gap-4 sm:grid-cols-3">
        <SummaryCard label="Visible bindings" value={String(filteredBindings.length)} accent={<Badge variant="info">Filtered</Badge>} />
        <SummaryCard label="Listening ports" value={String(listeningCount)} accent={<Badge variant="success">LISTEN</Badge>} />
        <SummaryCard label="Processes using ports" value={String(uniquePIDCount)} accent={<Badge variant="warning">{remotePeerCount} remote peers</Badge>} />
      </div>

      <div className="flex flex-wrap gap-2">
        <FilterChip active={filter === "all"} label="All" onClick={() => setFilter("all")} />
        <FilterChip active={filter === "tcp"} label="TCP" onClick={() => setFilter("tcp")} />
        <FilterChip active={filter === "udp"} label="UDP" onClick={() => setFilter("udp")} />
        <FilterChip active={filter === "listening"} label="Listening" onClick={() => setFilter("listening")} />
      </div>

      {selectedBinding ? (
        <Card className="space-y-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold tracking-tight text-foreground">
                {selectedBinding.process || selectedBinding.label || "Unknown process"}
              </h2>
              <p className="mt-1 text-sm text-secondary">
                Local endpoint {selectedBinding.local_addr}:{selectedBinding.local_port}. If a remote peer is present, this row represents an active network connection rather than a passive listener.
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Badge variant={selectedBinding.state === "LISTEN" ? "success" : "info"}>{selectedBinding.state || "OPEN"}</Badge>
              <Badge variant="neutral">{selectedBinding.protocol.toUpperCase()}</Badge>
              <Badge variant="neutral">PID {selectedBinding.pid}</Badge>
            </div>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <DetailTile label="Process" value={selectedBinding.process || selectedBinding.label || "Unknown"} valueClassName="break-words" />
            <DetailTile label="Local endpoint" value={`${selectedBinding.local_addr}:${selectedBinding.local_port}`} valueClassName="break-all" />
            <DetailTile label="Remote peer" value={formatRemote(selectedBinding)} valueClassName="break-all" />
            <DetailTile label="Usage meaning" value={portUsageMeaning(selectedBinding)} valueClassName="leading-snug" />
          </div>
        </Card>
      ) : null}

      {filteredBindings.length === 0 ? (
        <EmptyState icon={Network} title="No bindings match" description="Try a different process, PID, protocol, state, address, or port filter." />
      ) : null}

      <div className="grid gap-4 md:hidden">
        {filteredBindings.map((binding) => (
          <PortCard
            key={bindingKey(binding)}
            binding={binding}
            selected={selectedBinding !== null && bindingKey(selectedBinding) === bindingKey(binding)}
            onDetails={() => setSelectedBinding(binding)}
          />
        ))}
      </div>

      <Card className={filteredBindings.length === 0 ? "hidden" : "hidden overflow-x-auto md:block"}>
        <table className="min-w-full text-left text-sm">
          <thead className="text-secondary">
            <tr>
              <th className="pb-3 pr-4">Protocol</th>
              <th className="pb-3 pr-4">Local</th>
              <th className="pb-3 pr-4">Remote</th>
              <th className="pb-3 pr-4">State</th>
              <th className="pb-3 pr-4">PID</th>
              <th className="pb-3 pr-4">Process</th>
              <th className="pb-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredBindings.map((binding) => (
              <tr key={bindingKey(binding)} className={selectedBinding !== null && bindingKey(selectedBinding) === bindingKey(binding) ? "border-t border-border bg-background" : "border-t border-border"}>
                <td className="py-3 pr-4 uppercase text-secondary">{binding.protocol}</td>
                <td className="py-3 pr-4 text-foreground">{binding.local_addr}:{binding.local_port}</td>
                <td className="py-3 pr-4 text-secondary">{formatRemote(binding)}</td>
                <td className="py-3 pr-4 text-secondary">{binding.state || "OPEN"}</td>
                <td className="py-3 pr-4 font-mono text-secondary">{binding.pid}</td>
                <td className="py-3 pr-4 text-secondary">{binding.process || binding.label}</td>
                <td className="py-3">
                  <Button size="sm" variant="ghost" onClick={() => setSelectedBinding(binding)}>
                    Details
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
}

interface PortCardProps {
  binding: PortBinding;
  selected: boolean;
  onDetails: () => void;
}

function PortCard({ binding, selected, onDetails }: PortCardProps) {
  return (
    <Card className={selected ? "space-y-4 bg-background" : "space-y-4"}>
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-lg font-semibold tracking-tight text-foreground">{binding.process || binding.label || "Unknown process"}</div>
          <div className="mt-1 font-mono text-sm text-secondary">PID {binding.pid}</div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Badge variant={binding.state === "LISTEN" ? "success" : "info"}>{binding.state || "OPEN"}</Badge>
          <Badge variant="neutral">{binding.protocol.toUpperCase()}</Badge>
        </div>
      </div>
      <div className="grid gap-3 text-sm">
        <DetailTile label="Local" value={`${binding.local_addr}:${binding.local_port}`} valueClassName="break-all" />
        <DetailTile label="Remote" value={formatRemote(binding)} valueClassName="break-all" />
        <DetailTile label="Usage" value={portUsageMeaning(binding)} valueClassName="leading-snug" />
      </div>
      <Button type="button" variant="ghost" onClick={onDetails}>
        Details
      </Button>
    </Card>
  );
}

function bindingKey(binding: PortBinding) {
  return `${binding.protocol}-${binding.pid}-${binding.local_addr}-${binding.local_port}-${binding.remote_addr}-${binding.remote_port}-${binding.state}`;
}

function formatRemote(binding: PortBinding) {
  if (!hasRemotePeer(binding)) {
    return "No remote peer";
  }
  return `${binding.remote_addr}:${binding.remote_port}`;
}

function hasRemotePeer(binding: PortBinding) {
  return Boolean(binding.remote_addr) || binding.remote_port > 0;
}

function portUsageMeaning(binding: PortBinding) {
  if (binding.state === "LISTEN") {
    return "Waiting for inbound connections on the local port.";
  }
  if (binding.protocol.toLowerCase() === "udp") {
    return "Datagram socket used for connectionless traffic.";
  }
  if (hasRemotePeer(binding)) {
    return "Active connection between a local socket and a remote peer.";
  }
  return "Socket exists but no remote peer is currently attached.";
}

function compareBindings(left: PortBinding, right: PortBinding, sortKey: PortSortKey, direction: SortDirection) {
  let result = 0;
  switch (sortKey) {
    case "process":
      result = (left.process || left.label).localeCompare(right.process || right.label);
      break;
    case "pid":
      result = left.pid - right.pid;
      break;
    case "protocol":
      result = left.protocol.localeCompare(right.protocol);
      break;
    case "local":
      result = left.local_port - right.local_port;
      break;
    case "remote":
      result = left.remote_port - right.remote_port;
      break;
    case "state":
      result = (left.state || "").localeCompare(right.state || "");
      break;
  }
  return direction === "desc" ? result * -1 : result;
}
