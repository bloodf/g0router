import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerChatSessionsHandlers(page: Page, store: MockStore) {
  page.route("/api/chat-sessions", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, store.chatSessions);
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const session = { id: store.nextId(), created_at: new Date().toISOString(), updated_at: new Date().toISOString(), messages: [], ...body };
      store.chatSessions.push(session);
      return json(route, session);
    }
    return route.continue();
  });
  page.route(/\/api\/chat-sessions\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const s = store.chatSessions.find((cs) => cs.id === id);
      return s ? json(route, s) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const idx = store.chatSessions.findIndex((cs) => cs.id === id);
      if (idx === -1) return error(route, "Not found", 404);
      store.chatSessions[idx] = { ...store.chatSessions[idx], ...body, updated_at: new Date().toISOString() };
      return json(route, store.chatSessions[idx]);
    }
    if (method === "DELETE") {
      store.chatSessions = store.chatSessions.filter((cs) => cs.id !== id);
      return json(route, {});
    }
    return route.continue();
  });
}
