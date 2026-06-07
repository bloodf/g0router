import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Pricing", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("pricing page loads", async ({ page }) => {
    await page.goto("/pricing");
    await expect(page.locator("body")).toContainText("Pricing", { timeout: 10000 });
  });
});
