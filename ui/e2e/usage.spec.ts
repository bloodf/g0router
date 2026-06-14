import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Usage & Logs", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("usage page loads", async ({ page }) => {
    await page.goto("/usage");
    await expect(page.locator("body")).toContainText("Usage", { timeout: 10000 });
  });

  test("logs page loads", async ({ page }) => {
    await page.goto("/logs");
    await expect(page.locator("body")).toContainText("Logs", { timeout: 10000 });
  });

  test("usage overview shows metric cards", async ({ page }) => {
    await page.goto("/usage");
    await expect(page.locator("[data-testid='usage-metric']").first()).toBeVisible({ timeout: 10000 });
  });

  test("usage tabs switch between overview, logs and details", async ({ page }) => {
    await page.goto("/usage");
    await expect(page.locator("[data-testid='usage-tabs']")).toBeVisible({ timeout: 10000 });
    // Switch to logs tab -> a request-log table appears.
    await page.locator("[data-testid='usage-tabs'] [role='tab']", { hasText: /logs/i }).click();
    await expect(page.locator("[data-testid='request-log-table']")).toBeVisible({ timeout: 10000 });
    // Switch to details tab -> the request-details table appears.
    await page.locator("[data-testid='usage-tabs'] [role='tab']", { hasText: /details/i }).click();
    await expect(page.locator("[data-testid='request-details-table']")).toBeVisible({ timeout: 10000 });
  });

  test("period selector switches and re-fetches stats", async ({ page }) => {
    await page.goto("/usage");
    const period = page.locator("[data-testid='usage-period']");
    await expect(period).toBeVisible({ timeout: 10000 });
    // Pick a DIFFERENT period (default is 7d) so a new stats request fires.
    const reqPromise = page.waitForRequest((r) => r.url().includes("/api/usage/stats") && r.url().includes("period=30d"));
    await period.locator("[role='tab']", { hasText: /30d|30 d/i }).first().click();
    await reqPromise;
  });

  test("standalone logs route shows the request-log table", async ({ page }) => {
    await page.goto("/logs");
    await expect(page.locator("[data-testid='request-log-table']")).toBeVisible({ timeout: 10000 });
  });
});
