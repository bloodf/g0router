import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Aliases", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("page loads", async ({ page }) => {
    await page.goto("/aliases");
    await expect(page.locator("body")).toContainText("Aliases", { timeout: 10000 });
  });

  test("alias rows render from seed", async ({ page }) => {
    await page.goto("/aliases");
    const rows = page.locator("[data-testid='alias-row']");
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    await expect(rows).toHaveCount(3);
    await expect(page.locator("body")).toContainText("gpt4");
    await expect(page.locator("body")).toContainText("gpt-4o");
  });

  test("opening the alias modal and saving fires a POST", async ({ page }) => {
    await page.goto("/aliases");
    await expect(
      page.locator("[data-testid='alias-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='alias-new']").click();
    const modal = page.locator("[role='dialog']");
    await expect(modal).toBeVisible({ timeout: 10000 });
    await modal.locator("#alias-name").fill("fast");
    const postPromise = page.waitForRequest(
      (r) => r.url().includes("/api/aliases") && r.method() === "POST",
    );
    await modal.locator("[data-testid='alias-save']").click();
    await postPromise;
  });

  test("deleting an alias asks for confirmation", async ({ page }) => {
    await page.goto("/aliases");
    await expect(
      page.locator("[data-testid='alias-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='alias-delete']").first().click();
    await expect(page.locator("[role='dialog']")).toContainText("Delete", {
      timeout: 10000,
    });
  });
});
