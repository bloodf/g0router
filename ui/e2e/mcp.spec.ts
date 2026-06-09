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
});
