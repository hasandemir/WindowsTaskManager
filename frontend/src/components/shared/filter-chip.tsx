import { Button } from "../ui/button";
import { cn } from "../../lib/cn";

interface FilterChipProps {
  active: boolean;
  label: string;
  onClick: () => void;
}

export function FilterChip({ active, label, onClick }: FilterChipProps) {
  return (
    <Button
      type="button"
      variant={active ? "secondary" : "ghost"}
      size="sm"
      className={cn(
        "min-h-8 rounded-md px-2.5 py-1 text-[0.79rem]",
        active && "border-border-strong bg-surface text-foreground",
        !active && "text-secondary",
      )}
      onClick={onClick}
    >
      {label}
    </Button>
  );
}
