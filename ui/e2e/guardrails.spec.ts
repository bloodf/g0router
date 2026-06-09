import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Guardrails", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("guardrails page loads", async ({ page }) => {
    await page.goto("/guardrails");
    await expect(page.locator("body")).toContainText("Guardrails", { timeout: 10000 });
  });

  test("test guardrails prompt", async ({ page }) => {
    await page.goto("/guardrails");
    const input = page.locator('input[aria-label="Test prompt"]').first();
    await input.fill("my secret password");
    await page.locator('button:has-text("Test")').first().click();
    await expect(page.locator("body")).toContainText(/blocked/i, { timeout: 5000 });
  });
});
