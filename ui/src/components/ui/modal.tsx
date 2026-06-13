import * as React from "react";

import { cn } from "@/lib/utils";

const modalSizes = {
  sm: "max-w-sm",
  md: "max-w-md",
  lg: "max-w-lg",
  xl: "max-w-2xl",
} as const;

export interface ModalProps {
  open: boolean;
  onClose: () => void;
  title?: React.ReactNode;
  size?: keyof typeof modalSizes;
  children?: React.ReactNode;
}

function Modal({ open, onClose, title, size = "md", children }: ModalProps) {
  React.useEffect(() => {
    if (!open) return;

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
      }
    }

    document.addEventListener("keydown", onKeyDown);
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = previousOverflow;
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      data-testid="modal-overlay"
      onClick={onClose}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
    >
      <div
        role="dialog"
        aria-modal="true"
        onClick={(event) => event.stopPropagation()}
        className={cn(
          "w-full rounded-xl border border-border bg-card text-card-foreground shadow-lg",
          modalSizes[size]
        )}
      >
        <div className="flex items-center gap-3 border-b border-border px-4 py-3">
          <div data-testid="modal-traffic-lights" className="flex items-center gap-1.5">
            <span data-testid="traffic-dot" className="size-3 rounded-full bg-red-500" />
            <span data-testid="traffic-dot" className="size-3 rounded-full bg-yellow-500" />
            <span data-testid="traffic-dot" className="size-3 rounded-full bg-green-500" />
          </div>
          {title ? (
            <h2 className="text-sm font-semibold text-foreground">{title}</h2>
          ) : null}
        </div>
        <div className="p-4">{children}</div>
      </div>
    </div>
  );
}

export { Modal };
