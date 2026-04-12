import { LoaderCircle } from "lucide-react";
import { useEffect, useId } from "react";
import { createPortal } from "react-dom";
import { Button } from "../ui/button";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  cancelLabel?: string;
  tone?: "danger" | "neutral";
  isPending?: boolean;
  onConfirm: () => void;
  onOpenChange: (open: boolean) => void;
}

export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  cancelLabel = "Cancel",
  tone = "danger",
  isPending = false,
  onConfirm,
  onOpenChange,
}: ConfirmDialogProps) {
  const titleId = useId();
  const descriptionId = useId();

  useEffect(() => {
    if (!open) {
      return undefined;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !isPending) {
        onOpenChange(false);
      }
    };

    window.addEventListener("keydown", handleEscape);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", handleEscape);
    };
  }, [isPending, onOpenChange, open]);

  if (!open) {
    return null;
  }

  return createPortal(
    <div
      className="fixed inset-0 z-[300] flex items-center justify-center px-4 py-6"
      aria-labelledby={titleId}
      aria-describedby={descriptionId}
      aria-modal="true"
      role="alertdialog"
    >
      <button
        type="button"
        aria-label="Close confirmation dialog"
        className="absolute inset-0 bg-[color:var(--overlay)] backdrop-blur-sm"
        disabled={isPending}
        onClick={() => onOpenChange(false)}
      />
      <div className="relative z-[301] w-full max-w-lg rounded-3xl border border-border bg-surface p-6 shadow-lg">
        <h2 id={titleId} className="text-xl font-semibold tracking-tight text-foreground">
          {title}
        </h2>
        <p id={descriptionId} className="mt-3 text-sm leading-relaxed text-secondary">
          {description}
        </p>
        <div className="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <Button type="button" variant="secondary" disabled={isPending} onClick={() => onOpenChange(false)}>
            {cancelLabel}
          </Button>
          <Button
            type="button"
            variant={tone === "danger" ? "danger" : "primary"}
            disabled={isPending}
            onClick={onConfirm}
          >
            {isPending ? (
              <>
                <LoaderCircle className="mr-2 h-4 w-4 animate-spin motion-reduce:animate-none" />
                Working...
              </>
            ) : (
              confirmLabel
            )}
          </Button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
