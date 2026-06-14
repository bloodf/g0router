import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Proxy Pools", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("proxy pools page loads", async ({ page }) => {
    await page.goto("/proxy-pools");
    await expect(page.locator("body")).toContainText("Proxy Pools", { timeout: 10000 });
  });

  test("pool rows render from seed", async ({ page }) => {
    await page.goto("/proxy-pools");
    const rows = page.locator("[data-testid='proxy-pool-row']");
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    await expect(rows).toHaveCount(1);
    await expect(page.locator("body")).toContainText("US East");
    await expect(page.locator("body")).toContainText("us-east.proxy.example.com");
  });

  test("opening the pool modal and saving fires a POST", async ({ page }) => {
    await page.goto("/proxy-pools");
    await expect(
      page.locator("[data-testid='proxy-pool-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='proxy-pool-new']").click();
    const modal = page.locator("[role='dialog']");
    await expect(modal).toBeVisible({ timeout: 10000 });
    await expect(modal.locator("[data-testid='modal-traffic-lights']")).toBeVisible();
    await modal.locator("#proxy-pool-name").fill("EU West");
    const postPromise = page.waitForRequest(
      (r) =>
        r.url().endsWith("/api/proxy-pools") && r.method() === "POST",
    );
    await modal.locator("[data-testid='proxy-pool-save']").click();
    await postPromise;
  });

  test("testing a pool fires a POST to /test", async ({ page }) => {
    await page.goto("/proxy-pools");
    await expect(
      page.locator("[data-testid='proxy-pool-row']").first()
    ).toBeVisible({ timeout: 10000 });
    const postPromise = page.waitForRequest(
      (r) => /\/api\/proxy-pools\/[^/]+\/test$/.test(r.url()) && r.method() === "POST",
    );
    await page.locator("[data-testid='proxy-pool-test']").first().click();
    await postPromise;
  });

  test("deleting a pool asks for confirmation then fires DELETE", async ({ page }) => {
    await page.goto("/proxy-pools");
    await expect(
      page.locator("[data-testid='proxy-pool-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='proxy-pool-delete']").first().click();
    const dialog = page.locator("[role='dialog']");
    await expect(dialog).toContainText("Delete", { timeout: 10000 });
    const deletePromise = page.waitForRequest(
      (r) => /\/api\/proxy-pools\/[^/]+$/.test(r.url()) && r.method() === "DELETE",
    );
    await dialog.locator("button", { hasText: "Delete" }).click();
    await deletePromise;
  });
});
