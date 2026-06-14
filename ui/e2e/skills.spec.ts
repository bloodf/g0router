import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Skills", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("Skills page loads", async ({ page }) => {
    await page.goto("/skills");
    await expect(page.locator("body")).toContainText("Skills", {
      timeout: 10000,
    });
  });

  test("Skills page lists seeded skills grouped by category", async ({
    page,
  }) => {
    await page.goto("/skills");
    await expect(page.getByTestId("skill-row").first()).toBeVisible({
      timeout: 10000,
    });
    const rows = page.getByTestId("skill-row");
    await expect(rows).toHaveCount(2);
    await expect(page.locator("body")).toContainText("filesystem");
    await expect(page.locator("body")).toContainText("github");
    // grouped by category — the seed category heading is present
    await expect(page.locator("body")).toContainText("Endpoint Skills");
  });

  test("Skills page exposes a copy-to-clipboard control per skill", async ({
    page,
  }) => {
    await page.goto("/skills");
    await expect(page.getByTestId("skill-copy").first()).toBeVisible({
      timeout: 10000,
    });
    const copies = page.getByTestId("skill-copy");
    await expect(copies).toHaveCount(2);
  });
});
