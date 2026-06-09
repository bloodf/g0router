import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Alerts", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("alerts page loads", async ({ page }) => {
    await page.goto("/alerts");
    await expect(page.locator("body")).toContainText("Alerts", { timeout: 10000 });
  });
});
