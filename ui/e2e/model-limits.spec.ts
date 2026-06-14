import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Model Limits", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("model limits page loads", async ({ page }) => {
    await page.goto("/model-limits");
    await expect(page.locator("body")).toContainText("Model Limits", { timeout: 10000 });
  });

  test("limit rows render from seed", async ({ page }) => {
    await page.goto("/model-limits");
    const rows = page.locator("[data-testid='model-limit-row']");
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    await expect(rows).toHaveCount(2);
    await expect(page.locator("body")).toContainText("gpt-4o");
    await expect(page.locator("body")).toContainText("128000");
  });

  test("opening the model limit modal and saving fires a POST", async ({ page }) => {
    await page.goto("/model-limits");
    await expect(
      page.locator("[data-testid='model-limit-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='model-limit-new']").click();
    const modal = page.locator("[role='dialog']");
    await expect(modal).toBeVisible({ timeout: 10000 });
    await modal.locator("#model-limit-model").fill("gpt-4o-new");
    const postPromise = page.waitForRequest(
      (r) => r.url().includes("/api/model-limits") && r.method() === "POST",
    );
    await modal.locator("[data-testid='model-limit-save']").click();
    await postPromise;
  });
});
