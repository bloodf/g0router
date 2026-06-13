import { test, expect } from "@playwright/test";

test.describe("Shared UI primitives", () => {
  test("theme toggle cycles light, dark, system", async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem(
        "theme",
        JSON.stringify({ state: { theme: "light" }, version: 0 })
      );
    });
    await page.goto("/dashboard");

    const toggle = page
      .locator('[data-testid="theme-toggle-slot"]')
      .getByRole("button");
    await expect(toggle).toBeVisible();
    await expect(page.locator("html")).not.toHaveClass(/dark/);

    // light -> dark
    await toggle.click();
    await expect(page.locator("html")).toHaveClass(/dark/);
    await expect(toggle).toHaveAttribute("aria-label", /dark/i);

    // dark -> system
    await toggle.click();
    await expect(toggle).toHaveAttribute("aria-label", /system/i);
    await page.emulateMedia({ colorScheme: "dark" });
    await expect(page.locator("html")).toHaveClass(/dark/);
    await page.emulateMedia({ colorScheme: "light" });
    await expect(page.locator("html")).not.toHaveClass(/dark/);

    // system -> light
    await toggle.click();
    await expect(toggle).toHaveAttribute("aria-label", /light/i);
    await expect(page.locator("html")).not.toHaveClass(/dark/);
  });

  test('theme choice persists to themeStore key "theme"', async ({ page }) => {
    // Seed only once: the init script re-runs on reload, so guard it to avoid
    // clobbering the value the toggle persists below.
    await page.addInitScript(() => {
      if (!localStorage.getItem("theme")) {
        localStorage.setItem(
          "theme",
          JSON.stringify({ state: { theme: "light" }, version: 0 })
        );
      }
    });
    await page.goto("/dashboard");

    const toggle = page
      .locator('[data-testid="theme-toggle-slot"]')
      .getByRole("button");
    await toggle.click();
    await expect(page.locator("html")).toHaveClass(/dark/);

    const stored = await page.evaluate(() => localStorage.getItem("theme"));
    expect(stored).not.toBeNull();
    expect(JSON.parse(stored as string).state.theme).toBe("dark");

    await page.reload();
    await expect(page.locator("html")).toHaveClass(/dark/);
  });

  test("language switcher opens flag grid in a Modal with traffic lights; Escape and overlay close it; body scroll locks", async ({
    page,
  }) => {
    await page.goto("/dashboard");

    const trigger = page
      .locator('[data-testid="language-switcher-slot"]')
      .getByRole("button");
    await expect(trigger).toBeVisible();

    await trigger.click();
    const dialog = page.locator('[role="dialog"]');
    await expect(dialog).toBeVisible();
    await expect(
      page.locator('[data-testid="modal-traffic-lights"]')
    ).toBeVisible();
    const flagButtons = dialog.getByRole("button");
    expect(await flagButtons.count()).toBeGreaterThanOrEqual(8);
    await expect(page.locator("body")).toHaveCSS("overflow", "hidden");

    // Escape closes and restores scroll
    await page.keyboard.press("Escape");
    await expect(dialog).toHaveCount(0);
    await expect(page.locator("body")).not.toHaveCSS("overflow", "hidden");

    // reopen, overlay click closes (click a corner so the press lands on the
    // overlay backdrop, not the centered dialog panel).
    await trigger.click();
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    await page
      .locator('[data-testid="modal-overlay"]')
      .click({ position: { x: 5, y: 5 } });
    await expect(page.locator('[role="dialog"]')).toHaveCount(0);
  });

  test("selecting a flag POSTs /api/locale", async ({ page }) => {
    const posts: Array<{ method: string; body: string }> = [];
    await page.route("**/api/locale", async (route) => {
      const request = route.request();
      posts.push({ method: request.method(), body: request.postData() ?? "" });
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { locale: "pt-BR" }, error: null }),
      });
    });

    await page.goto("/dashboard");

    const trigger = page
      .locator('[data-testid="language-switcher-slot"]')
      .getByRole("button");
    await trigger.click();
    const dialog = page.locator('[role="dialog"]');
    await expect(dialog).toBeVisible();

    const ptButton = dialog.getByRole("button", { name: /pt-BR/i });
    await ptButton.click();

    await expect(dialog).toHaveCount(0);
    const localePosts = posts.filter((p) => p.method === "POST");
    expect(localePosts).toHaveLength(1);
    expect(JSON.parse(localePosts[0].body)).toEqual({ locale: "pt-BR" });
  });

  test("logout slot is still empty", async ({ page }) => {
    await page.goto("/dashboard");
    const slot = page.getByTestId("logout-slot");
    await expect(slot).toBeAttached();
    await expect(slot).toHaveText("");
  });
});
