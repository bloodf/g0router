import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Endpoint", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("page loads and shows the endpoint heading", async ({ page }) => {
    await page.goto("/endpoint");
    await expect(page.locator("body")).toContainText("Endpoint", { timeout: 10000 });
  });

  test("base-url block is visible (origin + /v1)", async ({ page }) => {
    await page.goto("/endpoint");
    const block = page.locator('[data-testid="base-url"]');
    await expect(block).toBeVisible({ timeout: 10000 });
    await expect(block).toContainText("/v1");
  });

  test("embedded keys widget renders a seeded key", async ({ page }) => {
    await page.goto("/endpoint");
    const rows = page.locator('[data-testid="api-key-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator("body")).toContainText("Default Key");
  });

  test("provider-node modal opens", async ({ page }) => {
    await page.goto("/endpoint");
    await page.locator('[data-testid="add-node-trigger"]').click();
    await expect(page.locator('[data-testid="modal-traffic-lights"]')).toBeVisible();
  });
});
