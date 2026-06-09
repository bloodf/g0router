import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Prompt Templates", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("prompts page loads", async ({ page }) => {
    await page.goto("/prompts");
    await expect(page.locator("body")).toContainText("Prompts", { timeout: 10000 });
  });
});
