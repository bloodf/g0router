import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerAuthHandlers(page: Page, store: MockStore) {
  page.route("/api/auth/status", async (route) => {
    if (route.request().method() === "GET") return json(route, store.auth);
    return route.continue();
  });
  page.route("/api/auth/login", async (route) => {
    if (route.request().method() === "POST") {
      const body = await route.request().postDataJSON();
      const user = store.users.find((u) => u.username === body.username);
      if (user && user.password === body.password) {
        store.auth.authenticated = true;
        store.auth.username = user.username;
        store.auth.display_name = user.display_name;
        store.auth.role = user.role;
        return json(route, { token: "mock-jwt-token" });
      }
      return error(route, "Invalid credentials", 401);
    }
    return route.continue();
  });
  page.route("/api/auth/logout", async (route) => {
    if (route.request().method() === "POST") {
      store.auth.authenticated = false;
      return json(route, {});
    }
    return route.continue();
  });
  page.route("/api/auth/setup", async (route) => {
    if (route.request().method() === "POST") {
      const body = await route.request().postDataJSON();
      const user = { id: store.nextId(), username: body.username, display_name: body.display_name || body.username, role: "admin" as const, password: body.password };
      store.users.push(user);
      store.auth.has_users = true;
      store.auth.authenticated = true;
      store.auth.username = user.username;
      store.auth.display_name = user.display_name;
      store.auth.role = "admin";
      return json(route, {});
    }
    return route.continue();
  });
  page.route("/api/auth/password", async (route) => {
    if (route.request().method() === "PUT") {
      const body = await route.request().postDataJSON();
      const user = store.users.find((u) => u.username === store.auth.username);
      if (!user) return error(route, "User not found", 404);
      if (user.password !== body.current_password) return error(route, "Current password is incorrect", 400);
      user.password = body.new_password;
      return json(route, {});
    }
    return route.continue();
  });
  page.route("/api/auth/users", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, store.users.map((u) => ({ ...u, password: undefined })));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      if (store.users.some((u) => u.username === body.username)) return error(route, "Username already exists", 409);
      const user = { id: store.nextId(), username: body.username, display_name: body.display_name || body.username, role: body.role || "user", password: body.password };
      store.users.push(user);
      return json(route, { ...user, password: undefined });
    }
    return route.continue();
  });
  page.route(/\/api\/auth\/users\/[^/]+$/, async (route) => {
    if (route.request().method() === "DELETE") {
      const id = route.request().url().split("/").pop()!;
      const idx = store.users.findIndex((u) => u.id === id);
      if (idx === -1) return error(route, "User not found", 404);
      store.users.splice(idx, 1);
      if (store.users.length === 0) store.auth.has_users = false;
      return json(route, {});
    }
    return route.continue();
  });
}
