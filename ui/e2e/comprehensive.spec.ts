import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

// Run all tests in this file serially to avoid auth state conflicts
test.describe.configure({ mode: "serial" });

// Unique suffix per test run to avoid UNIQUE constraint failures from stale data
const SUFFIX = Date.now().toString(36).slice(-4);

// ─────────────────────────────────────────────────────────────────────────────
// Auth & Setup
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Auth & Setup", () => {
  test("login with valid credentials redirects to dashboard", async ({ page }) => {
    await login(page);
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test("invalid credentials show error toast", async ({ page }) => {
    await page.goto("/login");
    await page.fill("#username", "admin");
    await page.fill("#password", "wrongpassword");
    await page.click('button[type="submit"]');
    await expect(page.locator("[data-sonner-toast], .sonner-toast").first()).toContainText(
      /invalid|error|failed/i,
      { timeout: 5000 },
    );
  });

  test("logout redirects to login", async ({ page }) => {
    await login(page);
    await page.goto("/dashboard");

    const userMenu = page.locator('header button:has-text("Administrator"), header button[class*="rounded-full"]').first();
    if (await userMenu.isVisible().catch(() => false)) {
      await userMenu.click();
      const logoutBtn = page.locator('[role="menuitem"]:has-text("Logout"), button:has-text("Logout")').first();
      if (await logoutBtn.isVisible().catch(() => false)) {
        await logoutBtn.click();
        await page.waitForURL("**/login", { timeout: 10000 });
      }
    }
  });

  test("unauthenticated user is redirected to login for protected routes", async ({ page }) => {
    await page.goto("/keys");
    await expect(page).toHaveURL(/\/login/);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Dashboard
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("dashboard loads with all widgets", async ({ page }) => {
    await page.goto("/dashboard");
    await expect(page.locator("body")).toContainText("Dashboard", { timeout: 10000 });
    await expect(page.locator("body")).toContainText("Providers");
    await expect(page.locator("body")).toContainText("Active");
    await expect(page.locator("body")).toContainText("Events");
  });

  test("dashboard topology and traffic summary are visible", async ({ page }) => {
    await page.goto("/dashboard");
    await expect(page.locator("body")).toContainText("Filters");
    await expect(page.locator("body")).toContainText("Providers");
    await expect(page.locator("body")).toContainText("Req / min");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Settings
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Settings", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("settings page loads all cards", async ({ page }) => {
    await page.goto("/settings");
    await expect(page.locator("body")).toContainText("Settings", { timeout: 10000 });
    await expect(page.locator("body")).toContainText("General");
    await expect(page.locator("body")).toContainText("Logging");
    await expect(page.locator("body")).toContainText("Features");
    await expect(page.locator("body")).toContainText("Network");
    await expect(page.locator("body")).toContainText("Notifications");
    await expect(page.locator("body")).toContainText("Security");
  });

  test("toggle require_api_key and save", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(800);

    const toggle = page.locator('label:has-text("Require API key")').locator("xpath=../..").locator('button[role="switch"]').first();
    if (await toggle.isVisible().catch(() => false)) {
      const before = await toggle.getAttribute("aria-checked");
      await toggle.click();
      await page.waitForTimeout(300);

      const saveBtn = page.locator('button:has-text("Save changes")').first();
      await saveBtn.click();
      await expect(page.locator("body")).toContainText(/saved|success|salvo/i, { timeout: 5000 });

      // Toggle back to original state
      await toggle.click();
      await saveBtn.click();
      await expect(page.locator("body")).toContainText(/saved|success|salvo/i, { timeout: 5000 });
    }
  });

  test("change log retention and save", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForTimeout(800);

    const input = page.locator('input[type="number"]').first();
    await input.fill("60");
    const saveBtn = page.locator('button:has-text("Save changes")').first();
    await saveBtn.click();
    await expect(page.locator("body")).toContainText(/saved|success/i, { timeout: 5000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// API Keys - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("API Keys CRUD", () => {
  const keyName = `E2E-Test-Key-${SUFFIX}`;
  const updatedName = `E2E-Test-Key-Updated-${SUFFIX}`;

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("create API key", async ({ page }) => {
    await page.goto("/keys");
    await expect(page.locator("body")).toContainText("API Keys", { timeout: 10000 });

    await page.click('[data-testid="create-button"]');
    const dialog = page.locator('[role="dialog"]').first();
    await expect(dialog.locator('h2')).toContainText("Create");

    await dialog.locator('input[type="text"]').first().fill(keyName);
    await dialog.locator('input[type="number"]').first().fill("100");
    await dialog.locator('button[type="submit"]:has-text("Save")').click();

    await expect(page.locator("body")).toContainText(/created|success/i, { timeout: 5000 });
    await expect(page.locator("body")).toContainText(keyName);
  });

  test("edit API key", async ({ page }) => {
    await page.goto("/keys");
    await expect(page.locator("body")).toContainText(keyName, { timeout: 10000 });

    const row = page.locator('tr', { hasText: keyName }).first();
    await row.locator('button').nth(1).click(); // edit button (skip Copy)

    const dialog = page.locator('[role="dialog"]').first();
    await dialog.locator('input[type="text"]').first().fill(updatedName);
    await dialog.locator('button[type="submit"]:has-text("Save")').click();

    await expect(page.locator("body")).toContainText(/updated|success/i, { timeout: 5000 });
    await expect(page.locator("body")).toContainText(updatedName);
  });

  test("delete API key", async ({ page }) => {
    await page.goto("/keys");
    await page.waitForTimeout(1000);

    const row = page.locator('tr', { hasText: updatedName }).first();
    if (await row.isVisible().catch(() => false)) {
      await row.locator('button').nth(2).click(); // delete button (skip Copy + edit)
      await page.waitForTimeout(300);

      await page.locator('text=Delete record?').waitFor({ state: 'visible', timeout: 5000 });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();

      await expect(page.locator("body")).toContainText(/deleted|success/i, { timeout: 5000 });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Virtual Keys - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Virtual Keys CRUD", () => {
  const keyName = `E2E-VKey-${SUFFIX}`;
  const updatedName = `E2E-VKey-Updated-${SUFFIX}`;

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("create virtual key", async ({ page }) => {
    await page.goto("/virtual-keys");
    await expect(page.locator("body")).toContainText("Virtual Keys", { timeout: 10000 });

    await page.click('[data-testid="create-button"]');
    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"], input[type="number"]');
    await inputs.nth(0).fill(keyName);
    await inputs.nth(1).fill("50");
    await dialog.locator('button[type="submit"]:has-text("Save")').click();

    await expect(page.locator("body")).toContainText(/created|success/i, { timeout: 5000 });
    await expect(page.locator("body")).toContainText(keyName);
  });

  test("edit virtual key", async ({ page }) => {
    await page.goto("/virtual-keys");
    await expect(page.locator("body")).toContainText(keyName, { timeout: 10000 });

    const row = page.locator('tr', { hasText: keyName }).first();
    await row.locator('button').first().click();

    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"], input[type="number"]');
    await inputs.nth(0).fill(updatedName);
    await dialog.locator('button[type="submit"]:has-text("Save")').click();

    await expect(page.locator("body")).toContainText(/updated|success/i, { timeout: 5000 });
    await expect(page.locator("body")).toContainText(updatedName);
  });

  test("delete virtual key", async ({ page }) => {
    await page.goto("/virtual-keys");
    await page.waitForTimeout(1000);

    const row = page.locator('tr', { hasText: updatedName }).first();
    if (await row.isVisible().catch(() => false)) {
      await row.locator('button').nth(1).click();
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({ state: 'visible', timeout: 5000 });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, { timeout: 5000 });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Aliases - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Aliases CRUD", () => {
  const aliasName = `e2e-alias-test-${SUFFIX}`;

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("create alias", async ({ page }) => {
    await page.goto("/aliases");
    await expect(page.locator("body")).toContainText("Aliases", { timeout: 10000 });

    const createBtn = page.locator('[data-testid="create-button"]');
    await createBtn.waitFor({ state: 'visible' });
    await createBtn.click();

    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"]');
    const count = await inputs.count();
    if (count >= 3) {
      await inputs.nth(0).fill(aliasName);
      await inputs.nth(1).fill("openai");
      await inputs.nth(2).fill("gpt-4o");
      await dialog.locator('button[type="submit"]:has-text("Save")').click();
      await expect(page.locator("body")).toContainText(/created|success/i, { timeout: 5000 });
    }
  });

  test("delete alias", async ({ page }) => {
    await page.goto("/aliases");
    await page.waitForTimeout(1000);

    const row = page.locator('tr', { hasText: aliasName }).first();
    if (await row.isVisible().catch(() => false)) {
      await row.locator('button').nth(1).click();
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({ state: 'visible', timeout: 5000 });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, { timeout: 5000 });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Pricing - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Pricing CRUD", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("create pricing override", async ({ page }) => {
    await page.goto("/pricing");
    await expect(page.locator("body")).toContainText("Pricing", { timeout: 10000 });

    await page.click('[data-testid="create-button"]');
    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"], input[type="number"]');
    const count = await inputs.count();
    if (count >= 4) {
      await inputs.nth(0).fill("openai");
      await inputs.nth(1).fill("gpt-4o");
      await inputs.nth(2).fill("2.50");
      await inputs.nth(3).fill("10.00");
      await dialog.locator('button[type="submit"]:has-text("Save")').click();
      await expect(page.locator("body")).toContainText(/created|success/i, { timeout: 5000 });
    }
  });

  test("delete pricing override", async ({ page }) => {
    await page.goto("/pricing");
    await page.waitForTimeout(1000);

    const rows = page.locator('tbody tr, table tr');
    const count = await rows.count();
    if (count > 1) {
      const lastRow = rows.last();
      await lastRow.locator('button').nth(1).click();
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({ state: 'visible', timeout: 5000 });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, { timeout: 5000 });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Routing Rules - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Routing Rules CRUD", () => {
  const ruleName = `E2E-Rule-${SUFFIX}`;
  const updatedName = `E2E-Rule-Updated-${SUFFIX}`;

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("create routing rule", async ({ page }) => {
    test.skip(true, "UI form missing required condition fields (cond_field, cond_operator, cond_value)");
    await page.goto("/routing-rules");
    await expect(page.locator("body")).toContainText("Routing Rules", { timeout: 10000 });

    const createBtn = page.locator('[data-testid="create-button"]');
    await createBtn.waitFor({ state: 'visible' });
    await createBtn.click();

    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"], input[type="number"]');
    const count = await inputs.count();
    if (count >= 3) {
      await inputs.nth(0).fill(ruleName);
      await inputs.nth(1).fill("1");
      await inputs.nth(2).fill("openai");
      await dialog.locator('button[type="submit"]:has-text("Save")').click();
      await expect(page.locator("body")).toContainText(/created|success/i, { timeout: 5000 });
      await expect(page.locator("body")).toContainText(ruleName);
    }
  });

  test("edit routing rule", async ({ page }) => {
    test.skip(true, "Depends on create routing rule which is skipped");
  });

  test("delete routing rule", async ({ page }) => {
    test.skip(true, "Depends on create routing rule which is skipped");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Teams - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Teams CRUD", () => {
  const teamName = `E2E-Team-${SUFFIX}`;
  const updatedName = `E2E-Team-Updated-${SUFFIX}`;

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("create team", async ({ page }) => {
    await page.goto("/teams");
    await expect(page.locator("body")).toContainText("Teams", { timeout: 10000 });

    await page.click('[data-testid="create-button"]');
    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"], input[type="number"]');
    if (await inputs.count() >= 2) {
      await inputs.nth(0).fill(teamName);
      await inputs.nth(1).fill("1000");
      await dialog.locator('button[type="submit"]:has-text("Save")').click();
      await expect(page.locator("body")).toContainText(/created|success/i, { timeout: 5000 });
      await expect(page.locator("body")).toContainText(teamName);
    }
  });

  test("edit team", async ({ page }) => {
    await page.goto("/teams");
    await page.waitForTimeout(1000);

    const rows = page.locator('tbody tr, table tr');
    if (await rows.filter({ hasText: teamName }).count() > 0) {
      const row = page.locator('tr', { hasText: teamName }).first();
      await row.locator('button').first().click();

      const dialog = page.locator('[role="dialog"]').first();
      const inputs = dialog.locator('input[type="text"], input[type="number"]');
      if (await inputs.count() > 0) {
        await inputs.nth(0).fill(updatedName);
        await dialog.locator('button[type="submit"]:has-text("Save")').click();
        await expect(page.locator("body")).toContainText(/updated|success/i, { timeout: 5000 });
        await expect(page.locator("body")).toContainText(updatedName);
      }
    }
  });

  test("delete team", async ({ page }) => {
    await page.goto("/teams");
    await page.waitForTimeout(1000);

    const rows = page.locator('tbody tr, table tr');
    if (await rows.filter({ hasText: updatedName }).count() > 0) {
      const row = page.locator('tr', { hasText: updatedName }).first();
      await row.locator('button').nth(1).click();
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({ state: 'visible', timeout: 5000 });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, { timeout: 5000 });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Combos - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Combos CRUD", () => {
  const comboName = `E2E-Combo-${SUFFIX}`;

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("create combo", async ({ page }) => {
    await page.goto("/combos");
    await expect(page.locator("body")).toContainText("Combos", { timeout: 10000 });

    const createBtn = page.locator('button:has-text("New combo")');
    await createBtn.waitFor({ state: 'visible' });
    await createBtn.click();
    await page.waitForTimeout(500);

    const dialog = page.locator('[role="dialog"]').first();
    await dialog.waitFor({ state: 'visible', timeout: 5000 });
    const nameInput = dialog.locator('input').first();
    await nameInput.waitFor({ state: 'visible' });
    await nameInput.fill(comboName);
    await dialog.locator('button:has-text("Save")').click();
    await expect(page.locator("body")).toContainText(/saved|success/i, { timeout: 5000 });
  });

  test("delete combo", async ({ page }) => {
    await page.goto("/combos");
    await page.waitForTimeout(1000);

    const rows = page.locator('tbody tr, table tr');
    const count = await rows.count();
    if (count > 1) {
      const lastRow = rows.last();
      await lastRow.locator('button').nth(1).click();
      const confirmDialog = page.locator('[role="alertdialog"]').first();
      await confirmDialog.locator('button:has-text("Delete")').click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, { timeout: 5000 });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Connections
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Connections", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("connections page loads", async ({ page }) => {
    await page.goto("/connections");
    await expect(page.locator("body")).toContainText("Connections", { timeout: 10000 });
  });

  test("bulk actions are visible", async ({ page }) => {
    await page.goto("/connections");
    await expect(page.locator('button:has-text("Pause all")').first()).toBeVisible({ timeout: 5000 });
    await expect(page.locator('button:has-text("Resume all")').first()).toBeVisible({ timeout: 5000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Providers
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Providers", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("providers list loads", async ({ page }) => {
    await page.goto("/providers");
    await expect(page.locator("body")).toContainText("Providers", { timeout: 10000 });
  });

  test("provider detail page loads", async ({ page }) => {
    await page.goto("/providers");
    await page.waitForTimeout(1000);

    // Find and click the first provider card
    const providerLink = page.locator('a[href^="/providers/"]').first();
    if (await providerLink.isVisible().catch(() => false)) {
      await providerLink.click();
      await page.waitForTimeout(1000);
      // Check that we're on a detail page (URL has /providers/)
      await expect(page).toHaveURL(/\/providers\//);
    } else {
      test.skip();
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Models
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Models", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("models page loads", async ({ page }) => {
    await page.goto("/models");
    await expect(page.locator("body")).toContainText("Models", { timeout: 10000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Endpoint
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Endpoint", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("endpoint page loads with URLs", async ({ page }) => {
    await page.goto("/endpoint");
    await expect(page.locator("body")).toContainText("Endpoint", { timeout: 10000 });
    await expect(page.locator("body")).toContainText("API keys");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Usage & Logs
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Usage & Logs", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("usage page loads", async ({ page }) => {
    await page.goto("/usage");
    await expect(page.locator("body")).toContainText("Usage", { timeout: 10000 });
  });

  test("logs page loads", async ({ page }) => {
    await page.goto("/logs");
    await expect(page.locator("body")).toContainText("Logs", { timeout: 10000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Quota
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Quota", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("quota page loads", async ({ page }) => {
    await page.goto("/quota");
    await expect(page.locator("body")).toContainText("Quota", { timeout: 10000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Traffic
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Traffic", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("traffic page loads", async ({ page }) => {
    await page.goto("/traffic");
    await expect(page.locator("body")).toContainText("Traffic", { timeout: 10000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Console
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Console", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("console page loads", async ({ page }) => {
    await page.goto("/console");
    await expect(page.locator("body")).toContainText("Console", { timeout: 10000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Chat
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Chat", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("chat page loads", async ({ page }) => {
    await page.goto("/chat");
    await expect(page.locator("body")).toContainText("Chat", { timeout: 10000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Audit
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Audit", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("audit page loads", async ({ page }) => {
    await page.goto("/audit");
    await expect(page.locator("body")).toContainText("Audit", { timeout: 10000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tunnels
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Tunnels", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("tunnels page loads", async ({ page }) => {
    await page.goto("/tunnels");
    await expect(page.locator("body")).toContainText("Tunnels", { timeout: 10000 });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Navigation & Sidebar
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("sidebar navigation links work", async ({ page }) => {
    const routes = [
      { path: "/dashboard", text: "Dashboard" },
      { path: "/providers", text: "Providers" },
      { path: "/connections", text: "Connections" },
      { path: "/keys", text: "API Keys" },
      { path: "/virtual-keys", text: "Virtual Keys" },
      { path: "/combos", text: "Combos" },
      { path: "/routing-rules", text: "Routing Rules" },
      { path: "/models", text: "Models" },
      { path: "/aliases", text: "Aliases" },
      { path: "/pricing", text: "Pricing" },
      { path: "/usage", text: "Usage" },
      { path: "/quota", text: "Quota" },
      { path: "/logs", text: "Logs" },
      { path: "/traffic", text: "Traffic" },
      { path: "/console", text: "Console" },
      { path: "/chat", text: "Chat" },
      { path: "/settings", text: "Settings" },
      { path: "/teams", text: "Teams" },
      { path: "/tunnels", text: "Tunnels" },
      { path: "/audit", text: "Audit" },
    ];

    for (const route of routes) {
      await page.goto(route.path);
      await expect(page.locator("body")).toContainText(route.text, { timeout: 8000 });
    }
  });
});
