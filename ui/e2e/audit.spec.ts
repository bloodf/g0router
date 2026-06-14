import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Audit", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("audit page loads", async ({ page }) => {
    await page.goto("/audit");
    await expect(page.locator("body")).toContainText("Audit", { timeout: 10000 });
  });

  test("audit rows render from seed", async ({ page }) => {
    await page.goto("/audit");
    const rows = page.locator('[data-testid="audit-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    expect(await rows.count()).toBeGreaterThanOrEqual(5);
    await expect(page.locator("body")).toContainText("create_key");
    await expect(page.locator("body")).toContainText("admin");
  });

  test("audit viewer exposes a limit/pagination control", async ({ page }) => {
    await page.goto("/audit");
    await expect(page.locator('[data-testid="audit-row"]').first()).toBeVisible({
      timeout: 10000,
    });
    await expect(page.locator('[data-testid="audit-limit"]')).toBeVisible();
  });
});
