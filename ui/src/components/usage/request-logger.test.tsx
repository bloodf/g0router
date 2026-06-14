import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderToString } from "react-dom/server";
import {
  RequestLogger,
  normalizeLogRow,
  startLogPolling,
} from "./request-logger";
import type { UsageLog } from "@/lib/types";

const sampleLog: UsageLog = {
  id: "log-1",
  timestamp: "2026-06-14T10:00:00Z",
  provider: "openai",
  model: "gpt-4o",
  api_key_id: "key-1",
  api_key_name: "Default Key",
  status: "success",
  status_code: 200,
  prompt_tokens: 100,
  completion_tokens: 50,
  total_tokens: 150,
  cost_usd: 0.01,
  latency_ms: 320,
  rtk_enabled: false,
  caveman_enabled: false,
};

describe("RequestLogger render (REST-driven)", () => {
  it("renders rows from a REST logs payload (UsageLog[])", () => {
    const html = renderToString(<RequestLogger initialLogs={[sampleLog]} />);
    expect(html).toContain("request-log-table");
    expect(html).toContain("gpt-4o");
    expect(html).toContain("openai");
  });

  it("normalizes a real-Go pipe-delimited string log into a row", () => {
    // Real Go (internal/usage/logs.go:41): "ts | model | PROVIDER | account | sent | received | status"
    const row = normalizeLogRow("14-06-2026 10:00:00 | gpt-4o | OPENAI | acc | 100 | 50 | success");
    expect(row.model).toBe("gpt-4o");
    expect(row.provider.toLowerCase()).toBe("openai");
    expect(row.status).toBe("success");
  });

  it("normalizes a structured UsageLog object into a row", () => {
    const row = normalizeLogRow(sampleLog);
    expect(row.model).toBe("gpt-4o");
    expect(row.provider).toBe("openai");
    expect(row.cost_usd).toBeCloseTo(0.01);
    expect(row.latency_ms).toBe(320);
  });
});

describe("startLogPolling (3s auto-refresh)", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("polls at the 3000ms interval and stops when the cleanup runs", () => {
    const fetchFn = vi.fn().mockResolvedValue([sampleLog]);
    const stop = startLogPolling(fetchFn, 3000);
    expect(fetchFn).toHaveBeenCalledTimes(0);
    vi.advanceTimersByTime(3000);
    expect(fetchFn).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(3000);
    expect(fetchFn).toHaveBeenCalledTimes(2);
    stop();
    vi.advanceTimersByTime(9000);
    expect(fetchFn).toHaveBeenCalledTimes(2);
  });
});
