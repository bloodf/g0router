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

  test("send message and receive mock response", async ({ page }) => {
    await page.goto("/chat");
    const textarea = page.locator('textarea').first();
    await textarea.fill("Hello mock assistant");
    await textarea.press("Enter");
    await expect(page.locator("body")).toContainText("mock assistant", { timeout: 15000 });
  });
});
