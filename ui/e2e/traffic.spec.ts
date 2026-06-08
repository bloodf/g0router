import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Traffic", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("traffic page loads", async ({ page }) => {
    await page.goto("/traffic");
    await expect(page.locator("body")).toContainText("Traffic", { timeout: 10000 });
  });
});
