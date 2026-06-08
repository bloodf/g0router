import { useEffect, useRef, useState, useCallback } from "react";
import type { TrafficEvent } from "@/lib/types";

export type ConnectionStatus = "connecting" | "open" | "closed" | "error";

export function useTrafficStream(opts: { enabled?: boolean } = {}) {
  const { enabled = true } = opts;
  const [events, setEvents] = useState<TrafficEvent[]>([]);
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
    const es = new EventSource("/api/traffic/stream");
    esRef.current = es;

    es.onopen = () => {
      setStatus("open");
      backoffRef.current = 1000;
    };

    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        const ev: TrafficEvent = {
          id: `${data.timestamp}-${Math.random().toString(36).slice(2, 8)}`,
          ...data,
        };
        setEvents((prev) => [ev, ...prev].slice(0, 50));
      } catch {
        // ignore malformed events
      }
    };

    es.onerror = () => {
      setStatus("error");
      es.close();
      esRef.current = null;
      // Exponential backoff capped at 30s
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

  const lastEvent = events[0] ?? null;

  return { events, lastEvent, status };
}
