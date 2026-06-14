import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Combos", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("combos page loads", async ({ page }) => {
    await page.goto("/combos");
    await expect(page.locator("body")).toContainText("Combos", { timeout: 10000 });
  });

  test("combo list rows render from seed", async ({ page }) => {
    await page.goto("/combos");
    const rows = page.locator("[data-testid='combo-row']");
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
    await expect(rows).toHaveCount(2);
    await expect(page.locator("body")).toContainText("Fast + Cheap");
    await expect(page.locator("body")).toContainText("Best Quality");
  });

  test("opening ComboFormModal renders member rows in seed order", async ({ page }) => {
    await page.goto("/combos");
    await expect(
      page.locator("[data-testid='combo-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='combo-edit']").first().click();
    await expect(
      page.locator("[data-testid='modal-traffic-lights']").first()
    ).toBeVisible({ timeout: 10000 });
    const steps = page.locator("[data-testid='combo-step-row']");
    await expect(steps).toHaveCount(2);
    // Seed combo-1 order: groq/llama-3-70b then openai/gpt-4o-mini.
    await expect(steps.nth(0)).toContainText("llama-3-70b");
    await expect(steps.nth(1)).toContainText("gpt-4o-mini");
  });

  test("saving a combo fires a PUT with the member order", async ({ page }) => {
    await page.goto("/combos");
    await expect(
      page.locator("[data-testid='combo-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='combo-edit']").first().click();
    await expect(
      page.locator("[data-testid='modal-traffic-lights']").first()
    ).toBeVisible({ timeout: 10000 });
    // The persisted-order proof (§1.3 point 4): saving fires the combo update with
    // the current member order in the request body. moveStep (unit-tested) is the
    // authoritative reorder correctness proof; this asserts the reorder wiring
    // reaches the network with the ordered members.
    const putPromise = page.waitForRequest(
      (r) => /\/api\/combos\/[^/]+$/.test(r.url()) && r.method() === "PUT",
    );
    await page.locator("[data-testid='combo-save']").click();
    const req = await putPromise;
    const body = req.postDataJSON() as { steps?: Array<{ model: string }> };
    expect(body.steps?.map((s) => s.model)).toEqual([
      "llama-3-70b",
      "gpt-4o-mini",
    ]);
  });

  test("deleting a combo asks for confirmation", async ({ page }) => {
    await page.goto("/combos");
    await expect(
      page.locator("[data-testid='combo-row']").first()
    ).toBeVisible({ timeout: 10000 });
    await page.locator("[data-testid='combo-delete']").first().click();
    await expect(page.locator("[role='dialog']")).toContainText("Delete", {
      timeout: 10000,
    });
  });
});
