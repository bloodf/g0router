import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("API Keys", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("page loads and shows keys", async ({ page }) => {
    await page.goto("/keys");
    await expect(page.locator("body")).toContainText("API Keys", { timeout: 10000 });
  });
});
