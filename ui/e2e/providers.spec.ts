import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Providers", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("list loads", async ({ page }) => {
    await page.goto("/providers");
    await expect(page.locator("body")).toContainText("Providers", { timeout: 10000 });
  });

  test("provider cards are visible", async ({ page }) => {
    await page.goto("/providers");
    // Provider cards should appear
    await expect(page.locator("[class*='card-elev']").first()).toBeVisible({ timeout: 10000 });
  });

  test("providers are grouped (OAuth / API-Key / Free / Compatible)", async ({ page }) => {
    await page.goto("/providers");
    const groups = page.locator("[data-testid='provider-group']");
    await expect(groups.first()).toBeVisible({ timeout: 10000 });
    // At least one card sits under a group.
    await expect(
      groups.first().locator("[class*='card-elev']").first()
    ).toBeVisible();
  });

  test("OAuth modal finalizes via the /callback popup relay", async ({ page }) => {
    let finalizeCount = 0;
    await page.route("**/api/oauth/*/start", async (route) =>
      route.fulfill({
        status: 200,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ data: { auth_url: "about:blank", state: "xyz" } }),
      })
    );
    await page.route("**/api/oauth/*/callback", async (route) => {
      finalizeCount += 1;
      return route.fulfill({
        status: 200,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ data: { id: "conn-new", kind: "oauth" } }),
      });
    });

    await page.goto("/providers");
    // Open a provider's OAuth modal. Gemini supports oauth in the catalog.
    await page.locator("[data-testid='provider-oauth-action']").first().click();
    const modal = page.locator("[data-testid='modal-traffic-lights']");
    await expect(modal.first()).toBeVisible({ timeout: 10000 });

    // Simulate the w6-c relay delivering an authorization code.
    await page.evaluate(() => {
      const channel = new BroadcastChannel("oauth_callback");
      channel.postMessage({ code: "abc", state: "xyz" });
      channel.close();
    });

    // The finalize POST fires exactly once and the modal closes.
    await expect
      .poll(() => finalizeCount, { timeout: 10000 })
      .toBe(1);
    await expect(modal.first()).toBeHidden({ timeout: 10000 });
  });

  test("provider detail panel shows connections + models", async ({ page }) => {
    await page.goto("/providers");
    // Open the detail panel for a provider with connections (openai has 2).
    await page.locator("[data-testid='provider-card-openai']").click();
    const panel = page.locator("[data-testid='provider-detail-panel']");
    await expect(panel).toBeVisible({ timeout: 10000 });
    await expect(
      panel.locator("[data-testid='connection-row']").first()
    ).toBeVisible();
  });

  test("manual config / edit-connection modals open", async ({ page }) => {
    await page.goto("/providers");
    await page.locator("[data-testid='provider-card-openai']").click();
    await expect(
      page.locator("[data-testid='provider-detail-panel']")
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='add-connection-action']").first().click();
    await expect(
      page.locator("[data-testid='modal-traffic-lights']").first()
    ).toBeVisible({ timeout: 10000 });
  });
});
