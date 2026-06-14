import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Tunnels", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("tunnels page loads", async ({ page }) => {
    await page.goto("/tunnels");
    await expect(page.locator("body")).toContainText("Tunnels", { timeout: 10000 });
  });

  test("tunnel cards render from seed", async ({ page }) => {
    await page.goto("/tunnels");
    const cards = page.locator("[data-testid='tunnel-card']");
    await expect(cards.first()).toBeVisible({ timeout: 10000 });
    await expect(cards).toHaveCount(2);
    await expect(page.locator("body")).toContainText("cloudflare");
    await expect(page.locator("body")).toContainText("tailscale");
    await expect(page.locator("body")).toContainText("trycloudflare.com");
  });

  test("enabling a tunnel fires a POST", async ({ page }) => {
    await page.goto("/tunnels");
    await expect(
      page.locator("[data-testid='tunnel-card']").first()
    ).toBeVisible({ timeout: 10000 });
    const postPromise = page.waitForRequest(
      (r) => /\/api\/tunnels\/[^/]+$/.test(r.url()) && r.method() === "POST",
    );
    await page.locator("[data-testid='tunnel-toggle']").first().click();
    await postPromise;
  });

  test("disabling an enabled tunnel fires a DELETE", async ({ page }) => {
    await page.goto("/tunnels");
    await expect(
      page.locator("[data-testid='tunnel-card']").first()
    ).toBeVisible({ timeout: 10000 });
    // Enable first (POST), then toggle again to disable (DELETE).
    const postPromise = page.waitForRequest(
      (r) => /\/api\/tunnels\/[^/]+$/.test(r.url()) && r.method() === "POST",
    );
    await page.locator("[data-testid='tunnel-toggle']").first().click();
    await postPromise;
    const deletePromise = page.waitForRequest(
      (r) => /\/api\/tunnels\/[^/]+$/.test(r.url()) && r.method() === "DELETE",
    );
    await page.locator("[data-testid='tunnel-toggle']").first().click();
    await deletePromise;
  });
});
