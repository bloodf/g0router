import { useEffect, useRef, useState, useCallback } from "react";
import type { ConsoleLogEntry } from "@/lib/types";

export type ConnectionStatus = "connecting" | "open" | "closed" | "error";

export function useConsoleStream(opts: { enabled?: boolean } = {}) {
  const { enabled = true } = opts;
  const [logs, setLogs] = useState<ConsoleLogEntry[]>([]);
  const [status, setStatus] = useState<ConnectionStatus>("closed");
  const esRef = useRef<EventSource | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const backoffRef = useRef(1000);

  const connect = useCallback(() => {
    if (!enabled) return;
    if (esRef.current) {
      esRef.current.close();
    }

    setStatus("connecting");
    const es = new EventSource("/api/console-logs/stream");
    esRef.current = es;

    es.onopen = () => {
      setStatus("open");
      backoffRef.current = 1000;
    };

    es.addEventListener("log", (e) => {
      try {
        const entry: ConsoleLogEntry = JSON.parse((e as MessageEvent).data);
        setLogs((prev) => [entry, ...prev].slice(0, 200));
      } catch {
        // ignore malformed events
      }
    });

    es.onerror = () => {
      setStatus("error");
      es.close();
      esRef.current = null;
      const delay = Math.min(backoffRef.current, 30_000);
      backoffRef.current *= 2;
      reconnectTimerRef.current = setTimeout(connect, delay);
    };
  }, [enabled]);

  useEffect(() => {
    connect();
    return () => {
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
      if (esRef.current) {
        esRef.current.close();
        esRef.current = null;
      }
    };
  }, [connect]);

  const clear = async () => {
    try {
      await fetch("/api/console-logs", { method: "DELETE", credentials: "same-origin" });
      setLogs([]);
    } catch {
      // ignore
    }
  };

  return { logs, clear, status };
}
