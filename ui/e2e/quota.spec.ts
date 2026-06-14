import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Quota", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("quota page loads", async ({ page }) => {
    await page.goto("/quota");
    await expect(page.locator("body")).toContainText("Quota", { timeout: 10000 });
  });

  test("provider-limit cards render with a used/limit indicator", async ({ page }) => {
    await page.goto("/quota");
    await expect(page.locator("[data-testid='quota-card']").first()).toBeVisible({ timeout: 10000 });
    // Each card shows a used/limit progress indicator.
    await expect(page.locator("[data-testid='quota-progress']").first()).toBeVisible({ timeout: 10000 });
  });
});
