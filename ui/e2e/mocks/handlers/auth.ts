import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

// Auth-mode + rate-limit are control knobs the spec drives via request headers
// (x-mock-auth-mode / x-mock-force-lockout) set with page.setExtraHTTPHeaders,
// since page.route intercepts page-context requests but not APIRequestContext.
// They mirror the real Go contract:
//   - GET /api/auth/status returns ONLY { auth_mode } (internal/admin/auth.go:177-179)
//   - 429 lockout returns { data:null, error:{ message, retry_after, reset_hint } }
//     plus a Retry-After header (internal/admin/auth.go:126-140)
// No frozen MockStore/types.ts field is added.
const resetHint =
  "Forgot password? Reset to default via g0router CLI: g0router reset-password";

export function registerAuthHandlers(page: Page, store: MockStore) {
  page.route("/api/auth/status", async (route) => {
    // Real Go Status returns ONLY { auth_mode } (auth.go:177-179).
    if (route.request().method() === "GET") {
      const mode = route.request().headers()["x-mock-auth-mode"] ?? "password";
      return json(route, { auth_mode: mode });
    }
    return route.continue();
  });
  page.route("/api/auth/login", async (route) => {
    if (route.request().method() === "POST") {
      // 429 lockout branch (auth.go:126-140): { error:{ message, retry_after, reset_hint } }
      const forceLockout = Number(
        route.request().headers()["x-mock-force-lockout"] ?? "0"
      );
      if (forceLockout > 0) {
        const retryAfter = forceLockout;
        return route.fulfill({
          status: 429,
          headers: {
            "Content-Type": "application/json",
            "Retry-After": String(retryAfter),
          },
          body: JSON.stringify({
            data: null,
            error: {
              message: `Too many failed attempts. Try again in ${retryAfter}s. ${resetHint}`,
              retry_after: retryAfter,
              reset_hint: resetHint,
            },
          }),
        });
      }
      const body = await route.request().postDataJSON();
      const user = store.users.find((u) => u.username === body.username);
      if (user && user.password === body.password) {
        store.auth.authenticated = true;
        store.auth.username = user.username;
        store.auth.display_name = user.display_name;
        store.auth.role = user.role;
        // Real Go login returns { data:{ token, user:{ id, username } } } (auth.go:120-123).
        return json(route, {
          token: "mock-jwt-token",
          user: { id: user.id, username: user.username },
        });
      }
      // Real Go 401 envelope is { error:{ message } } (auth.go:94); the frozen
      // apiFetch reads error.message, so the mock must use the object form.
      return route.fulfill({
        status: 401,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          data: null,
          error: { message: "invalid username or password" },
        }),
      });
    }
    return route.continue();
  });
  page.route("/api/auth/logout", async (route) => {
    if (route.request().method() === "POST") {
      store.auth.authenticated = false;
      // Real Go logout returns { data:{ logged_out:true } } (auth.go:153).
      return json(route, { logged_out: true });
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
      // Real Go AuthSetup returns { data:{ token, user:userDTO } } and strips the
      // password (internal/admin/usermgmt.go AuthSetup).
      return json(route, {
        token: "mock-jwt-token",
        user: { id: user.id, username: user.username, display_name: user.display_name, role: user.role },
      });
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
  // Real Go userDTO is { id, username, display_name, role } with the hash always
  // stripped (internal/admin/usermgmt.go userMgmtDTO).
  const toUserDTO = (u: { id: string; username: string; display_name: string; role: string }) => ({
    id: u.id,
    username: u.username,
    display_name: u.display_name,
    role: u.role,
  });
  page.route("/api/auth/users", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, store.users.map(toUserDTO));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      if (store.users.some((u) => u.username === body.username)) return error(route, "username already exists", 409);
      const user = { id: store.nextId(), username: body.username, display_name: body.display_name || body.username, role: body.role || "user", password: body.password };
      store.users.push(user);
      return json(route, toUserDTO(user));
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
