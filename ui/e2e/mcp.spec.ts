import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("MCP", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("MCP clients page loads", async ({ page }) => {
    await page.goto("/mcp");
    await expect(page.locator("body")).toContainText("MCP", { timeout: 10000 });
  });

  test("MCP tools page loads", async ({ page }) => {
    await page.goto("/mcp/tools");
    await expect(page.locator("body")).toContainText("Tools", { timeout: 10000 });
  });

  test("MCP page lists clients and instances from seed", async ({ page }) => {
    await page.goto("/mcp");
    await expect(page.getByTestId("mcp-instance-row").first()).toBeVisible({
      timeout: 10000,
    });
    await expect(page.locator("body")).toContainText("Filesystem Instance");
    // transport badge present on instance rows
    await expect(page.getByTestId("mcp-instance-row").first()).toContainText(
      "stdio"
    );
  });

  test("MCP marketplace opens and installs an instance", async ({ page }) => {
    await page.goto("/mcp");
    await page.getByTestId("mcp-marketplace-open").click();
    await expect(page.getByTestId("modal-traffic-lights")).toBeVisible();
    // marketplace browses /api/mcp/clients -> Filesystem + GitHub servers
    await expect(page.locator("body")).toContainText("GitHub");
    const installPost = page.waitForRequest(
      (req) =>
        req.url().includes("/api/mcp/instances") && req.method() === "POST"
    );
    await page.getByTestId("mcp-marketplace-install").first().click();
    await installPost;
  });

  test("MCP tools page lists tools with execute action", async ({ page }) => {
    await page.goto("/mcp/tools");
    await expect(page.getByTestId("mcp-tool-row").first()).toBeVisible({
      timeout: 10000,
    });
    await expect(page.locator("body")).toContainText("read_file");
    await expect(page.locator("body")).toContainText("write_file");
    const execPost = page.waitForRequest(
      (req) =>
        /\/api\/mcp\/tools\/[^/]+\/execute$/.test(req.url()) &&
        req.method() === "POST"
    );
    await page.getByTestId("mcp-tool-execute").first().click();
    await execPost;
    await expect(page.locator("body")).toContainText("Mock execution result");
  });

  test("MCP tools page lists tool-groups with toggle and create", async ({
    page,
  }) => {
    await page.goto("/mcp/tools");
    await expect(page.getByTestId("mcp-tool-group-row").first()).toBeVisible({
      timeout: 10000,
    });
    await expect(page.locator("body")).toContainText("File Operations");
    // New tool-group opens the modal
    await page.getByTestId("mcp-tool-group-new").click();
    await expect(page.getByTestId("modal-traffic-lights")).toBeVisible();
  });
});
