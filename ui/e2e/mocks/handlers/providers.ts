import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { getKnownProvider, getCatalogModels } from "../catalog";
import { json, error } from "./utils";

function providerList(store: MockStore) {
  const merged = new Map<string, ReturnType<typeof getKnownProvider>>();
  for (const p of store.providers.values()) {
    merged.set(p.id, { ...p, connection_count: Array.from(store.connections.values()).filter((c) => c.provider === p.id).length });
  }
  // Fill in any known providers not explicitly seeded so the matrix is complete.
  for (const p of [getKnownProvider("openai")!, getKnownProvider("anthropic")!, getKnownProvider("gemini")!, getKnownProvider("azure")!, getKnownProvider("bedrock")!, getKnownProvider("cerebras")!, getKnownProvider("cohere")!, getKnownProvider("deepseek")!, getKnownProvider("fireworks")!, getKnownProvider("groq")!, getKnownProvider("huggingface")!, getKnownProvider("minimax")!, getKnownProvider("mistral")!, getKnownProvider("nebius")!, getKnownProvider("nvidia")!, getKnownProvider("ollama")!, getKnownProvider("openrouter")!, getKnownProvider("perplexity")!, getKnownProvider("qwen")!, getKnownProvider("together")!, getKnownProvider("vertex")!, getKnownProvider("xai")!, getKnownProvider("alibaba")!, getKnownProvider("github-copilot")!, getKnownProvider("kimi")!, getKnownProvider("zhipu")!, getKnownProvider("cloudflare-ai-gateway")!, getKnownProvider("kagi")!, getKnownProvider("litellm")!, getKnownProvider("lm-studio")!, getKnownProvider("ollama-cloud")!, getKnownProvider("opencode")!, getKnownProvider("replicate")!, getKnownProvider("tavily")!, getKnownProvider("vllm")!].filter(Boolean)) {
    if (!merged.has(p.id)) {
      merged.set(p.id, { ...p, connection_count: Array.from(store.connections.values()).filter((c) => c.provider === p.id).length });
    }
  }
  return Array.from(merged.values()).sort((a, b) => a.display_name.localeCompare(b.display_name));
}

function getProvider(store: MockStore, id: string) {
  const seeded = store.providers.get(id);
  if (seeded) return { ...seeded, connection_count: Array.from(store.connections.values()).filter((c) => c.provider === id).length };
  const known = getKnownProvider(id);
  if (!known) return null;
  return { ...known, connection_count: Array.from(store.connections.values()).filter((c) => c.provider === id).length };
}

function getModels(store: MockStore, id: string) {
  const seeded = Array.from(store.models.values()).filter((m) => m.provider === id);
  if (seeded.length > 0) return seeded;
  return getCatalogModels(id);
}

export function registerProvidersHandlers(page: Page, store: MockStore) {
  page.route("/api/providers", async (route) => {
    if (route.request().method() === "GET") {
      return json(route, providerList(store));
    }
    return route.continue();
  });
  page.route(/\/api\/providers\/[^/]+$/, async (route) => {
    if (route.request().method() === "GET") {
      const id = route.request().url().split("/").pop()!;
      const p = getProvider(store, id);
      if (!p) return error(route, "Provider not found", 404);
      return json(route, p);
    }
    return route.continue();
  });
  page.route(/\/api\/providers\/[^/]+\/connections$/, async (route) => {
    if (route.request().method() === "GET") {
      const parts = route.request().url().split("/");
      const id = parts[parts.length - 2];
      return json(route, Array.from(store.connections.values()).filter((c) => c.provider === id));
    }
    return route.continue();
  });
  page.route(/\/api\/providers\/[^/]+\/models$/, async (route) => {
    if (route.request().method() === "GET") {
      const parts = route.request().url().split("/");
      const id = parts[parts.length - 2];
      return json(route, getModels(store, id));
    }
    return route.continue();
  });
  page.route(/\/api\/providers\/[^/]+\/suggested-models$/, async (route) => {
    if (route.request().method() === "GET") {
      const parts = route.request().url().split("/");
      const id = parts[parts.length - 2];
      const list = getModels(store, id).slice(0, 5);
      return json(route, list.map((m) => ({ id: m.id, name: m.name })));
    }
    return route.continue();
  });
  page.route("/api/providers/test-batch", async (route) => {
    if (route.request().method() === "POST") {
      const results = providerList(store).map((p) => ({ provider: p.id, ok: p.status === "active", latency_ms: Math.floor(Math.random() * 500) + 50 }));
      return json(route, { results });
    }
    return route.continue();
  });
}
