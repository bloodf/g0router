import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Teams", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("teams page loads", async ({ page }) => {
    await page.goto("/teams");
    await expect(page.locator("body")).toContainText("Teams", { timeout: 10000 });
  });

  test("team rows render from seed", async ({ page }) => {
    await page.goto("/teams");
    const rows = page.locator('[data-testid="team-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    expect(await rows.count()).toBeGreaterThanOrEqual(2);
    await expect(page.locator("body")).toContainText("Engineering");
    await expect(page.locator("body")).toContainText("Data Science");
  });

  test("opening the team form modal shows the traffic lights", async ({ page }) => {
    await page.goto("/teams");
    await page.locator('[data-testid="team-new"]').click();
    await expect(page.locator('[data-testid="modal-traffic-lights"]')).toBeVisible({
      timeout: 5000,
    });
  });

  test("creating a team via the form saves it", async ({ page }) => {
    await page.goto("/teams");
    await page.locator('[data-testid="team-new"]').click();
    await page.locator('#team-name').fill("Platform");
    await page.locator('[data-testid="team-save"]').click();
    await expect(page.locator('[data-testid="team-row"]').last()).toContainText(
      "Platform",
      { timeout: 5000 }
    );
  });

  test("deleting a team goes through the confirm modal", async ({ page }) => {
    await page.goto("/teams");
    const rows = page.locator('[data-testid="team-row"]');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    const before = await rows.count();
    await page.locator('[data-testid="team-delete"]').first().click();
    const dialog = page.locator('[role="dialog"]', { hasText: "Delete team" });
    await expect(dialog).toBeVisible({ timeout: 5000 });
    await dialog.locator('button:has-text("Delete")').click();
    await expect(rows).toHaveCount(before - 1, { timeout: 5000 });
  });

  test("the users panel lists the seeded admin user and a change-password control", async ({
    page,
  }) => {
    await page.goto("/teams");
    const userRows = page.locator('[data-testid="user-row"]');
    await expect(userRows.first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator("body")).toContainText("admin");
    await expect(
      page.locator('input[aria-label="New password"]')
    ).toBeVisible();
  });
});
