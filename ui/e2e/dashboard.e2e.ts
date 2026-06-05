import { expect, test, type Page, type Request } from "@playwright/test";

type RecordedAPIRequest = {
  authorization: string | null;
  body: unknown;
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

    await navigateTo(page, "Endpoint Setup");
    await expect(page.getByRole("heading", { name: "Endpoint configuration" })).toBeVisible();
    await expect(page.getByRole("table", { name: "API keys" })).toContainText("desktop-client");

    await navigateTo(page, "API Keys");
    await expect(page.getByRole("heading", { exact: true, name: "API Keys" })).toBeVisible();
    await expect(page.getByRole("table", { name: "API keys" })).toContainText("desktop-client");
    await expect(page.getByRole("button", { name: "Copy chat completions endpoint" })).not.toBeVisible();

    await navigateTo(page, "Providers");
    await expect(page.getByRole("heading", { name: "Providers" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Provider contract" })).toContainText("openai");
    await expect(page.getByRole("table", { name: "Provider connections" })).toContainText("OpenAI primary");
    await expect(page.getByText("e2e-provider-secret")).not.toBeVisible();

    await navigateTo(page, "Connections/Auth");
    await expect(page.getByRole("heading", { exact: true, name: "Connections/Auth" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Provider connections" })).toContainText("OpenAI primary");
    await expect(page.getByRole("table", { name: "Provider contract" })).not.toBeVisible();
    await expect(page.getByText("e2e-provider-secret")).not.toBeVisible();

    await navigateTo(page, "Aliases");
    await expect(page.getByRole("heading", { exact: true, name: "Aliases" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Model aliases" })).toContainText("fast");

    await navigateTo(page, "Models");
    await expect(page.getByRole("heading", { exact: true, name: "Models" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Provider models" })).toContainText("gpt-4o");
    await page.getByRole("combobox", { exact: true, name: "Provider" }).selectOption("anthropic");
    await expect(page.getByRole("table", { name: "Provider models" })).toContainText("claude-sonnet-4");

    await navigateTo(page, "Pricing");
    await expect(page.getByRole("heading", { exact: true, name: "Pricing" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Pricing overrides" })).toContainText("gpt-5-mini");

    await navigateTo(page, "Usage");
    await expect(page.getByRole("heading", { exact: true, name: "Usage" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Usage rows" })).toContainText("req_001");
    await expect(page.getByRole("table", { name: "Request logs" })).toContainText("codex");

    await navigateTo(page, "Logs");
    await expect(page.getByRole("heading", { exact: true, name: "Logs" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Request logs" })).toContainText("gpt-5-mini");

    await navigateTo(page, "Quotas");
    await expect(page.getByRole("heading", { exact: true, name: "Quotas" })).toBeVisible();
    await expect(page.getByText("No quota-capable providers")).toBeVisible();

    await navigateTo(page, "Combos/Routing");
    await expect(page.getByRole("heading", { exact: true, name: "Combos/Routing" })).toBeVisible();
    await expect(page.getByRole("table", { name: "Combo routes" })).toContainText("research-stack");

    await navigateTo(page, "MCP");
    await expect(page.getByRole("heading", { exact: true, name: "MCP" })).toBeVisible();
    await expect(page.getByRole("table", { name: "MCP instances" })).toContainText("linear-tools");
    await expect(page.getByRole("heading", { name: "Tools" })).toBeVisible();

    await navigateTo(page, "MCP Instances");
    await expect(page.getByRole("heading", { exact: true, name: "MCP Instances" })).toBeVisible();
    await expect(page.getByRole("table", { name: "MCP instances" })).toContainText("linear-tools");
    await expect(page.getByRole("heading", { name: "Start OAuth" })).not.toBeVisible();

    await navigateTo(page, "MCP Accounts");
    await expect(page.getByRole("heading", { exact: true, name: "MCP Accounts" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Start OAuth" })).toBeVisible();
    await expect(page.getByText("mcp@example.test")).toBeVisible();
    await expect(page.getByRole("table", { name: "MCP instances" })).not.toBeVisible();

    await navigateTo(page, "MCP Tools");
    await expect(page.getByRole("heading", { exact: true, name: "MCP Tools" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Execute tool" })).toBeVisible();
    await expect(page.getByRole("cell", { name: "mcp-1__linear-search" })).toBeVisible();
    await expect(page.getByRole("table", { name: "MCP instances" })).not.toBeVisible();

    await navigateTo(page, "Settings");
    await expect(page.getByRole("heading", { exact: true, name: "Settings" })).toBeVisible();
    await expect(page.getByLabel("Proxy URL")).toHaveValue("http://127.0.0.1:8080");

    await navigateTo(page, "Settings/Security");
    await expect(page.getByRole("heading", { exact: true, name: "Settings/Security" })).toBeVisible();
    await expect(page.getByLabel("Require API key")).toBeChecked();
    await expect(page.getByLabel("Enable request logs")).toBeChecked();

    await navigateTo(page, "Diagnostics");
    await expect(page.getByRole("heading", { exact: true, level: 2, name: "Diagnostics" })).toBeVisible();
    await expect(page.getByText("Control plane protected")).toBeVisible();
  });

  test("navigates to Quotas and renders quota state", async ({ page }) => {
    await mockAPI(page, { mode: "empty" });

    await page.goto("/");
    await navigateTo(page, "Quotas");

    await expect(page.getByRole("heading", { exact: true, name: "Quotas" })).toBeVisible();
    await expect(page.getByText("No quota-capable providers")).toBeVisible();
  });

  test("executes existing dashboard mutations with mocked API data", async ({ page }) => {
    await mockAPI(page);

    await page.goto("/");

    await navigateTo(page, "Endpoint Setup");
    await page.getByLabel("Key name").fill("automation-client");
    await page.getByRole("button", { name: "Create key" }).click();
    await expect(page.getByText("New gateway key")).toBeVisible();
    await expect(page.getByText("g0r_e2e_created_secret")).toBeVisible();
    await page.getByRole("button", { name: "Dismiss" }).click();
    await clickWithConfirm(page, "Delete API key automation-client?", () =>
      page.getByRole("button", { name: "Delete automation-client" }).click()
    );
    await expect(page.getByRole("table", { name: "API keys" })).not.toContainText("automation-client");

    await navigateTo(page, "Connections/Auth");
    await page.getByRole("combobox", { exact: true, name: "Provider" }).selectOption("openai");
    await page.getByLabel("Connection name").fill("OpenAI e2e");
    await page.getByLabel("Provider API key").fill("e2e-provider-secret");
    await page.getByRole("button", { name: "Add connection" }).click();
    await expect(page.getByRole("table", { name: "Provider connections" })).toContainText("OpenAI e2e");
    await expect(page.getByText("e2e-provider-secret")).not.toBeVisible();
    await page.getByRole("button", { name: "Test OpenAI e2e" }).click();
    await expect(page.getByText("OpenAI e2e is active")).toBeVisible();
    await page.getByRole("button", { name: "Deactivate OpenAI e2e" }).click();
    await expect(page.getByRole("row", { name: /OpenAI e2e openai local api_key inactive/i })).toBeVisible();
    await clickWithConfirm(page, "Delete provider connection OpenAI e2e?", () =>
      page.getByRole("button", { name: "Delete OpenAI e2e" }).click()
    );
    await expect(page.getByRole("table", { name: "Provider connections" })).not.toContainText("OpenAI e2e");

    await navigateTo(page, "Combos/Routing");
    await page.getByLabel("Combo name").fill("fast-fallback");
    await page.getByLabel("Step 1 provider").fill("openai");
    await page.getByLabel("Step 1 model").fill("gpt-5-mini");
    await page.getByRole("button", { name: "Create combo" }).click();
    await expect(page.getByRole("table", { name: "Combo routes" })).toContainText("fast-fallback");
    await page.getByRole("button", { name: "Edit fast-fallback" }).click();
    await page.getByLabel("Combo name").fill("fast-updated");
    await page.getByLabel("Step 1 model").fill("gpt-4o");
    await page.getByLabel("Active").uncheck();
    await page.getByRole("button", { name: "Update combo" }).click();
    await expect(page.getByRole("table", { name: "Combo routes" })).toContainText("fast-updated");
    await expect(page.getByRole("row", { name: /fast-updated openai \/ gpt-4o inactive/i })).toBeVisible();
    await clickWithConfirm(page, "Delete combo fast-updated?", () =>
      page.getByRole("button", { name: "Delete fast-updated" }).click()
    );
    await expect(page.getByRole("table", { name: "Combo routes" })).not.toContainText("fast-updated");

    await navigateTo(page, "Aliases");
    await page.getByRole("textbox", { exact: true, name: "Alias" }).fill("cheap");
    await page.getByRole("textbox", { exact: true, name: "Provider" }).fill("groq");
    await page.getByRole("textbox", { exact: true, name: "Model" }).fill("llama-3.3-70b-versatile");
    await page.getByRole("button", { name: "Create alias" }).click();
    await expect(page.getByRole("table", { name: "Model aliases" })).toContainText("cheap");
    await page.getByRole("button", { name: "Edit cheap" }).click();
    await page.getByRole("textbox", { exact: true, name: "Provider" }).fill("openai");
    await page.getByRole("textbox", { exact: true, name: "Model" }).fill("gpt-4o");
    await page.getByRole("button", { name: "Update alias" }).click();
    await expect(page.getByRole("row", { name: /cheap openai gpt-4o/i })).toBeVisible();
    await clickWithConfirm(page, "Delete alias cheap?", () => page.getByRole("button", { name: "Delete cheap" }).click());
    await expect(page.getByRole("table", { name: "Model aliases" })).not.toContainText("cheap");

    await navigateTo(page, "Pricing");
    await page.getByRole("textbox", { exact: true, name: "Provider" }).fill("anthropic");
    await page.getByRole("textbox", { exact: true, name: "Model" }).fill("claude-sonnet");
    await page.getByLabel("Input cost per token").fill("0.000003");
    await page.getByLabel("Output cost per token").fill("0.000015");
    await page.getByRole("button", { name: "Create override" }).click();
    await expect(page.getByRole("table", { name: "Pricing overrides" })).toContainText("claude-sonnet");
    await page.getByRole("button", { name: "Edit anthropic claude-sonnet" }).click();
    await page.getByLabel("Input cost per token").fill("0.000004");
    await page.getByLabel("Output cost per token").fill("0.000016");
    await page.getByRole("button", { name: "Update override" }).click();
    await expect(page.getByRole("row", { name: /anthropic claude-sonnet 0.000004 0.000016/i })).toBeVisible();
    await clickWithConfirm(page, "Delete pricing override anthropic/claude-sonnet?", () =>
      page.getByRole("button", { name: "Delete anthropic claude-sonnet" }).click()
    );
    await expect(page.getByRole("table", { name: "Pricing overrides" })).not.toContainText("claude-sonnet");

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
    await page.getByLabel("Callback URL").fill(
      "http://127.0.0.1:5173/api/mcp/oauth/callback?instance_id=mcp-created&code=e2e-code&state=e2e"
    );
    await page.getByRole("button", { name: "Complete OAuth" }).click();
    await expect(page.getByText("OAuth completed for e2e-account")).toBeVisible();
    await page.getByRole("combobox", { name: "Tool" }).selectOption("mcp-1__linear-search");
    await page.getByLabel("Arguments JSON").fill("{\"query\":\"release\"}");
    await page.getByRole("button", { name: "Execute tool" }).click();
    await expect(page.getByText(/linear issue found/i)).toBeVisible();
    await clickWithConfirm(page, "Delete MCP instance github-tools?", () =>
      page.getByRole("button", { name: "Delete github-tools" }).click()
    );
    await expect(page.getByRole("table", { name: "MCP instances" })).not.toContainText("github-tools");

    await navigateTo(page, "MCP Instances");
    await page.getByLabel("Instance name").fill("github-tools");
    await page.getByLabel("Server key").fill("github");
    await page.getByRole("textbox", { exact: true, name: "URL" }).fill("https://mcp.github.example.test");
    await page.getByRole("button", { name: "Create instance" }).click();
    await expect(page.getByRole("table", { name: "MCP instances" })).toContainText("github-tools");

    await navigateTo(page, "MCP Accounts");
    await page.getByRole("combobox", { name: /^Instance/ }).selectOption("mcp-created");
    await page.getByLabel("Authorization URL").fill("https://auth.example.test/authorize");
    await page.getByLabel("Resource URI").fill("https://mcp.github.example.test");
    await page.getByRole("button", { name: "Start OAuth" }).click();
    await expect(page.getByRole("link", { name: "Open authorization URL" })).toHaveAttribute(
      "href",
      "https://auth.example.test/authorize?state=e2e"
    );

    await navigateTo(page, "MCP Tools");
    await page.getByRole("combobox", { name: "Tool" }).selectOption("mcp-1__linear-search");
    await page.getByLabel("Arguments JSON").fill("{\"query\":\"release\"}");
    await page.getByRole("button", { name: "Execute tool" }).click();
    await expect(page.getByText(/linear issue found/i)).toBeVisible();

    await navigateTo(page, "Settings");
    await page.getByLabel("Proxy URL").fill("http://127.0.0.1:9090");
    await page.getByRole("button", { name: "Save settings" }).click();
    await expect(page.getByText("Settings saved")).toBeVisible();
    await expect(page.getByLabel("Proxy URL")).toHaveValue("http://127.0.0.1:9090");

    await navigateTo(page, "Settings/Security");
    await page.getByLabel("Enable request logs").check();
    await page.getByRole("button", { name: "Save settings" }).click();
    await expect(page.getByText("Settings saved")).toBeVisible();
  });

  test("filters, paginates, and searches the logs viewer", async ({ page }) => {
    await mockAPI(page);

    await page.goto("/");
    await navigateTo(page, "Logs");

    const logsTable = page.getByRole("table", { name: "Request logs" });
    await expect(logsTable).toContainText("gpt-5-mini");
    await expect(page.getByText("Showing 1–3 of 3")).toBeVisible();

    await page.getByLabel("Kind").selectOption("server_error");
    await expect(logsTable).toContainText("claude-sonnet-4");
    await expect(logsTable).not.toContainText("gpt-5-mini");
    await expect(page.getByText("Showing 1–1 of 1")).toBeVisible();

    await page.getByLabel("Kind").selectOption("");
    await page.getByLabel("Search logs").fill("llama-3.3-70b");
    await expect(logsTable).toContainText("llama-3.3-70b");
    await expect(logsTable).not.toContainText("gpt-5-mini");
    await expect(page.getByText("Showing 1–1 of 1")).toBeVisible();

    await page.getByLabel("Search logs").fill("");
    await expect(page.getByText("Showing 1–3 of 3")).toBeVisible();
    await expect(page.getByRole("button", { name: "Prev" })).toBeDisabled();
    await expect(page.getByRole("button", { name: "Next" })).toBeDisabled();
  });

  test("persists log retention in settings", async ({ page }) => {
    const apiRequests = await mockAPI(page);

    await page.goto("/");
    await navigateTo(page, "Settings");

    await expect(page.getByLabel("Log retention")).toHaveValue("30");
    await page.getByLabel("Log retention").selectOption("90");
    await page.getByRole("button", { name: "Save settings" }).click();
    await expect(page.getByText("Settings saved")).toBeVisible();

    await expect
      .poll(() => apiRequests.find((request) => request.method === "PUT" && request.path === "/api/settings")?.body)
      .toMatchObject({ log_retention_days: 90 });

    await expect(page.getByLabel("Log retention")).toHaveValue("90");
  });

  test("persists connection-source policy in settings", async ({ page }) => {
    const apiRequests = await mockAPI(page);

    await page.goto("/");
    await navigateTo(page, "Settings");

    await expect(page.getByLabel("Public web")).toBeChecked();
    await page.getByLabel("Public web").uncheck();
    await page.getByRole("button", { name: "Save settings" }).click();
    await expect(page.getByText("Settings saved")).toBeVisible();

    await expect
      .poll(() => apiRequests.find((request) => request.method === "PUT" && request.path === "/api/settings")?.body)
      .toMatchObject({ allowed_sources: ["local", "lan", "tailscale"] });
  });

  test("shows the API-key-required notice and usage attribution columns", async ({ page }) => {
    await mockAPI(page);

    await page.goto("/");
    await navigateTo(page, "API Keys");
    await expect(page.getByText("An API key is required to call the proxy")).toBeVisible();

    await navigateTo(page, "Usage");
    await expect(page.getByRole("heading", { exact: true, name: "Usage" })).toBeVisible();
    const usageTable = page.getByRole("table", { name: "Usage rows" });
    await expect(usageTable).toContainText("desktop-client");
    await expect(usageTable).toContainText("ops@example.test");
    await expect(usageTable.getByText("oauth").first()).toBeVisible();

    await page.getByLabel("Filter by auth type").selectOption("oauth");
    await expect(usageTable).toContainText("desktop-client");
  });

  test("creates MCP instances with advanced launch fields", async ({ page }) => {
    const apiRequests = await mockAPI(page);

    await page.goto("/");
    await navigateTo(page, "MCP Instances");
    await page.getByLabel("Instance name").fill("filesystem-tools");
    await page.getByLabel("Server key").fill("filesystem");
    await page.getByRole("combobox", { name: "Launch type" }).selectOption("command");
    await page.getByRole("textbox", { exact: true, name: "Command" }).fill("node");
    await page.getByLabel("Args JSON").fill("[\"server.js\",\"--stdio\"]");
    await page.getByLabel("Headers JSON").fill("{\"Authorization\":\"Bearer e2e-secret\"}");
    await page.getByLabel("Env JSON").fill("{\"API_KEY\":\"e2e-env-secret\"}");
    await page.getByLabel("Working directory").fill("/srv/mcp");
    await page.getByRole("button", { name: "Create instance" }).click();

    await expect.poll(() => apiRequests.find((request) => request.method === "POST" && request.path === "/api/mcp/instances")?.body).toMatchObject({
      args: ["server.js", "--stdio"],
      command: "node",
      cwd: "/srv/mcp",
      env: { API_KEY: "e2e-env-secret" },
      headers: { Authorization: "Bearer e2e-secret" },
      launch_type: "command",
      name: "filesystem-tools",
      server_key: "filesystem",
      transport: "stdio"
    });
    await expect(page.getByRole("table", { name: "MCP instances" })).toContainText("filesystem-tools");
    await expect(page.getByText("e2e-secret")).not.toBeVisible();
    await expect(page.getByText("e2e-env-secret")).not.toBeVisible();
  });

  test("starts MCP OAuth with Resource URI discovery and blank Authorization URL", async ({ page }) => {
    const apiRequests = await mockAPI(page);

    await page.goto("/");
    await navigateTo(page, "MCP Accounts");
    await page.getByRole("combobox", { name: /^Instance/ }).selectOption("mcp-1");
    await expect(page.getByLabel("Authorization URL")).not.toHaveAttribute("required", "");
    await page.getByLabel("Resource URI").fill("https://mcp.discovery.example.test");
    await page.getByRole("button", { name: "Start OAuth" }).click();

    await expect(page.getByRole("link", { name: "Open authorization URL" })).toHaveAttribute(
      "href",
      "https://auth.example.test/authorize?state=e2e"
    );
    await expect
      .poll(() => apiRequests.find((request) => request.method === "POST" && request.path === "/api/mcp/instances/mcp-1/auth/start")?.body)
      .toMatchObject({
        authorization_url: "",
        resource_uri: "https://mcp.discovery.example.test"
      });
  });

  test("connects provider OAuth on Connections/Auth with mocked exchange", async ({ page }) => {
    const apiRequests = await mockAPI(page);

    await page.goto("/");
    await navigateTo(page, "Connections/Auth");
    await page.getByRole("combobox", { name: "OAuth provider" }).selectOption("openai");
    await page.getByLabel("OAuth account label").fill("OpenAI OAuth e2e");
    await page.getByRole("button", { name: "Start OAuth" }).click();
    await expect(page.getByRole("link", { name: "Open authorization URL" })).toHaveAttribute(
      "href",
      "https://auth.openai.example.test/authorize?state=provider-oauth-e2e"
    );

    await page
      .getByLabel("Callback URL or code")
      .fill("http://127.0.0.1:5173/api/oauth/callback?provider=openai&state=provider-oauth-e2e&code=provider-e2e-code");
    await page.getByRole("button", { name: "Complete OAuth" }).click();

    await expect(page.getByText("OAuth connected OpenAI OAuth e2e")).toBeVisible();
    await expect(page.getByRole("table", { name: "Provider connections" })).toContainText("OpenAI OAuth e2e");
    await expect(page.getByText("provider-e2e-access")).not.toBeVisible();
    await expect(page.getByText("provider-e2e-refresh")).not.toBeVisible();
    await expect
      .poll(() => apiRequests.find((request) => request.method === "POST" && request.path === "/api/oauth/openai/authorize")?.body)
      .toMatchObject({ account_label: "OpenAI OAuth e2e" });
    await expect
      .poll(() => apiRequests.find((request) => request.method === "POST" && request.path === "/api/oauth/openai/exchange")?.body)
      .toMatchObject({ state: "provider-oauth-e2e", code: "provider-e2e-code" });
  });

  test("connects provider OAuth on Connections/Auth with mocked polling", async ({ page }) => {
    const apiRequests = await mockAPI(page);

    await page.goto("/");
    await navigateTo(page, "Connections/Auth");
    await page.getByRole("combobox", { name: "OAuth provider" }).selectOption("cursor");
    await page.getByLabel("OAuth account label").fill("Cursor OAuth e2e");
    await page.getByRole("button", { name: "Start OAuth" }).click();
    await expect(page.getByRole("link", { name: "Open authorization URL" })).toHaveAttribute(
      "href",
      "https://cursor.example.test/loginDeepControl?uuid=cursor-oauth-e2e"
    );
    await expect(page.getByText("Device code: cursor-oauth-e2e")).toBeVisible();
    await expect(page.getByText("Poll interval: 1s")).toBeVisible();

    await page.getByRole("button", { name: "Poll OAuth" }).click();

    await expect(page.getByText("OAuth connected Cursor OAuth e2e")).toBeVisible();
    await expect(page.getByRole("table", { name: "Provider connections" })).toContainText("Cursor OAuth e2e");
    await expect(page.getByText("cursor-e2e-access")).not.toBeVisible();
    await expect(page.getByText("cursor-e2e-refresh")).not.toBeVisible();
    await expect
      .poll(() => apiRequests.find((request) => request.method === "POST" && request.path === "/api/oauth/cursor/authorize")?.body)
      .toMatchObject({ account_label: "Cursor OAuth e2e" });
    expect(apiRequests.some((request) => request.method === "GET" && request.path === "/api/oauth/cursor/poll")).toBe(true);
  });

  test("handles endpoint copy, destructive cancellation, and mutation failure states", async ({ page, context }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"], { origin: "http://127.0.0.1:5173" });
    await mockAPI(page);

    await page.goto("/");
    await navigateTo(page, "Endpoint Setup");
    await page.getByRole("button", { name: "Copy chat completions endpoint" }).click();
    await expect(page.getByText("Endpoint copied")).toBeVisible();
    await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(
      `${new URL(page.url()).origin}/v1/chat/completions`
    );

    await page.getByLabel("Key name").fill("temporary-client");
    await page.getByRole("button", { name: "Create key" }).click();
    await page.getByRole("button", { name: "Dismiss" }).click();
    await clickWithConfirm(page, "Delete API key temporary-client?", () =>
      page.getByRole("button", { name: "Delete temporary-client" }).click(), "dismiss"
    );
    await expect(page.getByRole("table", { name: "API keys" })).toContainText("temporary-client");

    await mockAPI(page, { mode: "mutation-failure" });
    await page.goto("/");

    await navigateTo(page, "Endpoint Setup");
    await page.getByLabel("Key name").fill("broken-client");
    await page.getByRole("button", { name: "Create key" }).click();
    await expect(page.getByText("API key action failed")).toBeVisible();
    await expect(page.getByText("forced mutation failure")).toBeVisible();

    await navigateTo(page, "Aliases");
    await page.getByRole("textbox", { exact: true, name: "Alias" }).fill("broken");
    await page.getByRole("textbox", { exact: true, name: "Provider" }).fill("openai");
    await page.getByRole("textbox", { exact: true, name: "Model" }).fill("gpt-5-mini");
    await page.getByRole("button", { name: "Create alias" }).click();
    await expect(page.getByText("Could not change alias")).toBeVisible();

    await navigateTo(page, "Combos/Routing");
    await page.getByLabel("Combo name").fill("broken-combo");
    await page.getByLabel("Step 1 provider").fill("openai");
    await page.getByLabel("Step 1 model").fill("gpt-5-mini");
    await page.getByRole("button", { name: "Create combo" }).click();
    await expect(page.getByText("Could not change combo")).toBeVisible();

    await navigateTo(page, "Pricing");
    await page.getByRole("textbox", { exact: true, name: "Provider" }).fill("openai");
    await page.getByRole("textbox", { exact: true, name: "Model" }).fill("gpt-5-mini");
    await page.getByLabel("Input cost per token").fill("0.000001");
    await page.getByLabel("Output cost per token").fill("0.000002");
    await page.getByRole("button", { name: "Create override" }).click();
    await expect(page.getByText("Could not change pricing")).toBeVisible();

    await navigateTo(page, "Providers");
    await page.getByLabel("Connection name").fill("broken-provider");
    await page.getByLabel("Provider API key").fill("broken-provider-secret");
    await page.getByRole("button", { name: "Add connection" }).click();
    await expect(page.getByText("forced mutation failure")).toBeVisible();
    await expect(page.getByText("broken-provider-secret")).not.toBeVisible();

    await navigateTo(page, "MCP");
    await page.getByLabel("Instance name").fill("broken-mcp");
    await page.getByLabel("Server key").fill("broken");
    await page.getByRole("textbox", { exact: true, name: "URL" }).fill("https://mcp.example.test");
    await page.getByRole("button", { name: "Create instance" }).click();
    await expect(page.getByText("forced mutation failure")).toBeVisible();

    await navigateTo(page, "Settings");
    await page.getByLabel("Proxy URL").fill("http://127.0.0.1:9091");
    await page.getByRole("button", { name: "Save settings" }).click();
    await expect(page.getByText("Could not save settings")).toBeVisible();

    await navigateTo(page, "Settings/Security");
    await page.getByLabel("Enable request logs").check();
    await page.getByRole("button", { name: "Save settings" }).click();
    await expect(page.getByText("Could not save settings")).toBeVisible();
  });

  test("renders empty states for every dashboard section with mocked API data", async ({ page }) => {
    await mockAPI(page, { mode: "empty" });

    await page.goto("/");
    await expect(page.getByText("No overview data yet")).toBeVisible();

    await navigateTo(page, "Endpoint Setup");
    await expect(page.getByText("No API keys")).toBeVisible();

    await navigateTo(page, "API Keys");
    await expect(page.getByText("No API keys")).toBeVisible();

    await navigateTo(page, "Providers");
    await expect(page.getByText("No provider records")).toBeVisible();

    await navigateTo(page, "Aliases");
    await expect(page.getByText("No model aliases")).toBeVisible();

    await navigateTo(page, "Models");
    await expect(page.getByText("No model-capable providers")).toBeVisible();

    await navigateTo(page, "Pricing");
    await expect(page.getByText("No pricing overrides")).toBeVisible();

    await navigateTo(page, "Usage");
    await expect(page.getByText("No usage or logs yet")).toBeVisible();

    await navigateTo(page, "Logs");
    await expect(page.getByText("No request logs")).toBeVisible();

    await navigateTo(page, "Quotas");
    await expect(page.getByText("No quota-capable providers")).toBeVisible();

    await navigateTo(page, "Combos/Routing");
    await expect(page.getByText("No combo routes configured")).toBeVisible();

    await navigateTo(page, "MCP");
    await expect(page.getByText("No MCP data")).toBeVisible();

    await navigateTo(page, "MCP Instances");
    await expect(page.getByText("No MCP data")).toBeVisible();

    await navigateTo(page, "MCP Accounts");
    await expect(page.getByText("No MCP data")).toBeVisible();

    await navigateTo(page, "MCP Tools");
    await expect(page.getByText("No MCP data")).toBeVisible();

    await navigateTo(page, "Settings");
    await expect(page.getByText("No runtime settings returned")).toBeVisible();

    await navigateTo(page, "Settings/Security");
    await expect(page.getByText("No runtime settings returned")).toBeVisible();

    await navigateTo(page, "Diagnostics");
    await expect(page.getByText("No diagnostics data")).toBeVisible();
  });

  test("renders auth-expired states from protected mocked API responses", async ({ page }) => {
    await mockAPI(page, { mode: "auth-expired" });

    await page.goto("/");
    await expect(page.getByText("Session expired")).toBeVisible();

    await navigateTo(page, "Endpoint Setup");
    await expect(page.getByText("Authentication expired")).toBeVisible();

    await navigateTo(page, "API Keys");
    await expect(page.getByText("Authentication expired")).toBeVisible();

    await navigateTo(page, "Providers");
    await expect(page.getByText("Authentication expired")).toBeVisible();

    await navigateTo(page, "Aliases");
    await expect(page.getByText("Session expired")).toBeVisible();

    await navigateTo(page, "Models");
    await expect(page.getByText("Session expired")).toBeVisible();

    await navigateTo(page, "Pricing");
    await expect(page.getByText("Session expired")).toBeVisible();

    await navigateTo(page, "Usage");
    await expect(page.getByText("Session expired")).toBeVisible();

    await navigateTo(page, "Logs");
    await expect(page.getByText("Session expired")).toBeVisible();

    await navigateTo(page, "Quotas");
    await expect(page.getByText("Session expired")).toBeVisible();

    await navigateTo(page, "Combos/Routing");
    await expect(page.getByText("Session expired")).toBeVisible();

    await navigateTo(page, "MCP");
    await expect(page.getByText("MCP session expired")).toBeVisible();

    await navigateTo(page, "MCP Instances");
    await expect(page.getByText("MCP session expired")).toBeVisible();

    await navigateTo(page, "MCP Accounts");
    await expect(page.getByText("MCP session expired")).toBeVisible();

    await navigateTo(page, "MCP Tools");
    await expect(page.getByText("MCP session expired")).toBeVisible();

    await navigateTo(page, "Settings/Security");
    await expect(page.getByText("Session expired")).toBeVisible();

    await navigateTo(page, "Diagnostics");
    await expect(page.getByText("Session expired")).toBeVisible();
  });
});

async function navigateTo(page: Page, label: string) {
  await page.getByRole("button", { exact: true, name: label }).click();
}

type MockMode = "normal" | "empty" | "auth-expired" | "mutation-failure";

async function mockAPI(page: Page, options: { mode?: MockMode } = {}) {
  const apiRequests: RecordedAPIRequest[] = [];
  const mode = options.mode ?? "normal";
  const state = {
    apiKeys: [...apiKeys],
    aliases: [...aliases],
    connections: [...connections],
    combos: [...combos],
    mcpAccountsByInstance: { "mcp-1": [...mcpAccounts], "mcp-created": [] },
    mcpInstances: [...mcpInstances],
    pricing: [...pricing],
    settings: { ...settings }
  };

  await page.unroute("**/*").catch(() => undefined);
  await page.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());

    if (url.origin === "http://127.0.0.1:5173" && url.pathname.startsWith("/api/")) {
      const body = parseRequestBody(request);
      apiRequests.push(recordAPIRequest(request, url, body));
      const response = apiResponse(state, url.pathname, request.method(), body, mode, url.searchParams);
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

function recordAPIRequest(request: Request, url: URL, body: unknown): RecordedAPIRequest {
  return {
    authorization: request.headers().authorization ?? null,
    body,
    method: request.method(),
    path: url.pathname
  };
}

function parseRequestBody(request: Request): unknown {
  const postData = request.postData();
  if (!postData) {
    return undefined;
  }
  return request.postDataJSON();
}

function apiResponse(state: MockAPIState, path: string, method: string, body: unknown, mode: MockMode, params?: URLSearchParams): MockAPIResponse {
  if (mode === "auth-expired") {
    return { status: 401, body: { error: "control-plane auth required" } };
  }

  if (mode === "mutation-failure" && method !== "GET") {
    return { status: 500, body: { error: "forced mutation failure" } };
  }

  if (mode === "empty" && method === "GET") {
    return emptyAPIResponse(path);
  }

  if (method === "GET" && path === "/api/logs") {
    return logsResponse(params ?? new URLSearchParams());
  }

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

  if (method === "POST" && path === "/api/connections") {
    const request = body as { auth_type?: string; is_active?: boolean; name?: string; provider?: string };
    const connection = {
      ID: "conn-created",
      Provider: request.provider ?? "openai",
      Name: request.name ?? "created-provider",
      AuthType: request.auth_type ?? "api_key",
      IsActive: request.is_active ?? true,
      AccountID: null,
      Email: null,
      BackoffLevel: 0,
      CreatedAt: "2026-06-03T11:00:00Z",
      UpdatedAt: "2026-06-03T11:00:00Z"
    };
    state.connections = [...state.connections, connection];
    return { status: 201, body: connection };
  }

  if (method === "POST" && path === "/api/connections/conn-created/test") {
    return { status: 200, body: { ok: true, provider: "openai", name: "OpenAI e2e" } };
  }

  if (method === "PUT" && path.startsWith("/api/connections/")) {
    const id = decodeURIComponent(path.slice("/api/connections/".length));
    const request = body as { is_active?: boolean };
    state.connections = state.connections.map((connection) =>
      connection.ID === id ? { ...connection, IsActive: request.is_active ?? connection.IsActive } : connection
    );
    const connection = state.connections.find((entry) => entry.ID === id);
    return connection ? { status: 200, body: connection } : { status: 404, body: { error: "not found" } };
  }

  if (method === "POST" && path === "/api/oauth/openai/authorize") {
    return {
      status: 200,
      body: {
        provider: "openai",
        auth_url: "https://auth.openai.example.test/authorize?state=provider-oauth-e2e",
        session_id: "provider-oauth-e2e",
        user_code: "OPENAI-E2E",
        verification: "https://auth.openai.example.test/device"
      }
    };
  }

  if (method === "POST" && path === "/api/oauth/openai/exchange") {
    const request = body as { state?: string };
    const connection = {
      ID: "conn-provider-oauth",
      Provider: "openai",
      Name: "OpenAI OAuth e2e",
      AuthType: "oauth",
      IsActive: true,
      AccountID: null,
      Email: null,
      BackoffLevel: 0,
      CreatedAt: "2026-06-04T11:00:00Z",
      UpdatedAt: "2026-06-04T11:00:00Z",
      AccessToken: "provider-e2e-access",
      RefreshToken: "provider-e2e-refresh"
    };
    if (request.state === "provider-oauth-e2e") {
      state.connections = [...state.connections, connection];
    }
    return {
      status: 200,
      body: {
        id: connection.ID,
        provider: connection.Provider,
        name: connection.Name,
        auth_type: connection.AuthType,
        scopes: ["read"]
      }
    };
  }

  if (method === "POST" && path === "/api/oauth/cursor/authorize") {
    return {
      status: 200,
      body: {
        provider: "cursor",
        auth_url: "https://cursor.example.test/loginDeepControl?uuid=cursor-oauth-e2e",
        session_id: "cursor-oauth-e2e",
        user_code: "cursor-oauth-e2e",
        verification: "https://cursor.example.test/loginDeepControl?uuid=cursor-oauth-e2e",
        poll_interval: 1
      }
    };
  }

  if (method === "GET" && path === "/api/oauth/cursor/poll") {
    const connection = {
      ID: "conn-provider-cursor-oauth",
      Provider: "cursor",
      Name: "Cursor OAuth e2e",
      AuthType: "oauth",
      IsActive: true,
      AccountID: null,
      Email: null,
      BackoffLevel: 0,
      CreatedAt: "2026-06-04T11:00:00Z",
      UpdatedAt: "2026-06-04T11:00:00Z",
      AccessToken: "cursor-e2e-access",
      RefreshToken: "cursor-e2e-refresh"
    };
    state.connections = [...state.connections, connection];
    return {
      status: 200,
      body: {
        status: "complete",
        connection: {
          id: connection.ID,
          provider: connection.Provider,
          name: connection.Name,
          auth_type: connection.AuthType
        }
      }
    };
  }

  if (method === "DELETE" && path.startsWith("/api/connections/")) {
    const id = decodeURIComponent(path.slice("/api/connections/".length));
    state.connections = state.connections.filter((connection) => connection.ID !== id);
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

  if (method === "PUT" && path.startsWith("/api/combos/")) {
    const id = decodeURIComponent(path.slice("/api/combos/".length));
    const request = body as { is_active?: boolean; name?: string; steps?: Array<{ model: string; provider: string }> };
    state.combos = state.combos.map((combo) =>
      combo.ID === id
        ? {
            ...combo,
            Name: request.name ?? combo.Name,
            Steps: request.steps ?? combo.Steps,
            IsActive: request.is_active ?? combo.IsActive,
            UpdatedAt: "2026-06-03T11:30:00Z"
          }
        : combo
    );
    const combo = state.combos.find((entry) => entry.ID === id);
    return combo ? { status: 200, body: combo } : { status: 404, body: { error: "not found" } };
  }

  if (method === "POST" && path === "/api/aliases") {
    const request = body as { alias?: string; model?: string; provider?: string };
    const alias = { Alias: request.alias ?? "created", Provider: request.provider ?? "openai", Model: request.model ?? "gpt-5-mini" };
    state.aliases = [...state.aliases, alias];
    return { status: 201, body: alias };
  }

  if (method === "PUT" && path.startsWith("/api/aliases/")) {
    const aliasID = decodeURIComponent(path.slice("/api/aliases/".length));
    const request = body as { model?: string; provider?: string };
    state.aliases = state.aliases.map((alias) =>
      alias.Alias === aliasID
        ? { ...alias, Provider: request.provider ?? alias.Provider, Model: request.model ?? alias.Model }
        : alias
    );
    const alias = state.aliases.find((entry) => entry.Alias === aliasID);
    return alias ? { status: 200, body: alias } : { status: 404, body: { error: "not found" } };
  }

  if (method === "DELETE" && path.startsWith("/api/aliases/")) {
    const aliasID = decodeURIComponent(path.slice("/api/aliases/".length));
    state.aliases = state.aliases.filter((alias) => alias.Alias !== aliasID);
    return { status: 204, body: undefined };
  }

  if (method === "POST" && path === "/api/pricing") {
    const request = body as {
      input_cost_per_token?: number;
      model?: string;
      output_cost_per_token?: number;
      provider?: string;
    };
    const override = {
      Provider: request.provider ?? "openai",
      Model: request.model ?? "gpt-5-mini",
      InputCostPerToken: request.input_cost_per_token ?? 0,
      OutputCostPerToken: request.output_cost_per_token ?? 0
    };
    state.pricing = [...state.pricing, override];
    return { status: 201, body: override };
  }

  if (method === "PUT" && path.startsWith("/api/pricing/")) {
    const [provider, model] = path.slice("/api/pricing/".length).split("/").map(decodeURIComponent);
    const request = body as { input_cost_per_token?: number; output_cost_per_token?: number };
    state.pricing = state.pricing.map((override) =>
      override.Provider === provider && override.Model === model
        ? {
            ...override,
            InputCostPerToken: request.input_cost_per_token ?? override.InputCostPerToken,
            OutputCostPerToken: request.output_cost_per_token ?? override.OutputCostPerToken
          }
        : override
    );
    const override = state.pricing.find((entry) => entry.Provider === provider && entry.Model === model);
    return override ? { status: 200, body: override } : { status: 404, body: { error: "not found" } };
  }

  if (method === "DELETE" && path.startsWith("/api/pricing/")) {
    const [provider, model] = path.slice("/api/pricing/".length).split("/").map(decodeURIComponent);
    state.pricing = state.pricing.filter((override) => override.Provider !== provider || override.Model !== model);
    return { status: 204, body: undefined };
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
      args?: string[];
      cwd?: string;
      env?: Record<string, string>;
      headers?: Record<string, string>;
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
      Args: request.args ?? [],
      Command: request.command ?? null,
      Headers: request.headers ?? {},
      Env: request.env ?? {},
      CWD: request.cwd ?? null,
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

  if (method === "POST" && path === "/api/mcp/instances/mcp-created/oauth/complete") {
    const account = {
      id: "acct-created",
      instance_id: "mcp-created",
      account_label: "e2e-account",
      email: "e2e@example.test",
      resource_uri: "https://mcp.github.example.test"
    };
    state.mcpAccountsByInstance["mcp-created"] = [account];
    return { status: 200, body: account };
  }

  if (method === "POST" && path === "/api/mcp/tools/mcp-1__linear-search/execute") {
    return { status: 200, body: { content: [{ type: "text", text: "linear issue found" }] } };
  }

  if (method === "DELETE" && path.startsWith("/api/mcp/instances/")) {
    const id = decodeURIComponent(path.slice("/api/mcp/instances/".length));
    state.mcpInstances = state.mcpInstances.filter((instance) => instance.ID !== id);
    delete state.mcpAccountsByInstance[id];
    return { status: 204, body: undefined };
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
    case "/api/providers/openai/models":
      return { status: 200, body: { data: providerModels.openai } };
    case "/api/providers/anthropic/models":
      return { status: 200, body: { data: providerModels.anthropic } };
    case "/api/connections":
      return { status: 200, body: { data: state.connections } };
    case "/api/keys":
      return { status: 200, body: { data: state.apiKeys } };
    case "/api/aliases":
      return { status: 200, body: { data: state.aliases } };
    case "/api/pricing":
      return { status: 200, body: { data: state.pricing } };
    case "/api/usage":
    case "/api/logs":
      return { status: 200, body: { object: "list", data: usageRows, limit: 25, offset: 0, total: usageRows.length } };
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
      return { status: 200, body: { data: state.mcpAccountsByInstance["mcp-1"] ?? [] } };
    case "/api/mcp/instances/mcp-created/accounts":
      return { status: 200, body: { data: state.mcpAccountsByInstance["mcp-created"] ?? [] } };
    case "/api/mcp/tools":
      return { status: 200, body: { data: mcpTools } };
    case "/api/settings":
      return { status: 200, body: state.settings };
    default:
      return { status: 404, body: { error: `No E2E API fixture for ${path}` } };
  }
}

async function clickWithConfirm(page: Page, message: string, click: () => Promise<unknown>, action: "accept" | "dismiss" = "accept") {
  const dialogPromise = page.waitForEvent("dialog");
  const clickPromise = click();
  const dialog = await dialogPromise;
  expect(dialog.message()).toBe(message);
  if (action === "dismiss") {
    await dialog.dismiss();
  } else {
    await dialog.accept();
  }
  await clickPromise;
}

function emptyAPIResponse(path: string): MockAPIResponse {
  switch (path) {
    case "/api/providers":
    case "/api/providers/openai/models":
    case "/api/providers/anthropic/models":
    case "/api/connections":
    case "/api/keys":
    case "/api/aliases":
    case "/api/pricing":
    case "/api/combos":
    case "/api/mcp/clients":
    case "/api/mcp/instances":
    case "/api/mcp/instances/mcp-1/accounts":
    case "/api/mcp/instances/mcp-created/accounts":
    case "/api/mcp/tools":
      return { status: 200, body: { data: [] } };
    case "/api/usage":
    case "/api/logs":
      return { status: 200, body: { object: "list", data: [], limit: 25, offset: 0, total: 0 } };
    case "/api/usage/summary":
      return { status: 200, body: { request_count: 0, total_tokens: 0, total_cost_usd: 0 } };
    case "/api/settings":
      return { status: 200, body: null };
    default:
      return { status: 404, body: { error: `No empty E2E API fixture for ${path}` } };
  }
}

type MockAPIState = {
  apiKeys: typeof apiKeys;
  aliases: typeof aliases;
  connections: typeof connections;
  combos: typeof combos;
  mcpAccountsByInstance: Record<string, typeof mcpAccounts>;
  mcpInstances: typeof mcpInstances;
  pricing: typeof pricing;
  settings: typeof settings;
};

type MockAPIResponse = {
  body: unknown;
  status: number;
};

const providers = [
  {
    id: "openai",
    auth_types: ["oauth", "api_key"],
    refresh: false,
    registered_adapter: true,
    public_inference: true,
    direct_dispatch: true,
    inference: true,
    streaming: true,
    model_catalog: true,
    list_models: true,
    quota: false,
    public_status: "supported",
    notes: "E2E fixture"
  },
  {
    id: "cursor",
    auth_types: ["oauth"],
    refresh: true,
    registered_adapter: false,
    public_inference: false,
    direct_dispatch: false,
    inference: false,
    streaming: false,
    model_catalog: false,
    list_models: false,
    quota: false,
    public_status: "auth_only",
    notes: "E2E fixture"
  },
  {
    id: "anthropic",
    auth_types: ["api_key"],
    refresh: false,
    registered_adapter: true,
    public_inference: true,
    direct_dispatch: true,
    inference: true,
    streaming: true,
    model_catalog: true,
    list_models: true,
    quota: false,
    public_status: "supported",
    notes: "E2E fixture"
  }
];

const providerModels = {
  anthropic: [{ id: "claude-sonnet-4", object: "model", created: 0, owned_by: "anthropic" }],
  openai: [{ id: "gpt-4o", object: "model", created: 0, owned_by: "openai" }]
};

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

const aliases = [
  {
    Alias: "fast",
    Provider: "openai",
    Model: "gpt-5-mini"
  }
];

const pricing = [
  {
    Provider: "openai",
    Model: "gpt-5-mini",
    InputCostPerToken: 0.000001,
    OutputCostPerToken: 0.000002
  }
];

const usageRows = [
  {
    id: 1,
    request_id: "req_001",
    timestamp: "2026-06-03T10:00:00Z",
    provider: "openai",
    model: "gpt-5-mini",
    auth_type: "oauth",
    api_key_id: "key-1",
    api_key_name: "desktop-client",
    connection_provider: "openai",
    account_email: "ops@example.test",
    total_tokens: 1250,
    cost_usd: 0.034,
    latency_ms: 320,
    status_code: 200,
    source_format: "chat",
    target_format: "chat",
    client_tool: "codex"
  }
];

const logRecords = [
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
    source_format: "openai",
    target_format: "openai",
    client_tool: "codex",
    combo_name: "research-stack"
  },
  {
    id: 2,
    request_id: "req_500_oops",
    timestamp: "2026-06-03T10:05:00Z",
    provider: "anthropic",
    model: "claude-sonnet-4",
    auth_type: "oauth",
    total_tokens: 80,
    cost_usd: 0.002,
    latency_ms: 900,
    status_code: 500,
    error: "upstream exploded",
    source_format: "anthropic",
    target_format: "anthropic",
    client_tool: "claude-code",
    combo_name: "research-stack"
  },
  {
    id: 3,
    request_id: "req_paginated",
    timestamp: "2026-06-03T10:10:00Z",
    provider: "groq",
    model: "llama-3.3-70b",
    auth_type: "api_key",
    total_tokens: 12,
    cost_usd: 0.0001,
    latency_ms: 40,
    status_code: 200,
    source_format: "openai",
    target_format: "openai",
    client_tool: "curl",
    combo_name: "research-stack"
  }
];

function logsResponse(params: URLSearchParams): MockAPIResponse {
  const limit = Number(params.get("limit") ?? "50");
  const offset = Number(params.get("offset") ?? "0");
  const statusClass = params.get("status_class") ?? "";
  const provider = params.get("provider") ?? "";
  const search = params.get("search") ?? "";
  let filtered = logRecords;
  if (provider) {
    filtered = filtered.filter((row) => row.provider.includes(provider));
  }
  if (statusClass === "success") {
    filtered = filtered.filter((row) => row.status_code < 400);
  } else if (statusClass === "client_error") {
    filtered = filtered.filter((row) => row.status_code >= 400 && row.status_code < 500);
  } else if (statusClass === "server_error") {
    filtered = filtered.filter((row) => row.status_code >= 500);
  }
  if (search) {
    filtered = filtered.filter((row) => row.request_id.includes(search) || row.model.includes(search));
  }
  const total = filtered.length;
  const page = limit > 0 ? filtered.slice(offset, offset + limit) : filtered;
  return { status: 200, body: { object: "list", data: page, limit, offset, total } };
}

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
  require_api_key: true,
  rtk_enabled: true,
  caveman_enabled: false,
  caveman_level: "full",
  enable_request_logs: true,
  proxy_url: "http://127.0.0.1:8080",
  data_dir: "/var/lib/g0router",
  log_retention_days: 30,
  allowed_sources: ["local", "lan", "tailscale", "public"]
};
