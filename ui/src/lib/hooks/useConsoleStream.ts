import { useEffect, useRef, useState } from "react";
import type { ConsoleLogEntry } from "@/lib/types";

export function useConsoleStream(opts: { enabled?: boolean } = {}) {
  const { enabled = true } = opts;
  const [logs, setLogs] = useState<ConsoleLogEntry[]>([]);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!enabled) return;

    const es = new EventSource("/api/console-logs/stream");
    esRef.current = es;

    es.addEventListener("log", (e) => {
      try {
        const entry: ConsoleLogEntry = JSON.parse((e as MessageEvent).data);
        setLogs((prev) => [entry, ...prev].slice(0, 200));
      } catch {
        // ignore malformed events
      }
    });

    es.onerror = () => {
      // Auto-reconnect is built into EventSource
    };

    return () => {
      es.close();
      esRef.current = null;
    };
  }, [enabled]);

  const clear = async () => {
    try {
      await fetch("/api/console-logs", { method: "DELETE", credentials: "same-origin" });
      setLogs([]);
    } catch {
      // ignore
    }
  };

  return { logs, clear };
}
