import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Console", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("console page loads", async ({ page }) => {
    await page.goto("/console");
    await expect(page.locator("body")).toContainText("Console", { timeout: 10000 });
  });

  test("a fixture-driven log line appears", async ({ page }) => {
    await page.goto("/console");
    // The fixture MockEventSource pushes one of these synthetic lines every
    // 2500ms over /api/console-logs/stream (fixture.ts:78-97).
    const row = page.locator("[data-testid='console-log-row']").first();
    await expect(row).toBeVisible({ timeout: 15000 });
  });

  test("each log row shows a level badge", async ({ page }) => {
    await page.goto("/console");
    const badge = page
      .locator("[data-testid='console-log-row']")
      .first()
      .locator("[data-testid='console-log-level']");
    await expect(badge).toBeVisible({ timeout: 15000 });
  });
});
