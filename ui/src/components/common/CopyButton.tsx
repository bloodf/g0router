import { useState, type ReactNode } from "react";
import { Icon } from "./Icon";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

export function CopyButton({
  value,
  label,
  className,
  variant = "ghost",
  disabled,
  successMessage = "Copied to clipboard",
  errorMessage = "Couldn't copy to clipboard",
  onSuccess,
  onError,
}: {
  value: string;
  label?: string;
  className?: string;
  variant?: "ghost" | "outline" | "default";
  disabled?: boolean;
  successMessage?: string;
  errorMessage?: string;
  onSuccess?: () => void;
  onError?: (err: unknown) => void;
}) {
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);
  const onCopy = async () => {
    if (busy || disabled) return;
    setBusy(true);
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      toast.success(successMessage);
      onSuccess?.();
      setTimeout(() => setCopied(false), 1500);
    } catch (err) {
      toast.error(errorMessage);
      onError?.(err);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Button
      variant={variant}
      size="sm"
      onClick={onCopy}
      className={className}
      aria-label="Copy"
      disabled={disabled || busy}
    >
      <Icon name={busy ? "hourglass_empty" : copied ? "check" : "content_copy"} size={14} className={busy ? "animate-spin" : ""} />
      {label && <span className="ml-1.5">{label}</span>}
    </Button>
  );
}

export function CopyableCode({ value, children }: { value: string; children?: ReactNode }) {
  return (
    <div className="flex items-center gap-2 bg-surface-2 border border-border rounded-lg px-3 py-2 font-mono text-sm">
      <span className="flex-1 truncate">{children ?? value}</span>
      <CopyButton value={value} />
    </div>
  );
}
