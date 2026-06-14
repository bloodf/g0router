import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

// Mirrors the NEW Go provider-nodes admin API (internal/admin/nodes.go, plan
// §1.6b) plus the mock-only model-test/availability surfaces (plan §1.4 / §8
// ESC-3). Provider nodes are OpenAI-compatible custom providers; the list filters
// the providers store to type=="openai-compatible".
//   GET  /api/provider-nodes          -> {data:{nodes:[{id,name,base_url,type,enabled}]}}
//   POST /api/provider-nodes          -> {data:{node:{...}}}
//   POST /api/provider-nodes/validate -> {data:{valid,error?}}  (api_key NEVER stored)
//   POST /api/models/test             -> {data:{ok,latency_ms}}
//   GET  /api/models/availability     -> {data:{available:[...]}}
interface ProviderLike {
  id: string;
  name: string;
  type?: string;
  base_url?: string;
  enabled?: boolean;
}

function toNode(p: ProviderLike) {
  return {
    id: p.id,
    name: p.name,
    base_url: p.base_url ?? "",
    type: p.type ?? "openai-compatible",
    enabled: p.enabled ?? true,
  };
}

function isWellFormedURL(url: unknown): boolean {
  if (typeof url !== "string" || url === "") return false;
  try {
    const parsed = new URL(url);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

export function registerNodesHandlers(page: Page, store: MockStore) {
  page.route("/api/provider-nodes", async (route) => {
    const method = route.request().method();
    if (method === "GET") {
      const nodes = Array.from(store.providers.values())
        .filter((p) => (p as unknown as ProviderLike).type === "openai-compatible")
        .map((p) => toNode(p as unknown as ProviderLike));
      return json(route, { nodes });
    }
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const id = store.nextId();
      const node = toNode({
        id,
        name: body.name,
        base_url: body.baseUrl ?? body.base_url ?? "",
        type: "openai-compatible",
        enabled: true,
      });
      store.providers.set(id, { ...node, provider: node.id } as never);
      return json(route, { node });
    }
    return route.continue();
  });
  page.route("/api/provider-nodes/validate", async (route) => {
    if (route.request().method() === "POST") {
      const body = await route.request().postDataJSON();
      const url = body.baseUrl ?? body.base_url;
      if (isWellFormedURL(url)) return json(route, { valid: true });
      return json(route, { valid: false, error: "invalid url" });
    }
    return route.continue();
  });
  page.route("/api/models/test", async (route) => {
    if (route.request().method() === "POST") {
      return json(route, { ok: true, latency_ms: 42 });
    }
    return route.continue();
  });
  page.route("/api/models/availability", async (route) => {
    if (route.request().method() === "GET") {
      const available = Array.from(store.models.values()).map((m) => ({
        id: (m as { id: string }).id,
        available: true,
      }));
      return json(route, { available });
    }
    return route.continue();
  });
}
