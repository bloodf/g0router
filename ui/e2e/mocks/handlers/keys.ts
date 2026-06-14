import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

// Mirrors the REAL Go /api/keys CRUD (internal/admin/apikeys.go, plan §1.4 / §8
// ESC-2). Response bodies use the Go envelope shapes:
//   GET  /api/keys        -> {data:{keys:[apiKeyDTO]}}
//   POST /api/keys {name} -> {data:{key,name,id,machine_id}}
//   GET  /api/keys/{id}   -> {data:{key:apiKeyDTO}}
//   PUT  /api/keys/{id} {is_active} -> {data:{key:apiKeyDTO}}
//   DELETE /api/keys/{id} -> {data:{message}}
// There is no /{id}/regenerate on the Go side (reissue = delete+create), so it is
// not mocked.
function genKey(): string {
  return `sk-${Math.random().toString(36).slice(2, 12)}`;
}

export function registerKeysHandlers(page: Page, store: MockStore) {
  page.route("/api/keys", async (route) => {
    const method = route.request().method();
    if (method === "GET") {
      return json(route, { keys: Array.from(store.keys.values()) });
    }
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const id = store.nextId();
      const key = {
        id,
        key: genKey(),
        name: body.name,
        machine_id: `machine-${id}`,
        is_active: true,
        created_at: new Date().toISOString(),
      };
      store.keys.set(id, key as never);
      return json(route, { key: key.key, name: key.name, id: key.id, machine_id: key.machine_id });
    }
    return route.continue();
  });
  page.route(/\/api\/keys\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const k = store.keys.get(id);
      return k ? json(route, { key: k }) : error(route, "key not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.keys.get(id);
      if (!existing) return error(route, "key not found", 404);
      const updated = { ...existing, ...(body.is_active !== undefined ? { is_active: body.is_active } : {}) };
      store.keys.set(id, updated as never);
      return json(route, { key: updated });
    }
    if (method === "DELETE") {
      store.keys.delete(id);
      return json(route, { message: "Key deleted successfully" });
    }
    return route.continue();
  });
}
