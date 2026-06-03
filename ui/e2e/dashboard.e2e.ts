import { expect, test, type Page, type Request } from "@playwright/test";

type RecordedAPIRequest = {
  authorization: string | null;
  method: string;
  path: string;
};

test.describe("dashboard control plane", () => {
  test("saves control-plane auth and sends it on mocked API requests", async ({ page }) => {
    const apiRequests = await mockAPI(page);

    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Gateway overview" })).toBeVisible();
    await page.getByLabel("Control-plane API key").fill("test-control-plane-key");
    await page.getByRole("button", { name: "Save key" }).click();

    await expect
      .poll(() => apiRequests.some((request) => request.authorization === "Bearer test-control-plane-key"))
      .toBe(true);
    await expect(page.getByRole("button", { name: "Clear" })).toBeEnabled();
  });

  test("navigates existing dashboard pages with mocked API data", async ({ page }) => {
    await mockAPI(page);

    await page.goto("/");

    await expect(page.getByRole("heading", { name: "Gateway overview" })).toBeVisible();
    await expect(page.getByText("Active providers")).toBeVisible();
    await expect(page.getByText("research-stack")).toBeVisible();

    await navigateTo(page, "Endpoint");
    await expect(page.getByRole("heading", { name: "Endpoint configuration" })).toBeVisible();
    await expect(page.getByRole("table", { name: "API keys" })).toContainText("desktop-client");

    await navigateTo(page, "Providers");
    await expect(page.getByRole("heading", { name: "Providers" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Provider contract" })).toContainText("openai");
    await expect(page.getByRole("table", { name: "Provider connections" })).toContainText("OpenAI primary");

    await navigateTo(page, "Usage");
    await expect(page.getByRole("heading", { exact: true, name: "Usage" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Usage rows" })).toContainText("req_001");
    await expect(page.getByRole("table", { name: "Request logs" })).toContainText("codex");

    await navigateTo(page, "Quota");
    await expect(page.getByRole("heading", { exact: true, name: "Quota" })).toBeVisible();
    await expect(page.getByRole("article", { name: "openai quota" })).toContainText("850 remaining");

    await navigateTo(page, "Combos");
    await expect(page.getByRole("heading", { exact: true, name: "Combos" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Combo routes" })).toContainText("research-stack");

    await navigateTo(page, "MCP");
    await expect(page.getByRole("heading", { exact: true, name: "MCP" })).toBeVisible();
    await expect(page.getByRole("table", { name: "MCP instances" })).toContainText("linear-tools");
    await expect(page.getByRole("heading", { name: "Tools" })).toBeVisible();

    await navigateTo(page, "Settings");
    await expect(page.getByRole("heading", { exact: true, name: "Settings" })).toBeVisible();
    await expect(page.getByLabel("Proxy URL")).toHaveValue("http://127.0.0.1:8080");
  });

  test("executes existing dashboard mutations with mocked API data", async ({ page }) => {
    await mockAPI(page);

    await page.goto("/");

    await navigateTo(page, "Endpoint");
    await page.getByLabel("Key name").fill("automation-client");
    await page.getByRole("button", { name: "Create key" }).click();
    await expect(page.getByText("New gateway key")).toBeVisible();
    await expect(page.getByText("g0r_e2e_created_secret")).toBeVisible();
    await page.getByRole("button", { name: "Dismiss" }).click();
    await page.getByRole("button", { name: "Delete automation-client" }).click();
    await expect(page.getByRole("table", { name: "API keys" })).not.toContainText("automation-client");

    await navigateTo(page, "Combos");
    await page.getByLabel("Combo name").fill("fast-fallback");
    await page.getByLabel("Step provider").fill("openai");
    await page.getByLabel("Step model").fill("gpt-5-mini");
    await page.getByRole("button", { name: "Create combo" }).click();
    await expect(page.getByRole("table", { name: "Combo routes" })).toContainText("fast-fallback");
    await page.getByRole("button", { name: "Delete fast-fallback" }).click();
    await expect(page.getByRole("table", { name: "Combo routes" })).not.toContainText("fast-fallback");

    await navigateTo(page, "MCP");
    await page.getByLabel("Instance name").fill("github-tools");
    await page.getByLabel("Server key").fill("github");
    await page.getByRole("textbox", { exact: true, name: "URL" }).fill("https://mcp.github.example.test");
    await page.getByRole("button", { name: "Create instance" }).click();
    await expect(page.getByRole("table", { name: "MCP instances" })).toContainText("github-tools");
    await page.getByRole("combobox", { name: /^Instance/ }).selectOption("mcp-created");
    await page.getByLabel("Authorization URL").fill("https://auth.example.test/authorize");
    await page.getByLabel("Resource URI").fill("https://mcp.github.example.test");
    await page.getByRole("button", { name: "Start OAuth" }).click();
    await expect(page.getByRole("link", { name: "Open authorization URL" })).toHaveAttribute(
      "href",
      "https://auth.example.test/authorize?state=e2e"
    );

    await navigateTo(page, "Settings");
    await page.getByLabel("Proxy URL").fill("http://127.0.0.1:9090");
    await page.getByRole("button", { name: "Save settings" }).click();
    await expect(page.getByText("Settings saved")).toBeVisible();
    await expect(page.getByLabel("Proxy URL")).toHaveValue("http://127.0.0.1:9090");
  });
});

async function navigateTo(page: Page, label: string) {
  await page.getByRole("button", { name: label }).click();
}

async function mockAPI(page: Page) {
  const apiRequests: RecordedAPIRequest[] = [];
  const state = {
    apiKeys: [...apiKeys],
    combos: [...combos],
    mcpInstances: [...mcpInstances],
    settings: { ...settings }
  };

  await page.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());

    if (url.origin === "http://127.0.0.1:5173" && url.pathname.startsWith("/api/")) {
      apiRequests.push(recordAPIRequest(request, url));
      const response = apiResponse(state, url.pathname, request.method(), request.postDataJSON());
      await route.fulfill({
        contentType: "application/json",
        status: response.status,
        body: JSON.stringify(response.body)
      });
      return;
    }

    if (url.origin === "http://127.0.0.1:5173") {
      await route.continue();
      return;
    }

    await route.abort("blockedbyclient");
  });

  return apiRequests;
}

function recordAPIRequest(request: Request, url: URL): RecordedAPIRequest {
  return {
    authorization: request.headers().authorization ?? null,
    method: request.method(),
    path: url.pathname
  };
}

function apiResponse(state: MockAPIState, path: string, method: string, body: unknown): MockAPIResponse {
  if (method === "POST" && path === "/api/keys") {
    const request = body as { name?: string };
    const key = {
      ID: "key-created",
      Name: request.name ?? "created-key",
      Prefix: "g0r_e2e_created",
      IsActive: true,
      LastUsedAt: null,
      CreatedAt: "2026-06-03T11:00:00Z"
    };
    state.apiKeys = [...state.apiKeys, key];
    return { status: 201, body: { key, raw: "g0r_e2e_created_secret" } };
  }

  if (method === "DELETE" && path.startsWith("/api/keys/")) {
    const id = decodeURIComponent(path.slice("/api/keys/".length));
    state.apiKeys = state.apiKeys.filter((key) => key.ID !== id);
    return { status: 204, body: undefined };
  }

  if (method === "POST" && path === "/api/combos") {
    const request = body as { is_active?: boolean; name?: string; steps?: Array<{ model: string; provider: string }> };
    const combo = {
      ID: "combo-created",
      Name: request.name ?? "created-combo",
      Steps: request.steps ?? [],
      IsActive: request.is_active ?? true,
      CreatedAt: "2026-06-03T11:00:00Z",
      UpdatedAt: "2026-06-03T11:00:00Z"
    };
    state.combos = [...state.combos, combo];
    return { status: 201, body: combo };
  }

  if (method === "DELETE" && path.startsWith("/api/combos/")) {
    const id = decodeURIComponent(path.slice("/api/combos/".length));
    state.combos = state.combos.filter((combo) => combo.ID !== id);
    return { status: 204, body: undefined };
  }

  if (method === "POST" && path === "/api/mcp/instances") {
    const request = body as {
      account_label?: string;
      command?: string;
      is_active?: boolean;
      launch_type?: string;
      name?: string;
      server_key?: string;
      transport?: string;
      url?: string;
    };
    const instance = {
      ID: "mcp-created",
      Name: request.name ?? "created-mcp",
      ServerKey: request.server_key ?? "created",
      LaunchType: request.launch_type ?? "http",
      Transport: request.transport ?? "streamable-http",
      URL: request.url ?? null,
      Command: request.command ?? null,
      Headers: {},
      Env: {},
      AccountLabel: request.account_label ?? null,
      IsActive: request.is_active ?? true,
      HealthStatus: "starting",
      ToolManifest: { tools: [] },
      CreatedAt: "2026-06-03T11:00:00Z",
      UpdatedAt: "2026-06-03T11:00:00Z"
    };
    state.mcpInstances = [...state.mcpInstances, instance];
    return { status: 201, body: instance };
  }

  if (method === "POST" && path.endsWith("/auth/start")) {
    return { status: 201, body: { authorization_url: "https://auth.example.test/authorize?state=e2e", expires_at: "2026-06-03T12:00:00Z" } };
  }

  if (method === "PUT" && path === "/api/settings") {
    state.settings = body as typeof settings;
    return { status: 200, body: state.settings };
  }

  if (method !== "GET") {
    return { status: 404, body: { error: `No E2E API fixture for ${method} ${path}` } };
  }

  switch (path) {
    case "/api/providers":
      return { status: 200, body: { data: providers } };
    case "/api/connections":
      return { status: 200, body: { data: connections } };
    case "/api/keys":
      return { status: 200, body: { data: state.apiKeys } };
    case "/api/usage":
    case "/api/logs":
      return { status: 200, body: { object: "list", data: usageRows, limit: 25, offset: 0 } };
    case "/api/usage/summary":
      return { status: 200, body: { request_count: 2, total_tokens: 1250, total_cost_usd: 0.034 } };
    case "/api/usage/quota/openai":
      return { status: 200, body: { Provider: "openai", Limit: 1000, Used: 150, Remaining: 850 } };
    case "/api/combos":
      return { status: 200, body: { data: state.combos } };
    case "/api/mcp/clients":
      return { status: 200, body: { data: mcpClients } };
    case "/api/mcp/instances":
      return { status: 200, body: { data: state.mcpInstances } };
    case "/api/mcp/instances/mcp-1/accounts":
      return { status: 200, body: { data: mcpAccounts } };
    case "/api/mcp/instances/mcp-created/accounts":
      return { status: 200, body: { data: [] } };
    case "/api/mcp/tools":
      return { status: 200, body: { data: mcpTools } };
    case "/api/settings":
      return { status: 200, body: state.settings };
    default:
      return { status: 404, body: { error: `No E2E API fixture for ${path}` } };
  }
}

type MockAPIState = {
  apiKeys: typeof apiKeys;
  combos: typeof combos;
  mcpInstances: typeof mcpInstances;
  settings: typeof settings;
};

type MockAPIResponse = {
  body: unknown;
  status: number;
};

const providers = [
  {
    id: "openai",
    omp_id: "openai",
    router9_id: "openai",
    bifrost_id: "openai",
    auth_types: ["api_key"],
    refresh: false,
    registered_adapter: true,
    public_inference: true,
    direct_dispatch: true,
    inference: true,
    streaming: true,
    model_catalog: true,
    list_models: true,
    quota: true,
    public_status: "supported",
    notes: "E2E fixture"
  }
];

const connections = [
  {
    ID: "conn-1",
    Provider: "openai",
    Name: "OpenAI primary",
    AuthType: "api_key",
    IsActive: true,
    AccountID: "acct-openai",
    Email: "ops@example.test",
    BackoffLevel: 0,
    CreatedAt: "2026-06-03T10:00:00Z",
    UpdatedAt: "2026-06-03T10:00:00Z"
  }
];

const apiKeys = [
  {
    ID: "key-1",
    Name: "desktop-client",
    Prefix: "g0r_e2e",
    IsActive: true,
    LastUsedAt: "2026-06-03T10:00:00Z",
    CreatedAt: "2026-06-03T09:00:00Z"
  }
];

const usageRows = [
  {
    id: 1,
    request_id: "req_001",
    timestamp: "2026-06-03T10:00:00Z",
    provider: "openai",
    model: "gpt-5-mini",
    auth_type: "api_key",
    total_tokens: 1250,
    cost_usd: 0.034,
    latency_ms: 320,
    status_code: 200,
    source_format: "chat",
    target_format: "chat",
    client_tool: "codex"
  }
];

const combos = [
  {
    ID: "combo-1",
    Name: "research-stack",
    Steps: [{ provider: "openai", model: "gpt-5-mini" }],
    IsActive: true,
    CreatedAt: "2026-06-03T10:00:00Z",
    UpdatedAt: "2026-06-03T10:00:00Z"
  }
];

const mcpInstances = [
  {
    ID: "mcp-1",
    Name: "linear-tools",
    ServerKey: "linear",
    LaunchType: "http",
    Transport: "streamable-http",
    URL: "https://mcp.example.test",
    Headers: { "X-Mode": "readonly" },
    Env: { MODE: "readonly" },
    AccountLabel: "engineering",
    IsActive: true,
    HealthStatus: "healthy",
    ToolManifest: { tools: [{ name: "linear-search" }] },
    CreatedAt: "2026-06-03T10:00:00Z",
    UpdatedAt: "2026-06-03T10:00:00Z"
  }
];

const mcpClients = [
  {
    ID: "legacy-client",
    Name: "legacy-client",
    Transport: "stdio",
    Env: { MODE: "readonly" },
    IsActive: true,
    HealthStatus: "healthy",
    ToolManifest: { tools: [{ name: "legacy-read" }] },
    CreatedAt: "2026-06-03T09:00:00Z"
  }
];

const mcpAccounts = [
  {
    id: "acct-1",
    instance_id: "mcp-1",
    account_label: "engineering",
    email: "mcp@example.test",
    resource_uri: "https://mcp.example.test"
  }
];

const mcpTools = [
  {
    type: "function",
    function: {
      name: "mcp-1__linear-search",
      description: "Search linear issues"
    }
  }
];

const settings = {
  RequireAPIKey: true,
  RTKEnabled: true,
  CavemanEnabled: false,
  CavemanLevel: "full",
  EnableRequestLogs: true,
  ProxyURL: "http://127.0.0.1:8080",
  DataDir: "/var/lib/g0router"
};
