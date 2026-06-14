import * as React from "react";
import { Badge } from "@/components/ui/badge";
import type { ConsoleLogEntry } from "@/lib/types";

const STREAM_URL = "/api/console-logs/stream";
const MAX_ROWS = 200;

// Minimal EventSource surface so the subscription can be unit-tested with a
// hand-stubbed factory (w6-i §1.5 — the oauth-popup.ts precedent, no jsdom).
interface EventSourceLike {
  addEventListener(type: string, fn: (ev: unknown) => void): void;
  removeEventListener(type: string, fn: (ev: unknown) => void): void;
  close(): void;
}

export interface SubscribeConsoleLogsOptions {
  onEntry: (entry: ConsoleLogEntry) => void;
  onError?: (ev: unknown) => void;
  factory?: (url: string) => EventSourceLike;
}

// Pure SSE wiring for /api/console-logs/stream. Constructs the EventSource (via
// the injectable factory, defaulting to the global), forwards parsed message
// payloads to onEntry, swallows malformed frames, calls onError without throwing
// on the error event, and returns a cleanup that closes the stream.
export function subscribeConsoleLogs(
  opts: SubscribeConsoleLogsOptions
): () => void {
  const { onEntry, onError, factory } = opts;
  const make =
    factory ?? ((url: string) => new EventSource(url) as unknown as EventSourceLike);
  const es = make(STREAM_URL);

  const onMessage = (ev: unknown) => {
    const data = (ev as MessageEvent).data;
    try {
      const entry = JSON.parse(String(data)) as ConsoleLogEntry;
      onEntry(entry);
    } catch {
      // ignore malformed frames
    }
  };
  const handleError = (ev: unknown) => {
    onError?.(ev);
  };

  es.addEventListener("message", onMessage);
  es.addEventListener("error", handleError);

  return () => {
    es.removeEventListener("message", onMessage);
    es.removeEventListener("error", handleError);
    es.close();
  };
}

function levelVariant(level: string): "success" | "error" | "neutral" | "primary" {
  switch (level.toUpperCase()) {
    case "ERROR":
      return "error";
    case "WARN":
      return "primary";
    case "DEBUG":
      return "neutral";
    default:
      return "success";
  }
}

export interface ConsoleLogViewerProps {
  entries: ConsoleLogEntry[];
}

export function ConsoleLogViewer({ entries }: ConsoleLogViewerProps) {
  return (
    <div className="flex flex-col gap-1 font-mono text-xs">
      {entries.length === 0 ? (
        <p className="text-muted-foreground">Waiting for console output…</p>
      ) : (
        entries.map((entry, i) => (
          <div
            key={`${entry.timestamp}-${i}`}
            data-testid="console-log-row"
            className="flex items-center gap-2 border-b border-border/40 py-1"
          >
            <span className="whitespace-nowrap text-muted-foreground">
              {entry.timestamp}
            </span>
            <Badge
              data-testid="console-log-level"
              variant={levelVariant(entry.level)}
              size="sm"
            >
              {entry.level}
            </Badge>
            <span className="text-foreground">{entry.message}</span>
          </div>
        ))
      )}
    </div>
  );
}

// Stateful wrapper that opens the live stream and renders the rolling buffer.
// The streaming lifecycle is DOM-covered by the console e2e (the pure helper
// above is the unit-tested SSE proof).
export function LiveConsoleLogViewer() {
  const [entries, setEntries] = React.useState<ConsoleLogEntry[]>([]);

  React.useEffect(() => {
    if (typeof EventSource === "undefined") return;
    const stop = subscribeConsoleLogs({
      onEntry: (entry) =>
        setEntries((prev) => [...prev, entry].slice(-MAX_ROWS)),
    });
    return stop;
  }, []);

  return <ConsoleLogViewer entries={entries} />;
}
