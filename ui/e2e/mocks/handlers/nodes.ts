import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { error, json } from "./utils";

// Mirrors the NEW Go provider-nodes admin API (internal/admin/nodes.go,
// w7-platnodes) plus the mock-only model-test/availability surfaces (plan §1.4 /
// §8 ESC-3). Provider nodes are dynamic prefix-routing custom providers; the list
// filters the providers store to the node types and carries prefix/api_type.
//   GET    /api/provider-nodes          -> {data:{nodes:[{id,name,base_url,type,enabled,prefix,api_type}]}}
//   POST   /api/provider-nodes          -> {data:{node:{...}}}
//   GET    /api/provider-nodes/{id}     -> {data:{node:{...}}} | 404
//   PUT    /api/provider-nodes/{id}     -> {data:{node:{...}}} | 404
//   DELETE /api/provider-nodes/{id}     -> {data:{message}} | 404
//   POST   /api/provider-nodes/validate -> {data:{valid,error?}}  (api_key NEVER stored)
//   POST /api/models/test             -> {data:{ok,latency_ms}}
//   GET  /api/models/availability     -> {data:{available:[...]}}
const NODE_TYPES = ["openai-compatible", "anthropic-compatible", "custom-embedding"];

interface ProviderLike {
  id: string;
  name: string;
  type?: string;
  base_url?: string;
  enabled?: boolean;
  prefix?: string;
  api_type?: string;
}

function toNode(p: ProviderLike) {
  return {
    id: p.id,
    name: p.name,
    base_url: p.base_url ?? "",
    type: p.type ?? "openai-compatible",
    enabled: p.enabled ?? true,
    prefix: p.prefix ?? "",
    api_type: p.api_type ?? "",
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
        .filter((p) => NODE_TYPES.includes((p as unknown as ProviderLike).type ?? ""))
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
        type: body.type ?? "openai-compatible",
        enabled: true,
        prefix: body.prefix ?? "",
        api_type: body.apiType ?? body.api_type ?? "",
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
  // {id} CRUD mirrors the new Go surface (get-or-404 / merge-update / delete).
  page.route(/\/api\/provider-nodes\/[^/]+$/, async (route) => {
    const method = route.request().method();
    const url = new URL(route.request().url());
    const id = url.pathname.split("/").pop() ?? "";
    if (id === "validate") return route.continue();
    const existing = store.providers.get(id) as unknown as ProviderLike | undefined;
    if (method === "GET") {
      if (!existing) return error(route, "provider node not found", 404);
      return json(route, { node: toNode(existing) });
    }
    if (method === "PUT") {
      if (!existing) return error(route, "provider node not found", 404);
      const body = await route.request().postDataJSON();
      const node = toNode({
        id,
        name: body.name ?? existing.name,
        base_url: body.baseUrl ?? body.base_url ?? existing.base_url ?? "",
        type: body.type ?? existing.type ?? "openai-compatible",
        enabled: existing.enabled ?? true,
        prefix: body.prefix ?? existing.prefix ?? "",
        api_type: body.apiType ?? body.api_type ?? existing.api_type ?? "",
      });
      store.providers.set(id, { ...node, provider: node.id } as never);
      return json(route, { node });
    }
    if (method === "DELETE") {
      if (!existing) return error(route, "provider node not found", 404);
      store.providers.delete(id);
      return json(route, { message: "Provider node deleted successfully" });
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
