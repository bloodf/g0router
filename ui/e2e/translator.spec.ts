import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Translator", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("translator page loads", async ({ page }) => {
    await page.goto("/translator");
    await expect(page.locator("body")).toContainText("Translator", {
      timeout: 10000,
    });
  });

  test("step panels render with textareas", async ({ page }) => {
    await page.goto("/translator");
    const textareas = page.locator("[data-testid='translator-step'] textarea");
    await expect(textareas.first()).toBeVisible({ timeout: 10000 });
    const count = await textareas.count();
    expect(count).toBeGreaterThanOrEqual(1);
    // The first step's textarea carries a step aria-label.
    await expect(
      page.locator('textarea[aria-label="Client Request"]')
    ).toBeVisible({ timeout: 10000 });
  });

  test("loading a sample populates the first textarea", async ({ page }) => {
    await page.goto("/translator");
    const firstArea = page.locator('textarea[aria-label="Client Request"]');
    await expect(firstArea).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='translator-load']").first().click();
    // The /api/translator/load mock returns a JSON sample containing "gpt-4o".
    await expect(firstArea).toHaveValue(/gpt-4o/, { timeout: 10000 });
  });

  test("translate updates a downstream panel", async ({ page }) => {
    await page.goto("/translator");
    await expect(
      page.locator('textarea[aria-label="Client Request"]')
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='translator-load']").first().click();
    await expect(
      page.locator('textarea[aria-label="Client Request"]')
    ).toHaveValue(/gpt-4o/, { timeout: 10000 });
    await page.locator("[data-testid='translator-translate']").click();
    // The /api/translator/translate mock returns a payload marked "translated".
    await expect(page.locator("body")).toContainText("translated", {
      timeout: 10000,
    });
  });
});
