import { Button } from "../ui/button";

interface FilterChipProps {
  active: boolean;
  label: string;
  onClick: () => void;
}

export function FilterChip({ active, label, onClick }: FilterChipProps) {
  return (
    <Button type="button" variant={active ? "primary" : "secondary"} size="sm" onClick={onClick}>
      {label}
    </Button>
  );
}
