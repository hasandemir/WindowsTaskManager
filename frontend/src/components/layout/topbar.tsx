import { Wifi, WifiOff } from "lucide-react";
import { useMemo } from "react";
import { useLocation } from "react-router";
import { useSystemSnapshotQuery } from "../../lib/api-client";
import { useUIStore } from "../../stores/ui-store";
import { Badge } from "../ui/badge";
import { ThemeMenu } from "./theme-menu";

const pageTitles: Record<string, string> = {
  "/": "Overview",
  "/processes": "Processes",
  "/tree": "Process Tree",
  "/ports": "Ports",
  "/disks": "Disks",
  "/alerts": "Alerts",
  "/rules": "Rules",
  "/ai": "AI Advisor",
  "/about": "About",
};

export function Topbar() {
  const location = useLocation();
  const { data } = useSystemSnapshotQuery();
  const streamConnected = useUIStore((state) => state.streamConnected);
  const pageTitle = pageTitles[location.pathname] ?? "Dashboard";
  const cpuText = useMemo(() => `${data?.cpu.total_percent.toFixed(1) ?? "0.0"}% CPU`, [data]);

  return (
    <div className="flex w-full flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">{pageTitle}</h1>
        <p className="text-sm text-secondary">Live view of the local WTM service and machine state.</p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant="info">{cpuText}</Badge>
        <Badge variant={streamConnected ? "success" : "warning"}>
          {streamConnected ? <Wifi className="h-3.5 w-3.5" /> : <WifiOff className="h-3.5 w-3.5" />}
          {streamConnected ? "Streaming" : "Polling"}
        </Badge>
        <ThemeMenu />
      </div>
    </div>
  );
}
