import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Feature Flags", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("feature flags page loads", async ({ page }) => {
    await page.goto("/feature-flags");
    await expect(page.locator("body")).toContainText("Feature Flags", { timeout: 10000 });
  });

  test("feature flag rows render from seed", async ({ page }) => {
    await page.goto("/feature-flags");
    const rows = page.locator('[data-testid="feature-flag-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    expect(await rows.count()).toBeGreaterThanOrEqual(3);
    await expect(page.locator("body")).toContainText("mcp_gateway");
    await expect(page.locator("body")).toContainText("Enable MCP gateway");
  });

  test("toggling a flag fires a PUT", async ({ page }) => {
    await page.goto("/feature-flags");
    await expect(
      page.locator('[data-testid="feature-flag-row"]').first()
    ).toBeVisible({ timeout: 10000 });
    const putPromise = page.waitForRequest(
      (req) => /\/api\/feature-flags\/\d+$/.test(req.url()) && req.method() === "PUT"
    );
    await page
      .locator('[data-testid="feature-flag-row"] button[role="switch"]')
      .first()
      .click();
    await putPromise;
  });
});
