import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("API Keys", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("page loads and shows keys", async ({ page }) => {
    await page.goto("/keys");
    await expect(page.locator("body")).toContainText("API Keys", { timeout: 10000 });
  });

  test("key list renders seeded rows", async ({ page }) => {
    await page.goto("/keys");
    const rows = page.locator('[data-testid="api-key-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator("body")).toContainText("Default Key");
  });

  test("create key flow posts {name} and shows the returned key", async ({ page }) => {
    await page.goto("/keys");
    await page.locator('[data-testid="api-key-row"]').first().waitFor({ timeout: 10000 });

    const createPromise = page.waitForRequest(
      (req) => req.url().endsWith("/api/keys") && req.method() === "POST"
    );
    await page.locator('[data-testid="create-key-trigger"]').click();
    await page.locator('[data-testid="create-key-name"]').fill("CI Key");
    await page.locator('[data-testid="create-key-submit"]').click();

    const req = await createPromise;
    expect(JSON.parse(req.postData() || "{}")).toMatchObject({ name: "CI Key" });
    await expect(page.locator('[data-testid="created-key-value"]')).toBeVisible({ timeout: 10000 });
  });

  test("delete a key asks for confirmation", async ({ page }) => {
    await page.goto("/keys");
    await page.locator('[data-testid="api-key-row"]').first().waitFor({ timeout: 10000 });
    await page.locator('[data-testid="delete-key"]').first().click();
    await expect(page.locator('[data-testid="modal-traffic-lights"]')).toBeVisible();
  });
});
