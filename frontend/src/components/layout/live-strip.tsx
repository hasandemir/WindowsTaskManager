import { useMemo } from "react";
import { useLocation } from "react-router";
import { useLocalStorage } from "../../hooks/use-local-storage";
import { useSystemSnapshotQuery } from "../../lib/api-client";
import { formatBytes, formatRate } from "../../lib/format";
import type { InterfaceInfo } from "../../types/api";

function pickPrimaryInterface(interfaces: InterfaceInfo[]) {
  let best: InterfaceInfo | null = null;
  let bestScore = -1;
  for (const iface of interfaces) {
    const score = iface.in_bps + iface.out_bps + iface.in_pps + iface.out_pps;
    if (score > bestScore) {
      best = iface;
      bestScore = score;
    }
  }
  return best;
}

export function LiveStrip() {
  const location = useLocation();
  const { data } = useSystemSnapshotQuery();
  const [selectedInterface, setSelectedInterface] = useLocalStorage<string>("wtm-network-interface", "auto");
  const hidden = location.pathname === "/" || location.pathname === "/about";

  const activeInterface = useMemo(() => {
    if (!data) {
      return null;
    }
    if (selectedInterface !== "auto") {
      const manual = data.network.interfaces.find((iface) => iface.name === selectedInterface);
      if (manual) {
        return manual;
      }
    }
    return pickPrimaryInterface(data.network.interfaces);
  }, [data, selectedInterface]);

  const networkLabel = useMemo(() => {
    if (!data) {
      return "--";
    }
    if (activeInterface) {
      return `${formatRate(activeInterface.in_bps)} down / ${formatRate(activeInterface.out_bps)} up`;
    }
    return `${formatRate(data.network.total_down_bps)} down / ${formatRate(data.network.total_up_bps)} up`;
  }, [activeInterface, data]);

  if (hidden || !data) {
    return null;
  }

  return (
    <section className="page-padding mt-4">
      <div className="grid gap-3 rounded-3xl border border-border bg-background-subtle p-3 shadow-sm md:grid-cols-2 xl:grid-cols-[repeat(5,minmax(0,1fr))_16rem]">
        <Pill label="CPU" value={`${data.cpu.total_percent.toFixed(1)}%`} detail={`${data.cpu.num_logical} logical cores`} />
        <Pill
          label="Memory"
          value={`${data.memory.used_percent.toFixed(1)}%`}
          detail={`${formatBytes(data.memory.used_phys)} of ${formatBytes(data.memory.total_phys)}`}
        />
        <Pill
          label="GPU"
          value={data.gpu.available ? `${data.gpu.utilization.toFixed(0)}%` : "n/a"}
          detail={data.gpu.available ? data.gpu.name || "Graphics processor" : "GPU unavailable"}
        />
        <Pill label="Network" value={networkLabel} detail={activeInterface?.name ?? "All interfaces"} />
        <Pill label="Processes" value={String(data.processes.length)} detail="Tracked right now" />
        <label className="flex min-h-[5.75rem] min-w-0 flex-col justify-center gap-2 rounded-2xl border border-border bg-surface px-4 py-3 text-xs text-secondary">
          <span className="font-mono uppercase tracking-[0.18em]">Interface</span>
          <select
            aria-label="Select the network interface shown in the live strip"
            className="h-10 rounded-xl bg-background px-3 py-2 font-mono text-sm text-foreground focus-visible:ring-2 focus-visible:ring-[var(--ring)]"
            value={selectedInterface}
            onChange={(event) => setSelectedInterface(event.target.value)}
          >
            <option value="auto">Auto</option>
            {data.network.interfaces.map((iface) => (
              <option key={iface.name} value={iface.name}>
                {iface.name}
              </option>
            ))}
          </select>
        </label>
      </div>
    </section>
  );
}

interface PillProps {
  label: string;
  value: string;
  detail: string;
}

function Pill({ label, value, detail }: PillProps) {
  return (
    <div className="flex min-h-[5.75rem] min-w-0 flex-col justify-center rounded-2xl border border-border bg-surface px-4 py-3">
      <span className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</span>
      <span className="mt-2 truncate text-base font-semibold tracking-tight tabular-nums text-foreground md:text-lg">{value}</span>
      <span className="mt-1 truncate text-xs text-secondary">{detail}</span>
    </div>
  );
}
