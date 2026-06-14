import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Settings", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("settings form loads", async ({ page }) => {
    await page.goto("/settings");
    await expect(page.locator("body")).toContainText("Settings", { timeout: 10000 });
    // Check for form fields (exclude hidden mobile menu button)
    await expect(page.locator("input:not([type='hidden']), select, textarea, button:not(.md\\:hidden)").first()).toBeVisible();
  });

  test("toggle require_login and save", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(1000);

    // Find the require_login toggle
    const toggle = page.locator('label:has-text("Require login") + button, button[role="switch"]').first();
    if (await toggle.isVisible().catch(() => false)) {
      const before = await toggle.getAttribute("aria-checked");
      await toggle.click();
      await page.waitForTimeout(500);

      // Click save
      const saveBtn = page.locator('button:has-text("Save"), button:has-text("Salvar")').first();
      if (await saveBtn.isVisible().catch(() => false)) {
        await saveBtn.click();
        await expect(page.locator("body")).toContainText(/saved|success|salvo/i, { timeout: 5000 });
      }
    }
  });

  test("general and language panels render", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(1000);
    // Theme segmented control (general panel) and language select (language panel).
    await expect(page.getByTestId("theme-segmented")).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId("language-select")).toBeVisible();
  });

  test("OIDC config panel inputs render", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(1000);
    await expect(page.getByTestId("oidc-issuer-url")).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId("oidc-client-id")).toBeVisible();
    await expect(page.getByTestId("oidc-redirect-uri")).toBeVisible();
  });

  test("password change panel renders", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(1000);
    await expect(page.getByTestId("password-current")).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId("password-new")).toBeVisible();
    await expect(page.getByTestId("password-confirm")).toBeVisible();
  });

  test("DB info panel shows database data from the mock", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(1000);
    const dbPanel = page.getByTestId("db-info-panel");
    await expect(dbPanel).toBeVisible({ timeout: 10000 });
    // The mock returns a deterministic path; the panel must render it.
    await expect(dbPanel).toContainText("g0router.db");
  });

  test("about/version block shows the version from /api/version", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(1000);
    const about = page.getByTestId("about-version");
    await expect(about).toBeVisible({ timeout: 10000 });
    // The corrected version mock returns 0.9.0-mock.
    await expect(about).toContainText("0.9.0-mock");
  });

  test("View changelog opens the ChangelogModal", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(1000);
    await page.getByTestId("open-changelog").click();
    await expect(page.getByTestId("changelog-modal")).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId("changelog-modal")).toContainText(/changelog/i);
  });

  test("Donate opens the DonateModal", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(1000);
    await page.getByTestId("open-donate").click();
    await expect(page.getByTestId("donate-modal")).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId("donate-modal")).toContainText(/donate|support/i);
  });

  test("visiting /settings lights the sidebar update-badge", async ({ page }) => {
    await page.goto("/settings");
    // The version-check hook fetches /api/version (mock update_available:true,
    // latest_version:"v9.9.9") and calls settingsStore.setUpdateInfo, which lights
    // the FROZEN sidebar badge.
    await expect(page.getByTestId("update-badge")).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId("update-badge")).toContainText("v9.9.9");
  });
});
