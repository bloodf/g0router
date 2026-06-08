import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Usage & Logs", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("usage page loads", async ({ page }) => {
    await page.goto("/usage");
    await expect(page.locator("body")).toContainText("Usage", { timeout: 10000 });
  });

  test("logs page loads", async ({ page }) => {
    await page.goto("/logs");
    await expect(page.locator("body")).toContainText("Logs", { timeout: 10000 });
  });
});
