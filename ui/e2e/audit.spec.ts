import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Audit", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("audit page loads", async ({ page }) => {
    await page.goto("/audit");
    await expect(page.locator("body")).toContainText("Audit", { timeout: 10000 });
  });
});
