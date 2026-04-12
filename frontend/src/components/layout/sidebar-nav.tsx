import type { ComponentType } from "react";
import { Bot, Boxes, Cpu, GitBranch, HardDrive, Info, Network, Settings, ShieldAlert, Workflow } from "lucide-react";
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
      <div className="border-b border-border px-5 py-5">
        <div className="text-xs font-medium uppercase tracking-[0.24em] text-secondary">Windows Task Manager</div>
        <div className="mt-2 text-2xl font-semibold tracking-tight text-foreground">WTM</div>
        <div className="mt-1 text-sm text-secondary">Local system monitor and operator console</div>
      </div>
      <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
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
                  "flex min-h-11 items-center gap-3 rounded-xl px-3 py-2 text-sm font-medium text-secondary transition-colors focus-visible:ring-2 focus-visible:ring-[var(--ring)] focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                  "hover:bg-background-muted hover:text-foreground",
                  isActive && "bg-surface text-foreground shadow-sm ring-1 ring-border",
                )
              }
            >
              <Icon className="h-4 w-4 shrink-0" />
              <span>{item.label}</span>
            </NavLink>
          );
        })}
      </nav>
      <div className="border-t border-border px-4 py-4">
        <div className="flex items-center gap-2 rounded-xl bg-accent-muted px-3 py-3 text-sm text-secondary">
          <Settings className="h-4 w-4 text-accent" />
          Localhost operator console
        </div>
      </div>
    </div>
  );
}
