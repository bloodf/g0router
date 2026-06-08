import type { Page } from "@playwright/test";
import type { MockStore } from "./store";

export function setupMockApi(page: Page, store: MockStore) {
  page.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method();
    const pathname = url.pathname;

    // ── Auth ──
    if (pathname === "/api/auth/status" && method === "GET") {
      return json(route, store.auth);
    }
    if (pathname === "/api/auth/login" && method === "POST") {
      const body = await request.postDataJSON();
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
    if (pathname === "/api/auth/logout" && method === "POST") {
      store.auth.authenticated = false;
      return json(route, {});
    }
    if (pathname === "/api/auth/setup" && method === "POST") {
      const body = await request.postDataJSON();
      const user: typeof store.users[0] = {
        id: store.nextId(),
        username: body.username,
        display_name: body.display_name || body.username,
        role: "admin",
        password: body.password,
      };
      store.users.push(user);
      store.auth.has_users = true;
      store.auth.authenticated = true;
      store.auth.username = user.username;
      store.auth.display_name = user.display_name;
      store.auth.role = "admin";
      return json(route, {});
    }
    if (pathname === "/api/auth/password" && method === "POST") {
      return json(route, {});
    }
    if (pathname === "/api/auth/users" && method === "GET") {
      return json(route, store.users.map((u) => ({ ...u, password: undefined })));
    }

    // ── Settings ──
    if (pathname === "/api/settings" && method === "GET") {
      return json(route, store.settings);
    }
    if (pathname === "/api/settings" && method === "PUT") {
      const body = await request.postDataJSON();
      store.settings = { ...store.settings, ...body };
      return json(route, store.settings);
    }

    // ── Providers ──
    if (pathname === "/api/providers" && method === "GET") {
      const list = Array.from(store.providers.values());
      // Update connection_count from actual connections
      for (const p of list) {
        p.connection_count = Array.from(store.connections.values()).filter((c) => c.provider === p.id).length;
      }
      return json(route, list);
    }
    if (pathname.match(/^\/api\/providers\/[^/]+$/) && method === "GET") {
      const id = pathname.split("/").pop()!;
      const p = store.providers.get(id);
      if (!p) return error(route, "Provider not found", 404);
      p.connection_count = Array.from(store.connections.values()).filter((c) => c.provider === id).length;
      return json(route, p);
    }
    if (pathname.match(/^\/api\/providers\/[^/]+\/connections$/) && method === "GET") {
      const id = pathname.split("/")[3];
      const list = Array.from(store.connections.values()).filter((c) => c.provider === id);
      return json(route, list);
    }
    if (pathname.match(/^\/api\/providers\/[^/]+\/models$/) && method === "GET") {
      const id = pathname.split("/")[3];
      const list = Array.from(store.models.values()).filter((m) => m.provider === id);
      return json(route, list);
    }
    if (pathname === "/api/providers/test-batch" && method === "POST") {
      const results = Array.from(store.providers.values()).map((p) => ({
        provider: p.id,
        ok: p.status === "active",
        latency_ms: Math.floor(Math.random() * 500) + 50,
      }));
      return json(route, { results });
    }

    // ── Connections ──
    if (pathname === "/api/connections" && method === "GET") {
      return json(route, Array.from(store.connections.values()));
    }
    if (pathname.match(/^\/api\/connections\/[^/]+$/) && method === "GET") {
      const id = pathname.split("/").pop()!;
      const c = store.connections.get(id);
      return c ? json(route, c) : error(route, "Not found", 404);
    }
    if (pathname.match(/^\/api\/connections\/[^/]+$/) && method === "PUT") {
      const id = pathname.split("/").pop()!;
      const body = await request.postDataJSON();
      const existing = store.connections.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.connections.set(id, updated);
      return json(route, updated);
    }
    if (pathname.match(/^\/api\/connections\/[^/]+$/) && method === "DELETE") {
      const id = pathname.split("/").pop()!;
      store.connections.delete(id);
      return json(route, {});
    }
    if (pathname.match(/^\/api\/connections\/[^/]+\/test$/) && method === "POST") {
      return json(route, { ok: true, latency_ms: Math.floor(Math.random() * 300) + 50 });
    }
    if (pathname === "/api/connections/bulk-enable" && method === "POST") {
      for (const c of store.connections.values()) c.is_active = true;
      return json(route, {});
    }
    if (pathname === "/api/connections/bulk-disable" && method === "POST") {
      for (const c of store.connections.values()) c.is_active = false;
      return json(route, {});
    }

    // ── Keys ──
    if (pathname === "/api/keys" && method === "GET") {
      return json(route, Array.from(store.keys.values()));
    }
    if (pathname === "/api/keys" && method === "POST") {
      const body = await request.postDataJSON();
      const key = {
        id: store.nextId(),
        prefix: `sk-${Math.random().toString(36).slice(2, 8)}`,
        full_key: `sk-${Math.random().toString(36).slice(2, 12)}`,
        scopes: [],
        is_active: true,
        created_at: new Date().toISOString(),
        ...body,
      };
      store.keys.set(key.id, key);
      return json(route, key);
    }
    if (pathname.match(/^\/api\/keys\/[^/]+$/) && method === "PUT") {
      const id = pathname.split("/").pop()!;
      const body = await request.postDataJSON();
      const existing = store.keys.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.keys.set(id, updated);
      return json(route, updated);
    }
    if (pathname.match(/^\/api\/keys\/[^/]+$/) && method === "DELETE") {
      const id = pathname.split("/").pop()!;
      store.keys.delete(id);
      return json(route, {});
    }
    if (pathname.match(/^\/api\/keys\/[^/]+\/regenerate$/) && method === "POST") {
      const id = pathname.split("/")[3];
      const existing = store.keys.get(id);
      if (!existing) return error(route, "Not found", 404);
      existing.full_key = `sk-${Math.random().toString(36).slice(2, 12)}`;
      return json(route, existing);
    }

    // ── Virtual Keys ──
    if (pathname === "/api/virtual-keys" && method === "GET") {
      return json(route, Array.from(store.virtualKeys.values()));
    }
    if (pathname === "/api/virtual-keys" && method === "POST") {
      const body = await request.postDataJSON();
      const vk = { id: store.nextId(), prefix: `vk-${Math.random().toString(36).slice(2, 6)}`, budget_used_usd: 0, is_active: true, ...body };
      store.virtualKeys.set(vk.id, vk);
      return json(route, vk);
    }
    if (pathname.match(/^\/api\/virtual-keys\/[^/]+$/) && method === "PUT") {
      const id = pathname.split("/").pop()!;
      const body = await request.postDataJSON();
      const existing = store.virtualKeys.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.virtualKeys.set(id, updated);
      return json(route, updated);
    }
    if (pathname.match(/^\/api\/virtual-keys\/[^/]+$/) && method === "DELETE") {
      const id = pathname.split("/").pop()!;
      store.virtualKeys.delete(id);
      return json(route, {});
    }

    // ── Models ──
    if (pathname === "/api/models" && method === "GET") {
      return json(route, Array.from(store.models.values()));
    }

    // ── Combos ──
    if (pathname === "/api/combos" && method === "GET") {
      return json(route, Array.from(store.combos.values()));
    }
    if (pathname === "/api/combos" && method === "POST") {
      const body = await request.postDataJSON();
      const combo = { id: store.nextId(), is_active: true, ...body };
      store.combos.set(combo.id, combo);
      return json(route, combo);
    }
    if (pathname.match(/^\/api\/combos\/[^/]+$/) && method === "PUT") {
      const id = pathname.split("/").pop()!;
      const body = await request.postDataJSON();
      const existing = store.combos.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.combos.set(id, updated);
      return json(route, updated);
    }
    if (pathname.match(/^\/api\/combos\/[^/]+$/) && method === "DELETE") {
      const id = pathname.split("/").pop()!;
      store.combos.delete(id);
      return json(route, {});
    }

    // ── Aliases ──
    if (pathname === "/api/aliases" && method === "GET") {
      return json(route, Array.from(store.aliases.values()));
    }
    if (pathname === "/api/aliases" && method === "POST") {
      const body = await request.postDataJSON();
      const alias = { id: store.nextId(), ...body };
      store.aliases.set(alias.id, alias);
      return json(route, alias);
    }
    if (pathname.match(/^\/api\/aliases\/[^/]+$/) && method === "PUT") {
      const id = pathname.split("/").pop()!;
      const body = await request.postDataJSON();
      const existing = store.aliases.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.aliases.set(id, updated);
      return json(route, updated);
    }
    if (pathname.match(/^\/api\/aliases\/[^/]+$/) && method === "DELETE") {
      const id = pathname.split("/").pop()!;
      store.aliases.delete(id);
      return json(route, {});
    }

    // ── Pricing ──
    if (pathname === "/api/pricing" && method === "GET") {
      return json(route, Array.from(store.pricing.values()));
    }
    if (pathname === "/api/pricing" && method === "POST") {
      const body = await request.postDataJSON();
      const price = { id: store.nextId(), ...body };
      store.pricing.set(price.id, price);
      return json(route, price);
    }
    if (pathname.match(/^\/api\/pricing\/[^/]+$/) && method === "PUT") {
      const id = pathname.split("/").pop()!;
      const body = await request.postDataJSON();
      const existing = store.pricing.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.pricing.set(id, updated);
      return json(route, updated);
    }
    if (pathname.match(/^\/api\/pricing\/[^/]+$/) && method === "DELETE") {
      const id = pathname.split("/").pop()!;
      store.pricing.delete(id);
      return json(route, {});
    }

    // ── Routing Rules ──
    if (pathname === "/api/routing-rules" && method === "GET") {
      return json(route, Array.from(store.routingRules.values()));
    }
    if (pathname === "/api/routing-rules" && method === "POST") {
      const body = await request.postDataJSON();
      const rule = { id: store.nextId(), is_active: true, ...body };
      store.routingRules.set(rule.id, rule);
      return json(route, rule);
    }
    if (pathname.match(/^\/api\/routing-rules\/[^/]+$/) && method === "PUT") {
      const id = pathname.split("/").pop()!;
      const body = await request.postDataJSON();
      const existing = store.routingRules.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.routingRules.set(id, updated);
      return json(route, updated);
    }
    if (pathname.match(/^\/api\/routing-rules\/[^/]+$/) && method === "DELETE") {
      const id = pathname.split("/").pop()!;
      store.routingRules.delete(id);
      return json(route, {});
    }

    // ── Teams ──
    if (pathname === "/api/teams" && method === "GET") {
      return json(route, Array.from(store.teams.values()));
    }
    if (pathname === "/api/teams" && method === "POST") {
      const body = await request.postDataJSON();
      const team = { id: store.nextId(), budget_used_usd: 0, keys_count: 0, members: 0, ...body };
      store.teams.set(team.id, team);
      return json(route, team);
    }
    if (pathname.match(/^\/api\/teams\/[^/]+$/) && method === "PUT") {
      const id = pathname.split("/").pop()!;
      const body = await request.postDataJSON();
      const existing = store.teams.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.teams.set(id, updated);
      return json(route, updated);
    }
    if (pathname.match(/^\/api\/teams\/[^/]+$/) && method === "DELETE") {
      const id = pathname.split("/").pop()!;
      store.teams.delete(id);
      return json(route, {});
    }

    // ── Tunnels ──
    if (pathname === "/api/tunnels" && method === "GET") {
      return json(route, Array.from(store.tunnels.values()));
    }
    if (pathname.match(/^\/api\/tunnels\/[^/]+$/) && method === "POST") {
      const type = pathname.split("/").pop() as "cloudflare" | "tailscale";
      const t = store.tunnels.get(type);
      if (t) { t.is_enabled = true; t.status = "active"; }
      return json(route, t || {});
    }
    if (pathname.match(/^\/api\/tunnels\/[^/]+$/) && method === "DELETE") {
      const type = pathname.split("/").pop() as "cloudflare" | "tailscale";
      const t = store.tunnels.get(type);
      if (t) { t.is_enabled = false; t.status = "inactive"; }
      return json(route, {});
    }

    // ── Usage ──
    if (pathname === "/api/usage/summary" && method === "GET") {
      return json(route, {
        total_requests: store.usageLogs.length,
        total_tokens: store.usageLogs.reduce((s, l) => s + l.total_tokens, 0),
        total_cost: store.usageLogs.reduce((s, l) => s + l.cost_usd, 0),
        avg_latency_ms: Math.floor(store.usageLogs.reduce((s, l) => s + l.latency_ms, 0) / Math.max(store.usageLogs.length, 1)),
      });
    }
    if (pathname === "/api/usage/chart" && method === "GET") {
      const buckets = Array.from({ length: 7 }, (_, i) => new Date(Date.now() - (6 - i) * 86400000).toISOString().slice(0, 10));
      return json(route, {
        buckets,
        tokens_input: buckets.map(() => Math.floor(Math.random() * 50000)),
        tokens_output: buckets.map(() => Math.floor(Math.random() * 20000)),
        costs: buckets.map(() => Math.random() * 10),
        requests: buckets.map(() => Math.floor(Math.random() * 500)),
      });
    }
    if (pathname === "/api/usage" && method === "GET") {
      const limit = parseInt(url.searchParams.get("limit") || "100", 10);
      return json(route, { items: store.usageLogs.slice(0, limit), total: store.usageLogs.length });
    }

    // ── Quota ──
    if (pathname === "/api/quota" && method === "GET") {
      return json(route, store.quotas);
    }

    // ── Audit ──
    if (pathname === "/api/audit" && method === "GET") {
      const limit = parseInt(url.searchParams.get("limit") || "100", 10);
      return json(route, { items: store.auditLogs.slice(0, limit), total: store.auditLogs.length });
    }
    if (pathname === "/api/audit" && method === "POST") {
      const body = await request.postDataJSON();
      const log = { id: store.nextId(), timestamp: new Date().toISOString(), actor: store.auth.username || "admin", ...body };
      store.auditLogs.unshift(log);
      return json(route, log);
    }

    // ── Chat Sessions ──
    if (pathname === "/api/chat-sessions" && method === "GET") {
      return json(route, store.chatSessions);
    }

    // ── Version ──
    if (pathname === "/api/version" && method === "GET") {
      return json(route, { version: "0.9.0-mock", build_date: "2024-01-01" });
    }

    // ── Health ──
    if (pathname === "/healthz" && method === "GET") {
      return route.fulfill({ status: 200, body: JSON.stringify({ status: "ok" }) });
    }

    // ── Traffic SSE ──
    if (pathname === "/api/traffic/stream" && method === "GET") {
      const body = `data: {"timestamp":"${new Date().toISOString()}","key_id":"key-1","provider":"openai","model":"gpt-4o","status_class":"2xx","status_code":200,"latency_ms":120}\n\n`;
      return route.fulfill({
        status: 200,
        headers: { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" },
        body,
      });
    }

    // ── Console SSE ──
    if (pathname === "/api/console-logs/stream" && method === "GET") {
      const body = `event: log\ndata: {"timestamp":"${new Date().toISOString()}","level":"INFO","message":"Mock console stream active"}\n\n`;
      return route.fulfill({
        status: 200,
        headers: { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" },
        body,
      });
    }
    if (pathname === "/api/console-logs" && method === "DELETE") {
      store.consoleLogs = [];
      return json(route, {});
    }

    // ── Chat completions (inference) ──
    if (pathname === "/v1/chat/completions" && method === "POST") {
      const body = await request.postDataJSON();
      const messages = body.messages || [];
      const lastMsg = messages[messages.length - 1]?.content || "";
      const response = `Hello! I'm a mock assistant. You said: "${lastMsg.slice(0, 50)}..."`;

      const sseBody = [
        `data: {"id":"mock-chat","object":"chat.completion.chunk","created":${Math.floor(Date.now()/1000)},"model":"${body.model}","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
        ``,
        `data: {"id":"mock-chat","object":"chat.completion.chunk","created":${Math.floor(Date.now()/1000)},"model":"${body.model}","choices":[{"index":0,"delta":{"content":"${response}"},"finish_reason":null}]}`,
        ``,
        `data: {"id":"mock-chat","object":"chat.completion.chunk","created":${Math.floor(Date.now()/1000)},"model":"${body.model}","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
        ``,
        `data: [DONE]`,
        ``,
      ].join("\n");

      return route.fulfill({
        status: 200,
        headers: { "Content-Type": "text/event-stream" },
        body: sseBody,
      });
    }

    // Pass through everything else (static assets, etc.)
    await route.continue();
  });
}

function json(route: any, data: unknown) {
  return route.fulfill({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ data }),
  });
}

function error(route: any, message: string, status = 400) {
  return route.fulfill({
    status,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ error: message }),
  });
}
