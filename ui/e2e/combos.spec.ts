import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Combos", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("combos page loads", async ({ page }) => {
    await page.goto("/combos");
    await expect(page.locator("body")).toContainText("Combos", { timeout: 10000 });
  });
});
