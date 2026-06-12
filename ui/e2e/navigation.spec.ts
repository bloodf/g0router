import { test, expect } from "@playwright/test";

test.describe("Navigation", () => {
  test("root redirects to /dashboard", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL(/\/dashboard$/);
  });

  test("sidebar renders logo, traffic lights, and all 29 nav items", async ({ page }) => {
    await page.goto("/dashboard");
    await expect(page.getByTestId("traffic-lights")).toBeVisible();
    const navLinks = page.locator('[data-testid="desktop-sidebar"] nav a[href^="/"]');
    await expect(navLinks).toHaveCount(29);

    for (const href of ["/dashboard", "/virtual-keys", "/mcp", "/console"]) {
      const link = page.locator(`nav a[href="${href}"]`).first();
      await expect(link).toBeVisible();
      await link.click();
      await expect(page).toHaveURL(new RegExp(href.replace("/", "\\/") + "$"));
    }
  });

  test("sidebar shows update badge when settingsStore.updateAvailable", async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem(
        "settings",
        JSON.stringify({ state: { updateAvailable: true, latestVersion: "2.0.0" }, version: 0 })
      );
    });
    await page.goto("/dashboard");
    const badge = page.getByTestId("update-badge");
    await expect(badge).toBeVisible();
    await expect(badge).toHaveText("2.0.0");
  });

  test("header renders title, breadcrumbs, search, and null slots", async ({ page }) => {
    await page.goto("/dashboard");
    await expect(page.getByPlaceholder(/search/i)).toBeVisible();
    await page.getByPlaceholder(/search/i).fill("hello");
    await expect(page.getByPlaceholder(/search/i)).toHaveValue("hello");

    for (const id of ["theme-toggle-slot", "language-switcher-slot", "logout-slot"]) {
      const slot = page.getByTestId(id);
      await expect(slot).toBeAttached();
      await expect(slot).toHaveText("");
    }
  });

  test("toaster is mounted", async ({ page }) => {
    await page.goto("/dashboard");
    await expect(page.locator("[data-sonner-toaster]")).toBeAttached();
    await expect(page.locator("[data-sonner-toaster]")).toHaveCount(1);
  });

  test("theme=dark in localStorage applies .dark to <html>", async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem(
        "theme",
        JSON.stringify({ state: { theme: "dark" }, version: 0 })
      );
    });
    await page.goto("/dashboard");
    await expect(page.locator("html")).toHaveClass(/dark/);
  });

  test("theme=light removes .dark even when system prefers dark", async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem(
        "theme",
        JSON.stringify({ state: { theme: "light" }, version: 0 })
      );
    });
    await page.emulateMedia({ colorScheme: "dark" });
    await page.goto("/dashboard");
    await expect(page.locator("html")).not.toHaveClass(/dark/);
  });

  test("theme=system follows prefers-color-scheme", async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.removeItem("theme");
    });
    await page.emulateMedia({ colorScheme: "dark" });
    await page.goto("/dashboard");
    await expect(page.locator("html")).toHaveClass(/dark/);

    await page.emulateMedia({ colorScheme: "light" });
    await expect(page.locator("html")).not.toHaveClass(/dark/);
  });

  test("mobile viewport hides sidebar and hamburger opens overlay", async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto("/dashboard");

    await expect(page.getByTestId("desktop-sidebar")).not.toBeVisible();
    const hamburger = page.getByTestId("mobile-hamburger");
    await expect(hamburger).toBeVisible();

    await hamburger.click();
    await expect(page.getByTestId("mobile-sidebar")).toBeVisible();
    await expect(page.getByTestId("mobile-sidebar-overlay")).toBeVisible();

    await page.getByTestId("mobile-sidebar-overlay").click();
    await expect(page.getByTestId("mobile-sidebar")).not.toBeVisible();
    await expect(page.getByTestId("mobile-sidebar-overlay")).not.toBeVisible();
  });
});
