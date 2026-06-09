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
});
