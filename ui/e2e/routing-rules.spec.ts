import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Routing Rules", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("routing rules page loads", async ({ page }) => {
    await page.goto("/routing-rules");
    await expect(page.locator("body")).toContainText("Routing", { timeout: 10000 });
  });
});
