import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Model Limits", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("model limits page loads", async ({ page }) => {
    await page.goto("/model-limits");
    await expect(page.locator("body")).toContainText("Model Limits", { timeout: 10000 });
  });
});
