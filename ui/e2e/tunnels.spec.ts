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
    const toggle = page.locator("[data-testid='tunnel-toggle']").first();
    await expect(toggle).toBeVisible({ timeout: 10000 });
    // The mock store is worker-scoped, so the tunnel may already be enabled from
    // a prior test. Ensure it is enabled first (idempotent POST), then toggle off
    // and assert the DELETE fires.
    if ((await toggle.getAttribute("data-state")) !== "checked") {
      await toggle.click();
      await expect(toggle).toHaveAttribute("data-state", "checked", {
        timeout: 10000,
      });
    }
    const deletePromise = page.waitForRequest(
      (r) => /\/api\/tunnels\/[^/]+$/.test(r.url()) && r.method() === "DELETE",
    );
    await toggle.click();
    await deletePromise;
  });
});
