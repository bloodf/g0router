import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Chat", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("chat page loads", async ({ page }) => {
    await page.goto("/chat");
    await expect(page.locator("body")).toContainText("Chat", { timeout: 10000 });
  });
});
