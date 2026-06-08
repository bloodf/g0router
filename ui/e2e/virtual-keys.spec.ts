import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Virtual Keys", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("virtual keys page loads", async ({ page }) => {
    await page.goto("/virtual-keys");
    await expect(page.locator("body")).toContainText("Virtual Keys", { timeout: 10000 });
  });
});
