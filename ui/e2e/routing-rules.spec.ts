import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Routing Rules", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("routing rules page loads", async ({ page }) => {
    await page.goto("/routing-rules");
    await expect(page.locator("body")).toContainText("Routing", { timeout: 10000 });
  });

  test("rule rows render from seed", async ({ page }) => {
    await page.goto("/routing-rules");
    const rows = page.locator("[data-testid='routing-rule-row']");
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    await expect(rows).toHaveCount(2);
    await expect(page.locator("body")).toContainText("Route GPT-4 to OpenAI");
  });

  test("opening the rule modal and saving fires a POST", async ({ page }) => {
    await page.goto("/routing-rules");
    await expect(
      page.locator("[data-testid='routing-rule-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='routing-rule-new']").click();
    const modal = page.locator("[role='dialog']");
    await expect(modal).toBeVisible({ timeout: 10000 });
    await modal.locator("#routing-rule-name").fill("New rule");
    const postPromise = page.waitForRequest(
      (r) => r.url().includes("/api/routing-rules") && r.method() === "POST",
    );
    await modal.locator("[data-testid='routing-rule-save']").click();
    await postPromise;
  });

  test("deleting a rule asks for confirmation", async ({ page }) => {
    await page.goto("/routing-rules");
    await expect(
      page.locator("[data-testid='routing-rule-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='routing-rule-delete']").first().click();
    await expect(page.locator("[role='dialog']")).toContainText("Delete", {
      timeout: 10000,
    });
  });
});
