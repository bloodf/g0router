import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Auth flow", () => {
  test("login with valid credentials redirects to dashboard", async ({ page }) => {
    await login(page);
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test("invalid credentials show error toast", async ({ page }) => {
    await page.goto("/login");
    await page.fill("#username", "admin");
    await page.fill("#password", "wrongpassword");
    await page.click('button[type="submit"]');
    // Wait for toast or error to appear — sonner renders outside body, so search the whole page
    await expect(page.locator("[data-sonner-toast], .sonner-toast").first()).toContainText(/invalid|error|failed/i, { timeout: 5000 });
  });
});
