import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Aliases", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("page loads", async ({ page }) => {
    await page.goto("/aliases");
    await expect(page.locator("body")).toContainText("Aliases", { timeout: 10000 });
  });
});
