import { useEffect, useRef, useState } from "react";
import type { TrafficEvent } from "@/lib/types";

let _id = 0;
const nextId = () => `te-${++_id}`;

export function useTrafficStream(opts: { enabled?: boolean } = {}) {
  const { enabled = true } = opts;
  const [events, setEvents] = useState<TrafficEvent[]>([]);
  const lastEvent = useRef<TrafficEvent | null>(null);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!enabled) return;

    const es = new EventSource("/api/traffic/stream");
    esRef.current = es;

    es.onmessage = (e) => {
      try {
        const raw = JSON.parse(e.data);
        const ev: TrafficEvent = {
          id: nextId(),
          timestamp: raw.timestamp,
          key_id: raw.key_id,
          provider: raw.provider,
          model: raw.model,
          status_class: raw.status_class,
          status_code: raw.status_code,
          latency_ms: raw.latency_ms,
        };
        lastEvent.current = ev;
        setEvents((prev) => [ev, ...prev].slice(0, 50));
      } catch {
        // ignore malformed events
      }
    };

    es.onerror = () => {
      // Auto-reconnect is built into EventSource
    };

    return () => {
      es.close();
      esRef.current = null;
    };
  }, [enabled]);

  return { events, lastEvent: lastEvent.current };
}
