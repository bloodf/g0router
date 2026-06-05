import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TrafficPage } from "./TrafficPage";

// Helper: build a ReadableStream from a sequence of SSE chunks.
function makeStreamResponse(chunks: string[], status = 200): Response {
  const encoder = new TextEncoder();
  let chunkIndex = 0;
  const stream = new ReadableStream<Uint8Array>({
    pull(controller) {
      if (chunkIndex < chunks.length) {
        controller.enqueue(encoder.encode(chunks[chunkIndex++]));
      } else {
        controller.close();
      }
    }
  });
  return new Response(stream, {
    status,
    headers: { "Content-Type": "text/event-stream" }
  });
}

function makeEvent(overrides: Partial<{
  timestamp: string;
  key_id: string;
  provider: string;
  model: string;
  status_class: string;
  status_code: number;
  latency_ms: number;
}> = {}) {
  return {
    timestamp: "2026-06-05T10:00:00Z",
    key_id: "key-abc",
    provider: "openai",
    model: "gpt-4o",
    status_class: "2xx",
    status_code: 200,
    latency_ms: 150,
    ...overrides
  };
}

describe("TrafficPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders the gateway node immediately", () => {
    // Stream never resolves — stays in connecting state.
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<TrafficPage />);

    expect(screen.getByText("gateway")).toBeInTheDocument();
  });

  it("shows waiting state before any events arrive", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<TrafficPage />);

    expect(screen.getByText(/waiting for live traffic/i)).toBeInTheDocument();
  });

  it("renders provider node after receiving an event", async () => {
    const event = makeEvent({ provider: "openai", key_id: "key-abc" });
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve(
          makeStreamResponse([`data: ${JSON.stringify(event)}\n\n`])
        )
      )
    );

    render(<TrafficPage />);

    expect(await screen.findByText("openai")).toBeInTheDocument();
  });

  it("renders key node after receiving an event", async () => {
    const event = makeEvent({ provider: "anthropic", key_id: "key-xyz" });
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve(
          makeStreamResponse([`data: ${JSON.stringify(event)}\n\n`])
        )
      )
    );

    render(<TrafficPage />);

    expect(await screen.findByText("key-xyz")).toBeInTheDocument();
  });

  it("groups absent key_id as 'anonymous'", async () => {
    const event = makeEvent({ key_id: "" });
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve(
          makeStreamResponse([`data: ${JSON.stringify(event)}\n\n`])
        )
      )
    );

    render(<TrafficPage />);

    expect(await screen.findByText("anonymous")).toBeInTheDocument();
  });

  it("ignores SSE comment/ping lines", async () => {
    const event = makeEvent({ provider: "cohere", key_id: "key-q" });
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve(
          makeStreamResponse([
            `: ping\n\n`,
            `data: ${JSON.stringify(event)}\n\n`
          ])
        )
      )
    );

    render(<TrafficPage />);

    expect(await screen.findByText("cohere")).toBeInTheDocument();
  });

  it("shows error state on fetch failure (non-2xx)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve(
          new Response(JSON.stringify({ error: "stream unavailable" }), {
            status: 500,
            statusText: "Server Error",
            headers: { "Content-Type": "application/json" }
          })
        )
      )
    );

    render(<TrafficPage />);

    expect(await screen.findByText("Could not connect to traffic stream")).toBeInTheDocument();
  });

  it("shows auth-expired state on 401", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve(
          new Response(JSON.stringify({ error: "control-plane auth required" }), {
            status: 401,
            statusText: "Unauthorized",
            headers: { "Content-Type": "application/json" }
          })
        )
      )
    );

    render(<TrafficPage />);

    expect(await screen.findByText("Session expired")).toBeInTheDocument();
  });

  it("sends Authorization header using saved control-plane key", async () => {
    const fetch = vi.fn(() => new Promise<Response>(() => undefined));
    vi.stubGlobal("fetch", fetch);
    vi.stubGlobal("localStorage", {
      getItem: (k: string) => (k === "g0router.controlPlaneKey" ? "test-key-123" : null),
      setItem: vi.fn(),
      removeItem: vi.fn()
    });

    render(<TrafficPage />);

    // Wait for the fetch call.
    await vi.waitFor(() => expect(fetch).toHaveBeenCalled());
    const [, opts] = (fetch.mock.calls[0] as unknown) as [string, RequestInit];
    expect((opts.headers as Record<string, string>)["Authorization"]).toBe("Bearer test-key-123");
  });

  it("renders multiple provider nodes from multiple events", async () => {
    const ev1 = makeEvent({ provider: "openai", key_id: "k1" });
    const ev2 = makeEvent({ provider: "anthropic", key_id: "k2" });
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve(
          makeStreamResponse([
            `data: ${JSON.stringify(ev1)}\n\n`,
            `data: ${JSON.stringify(ev2)}\n\n`
          ])
        )
      )
    );

    render(<TrafficPage />);

    expect(await screen.findByText("openai")).toBeInTheDocument();
    expect(await screen.findByText("anthropic")).toBeInTheDocument();
  });
});
