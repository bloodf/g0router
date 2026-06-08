import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Audit", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("audit page loads", async ({ page }) => {
    await page.goto("/audit");
    await expect(page.locator("body")).toContainText("Audit", { timeout: 10000 });
  });
});
