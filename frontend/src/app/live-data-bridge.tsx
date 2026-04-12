import { useEffect } from "react";
import { queryClient } from "../lib/query-client";
import { useUIStore } from "../stores/ui-store";
import type { SystemSnapshot } from "../types/api";

export function LiveDataBridge() {
  const setStreamConnected = useUIStore((state) => state.setStreamConnected);
  const resetNetworkBanner = useUIStore((state) => state.resetNetworkBanner);

  useEffect(() => {
    const stream = new EventSource("/api/v1/stream");

    const onConnected = () => {
      setStreamConnected(true);
      resetNetworkBanner();
    };
    const onDisconnected = () => setStreamConnected(false);
    const onSnapshot = (event: MessageEvent<string>) => {
      onConnected();
      try {
        const snapshot = JSON.parse(event.data) as SystemSnapshot;
        queryClient.setQueryData(["system"], snapshot);
      } catch {
        // Keep the bridge resilient; polling remains the fallback path.
      }
    };
    const invalidateAlerts = () => {
      onConnected();
      void queryClient.invalidateQueries({ queryKey: ["alerts"] });
    };
    const invalidateAI = () => {
      onConnected();
      void queryClient.invalidateQueries({ queryKey: ["ai-status"] });
    };

    stream.addEventListener("hello", onConnected);
    stream.addEventListener("metrics.snapshot", onSnapshot as EventListener);
    stream.addEventListener("anomaly.raised", invalidateAlerts as EventListener);
    stream.addEventListener("anomaly.cleared", invalidateAlerts as EventListener);
    stream.addEventListener("ai.background", invalidateAI as EventListener);
    stream.onerror = onDisconnected;

    return () => {
      stream.close();
      setStreamConnected(false);
    };
  }, [resetNetworkBanner, setStreamConnected]);

  return null;
}
