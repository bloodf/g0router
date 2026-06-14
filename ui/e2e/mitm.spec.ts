import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("MITM", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("MITM page loads", async ({ page }) => {
    await page.goto("/mitm");
    await expect(page.locator("body")).toContainText("MITM", { timeout: 10000 });
  });

  test("status panel renders with a global enable toggle", async ({ page }) => {
    await page.goto("/mitm");
    await expect(
      page.locator("[data-testid='mitm-enable-toggle']")
    ).toBeVisible({ timeout: 10000 });
  });

  test("tool rows render from seed", async ({ page }) => {
    await page.goto("/mitm");
    const rows = page.locator("[data-testid='mitm-tool-row']");
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    await expect(rows).toHaveCount(2);
    await expect(page.locator("body")).toContainText("Request Inspector");
    await expect(page.locator("body")).toContainText("Response Modifier");
  });

  test("toggling a tool fires a POST", async ({ page }) => {
    await page.goto("/mitm");
    await expect(
      page.locator("[data-testid='mitm-tool-row']").first()
    ).toBeVisible({ timeout: 10000 });
    const postPromise = page.waitForRequest(
      (r) => /\/api\/mitm\/tools\/[^/]+$/.test(r.url()) && r.method() === "POST",
    );
    await page
      .locator("[data-testid='mitm-tool-toggle']")
      .first()
      .click();
    await postPromise;
  });

  test("a download CA certificate control is present", async ({ page }) => {
    await page.goto("/mitm");
    await expect(
      page.locator("[data-testid='mitm-ca-cert-download']")
    ).toBeVisible({ timeout: 10000 });
  });
});
