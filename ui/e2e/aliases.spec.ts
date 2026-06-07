import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Aliases", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("page loads", async ({ page }) => {
    await page.goto("/aliases");
    await expect(page.locator("body")).toContainText("Aliases", { timeout: 10000 });
  });
});
