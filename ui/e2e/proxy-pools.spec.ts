import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Proxy Pools", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("proxy pools page loads", async ({ page }) => {
    await page.goto("/proxy-pools");
    await expect(page.locator("body")).toContainText("Proxy Pools", { timeout: 10000 });
  });
});
