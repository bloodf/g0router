import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Alerts", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("alerts page loads", async ({ page }) => {
    await page.goto("/alerts");
    await expect(page.locator("body")).toContainText("Alerts", { timeout: 10000 });
  });

  test("alert channel rows render from seed", async ({ page }) => {
    await page.goto("/alerts");
    const rows = page.locator('[data-testid="alert-channel-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    expect(await rows.count()).toBeGreaterThanOrEqual(2);
    await expect(page.locator("body")).toContainText("Webhook Alerts");
    await expect(page.locator("body")).toContainText("webhook");
  });

  test("creating an alert channel via the form saves it", async ({ page }) => {
    await page.goto("/alerts");
    await page.locator('[data-testid="alert-channel-new"]').click();
    await expect(page.locator('[data-testid="modal-traffic-lights"]')).toBeVisible({
      timeout: 5000,
    });
    await page.locator('#alert-channel-name').fill("Slack Alerts");
    const postPromise = page.waitForRequest(
      (req) => req.url().endsWith("/api/alert-channels") && req.method() === "POST"
    );
    await page.locator('[data-testid="alert-channel-save"]').click();
    await postPromise;
  });

  test("per-channel test fires a test POST", async ({ page }) => {
    await page.goto("/alerts");
    await expect(
      page.locator('[data-testid="alert-channel-row"]').first()
    ).toBeVisible({ timeout: 10000 });
    const testPromise = page.waitForRequest(
      (req) => /\/api\/alert-channels\/[^/]+\/test$/.test(req.url()) && req.method() === "POST"
    );
    await page.locator('[data-testid="alert-channel-test"]').first().click();
    await testPromise;
  });

  test("deleting an alert channel goes through the confirm modal", async ({ page }) => {
    await page.goto("/alerts");
    await page.locator('[data-testid="alert-channel-delete"]').first().click();
    await expect(page.locator("body")).toContainText("Delete", { timeout: 5000 });
    await page.locator('button:has-text("Delete")').last().click();
    await expect(page.locator('[data-testid="alert-channel-row"]')).toHaveCount(1, {
      timeout: 5000,
    });
  });
});
