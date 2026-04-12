import { AlertTriangle, X } from "lucide-react";
import { useUIStore } from "../../stores/ui-store";
import { Button } from "../ui/button";

export function NetworkBanner() {
  const streamConnected = useUIStore((state) => state.streamConnected);
  const networkBannerDismissed = useUIStore((state) => state.networkBannerDismissed);
  const dismissNetworkBanner = useUIStore((state) => state.dismissNetworkBanner);

  if (streamConnected || networkBannerDismissed) {
    return null;
  }

  return (
    <div className="page-padding pt-4" aria-live="polite">
      <div className="flex items-start justify-between gap-3 rounded-2xl border border-warning bg-[color:var(--warning-bg)] px-4 py-3 text-sm text-warning">
        <div className="flex items-center gap-3">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span>Live stream disconnected. The dashboard is falling back to direct polling until SSE recovers.</span>
        </div>
        <Button
          type="button"
          size="icon"
          variant="ghost"
          className="h-9 min-h-9 w-9 shrink-0 text-warning hover:bg-background/50 hover:text-warning"
          aria-label="Dismiss connection warning"
          onClick={dismissNetworkBanner}
        >
          <X className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
