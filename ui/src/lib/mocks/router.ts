import { getStore, setStore } from "./store";
import type {
  AlertChannel,
  Alias,
  ApiKey,
  ChatSession,
  Combo,
  Connection,
  McpInstance,
  McpToolGroup,
  ModelLimit,
  PricingOverride,
  PromptTemplate,
  ProxyPool,
  RoutingRule,
  Team,
  VirtualKey,
} from "./types";

const id = () => Math.random().toString(36).slice(2, 10);
const now = () => new Date().toISOString();
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

function maybeError(): { error: string; status: number } | null {
  const s = getStore().settings;
  if (s.inject_errors && Math.random() < 0.1) {
    const choices = [
      { status: 500, error: "Internal mock error (chaos injection)" },
      { status: 429, error: "Rate limit hit", retry_after_seconds: 15 } as any,
      { status: 502, error: "Upstream provider unreachable" },
    ];
    return choices[Math.floor(Math.random() * choices.length)];
  }
  return null;
}

export class ApiError extends Error {
  status: number;
  payload: unknown;
  constructor(status: number, payload: unknown) {
    super(typeof payload === "string" ? payload : JSON.stringify(payload));
    this.status = status;
    this.payload = payload;
  }
}

type Handler = (
  params: Record<string, string>,
  body: any,
  query: URLSearchParams,
) => any;

interface Route {
  method: string;
  pattern: RegExp;
  keys: string[];
  handler: Handler;
}

const routes: Route[] = [];

function register(method: string, path: string, handler: Handler) {
  const keys: string[] = [];
  const pattern = new RegExp(
    "^" +
      path.replace(/:[a-zA-Z_]+/g, (m) => {
        keys.push(m.slice(1));
        return "([^/]+)";
      }) +
      "$",
  );
  routes.push({ method, pattern, keys, handler });
}

// ─── AUTH ────────────────────────────────────────────────────────────────
register("GET", "/api/auth/status", () => {
  const s = getStore();
  const u = s.users.find((x) => x.id === s.session_user_id);
  return {
    require_login: s.settings.require_login,
    has_users: s.users.length > 0,
    authenticated: !!u,
    username: u?.username,
    display_name: u?.display_name,
    role: u?.role,
  };
});

register("POST", "/api/auth/setup", (_p, body) => {
  const s = getStore();
  if (s.users.length > 0) throw new ApiError(409, "Users already exist");
  if (!body.username || !body.password)
    throw new ApiError(400, "username and password required");
  const user = {
    id: id(),
    username: body.username,
    display_name: body.display_name || body.username,
    role: "admin" as const,
    password: body.password,
  };
  setStore((st) => ({ ...st, users: [user], session_user_id: user.id }));
  return { ok: true };
});

register("POST", "/api/auth/login", (_p, body) => {
  const s = getStore();
  const u = s.users.find((x) => x.username === body.username);
  if (!u || u.password !== body.password)
    throw new ApiError(401, "Invalid credentials");
  setStore((st) => ({ ...st, session_user_id: u.id }));
  return { ok: true };
});

register("POST", "/api/auth/logout", () => {
  setStore((st) => ({ ...st, session_user_id: null as any }));
  return { ok: true };
});

register("PUT", "/api/auth/password", (_p, body) => {
  const s = getStore();
  const u = s.users.find((x) => x.id === s.session_user_id);
  if (!u) throw new ApiError(401, "Not logged in");
  if (u.password !== body.current_password)
    throw new ApiError(400, "Current password incorrect");
  u.password = body.new_password;
  return { ok: true };
});

register("GET", "/api/auth/users", () => getStore().users.map((u) => ({ ...u, password: undefined })));

// ─── PROVIDERS ───────────────────────────────────────────────────────────
register("GET", "/api/providers", () => getStore().providers);
register("GET", "/api/providers/:id", ({ id: pid }) => {
  const p = getStore().providers.find((x) => x.id === pid);
  if (!p) throw new ApiError(404, "Not found");
  return p;
});
register("GET", "/api/providers/:id/models", ({ id: pid }) =>
  getStore().models.filter((m) => m.provider === pid),
);
register("GET", "/api/providers/:id/connections", ({ id: pid }) =>
  getStore().connections.filter((c) => c.provider === pid),
);
register("POST", "/api/providers/:id/models/:model/test", () => ({
  ok: true,
  latency_ms: Math.floor(Math.random() * 800) + 200,
}));
register("POST", "/api/providers/test-batch", () => ({
  results: getStore().connections.map((c) => ({
    connection_id: c.id,
    ok: Math.random() > 0.15,
    latency_ms: Math.floor(Math.random() * 800) + 200,
  })),
}));
register("GET", "/api/providers/:id/suggested-models", ({ id: pid }) => {
  const have = new Set(getStore().models.filter((m) => m.provider === pid).map((m) => m.name));
  return [`${pid}-extra-1`, `${pid}-extra-2`].filter((n) => !have.has(n)).map((name) => ({ name }));
});

// ─── CONNECTIONS ─────────────────────────────────────────────────────────
register("GET", "/api/connections", () => getStore().connections);
register("POST", "/api/connections", (_p, body) => {
  const conn: Connection = {
    id: id(),
    provider: body.provider,
    name: body.name,
    auth_type: body.auth_type ?? "api_key",
    is_active: true,
    models: body.models ?? [],
    priority: getStore().connections.length,
    needs_reauth: false,
  };
  setStore((s) => ({ ...s, connections: [...s.connections, conn] }));
  return conn;
});
register("PUT", "/api/connections/:id", ({ id: cid }, body) => {
  setStore((s) => ({
    ...s,
    connections: s.connections.map((c) => (c.id === cid ? { ...c, ...body } : c)),
  }));
  return getStore().connections.find((c) => c.id === cid)!;
});
register("DELETE", "/api/connections/:id", ({ id: cid }) => {
  setStore((s) => ({ ...s, connections: s.connections.filter((c) => c.id !== cid) }));
  return { ok: true };
});
register("POST", "/api/connections/:id/test", () => ({
  ok: Math.random() > 0.15,
  latency_ms: Math.floor(Math.random() * 800) + 200,
}));
register("PUT", "/api/connections/:id/proxy", ({ id: cid }, body) => {
  setStore((s) => ({
    ...s,
    connections: s.connections.map((c) =>
      c.id === cid ? { ...c, proxy_id: body.proxy_id } : c,
    ),
  }));
  return { ok: true };
});
register("POST", "/api/connections/bulk-disable", () => {
  setStore((s) => ({
    ...s,
    connections: s.connections.map((c) => ({ ...c, is_active: false })),
  }));
  return { ok: true };
});
register("POST", "/api/connections/bulk-enable", () => {
  setStore((s) => ({
    ...s,
    connections: s.connections.map((c) => ({ ...c, is_active: true })),
  }));
  return { ok: true };
});

// ─── API KEYS ────────────────────────────────────────────────────────────
function recordAudit(action: string, target: string, details?: string) {
  const s = getStore();
  const u = s.users.find((x) => x.id === s.session_user_id);
  const actor = u?.username ?? "system";
  const entry = {
    id: id(),
    timestamp: now(),
    actor,
    action,
    target,
    details,
  };
  setStore((st) => ({ ...st, audit: [entry, ...st.audit] }));
  return entry;
}

register("GET", "/api/keys", () => getStore().keys);
register("POST", "/api/keys", (_p, body) => {
  const full = `sk-${id()}${id()}${id()}`;
  const k: ApiKey = {
    id: id(),
    name: body.name,
    prefix: full.slice(0, 10),
    full_key: full,
    scopes: body.scopes ?? ["chat"],
    expires_at: body.expires_at,
    rpm_limit: body.rpm_limit,
    tpm_limit: body.tpm_limit,
    daily_spend_cap: body.daily_spend_cap,
    is_active: true,
    created_at: now(),
  };
  setStore((s) => ({ ...s, keys: [...s.keys, k] }));
  recordAudit("create_key", `api_key:${k.name}`);
  return k;
});
register("DELETE", "/api/keys/:id", ({ id: kid }) => {
  const target = getStore().keys.find((k) => k.id === kid);
  setStore((s) => ({ ...s, keys: s.keys.filter((k) => k.id !== kid) }));
  recordAudit("delete_key", `api_key:${target?.name ?? kid}`);
  return { ok: true };
});
register("PUT", "/api/keys/:id", ({ id: kid }, body) => {
  setStore((s) => ({
    ...s,
    keys: s.keys.map((k) => (k.id === kid ? { ...k, ...body } : k)),
  }));
  const k = getStore().keys.find((k) => k.id === kid)!;
  if (typeof body?.is_active === "boolean") {
    recordAudit(body.is_active ? "enable_key" : "revoke_key", `api_key:${k.name}`);
  } else {
    recordAudit("update_key", `api_key:${k.name}`);
  }
  return k;
});
register("POST", "/api/keys/:id/regenerate", ({ id: kid }) => {
  const full = `sk-${id()}${id()}${id()}`;
  setStore((s) => ({
    ...s,
    keys: s.keys.map((k) =>
      k.id === kid ? { ...k, full_key: full, prefix: full.slice(0, 10), is_active: true } : k,
    ),
  }));
  const k = getStore().keys.find((k) => k.id === kid)!;
  recordAudit("regenerate_key", `api_key:${k.name}`, `new prefix ${k.prefix}`);
  return k;
});
register("POST", "/api/audit", (_p, body) => {
  if (!body?.action || !body?.target) throw new ApiError(400, "action and target required");
  return recordAudit(String(body.action), String(body.target), body.details ? String(body.details) : undefined);
});

// ─── VIRTUAL KEYS ────────────────────────────────────────────────────────
register("GET", "/api/virtual-keys", () => getStore().vkeys);
register("POST", "/api/virtual-keys", (_p, body) => {
  const vk: VirtualKey = {
    id: id(),
    name: body.name,
    prefix: `vk-${id().slice(0, 6)}`,
    budget_usd: body.budget_usd,
    budget_used_usd: 0,
    budget_period: body.budget_period ?? "monthly",
    rate_limit_rpm: body.rate_limit_rpm,
    rate_limit_tpm: body.rate_limit_tpm,
    team_id: body.team_id,
    is_active: true,
  };
  setStore((s) => ({ ...s, vkeys: [...s.vkeys, vk] }));
  return vk;
});
register("PUT", "/api/virtual-keys/:id", ({ id: vid }, body) => {
  setStore((s) => ({
    ...s,
    vkeys: s.vkeys.map((v) => (v.id === vid ? { ...v, ...body } : v)),
  }));
  return getStore().vkeys.find((v) => v.id === vid)!;
});
register("DELETE", "/api/virtual-keys/:id", ({ id: vid }) => {
  setStore((s) => ({ ...s, vkeys: s.vkeys.filter((v) => v.id !== vid) }));
  return { ok: true };
});

// ─── TEAMS ───────────────────────────────────────────────────────────────
register("GET", "/api/teams", () => getStore().teams);
register("POST", "/api/teams", (_p, body) => {
  const t: Team = {
    id: id(),
    name: body.name,
    budget_usd: body.budget_usd,
    budget_used_usd: 0,
    keys_count: 0,
    members: 1,
  };
  setStore((s) => ({ ...s, teams: [...s.teams, t] }));
  return t;
});
register("PUT", "/api/teams/:id", ({ id: tid }, body) => {
  setStore((s) => ({
    ...s,
    teams: s.teams.map((t) => (t.id === tid ? { ...t, ...body } : t)),
  }));
  return getStore().teams.find((t) => t.id === tid)!;
});
register("DELETE", "/api/teams/:id", ({ id: tid }) => {
  setStore((s) => ({ ...s, teams: s.teams.filter((t) => t.id !== tid) }));
  return { ok: true };
});

// ─── COMBOS & ROUTING ────────────────────────────────────────────────────
register("GET", "/api/combos", () => getStore().combos);
register("POST", "/api/combos", (_p, body) => {
  const c: Combo = {
    id: id(),
    name: body.name,
    strategy: body.strategy ?? "fallback",
    steps: body.steps ?? [],
    sticky_limit: body.sticky_limit,
    is_active: true,
  };
  setStore((s) => ({ ...s, combos: [...s.combos, c] }));
  return c;
});
register("PUT", "/api/combos/:id", ({ id: cid }, body) => {
  setStore((s) => ({
    ...s,
    combos: s.combos.map((c) => (c.id === cid ? { ...c, ...body } : c)),
  }));
  return getStore().combos.find((c) => c.id === cid)!;
});
register("DELETE", "/api/combos/:id", ({ id: cid }) => {
  setStore((s) => ({ ...s, combos: s.combos.filter((c) => c.id !== cid) }));
  return { ok: true };
});

register("GET", "/api/routing-rules", () => getStore().routingRules);
register("POST", "/api/routing-rules", (_p, body) => {
  const r: RoutingRule = { id: id(), ...body, is_active: true };
  setStore((s) => ({ ...s, routingRules: [...s.routingRules, r] }));
  return r;
});
register("PUT", "/api/routing-rules/:id", ({ id: rid }, body) => {
  setStore((s) => ({
    ...s,
    routingRules: s.routingRules.map((r) => (r.id === rid ? { ...r, ...body } : r)),
  }));
  return getStore().routingRules.find((r) => r.id === rid)!;
});
register("DELETE", "/api/routing-rules/:id", ({ id: rid }) => {
  setStore((s) => ({ ...s, routingRules: s.routingRules.filter((r) => r.id !== rid) }));
  return { ok: true };
});

// ─── ALIASES ─────────────────────────────────────────────────────────────
register("GET", "/api/aliases", () => getStore().aliases);
register("POST", "/api/aliases", (_p, body) => {
  const a: Alias = { id: id(), created_at: now(), ...body };
  setStore((s) => ({ ...s, aliases: [...s.aliases, a] }));
  return a;
});
register("PUT", "/api/aliases/:id", ({ id: aid }, body) => {
  setStore((s) => ({
    ...s,
    aliases: s.aliases.map((a) => (a.id === aid ? { ...a, ...body } : a)),
  }));
  return getStore().aliases.find((a) => a.id === aid)!;
});
register("DELETE", "/api/aliases/:id", ({ id: aid }) => {
  setStore((s) => ({ ...s, aliases: s.aliases.filter((a) => a.id !== aid) }));
  return { ok: true };
});

// ─── PRICING ─────────────────────────────────────────────────────────────
register("GET", "/api/pricing", () => getStore().pricing);
register("POST", "/api/pricing", (_p, body) => {
  const po: PricingOverride = { id: id(), ...body };
  setStore((s) => ({ ...s, pricing: [...s.pricing, po] }));
  return po;
});
register("PUT", "/api/pricing/:id", ({ id: pid }, body) => {
  setStore((s) => ({
    ...s,
    pricing: s.pricing.map((p) => (p.id === pid ? { ...p, ...body } : p)),
  }));
  return getStore().pricing.find((p) => p.id === pid)!;
});
register("DELETE", "/api/pricing/:id", ({ id: pid }) => {
  setStore((s) => ({ ...s, pricing: s.pricing.filter((p) => p.id !== pid) }));
  return { ok: true };
});

// ─── USAGE ───────────────────────────────────────────────────────────────
register("GET", "/api/usage/summary", (_p, _b, q) => {
  const period = q.get("period") || "today";
  const ms = period === "today" || period === "24h" ? 86400000 : period === "7d" ? 7 * 86400000 : period === "30d" ? 30 * 86400000 : 60 * 86400000;
  const cut = Date.now() - ms;
  const logs = getStore().usageLogs.filter((l) => new Date(l.timestamp).getTime() > cut);
  const total_tokens = logs.reduce((s, l) => s + l.total_tokens, 0);
  const total_cost = logs.reduce((s, l) => s + l.cost_usd, 0);
  const avg_latency = logs.length ? logs.reduce((s, l) => s + l.latency_ms, 0) / logs.length : 0;
  return {
    total_requests: logs.length,
    total_tokens,
    total_cost,
    avg_latency_ms: Math.round(avg_latency),
  };
});

register("GET", "/api/usage", (_p, _b, q) => {
  const offset = parseInt(q.get("offset") || "0", 10);
  const limit = parseInt(q.get("limit") || "50", 10);
  const all = [...getStore().usageLogs].sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
  );
  return { items: all.slice(offset, offset + limit), total: all.length };
});

register("GET", "/api/usage/chart", (_p, _b, q) => {
  const period = q.get("period") || "7d";
  const granularity = q.get("granularity") || (period === "today" || period === "24h" ? "hour" : "day");
  const bucketMs = granularity === "hour" ? 3600000 : 86400000;
  const totalMs = period === "today" || period === "24h" ? 86400000 : period === "7d" ? 7 * 86400000 : period === "30d" ? 30 * 86400000 : 60 * 86400000;
  const buckets = Math.floor(totalMs / bucketMs);
  const start = Date.now() - totalMs;
  const labels: string[] = [];
  const requests: number[] = [];
  const tokens_input: number[] = [];
  const tokens_output: number[] = [];
  const costs: number[] = [];

  const logs = getStore().usageLogs.filter((l) => new Date(l.timestamp).getTime() > start);
  for (let i = 0; i < buckets; i++) {
    const from = start + i * bucketMs;
    const to = from + bucketMs;
    const inBucket = logs.filter((l) => {
      const t = new Date(l.timestamp).getTime();
      return t >= from && t < to;
    });
    labels.push(new Date(from).toISOString());
    requests.push(inBucket.length);
    tokens_input.push(inBucket.reduce((s, l) => s + l.prompt_tokens, 0));
    tokens_output.push(inBucket.reduce((s, l) => s + l.completion_tokens, 0));
    costs.push(inBucket.reduce((s, l) => s + l.cost_usd, 0));
  }
  return { buckets: labels, requests, tokens_input, tokens_output, costs };
});

register("GET", "/api/logs/:id", ({ id: lid }) => {
  const l = getStore().usageLogs.find((x) => x.id === lid);
  if (!l) throw new ApiError(404, "Not found");
  return {
    ...l,
    request: { model: l.model, messages: [{ role: "user", content: "(redacted)" }] },
    response: { id: l.id, choices: [{ message: { role: "assistant", content: "(redacted)" } }] },
  };
});

// ─── QUOTA ───────────────────────────────────────────────────────────────
register("GET", "/api/quota", () => getStore().quotas);

// ─── MODELS ──────────────────────────────────────────────────────────────
register("GET", "/api/models", () => getStore().models);
register("GET", "/api/models/disabled", () =>
  getStore().models.filter((m) => m.is_disabled),
);
register("POST", "/api/models/disabled", (_p, body) => {
  setStore((s) => ({
    ...s,
    models: s.models.map((m) =>
      m.provider === body.provider && m.name === body.model ? { ...m, is_disabled: true } : m,
    ),
  }));
  return { ok: true };
});
register("DELETE", "/api/models/disabled", (_p, body) => {
  setStore((s) => ({
    ...s,
    models: s.models.map((m) =>
      m.provider === body.provider && m.name === body.model ? { ...m, is_disabled: false } : m,
    ),
  }));
  return { ok: true };
});
register("POST", "/api/models/custom", (_p, body) => {
  const m = {
    id: `${body.provider}/${body.model}`,
    provider: body.provider,
    name: body.model,
    input_cost: body.input_cost ?? 1,
    output_cost: body.output_cost ?? 1,
    context_window: body.context_window ?? 8192,
    is_disabled: false,
    is_custom: true,
  };
  setStore((s) => ({ ...s, models: [...s.models, m] }));
  return m;
});
register("DELETE", "/api/models/custom/:id", ({ id: mid }) => {
  setStore((s) => ({ ...s, models: s.models.filter((m) => m.id !== mid) }));
  return { ok: true };
});

// ─── CHAT SESSIONS ───────────────────────────────────────────────────────
register("GET", "/api/chat-sessions", () =>
  getStore().sessions.map(({ messages: _m, ...rest }) => rest),
);
register("GET", "/api/chat-sessions/:id", ({ id: sid }) => {
  const s = getStore().sessions.find((x) => x.id === sid);
  if (!s) throw new ApiError(404, "Not found");
  return s;
});
register("POST", "/api/chat-sessions", (_p, body) => {
  const s: ChatSession = {
    id: id(),
    title: body.title || "New chat",
    provider: body.provider,
    model: body.model,
    messages: [],
    created_at: now(),
    updated_at: now(),
  };
  setStore((st) => ({ ...st, sessions: [s, ...st.sessions] }));
  return s;
});
register("PUT", "/api/chat-sessions/:id", ({ id: sid }, body) => {
  setStore((s) => ({
    ...s,
    sessions: s.sessions.map((x) =>
      x.id === sid ? { ...x, ...body, updated_at: now() } : x,
    ),
  }));
  return getStore().sessions.find((x) => x.id === sid)!;
});
register("DELETE", "/api/chat-sessions/:id", ({ id: sid }) => {
  setStore((s) => ({ ...s, sessions: s.sessions.filter((x) => x.id !== sid) }));
  return { ok: true };
});

// ─── PROXY POOLS ─────────────────────────────────────────────────────────
register("GET", "/api/proxy-pools", () => getStore().proxyPools);
register("POST", "/api/proxy-pools", (_p, body) => {
  const p: ProxyPool = { id: id(), is_active: true, ...body };
  setStore((s) => ({ ...s, proxyPools: [...s.proxyPools, p] }));
  return p;
});
register("PUT", "/api/proxy-pools/:id", ({ id: pid }, body) => {
  setStore((s) => ({
    ...s,
    proxyPools: s.proxyPools.map((p) => (p.id === pid ? { ...p, ...body } : p)),
  }));
  return getStore().proxyPools.find((p) => p.id === pid)!;
});
register("DELETE", "/api/proxy-pools/:id", ({ id: pid }) => {
  setStore((s) => ({ ...s, proxyPools: s.proxyPools.filter((p) => p.id !== pid) }));
  return { ok: true };
});
register("POST", "/api/proxy-pools/:id/test", () => ({
  ok: Math.random() > 0.2,
  latency_ms: Math.floor(Math.random() * 1000) + 100,
}));
register("POST", "/api/proxy-pools/batch", (_p, body) => {
  const list: string[] = body.proxies || [];
  const created: ProxyPool[] = list.map((line, i) => {
    const [host, port] = line.split(":");
    return {
      id: id(),
      name: `imported-${i + 1}`,
      protocol: "http",
      host: host || "proxy.example.com",
      port: parseInt(port || "8080", 10),
      is_active: true,
    };
  });
  setStore((s) => ({ ...s, proxyPools: [...s.proxyPools, ...created] }));
  return { created: created.length };
});

// ─── TUNNELS ─────────────────────────────────────────────────────────────
register("GET", "/api/tunnels", () => getStore().tunnels);
register("POST", "/api/tunnels/cloudflare", () => {
  setStore((s) => ({
    ...s,
    tunnels: s.tunnels.map((t) =>
      t.type === "cloudflare"
        ? { ...t, is_enabled: true, status: "active", url: `https://${id()}.trycloudflare.com` }
        : t,
    ),
  }));
  return getStore().tunnels.find((t) => t.type === "cloudflare");
});
register("DELETE", "/api/tunnels/cloudflare", () => {
  setStore((s) => ({
    ...s,
    tunnels: s.tunnels.map((t) =>
      t.type === "cloudflare" ? { ...t, is_enabled: false, status: "inactive", url: undefined } : t,
    ),
  }));
  return { ok: true };
});
register("POST", "/api/tunnels/tailscale", () => {
  setStore((s) => ({
    ...s,
    tunnels: s.tunnels.map((t) =>
      t.type === "tailscale"
        ? { ...t, is_enabled: true, status: "active", url: `https://g0router.tailnet-${id().slice(0, 5)}.ts.net` }
        : t,
    ),
  }));
  return getStore().tunnels.find((t) => t.type === "tailscale");
});
register("DELETE", "/api/tunnels/tailscale", () => {
  setStore((s) => ({
    ...s,
    tunnels: s.tunnels.map((t) =>
      t.type === "tailscale" ? { ...t, is_enabled: false, status: "inactive", url: undefined } : t,
    ),
  }));
  return { ok: true };
});
register("GET", "/api/tunnels/health", () => ({ ok: true }));

// ─── MITM ────────────────────────────────────────────────────────────────
register("GET", "/api/mitm/status", () => ({
  enabled: getStore().mitm.some((t) => t.enabled),
  tools: getStore().mitm,
  cert_installed: false,
}));
register("POST", "/api/mitm/toggle", (_p, body) => ({ enabled: !!body.enabled }));
register("GET", "/api/mitm/ca-cert", () => ({ cert: "-----BEGIN CERTIFICATE-----\nMOCK_CERT_DATA\n-----END CERTIFICATE-----" }));
register("PUT", "/api/mitm/tools/:tool", ({ tool }, body) => {
  setStore((s) => ({
    ...s,
    mitm: s.mitm.map((t) =>
      t.id === tool ? { ...t, ...body, status: body.enabled ? "active" : "inactive" } : t,
    ),
  }));
  return getStore().mitm.find((t) => t.id === tool)!;
});

// ─── MCP ─────────────────────────────────────────────────────────────────
register("GET", "/api/mcp/clients", () =>
  getStore().mcpInstances.map((i) => ({
    name: i.name,
    transport: i.type,
    status: i.status,
    tools_count: i.tools_count,
  })),
);
register("GET", "/api/mcp/instances", () => getStore().mcpInstances);
register("POST", "/api/mcp/instances", (_p, body) => {
  const i: McpInstance = {
    id: id(),
    status: "stopped",
    health: "unhealthy",
    tools_count: 0,
    ...body,
  };
  setStore((s) => ({ ...s, mcpInstances: [...s.mcpInstances, i] }));
  return i;
});
register("PUT", "/api/mcp/instances/:id", ({ id: iid }, body) => {
  setStore((s) => ({
    ...s,
    mcpInstances: s.mcpInstances.map((i) => (i.id === iid ? { ...i, ...body } : i)),
  }));
  return getStore().mcpInstances.find((i) => i.id === iid)!;
});
register("DELETE", "/api/mcp/instances/:id", ({ id: iid }) => {
  setStore((s) => ({ ...s, mcpInstances: s.mcpInstances.filter((i) => i.id !== iid) }));
  return { ok: true };
});
register("GET", "/api/mcp/accounts", () => getStore().mcpAccounts);
register("GET", "/api/mcp/tools", () => getStore().mcpTools);
register("POST", "/api/mcp/tools/:name/execute", ({ name }, body) => ({
  tool: name,
  args: body,
  result: { ok: true, output: `Executed ${name} with mock result.` },
  duration_ms: Math.floor(Math.random() * 500) + 80,
}));
register("GET", "/api/mcp/tool-groups", () => getStore().mcpToolGroups);
register("POST", "/api/mcp/tool-groups", (_p, body) => {
  const g: McpToolGroup = { id: id(), tools: [], ...body };
  setStore((s) => ({ ...s, mcpToolGroups: [...s.mcpToolGroups, g] }));
  return g;
});
register("PUT", "/api/mcp/tool-groups/:id", ({ id: gid }, body) => {
  setStore((s) => ({
    ...s,
    mcpToolGroups: s.mcpToolGroups.map((g) => (g.id === gid ? { ...g, ...body } : g)),
  }));
  return getStore().mcpToolGroups.find((g) => g.id === gid)!;
});
register("DELETE", "/api/mcp/tool-groups/:id", ({ id: gid }) => {
  setStore((s) => ({ ...s, mcpToolGroups: s.mcpToolGroups.filter((g) => g.id !== gid) }));
  return { ok: true };
});

// ─── GUARDRAILS ──────────────────────────────────────────────────────────
register("GET", "/api/guardrails", () => getStore().guardrails);
register("PUT", "/api/guardrails", (_p, body) => {
  setStore((s) => ({ ...s, guardrails: { ...s.guardrails, ...body } }));
  return getStore().guardrails;
});
register("POST", "/api/guardrails/test", (_p, body) => {
  const g = getStore().guardrails;
  let out: string = body.prompt || "";
  if (g.pii_redaction) {
    out = out.replace(/[\w._%+-]+@[\w.-]+\.\w+/g, "[email]");
    out = out.replace(/\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b/g, "[phone]");
  }
  const blocked = g.blocklist.some((w) => out.toLowerCase().includes(w.toLowerCase()));
  return { filtered: out, blocked };
});

// ─── MODEL LIMITS ────────────────────────────────────────────────────────
register("GET", "/api/model-limits", () => getStore().modelLimits);
register("POST", "/api/model-limits", (_p, body) => {
  const ml: ModelLimit = { id: id(), allowed_keys: [], ...body };
  setStore((s) => ({ ...s, modelLimits: [...s.modelLimits, ml] }));
  return ml;
});
register("PUT", "/api/model-limits/:id", ({ id: mid }, body) => {
  setStore((s) => ({
    ...s,
    modelLimits: s.modelLimits.map((m) => (m.id === mid ? { ...m, ...body } : m)),
  }));
  return getStore().modelLimits.find((m) => m.id === mid)!;
});
register("DELETE", "/api/model-limits/:id", ({ id: mid }) => {
  setStore((s) => ({ ...s, modelLimits: s.modelLimits.filter((m) => m.id !== mid) }));
  return { ok: true };
});

// ─── PROMPTS ─────────────────────────────────────────────────────────────
register("GET", "/api/prompt-templates", () => getStore().prompts);
register("POST", "/api/prompt-templates", (_p, body) => {
  const p: PromptTemplate = { id: id(), variables: [], ...body };
  setStore((s) => ({ ...s, prompts: [...s.prompts, p] }));
  return p;
});
register("PUT", "/api/prompt-templates/:id", ({ id: pid }, body) => {
  setStore((s) => ({
    ...s,
    prompts: s.prompts.map((p) => (p.id === pid ? { ...p, ...body } : p)),
  }));
  return getStore().prompts.find((p) => p.id === pid)!;
});
register("DELETE", "/api/prompt-templates/:id", ({ id: pid }) => {
  setStore((s) => ({ ...s, prompts: s.prompts.filter((p) => p.id !== pid) }));
  return { ok: true };
});

// ─── ALERT CHANNELS ──────────────────────────────────────────────────────
register("GET", "/api/alert-channels", () => getStore().alerts);
register("POST", "/api/alert-channels", (_p, body) => {
  const a: AlertChannel = { id: id(), is_active: true, config: {}, ...body };
  setStore((s) => ({ ...s, alerts: [...s.alerts, a] }));
  return a;
});
register("PUT", "/api/alert-channels/:id", ({ id: aid }, body) => {
  setStore((s) => ({
    ...s,
    alerts: s.alerts.map((a) => (a.id === aid ? { ...a, ...body } : a)),
  }));
  return getStore().alerts.find((a) => a.id === aid)!;
});
register("DELETE", "/api/alert-channels/:id", ({ id: aid }) => {
  setStore((s) => ({ ...s, alerts: s.alerts.filter((a) => a.id !== aid) }));
  return { ok: true };
});
register("POST", "/api/alert-channels/:id/test", () => ({ ok: true, message: "Test alert sent (mock)" }));

// ─── FEATURE FLAGS ───────────────────────────────────────────────────────
register("GET", "/api/feature-flags", () => getStore().flags);
register("PUT", "/api/feature-flags/:id", ({ id: fid }, body) => {
  setStore((s) => ({
    ...s,
    flags: s.flags.map((f) => (f.id === fid ? { ...f, ...body } : f)),
  }));
  return getStore().flags.find((f) => f.id === fid)!;
});

// ─── SETTINGS ────────────────────────────────────────────────────────────
register("GET", "/api/settings", () => getStore().settings);
register("PUT", "/api/settings", (_p, body) => {
  setStore((s) => ({ ...s, settings: { ...s.settings, ...body } }));
  return getStore().settings;
});
register("POST", "/api/settings/backup", () => getStore());
register("POST", "/api/settings/restore", (_p, body) => {
  setStore(() => body);
  return { ok: true };
});
register("POST", "/api/settings/proxy-test", () => ({
  ok: Math.random() > 0.3,
  latency_ms: Math.floor(Math.random() * 600) + 100,
}));

// ─── AUDIT ───────────────────────────────────────────────────────────────
register("GET", "/api/audit", (_p, _b, q) => {
  const offset = parseInt(q.get("offset") || "0", 10);
  const limit = parseInt(q.get("limit") || "50", 10);
  const items = [...getStore().audit].sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
  );
  return { items: items.slice(offset, offset + limit), total: items.length };
});

// ─── DIAGNOSTICS ─────────────────────────────────────────────────────────
register("GET", "/api/diagnostics", () => ({
  checks: [
    { name: "Database", status: "ok", latency_ms: 4 },
    { name: "Inference engine", status: "ok", latency_ms: 2 },
    { name: "MCP manager", status: "ok", latency_ms: 1 },
    { name: "Cache", status: "ok", latency_ms: 1 },
  ],
  readiness: [
    { item: "Auth configured", ok: true },
    { item: "At least one provider", ok: getStore().providers.some((p) => p.connection_count > 0) },
    { item: "API key created", ok: getStore().keys.length > 0 },
    { item: "Logs enabled", ok: getStore().settings.enable_request_logs },
  ],
  version: { app: "0.9.0-mock", go: "1.23.4", build_date: "2026-06-06" },
  resources: { memory_mb: 142, goroutines: 87, uptime_seconds: 3600 },
}));

// ─── SKILLS ──────────────────────────────────────────────────────────────
register("GET", "/api/skills", () => getStore().skills);

// ─── LOCALE / VERSION / UPDATE ───────────────────────────────────────────
register("GET", "/api/locale", () => ({ locale: getStore().settings.language || "en" }));
register("POST", "/api/locale", (_p, body) => {
  setStore((s) => ({ ...s, settings: { ...s.settings, language: body.locale } }));
  return { ok: true };
});
register("GET", "/api/version", () => ({ version: "0.9.0-mock", go_version: "1.23.4", build_date: "2026-06-06" }));
register("POST", "/api/update/check", () => ({
  current: "0.9.0-mock",
  latest: "0.9.0-mock",
  update_available: false,
  changelog_url: "https://example.com/changelog",
}));
register("POST", "/api/update/apply", () => ({ ok: true, message: "Update staged (mock)" }));

// ─── HEALTH ──────────────────────────────────────────────────────────────
register("GET", "/healthz", () => ({ ok: true }));

// ─── DISPATCHER ──────────────────────────────────────────────────────────
export async function dispatch(
  method: string,
  url: string,
  body?: any,
): Promise<any> {
  await sleep(Math.floor(Math.random() * 250) + 80);
  const err = maybeError();
  if (err) throw new ApiError(err.status, err);

  const u = new URL(url, "http://mock");
  const path = u.pathname;
  const query = u.searchParams;

  for (const r of routes) {
    if (r.method !== method) continue;
    const m = r.pattern.exec(path);
    if (!m) continue;
    const params: Record<string, string> = {};
    r.keys.forEach((k, i) => (params[k] = decodeURIComponent(m[i + 1])));
    return r.handler(params, body, query);
  }
  throw new ApiError(404, `No mock for ${method} ${path}`);
}

export function isAuthRequired(path: string) {
  return !path.startsWith("/api/auth/") && path !== "/healthz" && path !== "/api/locale" && path !== "/api/version";
}
