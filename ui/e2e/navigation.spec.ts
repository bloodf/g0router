import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("sidebar links work", async ({ page }) => {
    await page.goto("/dashboard");
    // Try clicking on a few sidebar links
    const links = ["/providers", "/connections", "/keys", "/settings"];
    for (const href of links) {
      const link = page.locator(`a[href="${href}"]`).first();
      if (await link.isVisible().catch(() => false)) {
        await link.click();
        await page.waitForLoadState("networkidle");
        await expect(page).toHaveURL(new RegExp(href.replace("/", "\\/")));
      }
    }
  });
});
