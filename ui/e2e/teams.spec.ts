import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Teams", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("teams page loads", async ({ page }) => {
    await page.goto("/teams");
    await expect(page.locator("body")).toContainText("Teams", { timeout: 10000 });
  });
});
