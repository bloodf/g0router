import { Button } from "@/components/ui/button";

export interface TranslatorStepProps {
  label: string;
  description: string;
  value: string;
  onChange: (value: string) => void;
  onLoad?: () => void;
}

// One step panel of the request/response transformation inspector. The editor is
// a plain monospaced <textarea> (NO Monaco/CodeMirror — neither is installed;
// w6-i §1.6 textarea variant). The textarea carries the step label as its
// aria-label (the translator e2e selector).
export function TranslatorStep({
  label,
  description,
  value,
  onChange,
  onLoad,
}: TranslatorStepProps) {
  return (
    <div
      data-testid="translator-step"
      className="flex flex-col gap-2 rounded-lg border border-border bg-card p-3"
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-col">
          <span className="text-sm font-medium text-foreground">{label}</span>
          <span className="text-xs text-muted-foreground">{description}</span>
        </div>
        {onLoad ? (
          <Button
            type="button"
            variant="secondary"
            size="sm"
            data-testid="translator-load"
            onClick={onLoad}
          >
            Load
          </Button>
        ) : null}
      </div>
      <textarea
        aria-label={label}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={6}
        spellCheck={false}
        className="w-full rounded-md border border-input bg-transparent p-2 font-mono text-xs text-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
      />
    </div>
  );
}
