import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerMcpHandlers(page: Page, store: MockStore) {
  page.route("/api/mcp/clients", async (route) => {
    if (route.request().method() === "GET") return json(route, Array.from(store.mcpClients.values()));
    return route.continue();
  });
  page.route(/\/api\/mcp\/clients\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    if (route.request().method() === "GET") {
      const c = store.mcpClients.get(id);
      return c ? json(route, c) : error(route, "Not found", 404);
    }
    return route.continue();
  });
  page.route("/api/mcp/instances", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.mcpInstances.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const inst = { ID: store.nextId(), CreatedAt: new Date().toISOString(), UpdatedAt: new Date().toISOString(), IsActive: true, ...body };
      store.mcpInstances.set(inst.ID, inst);
      return json(route, inst);
    }
    return route.continue();
  });
  page.route(/\/api\/mcp\/instances\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const inst = store.mcpInstances.get(id);
      return inst ? json(route, inst) : error(route, "Not found", 404);
    }
    if (method === "DELETE") {
      store.mcpInstances.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
  page.route(/\/api\/mcp\/instances\/[^/]+\/accounts$/, async (route) => {
    if (route.request().method() === "GET") return json(route, []);
    return route.continue();
  });
  page.route(/\/api\/mcp\/instances\/[^/]+\/auth\/start$/, async (route) => {
    if (route.request().method() === "POST") return json(route, { url: "https://mock-oauth.example.com/authorize" });
    return route.continue();
  });
  page.route("/api/mcp/tools", async (route) => {
    if (route.request().method() === "GET") return json(route, store.mcpTools);
    return route.continue();
  });
  page.route(/\/api\/mcp\/tools\/[^/]+\/execute$/, async (route) => {
    if (route.request().method() === "POST") {
      const name = route.request().url().split("/")[4];
      return json(route, { result: `Mock execution result for ${name}` });
    }
    return route.continue();
  });
  page.route("/api/mcp/tool-groups", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.mcpToolGroups.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const tg = { id: Date.now(), created_at: new Date().toISOString(), updated_at: new Date().toISOString(), ...body };
      store.mcpToolGroups.set(String(tg.id), tg);
      return json(route, tg);
    }
    return route.continue();
  });
  page.route(/\/api\/mcp\/tool-groups\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const tg = store.mcpToolGroups.get(id);
      return tg ? json(route, tg) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.mcpToolGroups.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body, updated_at: new Date().toISOString() };
      store.mcpToolGroups.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.mcpToolGroups.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
}
