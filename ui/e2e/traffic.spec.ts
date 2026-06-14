import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Traffic", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("traffic page loads", async ({ page }) => {
    await page.goto("/traffic");
    await expect(page.locator("body")).toContainText("Traffic", { timeout: 10000 });
  });

  test("a live traffic row appears from the stream", async ({ page }) => {
    await page.goto("/traffic");
    // MockEventSource (fixture.ts) pushes /api/traffic/stream rows every ~1.2s.
    await expect(page.locator("[data-testid='traffic-row']").first()).toBeVisible({ timeout: 15000 });
  });
});
