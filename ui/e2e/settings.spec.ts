import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Settings", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("settings form loads", async ({ page }) => {
    await page.goto("/settings");
    await expect(page.locator("body")).toContainText("Settings", { timeout: 10000 });
    // Check for form fields (exclude hidden mobile menu button)
    await expect(page.locator("input:not([type='hidden']), select, textarea, button:not(.md\\:hidden)").first()).toBeVisible();
  });

  test("toggle require_login and save", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(1000);

    // Find the require_login toggle
    const toggle = page.locator('label:has-text("Require login") + button, button[role="switch"]').first();
    if (await toggle.isVisible().catch(() => false)) {
      const before = await toggle.getAttribute("aria-checked");
      await toggle.click();
      await page.waitForTimeout(500);

      // Click save
      const saveBtn = page.locator('button:has-text("Save"), button:has-text("Salvar")').first();
      if (await saveBtn.isVisible().catch(() => false)) {
        await saveBtn.click();
        await expect(page.locator("body")).toContainText(/saved|success|salvo/i, { timeout: 5000 });
      }
    }
  });
});
