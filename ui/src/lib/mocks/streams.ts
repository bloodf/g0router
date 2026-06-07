import { useEffect, useRef, useState } from "react";
import { getStore } from "../mocks/store";
import type { ConsoleLogEntry, TrafficEvent } from "../mocks/types";

const id = () => Math.random().toString(36).slice(2, 10);
const pick = <T>(arr: T[]): T => arr[Math.floor(Math.random() * arr.length)];
const rand = (min: number, max: number) =>
  Math.floor(Math.random() * (max - min + 1)) + min;

/** Mock SSE: traffic events. Emits every 400-1500ms. */
export function useTrafficStream(opts: { enabled?: boolean; speed?: number } = {}) {
  const { enabled = true, speed = 1 } = opts;
  const [events, setEvents] = useState<TrafficEvent[]>([]);
  const lastEvent = useRef<TrafficEvent | null>(null);

  useEffect(() => {
    if (!enabled) return;
    let timer: ReturnType<typeof setTimeout>;
    const tick = () => {
      const s = getStore();
      const conns = s.connections.filter((c) => c.is_active);
      const keys = s.keys.filter((k) => k.is_active);
      if (conns.length && keys.length) {
        const conn = pick(conns);
        const key = pick(keys);
        const combo = Math.random() > 0.6 ? pick(s.combos) : undefined;
        const ev: TrafficEvent = {
          id: id(),
          timestamp: new Date().toISOString(),
          api_key_id: key.id,
          api_key_name: key.name,
          provider: conn.provider,
          model: conn.models[0] ?? "unknown",
          combo_id: combo?.id,
          status: Math.random() > 0.1 ? "success" : "error",
          tokens: rand(100, 5000),
          latency_ms: rand(120, 3500),
          cost_usd: Math.random() * 0.05,
        };
        lastEvent.current = ev;
        setEvents((prev) => [ev, ...prev].slice(0, 50));
      }
      timer = setTimeout(tick, rand(400, 1500) / speed);
    };
    timer = setTimeout(tick, 600 / speed);
    return () => clearTimeout(timer);
  }, [enabled, speed]);

  return { events, lastEvent: lastEvent.current };
}

const LOG_TEMPLATES = [
  { level: "INFO" as const, msg: "router: forwarded request to {provider}/{model}" },
  { level: "LOG" as const, msg: "cache: hit on prompt fingerprint {hash}" },
  { level: "WARN" as const, msg: "rate-limit: {provider} approaching threshold ({pct}%)" },
  { level: "ERROR" as const, msg: "connection {conn}: 401 Unauthorized — needs re-auth" },
  { level: "DEBUG" as const, msg: "combo {combo}: step 1/3 selected" },
  { level: "INFO" as const, msg: "auth: user 'admin' verified session" },
  { level: "LOG" as const, msg: "metrics: {n} reqs in last minute" },
];

export function useConsoleStream(opts: { enabled?: boolean } = {}) {
  const { enabled = true } = opts;
  const [logs, setLogs] = useState<ConsoleLogEntry[]>([]);

  useEffect(() => {
    if (!enabled) return;
    let timer: ReturnType<typeof setTimeout>;
    const tick = () => {
      const t = pick(LOG_TEMPLATES);
      const s = getStore();
      const message = t.msg
        .replace("{provider}", pick(s.providers).id)
        .replace("{model}", "gpt-4o")
        .replace("{hash}", id())
        .replace("{pct}", String(rand(70, 95)))
        .replace("{conn}", pick(s.connections)?.name ?? "default")
        .replace("{combo}", pick(s.combos)?.name ?? "default")
        .replace("{n}", String(rand(5, 200)));
      const entry: ConsoleLogEntry = {
        id: id(),
        timestamp: new Date().toISOString(),
        level: t.level,
        message,
      };
      setLogs((prev) => [entry, ...prev].slice(0, 200));
      timer = setTimeout(tick, rand(500, 2000));
    };
    timer = setTimeout(tick, 700);
    return () => clearTimeout(timer);
  }, [enabled]);

  const clear = () => setLogs([]);
  return { logs, clear };
}

/** Mock chat WS — stream a word-by-word response. */
export function streamMockChat(
  prompt: string,
  onDelta: (chunk: string) => void,
  onDone: (full: string) => void,
  signal?: AbortSignal,
) {
  const responses = [
    "That's a great question. Let me break it down step by step.\n\n1. First, consider the core idea.\n2. Then, examine the trade-offs.\n3. Finally, choose what fits your context.",
    "Here's what I'd recommend:\n\n- Start with the simplest possible version.\n- Measure before optimizing.\n- Iterate based on feedback.\n\n`Simple is better than clever.`",
    `Based on \`${prompt.slice(0, 30)}\`, I'd suggest looking at three angles: clarity, scalability, and developer experience. Each matters differently depending on your team size.`,
    "I can help with that. Could you clarify whether you want a code-first walkthrough, a high-level concept overview, or both?",
  ];
  const full = responses[Math.floor(Math.random() * responses.length)];
  const words = full.split(/(\s+)/);
  let i = 0;
  let acc = "";
  const tick = () => {
    if (signal?.aborted) return;
    if (i >= words.length) {
      onDone(acc);
      return;
    }
    acc += words[i];
    onDelta(words[i]);
    i++;
    setTimeout(tick, 20 + Math.random() * 60);
  };
  setTimeout(tick, 300);
}
