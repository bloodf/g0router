import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Providers", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("list loads", async ({ page }) => {
    await page.goto("/providers");
    await expect(page.locator("body")).toContainText("Providers", { timeout: 10000 });
  });

  test("provider cards are visible", async ({ page }) => {
    await page.goto("/providers");
    // Provider cards should appear
    await expect(page.locator("[class*='card-elev']").first()).toBeVisible({ timeout: 10000 });
  });
});
