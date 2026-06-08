import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("page loads without errors", async ({ page }) => {
    await page.goto("/dashboard");
    // Check that the page title or some expected element is present
    await expect(page.locator("body")).toBeVisible();
    // No error toast or overlay
    await expect(page.locator("[role='alert']")).not.toBeVisible();
  });

  test("metrics cards are visible", async ({ page }) => {
    await page.goto("/dashboard");
    // Metric cards use translation keys - check for card structure
    await expect(page.locator("[class*='grid']").first()).toBeVisible();
  });
});
