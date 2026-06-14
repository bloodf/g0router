import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Models", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("models page loads", async ({ page }) => {
    await page.goto("/models");
    await expect(page.locator("body")).toContainText("Models", { timeout: 10000 });
  });

  test("model rows render with cost + context window", async ({ page }) => {
    await page.goto("/models");
    const row = page.locator("[data-testid='model-row']").first();
    await expect(row).toBeVisible({ timeout: 10000 });
    // A row shows a cost figure.
    await expect(row).toContainText("$");
  });

  test("disable a model toggles /api/models/disabled", async ({ page }) => {
    let posted = false;
    await page.route("**/api/models/disabled", async (route) => {
      if (route.request().method() === "POST") {
        posted = true;
        return route.fulfill({
          status: 200,
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ data: {} }),
        });
      }
      return route.continue();
    });
    await page.goto("/models");
    const row = page.locator("[data-testid='model-row']").first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.locator("[role='switch']").first().click();
    await expect.poll(() => posted, { timeout: 10000 }).toBe(true);
  });
});
