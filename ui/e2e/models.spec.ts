import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Models", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("models page loads", async ({ page }) => {
    await page.goto("/models");
    await expect(page.locator("body")).toContainText("Models", { timeout: 10000 });
  });
});
