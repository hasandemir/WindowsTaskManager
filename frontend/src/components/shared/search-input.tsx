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
    <input
      aria-label={ariaLabel}
      className={`min-h-11 w-full rounded-2xl border border-border bg-surface px-4 py-2 text-sm text-foreground outline-none transition-colors duration-150 placeholder:text-muted focus-visible:ring-2 focus-visible:ring-[var(--ring)] focus-visible:ring-offset-2 focus-visible:ring-offset-background ${widthClassName}`}
      placeholder={placeholder}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}
