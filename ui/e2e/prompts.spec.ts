import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Prompt Templates", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("prompts page loads", async ({ page }) => {
    await page.goto("/prompts");
    await expect(page.locator("body")).toContainText("Prompts", { timeout: 10000 });
  });

  test("prompt rows render from seed", async ({ page }) => {
    await page.goto("/prompts");
    const rows = page.locator('[data-testid="prompt-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    expect(await rows.count()).toBeGreaterThanOrEqual(2);
    await expect(page.locator("body")).toContainText("Code Review");
    await expect(page.locator("body")).toContainText("gpt-4o");
  });

  test("creating a prompt via the form saves it", async ({ page }) => {
    await page.goto("/prompts");
    await page.locator('[data-testid="prompt-new"]').click();
    await expect(page.locator('[data-testid="modal-traffic-lights"]')).toBeVisible({
      timeout: 5000,
    });
    await page.locator('#prompt-name').fill("Summarizer");
    const postPromise = page.waitForRequest(
      (req) => req.url().endsWith("/api/prompt-templates") && req.method() === "POST"
    );
    await page.locator('[data-testid="prompt-save"]').click();
    await postPromise;
  });

  test("deleting a prompt goes through the confirm modal", async ({ page }) => {
    await page.goto("/prompts");
    const rows = page.locator('[data-testid="prompt-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    const before = await rows.count();
    await page.locator('[data-testid="prompt-delete"]').first().click();
    const dialog = page.locator('[role="dialog"]', { hasText: "Delete prompt" });
    await expect(dialog).toBeVisible({ timeout: 5000 });
    await dialog.locator('button:has-text("Delete")').click();
    await expect(rows).toHaveCount(before - 1, { timeout: 5000 });
  });
});
