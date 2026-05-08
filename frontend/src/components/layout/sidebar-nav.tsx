import type { ComponentType } from "react";
import { Bot, Boxes, Cpu, GitBranch, HardDrive, Info, LayoutDashboard, Network, Settings, ShieldAlert, Sparkles, Workflow } from "lucide-react";
import { NavLink } from "react-router";
import type { AppRoutePath } from "../../app/route-map";
import { prefetchRoute } from "../../app/route-map";
import { cn } from "../../lib/cn";

interface SidebarNavProps {
  onNavigate: () => void;
}

interface NavItem {
  to: AppRoutePath;
  label: string;
  icon: ComponentType<{ className?: string }>;
}

const navItems: NavItem[] = [
  { to: "/", label: "Overview", icon: Cpu },
  { to: "/overview", label: "Metrics", icon: LayoutDashboard },
  { to: "/processes", label: "Processes", icon: Boxes },
  { to: "/tree", label: "Tree", icon: GitBranch },
  { to: "/ports", label: "Ports", icon: Network },
  { to: "/disks", label: "Disks", icon: HardDrive },
  { to: "/alerts", label: "Alerts", icon: ShieldAlert },
  { to: "/rules", label: "Rules", icon: Workflow },
  { to: "/ai", label: "AI", icon: Bot },
  { to: "/settings", label: "Settings", icon: Settings },
  { to: "/about", label: "About", icon: Info },
];

export function SidebarNav({ onNavigate }: SidebarNavProps) {
  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-border px-3.5 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-[0.65rem] border border-border bg-surface text-accent">
            <Sparkles className="h-3 w-3" />
          </div>
          <div>
            <div className="text-[0.68rem] font-medium uppercase tracking-[0.18em] text-secondary">Windows Task Manager</div>
            <div className="mt-0.5 text-[1.05rem] font-semibold tracking-tight text-foreground">WTM</div>
          </div>
        </div>
        <div className="mt-3 border-l-2 border-accent/45 pl-3 text-[0.8rem] leading-5 text-secondary">
          Local operator console for process triage, rules, alerts, ports, and guarded actions.
        </div>
      </div>
      <nav className="flex-1 space-y-1 overflow-y-auto px-2 py-3">
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/"}
              onClick={onNavigate}
              onMouseEnter={() => prefetchRoute(item.to)}
              className={({ isActive }) =>
                cn(
                  "flex min-h-9 items-center gap-2.5 rounded-[0.7rem] px-2.5 py-2 text-[0.88rem] font-medium text-secondary transition-colors focus-visible:ring-2 focus-visible:ring-[var(--ring)] focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                  "hover:bg-background-muted hover:text-foreground",
                  isActive && "bg-surface text-foreground ring-1 ring-border",
                )
              }
            >
              <div className={cn("flex h-6.5 w-6.5 items-center justify-center rounded-[0.55rem] bg-background-muted text-secondary")}>
                <Icon className="h-3.25 w-3.25 shrink-0" />
              </div>
              <span>{item.label}</span>
            </NavLink>
          );
        })}
      </nav>
      <div className="border-t border-border px-3.5 py-3.5">
        <div className="rounded-[0.8rem] border border-border bg-surface px-3 py-2.5">
          <div className="eyebrow">Local service</div>
          <div className="mt-1.5 text-sm font-semibold text-foreground">localhost operator console</div>
          <div className="mt-1 text-[0.8rem] leading-5 text-secondary">Realtime stream first, polling fallback when transport blinks.</div>
        </div>
      </div>
    </div>
  );
}
