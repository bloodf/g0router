import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("MITM", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("MITM page loads", async ({ page }) => {
    await page.goto("/mitm");
    await expect(page.locator("body")).toContainText("MITM", { timeout: 10000 });
  });
});
