import { describe, it, expect, vi } from "vitest";
import { renderToString } from "react-dom/server";
import {
  ConsoleLogViewer,
  subscribeConsoleLogs,
} from "./console-log-viewer";
import type { ConsoleLogEntry } from "@/lib/types";

// The authoritative console-SSE proof (w6-i §1.5). The component opens a real
// EventSource("/api/console-logs/stream") inside an effect (DOM-covered by the
// e2e). The wiring itself is extracted into the pure `subscribeConsoleLogs`
// helper so it is testable in plain node with a hand-stubbed EventSource factory
// (the w6-a oauth-popup.ts / theme.test.ts precedent — no jsdom).

interface FakeES {
  url: string;
  listeners: Record<string, ((ev: unknown) => void)[]>;
  closed: boolean;
}

function makeFactory() {
  const created: FakeES[] = [];
  const factory = (url: string) => {
    const es: FakeES = { url, listeners: {}, closed: false };
    created.push(es);
    return {
      addEventListener(type: string, fn: (ev: unknown) => void) {
        (es.listeners[type] ||= []).push(fn);
      },
      removeEventListener(type: string, fn: (ev: unknown) => void) {
        es.listeners[type] = (es.listeners[type] || []).filter((f) => f !== fn);
      },
      close() {
        es.closed = true;
      },
    };
  };
  const fire = (idx: number, type: string, ev: unknown) =>
    (created[idx].listeners[type] || []).forEach((fn) => fn(ev));
  return { factory, created, fire };
}

describe("subscribeConsoleLogs (SSE wiring)", () => {
  it("constructs EventSource('/api/console-logs/stream') on subscribe", () => {
    const { factory, created } = makeFactory();
    const stop = subscribeConsoleLogs({ onEntry: () => {}, factory });
    expect(created).toHaveLength(1);
    expect(created[0].url).toBe("/api/console-logs/stream");
    stop();
  });

  it("appends an injected message payload via onEntry", () => {
    const { factory, fire } = makeFactory();
    const seen: ConsoleLogEntry[] = [];
    const stop = subscribeConsoleLogs({
      onEntry: (e) => seen.push(e),
      factory,
    });
    fire(0, "message", {
      data: JSON.stringify({
        timestamp: "2026-06-14T10:00:00Z",
        level: "INFO",
        message: "Request routed to provider",
      }),
    });
    expect(seen).toHaveLength(1);
    expect(seen[0].message).toBe("Request routed to provider");
    expect(seen[0].level).toBe("INFO");
    stop();
  });

  it("does not throw on an error event", () => {
    const { factory, fire } = makeFactory();
    const onError = vi.fn();
    const stop = subscribeConsoleLogs({
      onEntry: () => {},
      onError,
      factory,
    });
    expect(() => fire(0, "error", new Event("error"))).not.toThrow();
    expect(onError).toHaveBeenCalledTimes(1);
    stop();
  });

  it("calls close() on the EventSource when the subscription is stopped", () => {
    const { factory, created } = makeFactory();
    const stop = subscribeConsoleLogs({ onEntry: () => {}, factory });
    expect(created[0].closed).toBe(false);
    stop();
    expect(created[0].closed).toBe(true);
  });

  it("ignores a malformed (non-JSON) message frame without throwing", () => {
    const { factory, fire } = makeFactory();
    const seen: ConsoleLogEntry[] = [];
    const stop = subscribeConsoleLogs({
      onEntry: (e) => seen.push(e),
      factory,
    });
    expect(() => fire(0, "message", { data: "not json" })).not.toThrow();
    expect(seen).toHaveLength(0);
    stop();
  });
});

describe("ConsoleLogViewer render", () => {
  it("renders each entry as a row with a level badge", () => {
    const entries: ConsoleLogEntry[] = [
      { timestamp: "2026-06-14T10:00:00Z", level: "INFO", message: "Cache hit" },
      { timestamp: "2026-06-14T10:00:01Z", level: "WARN", message: "Slow upstream" },
    ];
    const html = renderToString(<ConsoleLogViewer entries={entries} />);
    expect(html).toContain("console-log-row");
    expect(html).toContain("console-log-level");
    expect(html).toContain("Cache hit");
    expect(html).toContain("Slow upstream");
    expect(html).toContain("INFO");
    expect(html).toContain("WARN");
  });
});
