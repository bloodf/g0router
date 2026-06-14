import { describe, it, expect, vi } from "vitest";
import { renderToString } from "react-dom/server";
import { ChatWindow, streamChatCompletion } from "./chat-window";

// The chat send/receive proof (w6-i §1.3). The stream consumption is extracted
// into the pure `streamChatCompletion` helper so it can be unit-tested with a
// stubbed fetch returning a ReadableStream of OpenAI chunks — the same shape the
// e2e inference.ts mock emits (chat.completion.chunk with choices[].delta.content).
// The chosen approach is a plain-fetch ReadableStream reader (NOT @ai-sdk/react),
// because @ai-sdk/react@3's DefaultChatTransport expects the AI SDK UI-message
// stream protocol, not raw OpenAI SSE — pointing it at the raw route would need a
// new adapter dependency (forbidden). No dependency added either way (§1.3 point 2).

function sseResponse(chunks: string[]): Response {
  const encoder = new TextEncoder();
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const c of chunks) controller.enqueue(encoder.encode(c));
      controller.close();
    },
  });
  return new Response(body, {
    status: 200,
    headers: { "Content-Type": "text/event-stream" },
  });
}

describe("streamChatCompletion", () => {
  it("POSTs /v1/chat/completions and streams delta.content via onDelta", async () => {
    const chunks = [
      `data: {"choices":[{"delta":{"role":"assistant"}}]}\n\n`,
      `data: {"choices":[{"delta":{"content":"Hello! I'm a "}}]}\n\n`,
      `data: {"choices":[{"delta":{"content":"mock assistant"}}]}\n\n`,
      `data: {"choices":[{"delta":{},"finish_reason":"stop"}]}\n\n`,
      `data: [DONE]\n\n`,
    ];
    const fetchFn = vi.fn().mockResolvedValue(sseResponse(chunks));
    const deltas: string[] = [];
    await streamChatCompletion({
      url: "/v1/chat/completions",
      body: { model: "gpt-4o", messages: [{ role: "user", content: "hi" }] },
      onDelta: (d) => deltas.push(d),
      fetchFn,
    });
    expect(fetchFn).toHaveBeenCalledTimes(1);
    const [calledUrl, init] = fetchFn.mock.calls[0];
    expect(String(calledUrl)).toContain("/v1/chat/completions");
    expect(init.method).toBe("POST");
    expect(deltas.join("")).toBe("Hello! I'm a mock assistant");
  });

  it("stops at [DONE] without emitting empty deltas", async () => {
    const fetchFn = vi
      .fn()
      .mockResolvedValue(
        sseResponse([
          `data: {"choices":[{"delta":{"content":"x"}}]}\n\n`,
          `data: [DONE]\n\n`,
        ])
      );
    const deltas: string[] = [];
    await streamChatCompletion({
      url: "/v1/chat/completions",
      body: { model: "m", messages: [] },
      onDelta: (d) => deltas.push(d),
      fetchFn,
    });
    expect(deltas).toEqual(["x"]);
  });
});

describe("ChatWindow render", () => {
  it("renders a Message input and the message list", () => {
    const html = renderToString(<ChatWindow />);
    expect(html).toContain('aria-label="Message"');
    expect(html).toContain("chat-model-select");
  });
});
