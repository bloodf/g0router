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
    const textarea = page.locator('textarea').first();
    await textarea.fill("my secret password");
    await page.locator('button:has-text("Test")').first().click();
    await expect(page.locator("body")).toContainText("blocked", { timeout: 5000 });
  });
});
