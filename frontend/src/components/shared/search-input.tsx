import { Search, X } from "lucide-react";
import { Button } from "../ui/button";

interface SearchInputProps {
  ariaLabel: string;
  placeholder: string;
  value: string;
  widthClassName?: string;
  onChange: (value: string) => void;
}

export function SearchInput({
  ariaLabel,
  placeholder,
  value,
  widthClassName = "sm:w-72",
  onChange,
}: SearchInputProps) {
  return (
    <div className={`flex min-h-9 w-full items-center gap-2 rounded-lg border border-border bg-background px-3 text-sm text-foreground ${widthClassName}`}>
      <Search className="h-3.5 w-3.5 shrink-0 text-secondary" />
      <input
        aria-label={ariaLabel}
        className="h-9 min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted"
        placeholder={placeholder}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
      {value ? (
        <Button type="button" size="icon" variant="ghost" className="h-8 min-h-8 w-8" aria-label={`Clear ${ariaLabel}`} onClick={() => onChange("")}>
          <X className="h-3.5 w-3.5" />
        </Button>
      ) : null}
    </div>
  );
}
