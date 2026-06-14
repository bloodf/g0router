import type { Page } from "@playwright/test";

export function registerInferenceHandlers(page: Page) {
  page.route("/v1/chat/completions", async (route) => {
    if (route.request().method() === "POST") {
      const body = await route.request().postDataJSON();
      const messages = body.messages || [];
      const lastMsg = messages[messages.length - 1]?.content || "";
      const response = `Hello! I'm a mock assistant. You said: "${lastMsg.slice(0, 50)}..."`;
      const created = Math.floor(Date.now() / 1000);
      const chunk = (delta: Record<string, unknown>, finish: string | null) =>
        `data: ${JSON.stringify({
          id: "mock-chat",
          object: "chat.completion.chunk",
          created,
          model: body.model,
          choices: [{ index: 0, delta, finish_reason: finish }],
        })}`;
      const sseBody = [
        chunk({ role: "assistant" }, null),
        ``,
        chunk({ content: response }, null),
        ``,
        chunk({}, "stop"),
        ``,
        `data: [DONE]`,
        ``,
      ].join("\n");
      return route.fulfill({ status: 200, headers: { "Content-Type": "text/event-stream" }, body: sseBody });
    }
    return route.continue();
  });
}
