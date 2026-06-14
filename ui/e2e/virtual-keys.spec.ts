import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Virtual Keys", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("virtual keys page loads", async ({ page }) => {
    await page.goto("/virtual-keys");
    await expect(page.locator("body")).toContainText("Virtual Keys", { timeout: 10000 });
  });

  test("virtual key list renders seeded rows", async ({ page }) => {
    await page.goto("/virtual-keys");
    const rows = page.locator('[data-testid="virtual-key-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator("body")).toContainText("Team Alpha");
  });

  test("opening the form modal shows the KeyIDs editor", async ({ page }) => {
    await page.goto("/virtual-keys");
    await page.locator('[data-testid="virtual-key-row"]').first().waitFor({ timeout: 10000 });
    await page.locator('[data-testid="create-vk-trigger"]').click();
    await expect(page.locator('[data-testid="modal-traffic-lights"]')).toBeVisible();
    await expect(page.locator('[data-testid="key-ids-editor"]')).toBeVisible();
  });

  test("saving a virtual key posts provider_configs with key_ids", async ({ page }) => {
    await page.goto("/virtual-keys");
    await page.locator('[data-testid="virtual-key-row"]').first().waitFor({ timeout: 10000 });
    await page.locator('[data-testid="create-vk-trigger"]').click();
    await page.locator('[data-testid="vk-name"]').fill("Team Gamma");

    // Pick a provider, a model, and a key id in the KeyIDs editor.
    await page.locator('[data-testid="vk-provider-select"]').selectOption("openai");
    await page.locator('[data-testid="vk-model-option"]').first().click();
    await page.locator('[data-testid="vk-keyid-option"]').first().click();

    const savePromise = page.waitForRequest(
      (req) => req.url().endsWith("/api/virtual-keys") && req.method() === "POST"
    );
    await page.locator('[data-testid="vk-save"]').click();
    const req = await savePromise;
    const body = JSON.parse(req.postData() || "{}");
    expect(Array.isArray(body.provider_configs)).toBe(true);
    expect(body.provider_configs[0].key_ids.length).toBeGreaterThan(0);
  });
});
