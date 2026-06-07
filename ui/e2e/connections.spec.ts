import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Connections", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("page loads", async ({ page }) => {
    await page.goto("/connections");
    await expect(page.locator("body")).toContainText("Connections", { timeout: 10000 });
  });
});
