import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Console", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("console page loads", async ({ page }) => {
    await page.goto("/console");
    await expect(page.locator("body")).toContainText("Console", { timeout: 10000 });
  });
});
