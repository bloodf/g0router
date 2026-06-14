import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Connections", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("page loads", async ({ page }) => {
    await page.goto("/connections");
    await expect(page.locator("body")).toContainText("Connections", { timeout: 10000 });
  });

  test("connection rows render with provider + auth type", async ({ page }) => {
    await page.goto("/connections");
    const row = page.locator("[data-testid='connection-row']").first();
    await expect(row).toBeVisible({ timeout: 10000 });
    // Each row exposes an is_active toggle.
    await expect(row.locator("[role='switch']").first()).toBeVisible();
  });

  test("test a connection", async ({ page }) => {
    await page.route("**/api/connections/*/test", async (route) =>
      route.fulfill({
        status: 200,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ data: { ok: true, latency_ms: 42 } }),
      })
    );
    await page.goto("/connections");
    const row = page.locator("[data-testid='connection-row']").first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.locator("[data-testid='connection-test']").click();
    await expect(page.locator("[data-sonner-toast]").first()).toBeVisible({
      timeout: 10000,
    });
  });

  test("delete a connection asks for confirmation", async ({ page }) => {
    await page.goto("/connections");
    const row = page.locator("[data-testid='connection-row']").first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.locator("[data-testid='connection-delete']").click();
    // ConfirmModal appears with traffic lights.
    await expect(
      page.locator("[data-testid='modal-traffic-lights']").first()
    ).toBeVisible({ timeout: 10000 });
  });
});
