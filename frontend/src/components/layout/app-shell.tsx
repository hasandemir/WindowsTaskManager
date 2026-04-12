import { Menu } from "lucide-react";
import { Outlet } from "react-router";
import { useUIStore } from "../../stores/ui-store";
import { NetworkBanner } from "../shared/network-banner";
import { Button } from "../ui/button";
import { LiveStrip } from "./live-strip";
import { SidebarNav } from "./sidebar-nav";
import { Topbar } from "./topbar";

export function AppShell() {
  const sidebarOpen = useUIStore((state) => state.sidebarOpen);
  const setSidebarOpen = useUIStore((state) => state.setSidebarOpen);

  return (
    <div className="app-shell">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[600] focus:rounded-xl focus:bg-accent focus:px-4 focus:py-3 focus:text-accent-foreground"
      >
        Skip to main content
      </a>
      <aside
        className={[
          "fixed inset-y-0 left-0 z-[200] w-64 border-r border-border bg-background-subtle transition-transform duration-150 lg:translate-x-0",
          sidebarOpen ? "translate-x-0" : "-translate-x-full",
        ].join(" ")}
      >
        <SidebarNav onNavigate={() => setSidebarOpen(false)} />
      </aside>
      <div
        className={[
          "fixed inset-0 z-[150] bg-[color:var(--overlay)] transition-opacity duration-150 lg:hidden",
          sidebarOpen ? "opacity-100" : "pointer-events-none opacity-0",
        ].join(" ")}
        onClick={() => setSidebarOpen(false)}
      />
      <main id="main-content" className="min-h-screen lg:pl-64" tabIndex={-1}>
        <header className="sticky top-0 z-[100] border-b border-border bg-background/85 backdrop-blur-sm">
          <div className="page-padding flex min-h-16 items-center gap-3 py-3">
            <Button
              variant="ghost"
              size="icon"
              className="lg:hidden"
              aria-label="Open navigation"
              onClick={() => setSidebarOpen(true)}
            >
              <Menu className="h-5 w-5" />
            </Button>
            <Topbar />
          </div>
        </header>
        <NetworkBanner />
        <LiveStrip />
        <div className="page-padding py-4 sm:py-6 lg:py-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
