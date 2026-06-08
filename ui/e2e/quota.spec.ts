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
});
