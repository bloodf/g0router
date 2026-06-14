import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Pricing", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("pricing page loads", async ({ page }) => {
    await page.goto("/pricing");
    await expect(page.locator("body")).toContainText("Pricing", { timeout: 10000 });
  });

  test("pricing table rows render", async ({ page }) => {
    await page.goto("/pricing");
    await expect(page.locator("[data-testid='pricing-row']").first()).toBeVisible({ timeout: 10000 });
  });

  test("opening the pricing modal shows the rate fields and saving fires a PATCH", async ({ page }) => {
    await page.goto("/pricing");
    await page.locator("[data-testid='pricing-edit']").first().click();
    const modal = page.locator("[role='dialog']");
    await expect(modal).toBeVisible({ timeout: 10000 });
    // The modal exposes the real Go rate fields.
    await expect(modal.locator("#pricing-input")).toBeVisible();
    await expect(modal.locator("#pricing-output")).toBeVisible();
    // Saving fires a PATCH to /api/pricing.
    const patchPromise = page.waitForRequest(
      (r) => r.url().includes("/api/pricing") && r.method() === "PATCH",
    );
    await modal.locator("[data-testid='pricing-save']").click();
    await patchPromise;
  });
});
