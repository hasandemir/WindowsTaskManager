import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { Check, Monitor, Moon, Sun } from "lucide-react";
import type { Theme } from "../../hooks/use-theme";
import { useTheme } from "../../hooks/use-theme";
import { Button } from "../ui/button";

interface ThemeOption {
  label: string;
  value: Theme;
  icon: typeof Sun;
}

const themeOptions: ThemeOption[] = [
  { label: "System", value: "system", icon: Monitor },
  { label: "Dark", value: "dark", icon: Moon },
  { label: "Light", value: "light", icon: Sun },
];

export function ThemeMenu() {
  const { theme, setTheme } = useTheme();
  const ActiveIcon = theme === "system" ? Monitor : theme === "dark" ? Moon : Sun;

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <Button
          type="button"
          variant="secondary"
          size="icon"
          aria-label="Switch theme"
          title="Switch theme"
        >
          <ActiveIcon className="h-4 w-4" />
        </Button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          sideOffset={10}
          align="end"
          className="z-[400] min-w-44 rounded-2xl border border-border bg-surface p-2 shadow-lg"
        >
          {themeOptions.map((option) => {
            const Icon = option.icon;
            const selected = theme === option.value;

            return (
              <DropdownMenu.Item
                key={option.value}
                className="flex min-h-11 cursor-pointer items-center justify-between rounded-xl px-3 py-2 text-sm text-foreground outline-none transition-colors duration-150 hover:bg-background-muted focus:bg-background-muted"
                onSelect={() => setTheme(option.value)}
              >
                <span className="flex items-center gap-3">
                  <Icon className="h-4 w-4 text-secondary" />
                  {option.label}
                </span>
                {selected ? <Check className="h-4 w-4 text-accent" /> : null}
              </DropdownMenu.Item>
            );
          })}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
