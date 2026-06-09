import { test as base, expect } from "@playwright/test";
import { MockStore } from "./store";
import { setupMockApi } from "./handlers/index";

export type MockFixtures = {
  mockStore: MockStore;
};

// Worker-scoped store so serial tests can share entity data.
// Auth state is reset before each test; tests that need auth must call login().
export const test = base.extend<MockFixtures, { workerStore: MockStore }>({
  workerStore: [async ({}, use) => {
    const store = new MockStore();
    store.seedAll();
    await use(store);
  }, { scope: "worker" }],

  mockStore: async ({ workerStore }, use) => {
    // Reset auth to default (logged out) but keep all seeded entity data
    workerStore.auth = {
      require_login: true,
      has_users: true,
      authenticated: false,
      username: "admin",
      display_name: "Administrator",
      role: "admin",
    };
    await use(workerStore);
  },

  page: async ({ page, mockStore }, use) => {
    setupMockApi(page, mockStore);
    await page.addInitScript(() => {
      const originalEventSource = window.EventSource;
      class MockEventSource extends EventTarget {
        url: string;
        readyState = 0;
        private timer: ReturnType<typeof setInterval> | null = null;
        private trafficProviders = ["openai", "anthropic", "gemini", "groq", "mistral", "deepseek"];
        private trafficModels: Record<string, string[]> = {
          openai: ["gpt-4o", "gpt-4o-mini"],
          anthropic: ["claude-sonnet-4", "claude-haiku"],
          gemini: ["gemini-2.5-flash", "gemini-2.5-pro"],
          groq: ["llama-3.3-70b-versatile", "llama-3.1-8b-instant"],
          mistral: ["mistral-large-latest", "mistral-small-latest"],
          deepseek: ["deepseek-chat", "deepseek-reasoner"],
        };

        constructor(url: string | URL, _init?: EventSourceInit) {
          super();
          this.url = String(url);
          setTimeout(() => {
            this.readyState = 1;
            this.dispatchEvent(new Event("open"));
            this.startStreaming();
          }, 50);
        }

        private startStreaming() {
          if (this.url.includes("/api/traffic/stream")) {
            this.timer = setInterval(() => {
              const provider = this.trafficProviders[Math.floor(Math.random() * this.trafficProviders.length)];
              const models = this.trafficModels[provider] || ["unknown"];
              const model = models[Math.floor(Math.random() * models.length)];
              const ev = new MessageEvent("message", {
                data: JSON.stringify({
                  timestamp: new Date().toISOString(),
                  key_id: `key-${Math.floor(Math.random() * 5) + 1}`,
                  provider,
                  model,
                  status_class: Math.random() > 0.1 ? "2xx" : "5xx",
                  status_code: Math.random() > 0.1 ? 200 : 500,
                  latency_ms: Math.floor(Math.random() * 400) + 50,
                }),
              });
              this.dispatchEvent(ev);
            }, 1200);
          } else if (this.url.includes("/api/console-logs/stream")) {
            const levels = ["INFO", "DEBUG", "WARN"];
            const messages = [
              "Request routed to provider",
              "Cache hit",
              "Token usage recorded",
              "Connection tested OK",
              "Model list refreshed",
            ];
            this.timer = setInterval(() => {
              const ev = new MessageEvent("message", {
                data: JSON.stringify({
                  timestamp: new Date().toISOString(),
                  level: levels[Math.floor(Math.random() * levels.length)],
                  message: messages[Math.floor(Math.random() * messages.length)],
                }),
              });
              this.dispatchEvent(ev);
            }, 2500);
          }
        }

        close() {
          this.readyState = 2;
          if (this.timer) {
            clearInterval(this.timer);
            this.timer = null;
          }
        }
      }
      (MockEventSource as any).CONNECTING = 0;
      (MockEventSource as any).OPEN = 1;
      (MockEventSource as any).CLOSED = 2;
      window.EventSource = MockEventSource as any;
    });
    await use(page);
  },
});

export { expect };
