import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Tunnels", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("tunnels page loads", async ({ page }) => {
    await page.goto("/tunnels");
    await expect(page.locator("body")).toContainText("Tunnels", { timeout: 10000 });
  });
});
