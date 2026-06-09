import type { Page } from "@playwright/test";
import type { MockStore } from "../store";

export function registerStreamsHandlers(page: Page, store: MockStore) {
  page.route("/api/traffic/stream", async (route) => {
    if (route.request().method() === "GET") {
      const body = `data: {"timestamp":"${new Date().toISOString()}","key_id":"key-1","provider":"openai","model":"gpt-4o","status_class":"2xx","status_code":200,"latency_ms":120}\n\n`;
      return route.fulfill({ status: 200, headers: { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" }, body });
    }
    return route.continue();
  });
  page.route("/api/console-logs/stream", async (route) => {
    if (route.request().method() === "GET") {
      const body = `event: log\ndata: {"timestamp":"${new Date().toISOString()}","level":"INFO","message":"Mock console stream active"}\n\n`;
      return route.fulfill({ status: 200, headers: { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" }, body });
    }
    return route.continue();
  });
  page.route("/api/console-logs", async (route) => {
    if (route.request().method() === "DELETE") {
      store.consoleLogs = [];
      return route.fulfill({ status: 200, body: JSON.stringify({ data: {} }) });
    }
    return route.continue();
  });
}
