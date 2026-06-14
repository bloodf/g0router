import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderToString } from "react-dom/server";
import {
  UsageStats,
  mergeUsageStats,
  subscribeUsageStream,
  type UsageStatsData,
} from "./usage-stats";

// Plain-node unit tests (no jsdom) — hand-stub EventSource, following the w6-a
// stores/theme.test.ts precedent. The SSE behaviour lives in pure helpers
// (mergeUsageStats / subscribeUsageStream) so it is provable without effects.

// A controllable EventSource stub that mirrors the e2e MockEventSource
// (fixture.ts) — an EventTarget driven by addEventListener/dispatchEvent. This
// is exactly why subscribeUsageStream must use addEventListener (not .onmessage=).
class FakeEventSource extends EventTarget {
  static last: FakeEventSource | null = null;
  static instances: FakeEventSource[] = [];
  url: string;
  closed = false;
  constructor(url: string) {
    super();
    this.url = url;
    FakeEventSource.last = this;
    FakeEventSource.instances.push(this);
  }
  emit(payload: unknown) {
    this.dispatchEvent(new MessageEvent("message", { data: JSON.stringify(payload) }));
  }
  fail() {
    this.dispatchEvent(new Event("error"));
  }
  close() {
    this.closed = true;
  }
}

function stubEventSource() {
  FakeEventSource.last = null;
  FakeEventSource.instances = [];
  (globalThis as Record<string, unknown>).EventSource = FakeEventSource as unknown as typeof EventSource;
}

const baseStats: UsageStatsData = {
  total_requests: 42,
  total_prompt_tokens: 1000,
  total_completion_tokens: 500,
  total_cost: 1.23,
  by_provider: { openai: { requests: 42, prompt_tokens: 1000, completion_tokens: 500, cost: 1.23 } },
  by_model: {},
  active_requests: [],
  recent_requests: [],
  pending: {},
  error_provider: "",
};

beforeEach(() => {
  stubEventSource();
});

describe("UsageStats render (REST-driven)", () => {
  it("renders overview metric cards from a REST stats payload", () => {
    const html = renderToString(<UsageStats period="all" initialStats={baseStats} />);
    // Metric cards expose data-testid="usage-metric"; the request count shows.
    expect(html).toContain("usage-metric");
    expect(html).toContain("42");
  });
});

describe("subscribeUsageStream (SSE proof)", () => {
  it("constructs an EventSource at /api/usage/stream", () => {
    const cleanup = subscribeUsageStream({ onData: () => {}, onError: () => {} });
    expect(FakeEventSource.last).not.toBeNull();
    expect(FakeEventSource.last!.url).toBe("/api/usage/stream");
    cleanup();
  });

  it("parses an injected message and delivers it to onData", () => {
    const onData = vi.fn();
    const cleanup = subscribeUsageStream({ onData, onError: () => {} });
    FakeEventSource.last!.emit({ active_requests: [{ model: "gpt-4o", provider: "openai", account: "a", count: 3 }] });
    expect(onData).toHaveBeenCalledTimes(1);
    expect(onData.mock.calls[0][0].active_requests[0].count).toBe(3);
    cleanup();
  });

  it("survives an error event without throwing and notifies onError", () => {
    const onError = vi.fn();
    const cleanup = subscribeUsageStream({ onData: () => {}, onError });
    expect(() => FakeEventSource.last!.fail()).not.toThrow();
    expect(onError).toHaveBeenCalledTimes(1);
    cleanup();
  });

  it("closes the EventSource when the cleanup runs (unmount)", () => {
    const cleanup = subscribeUsageStream({ onData: () => {}, onError: () => {} });
    const es = FakeEventSource.last!;
    cleanup();
    expect(es.closed).toBe(true);
  });
});

describe("mergeUsageStats (additive overlay)", () => {
  it("overlays live active_requests/recent_requests/pending/error_provider onto base", () => {
    const merged = mergeUsageStats(baseStats, {
      active_requests: [{ model: "gpt-4o", provider: "openai", account: "a", count: 7 }],
      recent_requests: [{ timestamp: "t", model: "gpt-4o", provider: "openai", prompt_tokens: 1, completion_tokens: 2, status: "success" }],
      pending: { openai: 2 },
      error_provider: "anthropic",
    });
    // Live fields overlaid.
    expect(merged.active_requests[0].count).toBe(7);
    expect(merged.recent_requests).toHaveLength(1);
    expect(merged.pending.openai).toBe(2);
    expect(merged.error_provider).toBe("anthropic");
    // Aggregate REST fields preserved.
    expect(merged.total_requests).toBe(42);
    expect(merged.total_cost).toBeCloseTo(1.23);
  });

  it("ignores absent live fields and keeps the base values", () => {
    const merged = mergeUsageStats(baseStats, {});
    expect(merged.total_requests).toBe(42);
    expect(merged.active_requests).toHaveLength(0);
  });
});
