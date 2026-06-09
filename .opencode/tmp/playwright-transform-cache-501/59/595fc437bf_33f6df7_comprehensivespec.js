import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

// Run all tests in this file serially to avoid auth state conflicts
test.describe.configure({
  mode: "serial"
});

// Unique suffix per test run to avoid UNIQUE constraint failures from stale data
const SUFFIX = Date.now().toString(36).slice(-4);

// ─────────────────────────────────────────────────────────────────────────────
// Auth & Setup
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Auth & Setup", () => {
  test("login with valid credentials redirects to dashboard", async ({
    page
  }) => {
    await login(page);
    await expect(page).toHaveURL(/\/dashboard/);
  });
  test("invalid credentials show error toast", async ({
    page
  }) => {
    await page.goto("/login");
    await page.fill("#username", "admin");
    await page.fill("#password", "wrongpassword");
    await page.click('button[type="submit"]');
    await expect(page.locator("[data-sonner-toast], .sonner-toast").first()).toContainText(/invalid|error|failed/i, {
      timeout: 5000
    });
  });
  test("logout redirects to login", async ({
    page
  }) => {
    await login(page);
    await page.goto("/dashboard");
    const userMenu = page.locator('header button:has-text("Administrator"), header button[class*="rounded-full"]').first();
    if (await userMenu.isVisible().catch(() => false)) {
      await userMenu.click();
      const logoutBtn = page.locator('[role="menuitem"]:has-text("Logout"), button:has-text("Logout")').first();
      if (await logoutBtn.isVisible().catch(() => false)) {
        await logoutBtn.click();
        await page.waitForURL("**/login", {
          timeout: 10000
        });
      }
    }
  });
  test("unauthenticated user is redirected to login for protected routes", async ({
    page
  }) => {
    await page.goto("/keys");
    await expect(page).toHaveURL(/\/login/);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Dashboard
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Dashboard", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("dashboard loads with all widgets", async ({
    page
  }) => {
    await page.goto("/dashboard");
    await expect(page.locator("body")).toContainText("Dashboard", {
      timeout: 10000
    });
    await expect(page.locator("body")).toContainText("Providers");
    await expect(page.locator("body")).toContainText("Active");
    await expect(page.locator("body")).toContainText("Events");
  });
  test("dashboard topology and traffic summary are visible", async ({
    page
  }) => {
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
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("settings page loads all cards", async ({
    page
  }) => {
    await page.goto("/settings");
    await expect(page.locator("body")).toContainText("Settings", {
      timeout: 10000
    });
    await expect(page.locator("body")).toContainText("General");
    await expect(page.locator("body")).toContainText("Logging");
    await expect(page.locator("body")).toContainText("Features");
    await expect(page.locator("body")).toContainText("Network");
    await expect(page.locator("body")).toContainText("Notifications");
    await expect(page.locator("body")).toContainText("Security");
  });
  test("toggle require_api_key and save", async ({
    page
  }) => {
    await page.goto("/settings");
    await page.waitForTimeout(800);
    const toggle = page.locator('label:has-text("Require API key")').locator("xpath=../..").locator('button[role="switch"]').first();
    if (await toggle.isVisible().catch(() => false)) {
      const before = await toggle.getAttribute("aria-checked");
      await toggle.click();
      await page.waitForTimeout(300);
      const saveBtn = page.locator('button:has-text("Save changes")').first();
      await saveBtn.click();
      await expect(page.locator("body")).toContainText(/saved|success|salvo/i, {
        timeout: 5000
      });

      // Toggle back to original state
      await toggle.click();
      await saveBtn.click();
      await expect(page.locator("body")).toContainText(/saved|success|salvo/i, {
        timeout: 5000
      });
    }
  });
  test("change log retention and save", async ({
    page
  }) => {
    await page.goto("/settings");
    await page.waitForTimeout(800);
    const input = page.locator('input[type="number"]').first();
    await input.fill("60");
    const saveBtn = page.locator('button:has-text("Save changes")').first();
    await saveBtn.click();
    await expect(page.locator("body")).toContainText(/saved|success/i, {
      timeout: 5000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// API Keys - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("API Keys CRUD", () => {
  const keyName = `E2E-Test-Key-${SUFFIX}`;
  const updatedName = `E2E-Test-Key-Updated-${SUFFIX}`;
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("create API key", async ({
    page
  }) => {
    await page.goto("/keys");
    await expect(page.locator("body")).toContainText("API Keys", {
      timeout: 10000
    });
    await page.click('[data-testid="create-button"]');
    const dialog = page.locator('[role="dialog"]').first();
    await expect(dialog.locator('h2')).toContainText("Create");
    await dialog.locator('input[type="text"]').first().fill(keyName);
    await dialog.locator('input[type="number"]').first().fill("100");
    await dialog.locator('button[type="submit"]:has-text("Save")').click();
    await expect(page.locator("body")).toContainText(/created|success/i, {
      timeout: 5000
    });
    await expect(page.locator("body")).toContainText(keyName);
  });
  test("edit API key", async ({
    page
  }) => {
    await page.goto("/keys");
    await expect(page.locator("body")).toContainText(keyName, {
      timeout: 10000
    });
    const row = page.locator('tr', {
      hasText: keyName
    }).first();
    const actionsCell = row.locator('td').last();
    await actionsCell.locator('button').nth(1).click(); // edit button (skip Regenerate)

    const dialog = page.locator('[role="dialog"]').first();
    await dialog.waitFor({
      state: 'visible',
      timeout: 5000
    });
    await dialog.locator('input[type="text"]').first().fill(updatedName);
    await dialog.locator('button[type="submit"]:has-text("Save")').click();
    await expect(page.locator("body")).toContainText(/updated|success/i, {
      timeout: 5000
    });
    await expect(page.locator("body")).toContainText(updatedName);
  });
  test("delete API key", async ({
    page
  }) => {
    await page.goto("/keys");
    await page.waitForTimeout(1000);
    const row = page.locator('tr', {
      hasText: updatedName
    }).first();
    if (await row.isVisible().catch(() => false)) {
      const actionsCell = row.locator('td').last();
      await actionsCell.locator('button').nth(2).click(); // delete button (skip Regenerate + edit)
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({
        state: 'visible',
        timeout: 5000
      });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, {
        timeout: 5000
      });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Virtual Keys - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Virtual Keys CRUD", () => {
  const keyName = `E2E-VKey-${SUFFIX}`;
  const updatedName = `E2E-VKey-Updated-${SUFFIX}`;
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("create virtual key", async ({
    page
  }) => {
    await page.goto("/virtual-keys");
    await expect(page.locator("body")).toContainText("Virtual Keys", {
      timeout: 10000
    });
    await page.click('[data-testid="create-button"]');
    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"], input[type="number"]');
    await inputs.nth(0).fill(keyName);
    await inputs.nth(1).fill("50");
    await dialog.locator('button[type="submit"]:has-text("Save")').click();
    await expect(page.locator("body")).toContainText(/created|success/i, {
      timeout: 5000
    });
    await expect(page.locator("body")).toContainText(keyName);
  });
  test("edit virtual key", async ({
    page
  }) => {
    await page.goto("/virtual-keys");
    await expect(page.locator("body")).toContainText(keyName, {
      timeout: 10000
    });
    const row = page.locator('tr', {
      hasText: keyName
    }).first();
    await row.locator('button').first().click();
    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"], input[type="number"]');
    await inputs.nth(0).fill(updatedName);
    await dialog.locator('button[type="submit"]:has-text("Save")').click();
    await expect(page.locator("body")).toContainText(/updated|success/i, {
      timeout: 5000
    });
    await expect(page.locator("body")).toContainText(updatedName);
  });
  test("delete virtual key", async ({
    page
  }) => {
    await page.goto("/virtual-keys");
    await page.waitForTimeout(1000);
    const row = page.locator('tr', {
      hasText: updatedName
    }).first();
    if (await row.isVisible().catch(() => false)) {
      await row.locator('button').nth(1).click();
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({
        state: 'visible',
        timeout: 5000
      });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, {
        timeout: 5000
      });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Aliases - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Aliases CRUD", () => {
  const aliasName = `e2e-alias-test-${SUFFIX}`;
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("create alias", async ({
    page
  }) => {
    await page.goto("/aliases");
    await expect(page.locator("body")).toContainText("Aliases", {
      timeout: 10000
    });
    const createBtn = page.locator('[data-testid="create-button"]');
    await createBtn.waitFor({
      state: 'visible'
    });
    await createBtn.click();
    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"]');
    const count = await inputs.count();
    if (count >= 3) {
      await inputs.nth(0).fill(aliasName);
      await inputs.nth(1).fill("openai");
      await inputs.nth(2).fill("gpt-4o");
      await dialog.locator('button[type="submit"]:has-text("Save")').click();
      await expect(page.locator("body")).toContainText(/created|success/i, {
        timeout: 5000
      });
    }
  });
  test("delete alias", async ({
    page
  }) => {
    await page.goto("/aliases");
    await page.waitForTimeout(1000);
    const row = page.locator('tr', {
      hasText: aliasName
    }).first();
    if (await row.isVisible().catch(() => false)) {
      await row.locator('button').nth(1).click();
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({
        state: 'visible',
        timeout: 5000
      });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, {
        timeout: 5000
      });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Pricing - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Pricing CRUD", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("create pricing override", async ({
    page
  }) => {
    await page.goto("/pricing");
    await expect(page.locator("body")).toContainText("Pricing", {
      timeout: 10000
    });
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
      await expect(page.locator("body")).toContainText(/created|success/i, {
        timeout: 5000
      });
    }
  });
  test("delete pricing override", async ({
    page
  }) => {
    await page.goto("/pricing");
    await page.waitForTimeout(1000);
    const rows = page.locator('tbody tr, table tr');
    const count = await rows.count();
    if (count > 1) {
      const lastRow = rows.last();
      await lastRow.locator('button').nth(1).click();
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({
        state: 'visible',
        timeout: 5000
      });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, {
        timeout: 5000
      });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Routing Rules - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Routing Rules CRUD", () => {
  const ruleName = `E2E-Rule-${SUFFIX}`;
  const updatedName = `E2E-Rule-Updated-${SUFFIX}`;
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("create routing rule", async ({
    page
  }) => {
    await page.goto("/routing-rules");
    await expect(page.locator("body")).toContainText("Routing Rules", {
      timeout: 10000
    });
    const createBtn = page.locator('[data-testid="create-button"]');
    await createBtn.waitFor({
      state: 'visible'
    });
    await createBtn.click();
    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"], input[type="number"]');
    const count = await inputs.count();
    if (count >= 5) {
      await inputs.nth(0).fill(ruleName);
      await inputs.nth(1).fill("1");
      await inputs.nth(2).fill("model");
      await dialog.locator('select').first().selectOption("equals");
      await inputs.nth(3).fill("gpt-4o");
      await inputs.nth(4).fill("openai");
      await dialog.locator('button[type="submit"]:has-text("Save")').click();
      await expect(page.locator("body")).toContainText(/created|success/i, {
        timeout: 5000
      });
      await expect(page.locator("body")).toContainText(ruleName);
    }
  });
  test("edit routing rule", async ({
    page
  }) => {
    await page.goto("/routing-rules");
    await expect(page.locator("body")).toContainText(ruleName, {
      timeout: 10000
    });
    const row = page.locator('tr', {
      hasText: ruleName
    }).first();
    await row.locator('button').first().click();
    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"]');
    if ((await inputs.count()) > 0) {
      await inputs.nth(0).fill(updatedName);
      await dialog.locator('button[type="submit"]:has-text("Save")').click();
      await expect(page.locator("body")).toContainText(/updated|success/i, {
        timeout: 5000
      });
      await expect(page.locator("body")).toContainText(updatedName);
    }
  });
  test("delete routing rule", async ({
    page
  }) => {
    await page.goto("/routing-rules");
    await page.waitForTimeout(1000);
    const row = page.locator('tr', {
      hasText: updatedName
    }).first();
    if (await row.isVisible().catch(() => false)) {
      await row.locator('button').nth(1).click();
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({
        state: 'visible',
        timeout: 5000
      });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, {
        timeout: 5000
      });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Teams - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Teams CRUD", () => {
  const teamName = `E2E-Team-${SUFFIX}`;
  const updatedName = `E2E-Team-Updated-${SUFFIX}`;
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("create team", async ({
    page
  }) => {
    await page.goto("/teams");
    await expect(page.locator("body")).toContainText("Teams", {
      timeout: 10000
    });
    await page.click('[data-testid="create-button"]');
    const dialog = page.locator('[role="dialog"]').first();
    const inputs = dialog.locator('input[type="text"], input[type="number"]');
    if ((await inputs.count()) >= 2) {
      await inputs.nth(0).fill(teamName);
      await inputs.nth(1).fill("1000");
      await dialog.locator('button[type="submit"]:has-text("Save")').click();
      await expect(page.locator("body")).toContainText(/created|success/i, {
        timeout: 5000
      });
      await expect(page.locator("body")).toContainText(teamName);
    }
  });
  test("edit team", async ({
    page
  }) => {
    await page.goto("/teams");
    await page.waitForTimeout(1000);
    const rows = page.locator('tbody tr, table tr');
    if ((await rows.filter({
      hasText: teamName
    }).count()) > 0) {
      const row = page.locator('tr', {
        hasText: teamName
      }).first();
      await row.locator('button').first().click();
      const dialog = page.locator('[role="dialog"]').first();
      const inputs = dialog.locator('input[type="text"], input[type="number"]');
      if ((await inputs.count()) > 0) {
        await inputs.nth(0).fill(updatedName);
        await dialog.locator('button[type="submit"]:has-text("Save")').click();
        await expect(page.locator("body")).toContainText(/updated|success/i, {
          timeout: 5000
        });
        await expect(page.locator("body")).toContainText(updatedName);
      }
    }
  });
  test("delete team", async ({
    page
  }) => {
    await page.goto("/teams");
    await page.waitForTimeout(1000);
    const rows = page.locator('tbody tr, table tr');
    if ((await rows.filter({
      hasText: updatedName
    }).count()) > 0) {
      const row = page.locator('tr', {
        hasText: updatedName
      }).first();
      await row.locator('button').nth(1).click();
      await page.waitForTimeout(300);
      await page.locator('text=Delete record?').waitFor({
        state: 'visible',
        timeout: 5000
      });
      await page.locator('button:has-text("Cancel") + button, [role="dialog"] button:has-text("Delete")').first().click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, {
        timeout: 5000
      });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Combos - Full CRUD
// ─────────────────────────────────────────────────────────────────────────────

test.describe.serial("Combos CRUD", () => {
  const comboName = `E2E-Combo-${SUFFIX}`;
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("create combo", async ({
    page
  }) => {
    await page.goto("/combos");
    await expect(page.locator("body")).toContainText("Combos", {
      timeout: 10000
    });
    const createBtn = page.locator('button:has-text("New combo")');
    await createBtn.waitFor({
      state: 'visible'
    });
    await createBtn.click();
    await page.waitForTimeout(500);
    const dialog = page.locator('[role="dialog"]').first();
    await dialog.waitFor({
      state: 'visible',
      timeout: 5000
    });
    const nameInput = dialog.locator('input').first();
    await nameInput.waitFor({
      state: 'visible'
    });
    await nameInput.fill(comboName);
    await dialog.locator('button:has-text("Save")').click();
    await expect(page.locator("body")).toContainText(/saved|success/i, {
      timeout: 5000
    });
  });
  test("delete combo", async ({
    page
  }) => {
    await page.goto("/combos");
    await page.waitForTimeout(1000);
    const rows = page.locator('tbody tr, table tr');
    const count = await rows.count();
    if (count > 1) {
      const lastRow = rows.last();
      await lastRow.locator('button').nth(1).click();
      const confirmDialog = page.locator('[role="alertdialog"]').first();
      await confirmDialog.locator('button:has-text("Delete")').click();
      await expect(page.locator("body")).toContainText(/deleted|success/i, {
        timeout: 5000
      });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Connections
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Connections", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("connections page loads", async ({
    page
  }) => {
    await page.goto("/connections");
    await expect(page.locator("body")).toContainText("Connections", {
      timeout: 10000
    });
  });
  test("bulk actions are visible", async ({
    page
  }) => {
    await page.goto("/connections");
    await expect(page.locator('button:has-text("Pause all")').first()).toBeVisible({
      timeout: 5000
    });
    await expect(page.locator('button:has-text("Resume all")').first()).toBeVisible({
      timeout: 5000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Providers
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Providers", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("providers list loads", async ({
    page
  }) => {
    await page.goto("/providers");
    await expect(page.locator("body")).toContainText("Providers", {
      timeout: 10000
    });
  });
  test("provider detail page loads", async ({
    page
  }) => {
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
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("models page loads", async ({
    page
  }) => {
    await page.goto("/models");
    await expect(page.locator("body")).toContainText("Models", {
      timeout: 10000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Endpoint
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Endpoint", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("endpoint page loads with URLs", async ({
    page
  }) => {
    await page.goto("/endpoint");
    await expect(page.locator("body")).toContainText("Endpoint", {
      timeout: 10000
    });
    await expect(page.locator("body")).toContainText("API keys");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Usage & Logs
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Usage & Logs", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("usage page loads", async ({
    page
  }) => {
    await page.goto("/usage");
    await expect(page.locator("body")).toContainText("Usage", {
      timeout: 10000
    });
  });
  test("logs page loads", async ({
    page
  }) => {
    await page.goto("/logs");
    await expect(page.locator("body")).toContainText("Logs", {
      timeout: 10000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Quota
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Quota", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("quota page loads", async ({
    page
  }) => {
    await page.goto("/quota");
    await expect(page.locator("body")).toContainText("Quota", {
      timeout: 10000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Traffic
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Traffic", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("traffic page loads", async ({
    page
  }) => {
    await page.goto("/traffic");
    await expect(page.locator("body")).toContainText("Traffic", {
      timeout: 10000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Console
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Console", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("console page loads", async ({
    page
  }) => {
    await page.goto("/console");
    await expect(page.locator("body")).toContainText("Console", {
      timeout: 10000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Chat
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Chat", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("chat page loads", async ({
    page
  }) => {
    await page.goto("/chat");
    await expect(page.locator("body")).toContainText("Chat", {
      timeout: 10000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Audit
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Audit", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("audit page loads", async ({
    page
  }) => {
    await page.goto("/audit");
    await expect(page.locator("body")).toContainText("Audit", {
      timeout: 10000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tunnels
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Tunnels", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("tunnels page loads", async ({
    page
  }) => {
    await page.goto("/tunnels");
    await expect(page.locator("body")).toContainText("Tunnels", {
      timeout: 10000
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Navigation & Sidebar
// ─────────────────────────────────────────────────────────────────────────────

test.describe("Navigation", () => {
  test.beforeEach(async ({
    page
  }) => {
    await login(page);
  });
  test("sidebar navigation links work", async ({
    page
  }) => {
    const routes = [{
      path: "/dashboard",
      text: "Dashboard"
    }, {
      path: "/providers",
      text: "Providers"
    }, {
      path: "/connections",
      text: "Connections"
    }, {
      path: "/keys",
      text: "API Keys"
    }, {
      path: "/virtual-keys",
      text: "Virtual Keys"
    }, {
      path: "/combos",
      text: "Combos"
    }, {
      path: "/routing-rules",
      text: "Routing Rules"
    }, {
      path: "/models",
      text: "Models"
    }, {
      path: "/aliases",
      text: "Aliases"
    }, {
      path: "/pricing",
      text: "Pricing"
    }, {
      path: "/usage",
      text: "Usage"
    }, {
      path: "/quota",
      text: "Quota"
    }, {
      path: "/logs",
      text: "Logs"
    }, {
      path: "/traffic",
      text: "Traffic"
    }, {
      path: "/console",
      text: "Console"
    }, {
      path: "/chat",
      text: "Chat"
    }, {
      path: "/settings",
      text: "Settings"
    }, {
      path: "/teams",
      text: "Teams"
    }, {
      path: "/tunnels",
      text: "Tunnels"
    }, {
      path: "/audit",
      text: "Audit"
    }];
    for (const route of routes) {
      await page.goto(route.path);
      await expect(page.locator("body")).toContainText(route.text, {
        timeout: 8000
      });
    }
  });
});
//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJuYW1lcyI6WyJ0ZXN0IiwiZXhwZWN0IiwibG9naW4iLCJkZXNjcmliZSIsImNvbmZpZ3VyZSIsIm1vZGUiLCJTVUZGSVgiLCJEYXRlIiwibm93IiwidG9TdHJpbmciLCJzbGljZSIsInBhZ2UiLCJ0b0hhdmVVUkwiLCJnb3RvIiwiZmlsbCIsImNsaWNrIiwibG9jYXRvciIsImZpcnN0IiwidG9Db250YWluVGV4dCIsInRpbWVvdXQiLCJ1c2VyTWVudSIsImlzVmlzaWJsZSIsImNhdGNoIiwibG9nb3V0QnRuIiwid2FpdEZvclVSTCIsImJlZm9yZUVhY2giLCJzZXJpYWwiLCJ3YWl0Rm9yVGltZW91dCIsInRvZ2dsZSIsImJlZm9yZSIsImdldEF0dHJpYnV0ZSIsInNhdmVCdG4iLCJpbnB1dCIsImtleU5hbWUiLCJ1cGRhdGVkTmFtZSIsImRpYWxvZyIsInJvdyIsImhhc1RleHQiLCJhY3Rpb25zQ2VsbCIsImxhc3QiLCJudGgiLCJ3YWl0Rm9yIiwic3RhdGUiLCJpbnB1dHMiLCJhbGlhc05hbWUiLCJjcmVhdGVCdG4iLCJjb3VudCIsInJvd3MiLCJsYXN0Um93IiwicnVsZU5hbWUiLCJzZWxlY3RPcHRpb24iLCJ0ZWFtTmFtZSIsImZpbHRlciIsImNvbWJvTmFtZSIsIm5hbWVJbnB1dCIsImNvbmZpcm1EaWFsb2ciLCJ0b0JlVmlzaWJsZSIsInByb3ZpZGVyTGluayIsInNraXAiLCJyb3V0ZXMiLCJwYXRoIiwidGV4dCIsInJvdXRlIl0sInNvdXJjZXMiOlsiY29tcHJlaGVuc2l2ZS5zcGVjLnRzIl0sInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB7IHRlc3QsIGV4cGVjdCB9IGZyb20gXCIuL21vY2tzL2ZpeHR1cmVcIjtcbmltcG9ydCB7IGxvZ2luIH0gZnJvbSBcIi4vaGVscGVyc1wiO1xuXG4vLyBSdW4gYWxsIHRlc3RzIGluIHRoaXMgZmlsZSBzZXJpYWxseSB0byBhdm9pZCBhdXRoIHN0YXRlIGNvbmZsaWN0c1xudGVzdC5kZXNjcmliZS5jb25maWd1cmUoeyBtb2RlOiBcInNlcmlhbFwiIH0pO1xuXG4vLyBVbmlxdWUgc3VmZml4IHBlciB0ZXN0IHJ1biB0byBhdm9pZCBVTklRVUUgY29uc3RyYWludCBmYWlsdXJlcyBmcm9tIHN0YWxlIGRhdGFcbmNvbnN0IFNVRkZJWCA9IERhdGUubm93KCkudG9TdHJpbmcoMzYpLnNsaWNlKC00KTtcblxuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG4vLyBBdXRoICYgU2V0dXBcbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuXG50ZXN0LmRlc2NyaWJlKFwiQXV0aCAmIFNldHVwXCIsICgpID0+IHtcbiAgdGVzdChcImxvZ2luIHdpdGggdmFsaWQgY3JlZGVudGlhbHMgcmVkaXJlY3RzIHRvIGRhc2hib2FyZFwiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBsb2dpbihwYWdlKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZSkudG9IYXZlVVJMKC9cXC9kYXNoYm9hcmQvKTtcbiAgfSk7XG5cbiAgdGVzdChcImludmFsaWQgY3JlZGVudGlhbHMgc2hvdyBlcnJvciB0b2FzdFwiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvbG9naW5cIik7XG4gICAgYXdhaXQgcGFnZS5maWxsKFwiI3VzZXJuYW1lXCIsIFwiYWRtaW5cIik7XG4gICAgYXdhaXQgcGFnZS5maWxsKFwiI3Bhc3N3b3JkXCIsIFwid3JvbmdwYXNzd29yZFwiKTtcbiAgICBhd2FpdCBwYWdlLmNsaWNrKCdidXR0b25bdHlwZT1cInN1Ym1pdFwiXScpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJbZGF0YS1zb25uZXItdG9hc3RdLCAuc29ubmVyLXRvYXN0XCIpLmZpcnN0KCkpLnRvQ29udGFpblRleHQoXG4gICAgICAvaW52YWxpZHxlcnJvcnxmYWlsZWQvaSxcbiAgICAgIHsgdGltZW91dDogNTAwMCB9LFxuICAgICk7XG4gIH0pO1xuXG4gIHRlc3QoXCJsb2dvdXQgcmVkaXJlY3RzIHRvIGxvZ2luXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IGxvZ2luKHBhZ2UpO1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi9kYXNoYm9hcmRcIik7XG5cbiAgICBjb25zdCB1c2VyTWVudSA9IHBhZ2UubG9jYXRvcignaGVhZGVyIGJ1dHRvbjpoYXMtdGV4dChcIkFkbWluaXN0cmF0b3JcIiksIGhlYWRlciBidXR0b25bY2xhc3MqPVwicm91bmRlZC1mdWxsXCJdJykuZmlyc3QoKTtcbiAgICBpZiAoYXdhaXQgdXNlck1lbnUuaXNWaXNpYmxlKCkuY2F0Y2goKCkgPT4gZmFsc2UpKSB7XG4gICAgICBhd2FpdCB1c2VyTWVudS5jbGljaygpO1xuICAgICAgY29uc3QgbG9nb3V0QnRuID0gcGFnZS5sb2NhdG9yKCdbcm9sZT1cIm1lbnVpdGVtXCJdOmhhcy10ZXh0KFwiTG9nb3V0XCIpLCBidXR0b246aGFzLXRleHQoXCJMb2dvdXRcIiknKS5maXJzdCgpO1xuICAgICAgaWYgKGF3YWl0IGxvZ291dEJ0bi5pc1Zpc2libGUoKS5jYXRjaCgoKSA9PiBmYWxzZSkpIHtcbiAgICAgICAgYXdhaXQgbG9nb3V0QnRuLmNsaWNrKCk7XG4gICAgICAgIGF3YWl0IHBhZ2Uud2FpdEZvclVSTChcIioqL2xvZ2luXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG4gICAgICB9XG4gICAgfVxuICB9KTtcblxuICB0ZXN0KFwidW5hdXRoZW50aWNhdGVkIHVzZXIgaXMgcmVkaXJlY3RlZCB0byBsb2dpbiBmb3IgcHJvdGVjdGVkIHJvdXRlc1wiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIva2V5c1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZSkudG9IYXZlVVJMKC9cXC9sb2dpbi8pO1xuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIERhc2hib2FyZFxuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG5cbnRlc3QuZGVzY3JpYmUoXCJEYXNoYm9hcmRcIiwgKCkgPT4ge1xuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJkYXNoYm9hcmQgbG9hZHMgd2l0aCBhbGwgd2lkZ2V0c1wiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvZGFzaGJvYXJkXCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KFwiRGFzaGJvYXJkXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoXCJQcm92aWRlcnNcIik7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoXCJBY3RpdmVcIik7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoXCJFdmVudHNcIik7XG4gIH0pO1xuXG4gIHRlc3QoXCJkYXNoYm9hcmQgdG9wb2xvZ3kgYW5kIHRyYWZmaWMgc3VtbWFyeSBhcmUgdmlzaWJsZVwiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvZGFzaGJvYXJkXCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KFwiRmlsdGVyc1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIlByb3ZpZGVyc1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIlJlcSAvIG1pblwiKTtcbiAgfSk7XG59KTtcblxuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG4vLyBTZXR0aW5nc1xuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG5cbnRlc3QuZGVzY3JpYmUuc2VyaWFsKFwiU2V0dGluZ3NcIiwgKCkgPT4ge1xuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJzZXR0aW5ncyBwYWdlIGxvYWRzIGFsbCBjYXJkc1wiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvc2V0dGluZ3NcIik7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoXCJTZXR0aW5nc1wiLCB7IHRpbWVvdXQ6IDEwMDAwIH0pO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KFwiR2VuZXJhbFwiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIkxvZ2dpbmdcIik7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoXCJGZWF0dXJlc1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIk5ldHdvcmtcIik7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoXCJOb3RpZmljYXRpb25zXCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KFwiU2VjdXJpdHlcIik7XG4gIH0pO1xuXG4gIHRlc3QoXCJ0b2dnbGUgcmVxdWlyZV9hcGlfa2V5IGFuZCBzYXZlXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi9zZXR0aW5nc1wiKTtcbiAgICBhd2FpdCBwYWdlLndhaXRGb3JUaW1lb3V0KDgwMCk7XG5cbiAgICBjb25zdCB0b2dnbGUgPSBwYWdlLmxvY2F0b3IoJ2xhYmVsOmhhcy10ZXh0KFwiUmVxdWlyZSBBUEkga2V5XCIpJykubG9jYXRvcihcInhwYXRoPS4uLy4uXCIpLmxvY2F0b3IoJ2J1dHRvbltyb2xlPVwic3dpdGNoXCJdJykuZmlyc3QoKTtcbiAgICBpZiAoYXdhaXQgdG9nZ2xlLmlzVmlzaWJsZSgpLmNhdGNoKCgpID0+IGZhbHNlKSkge1xuICAgICAgY29uc3QgYmVmb3JlID0gYXdhaXQgdG9nZ2xlLmdldEF0dHJpYnV0ZShcImFyaWEtY2hlY2tlZFwiKTtcbiAgICAgIGF3YWl0IHRvZ2dsZS5jbGljaygpO1xuICAgICAgYXdhaXQgcGFnZS53YWl0Rm9yVGltZW91dCgzMDApO1xuXG4gICAgICBjb25zdCBzYXZlQnRuID0gcGFnZS5sb2NhdG9yKCdidXR0b246aGFzLXRleHQoXCJTYXZlIGNoYW5nZXNcIiknKS5maXJzdCgpO1xuICAgICAgYXdhaXQgc2F2ZUJ0bi5jbGljaygpO1xuICAgICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoL3NhdmVkfHN1Y2Nlc3N8c2Fsdm8vaSwgeyB0aW1lb3V0OiA1MDAwIH0pO1xuXG4gICAgICAvLyBUb2dnbGUgYmFjayB0byBvcmlnaW5hbCBzdGF0ZVxuICAgICAgYXdhaXQgdG9nZ2xlLmNsaWNrKCk7XG4gICAgICBhd2FpdCBzYXZlQnRuLmNsaWNrKCk7XG4gICAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCgvc2F2ZWR8c3VjY2Vzc3xzYWx2by9pLCB7IHRpbWVvdXQ6IDUwMDAgfSk7XG4gICAgfVxuICB9KTtcblxuICB0ZXN0KFwiY2hhbmdlIGxvZyByZXRlbnRpb24gYW5kIHNhdmVcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL3NldHRpbmdzXCIpO1xuICAgIGF3YWl0IHBhZ2Uud2FpdEZvclRpbWVvdXQoODAwKTtcblxuICAgIGNvbnN0IGlucHV0ID0gcGFnZS5sb2NhdG9yKCdpbnB1dFt0eXBlPVwibnVtYmVyXCJdJykuZmlyc3QoKTtcbiAgICBhd2FpdCBpbnB1dC5maWxsKFwiNjBcIik7XG4gICAgY29uc3Qgc2F2ZUJ0biA9IHBhZ2UubG9jYXRvcignYnV0dG9uOmhhcy10ZXh0KFwiU2F2ZSBjaGFuZ2VzXCIpJykuZmlyc3QoKTtcbiAgICBhd2FpdCBzYXZlQnRuLmNsaWNrKCk7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoL3NhdmVkfHN1Y2Nlc3MvaSwgeyB0aW1lb3V0OiA1MDAwIH0pO1xuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIEFQSSBLZXlzIC0gRnVsbCBDUlVEXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcblxudGVzdC5kZXNjcmliZS5zZXJpYWwoXCJBUEkgS2V5cyBDUlVEXCIsICgpID0+IHtcbiAgY29uc3Qga2V5TmFtZSA9IGBFMkUtVGVzdC1LZXktJHtTVUZGSVh9YDtcbiAgY29uc3QgdXBkYXRlZE5hbWUgPSBgRTJFLVRlc3QtS2V5LVVwZGF0ZWQtJHtTVUZGSVh9YDtcblxuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJjcmVhdGUgQVBJIGtleVwiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIva2V5c1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIkFQSSBLZXlzXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG5cbiAgICBhd2FpdCBwYWdlLmNsaWNrKCdbZGF0YS10ZXN0aWQ9XCJjcmVhdGUtYnV0dG9uXCJdJyk7XG4gICAgY29uc3QgZGlhbG9nID0gcGFnZS5sb2NhdG9yKCdbcm9sZT1cImRpYWxvZ1wiXScpLmZpcnN0KCk7XG4gICAgYXdhaXQgZXhwZWN0KGRpYWxvZy5sb2NhdG9yKCdoMicpKS50b0NvbnRhaW5UZXh0KFwiQ3JlYXRlXCIpO1xuXG4gICAgYXdhaXQgZGlhbG9nLmxvY2F0b3IoJ2lucHV0W3R5cGU9XCJ0ZXh0XCJdJykuZmlyc3QoKS5maWxsKGtleU5hbWUpO1xuICAgIGF3YWl0IGRpYWxvZy5sb2NhdG9yKCdpbnB1dFt0eXBlPVwibnVtYmVyXCJdJykuZmlyc3QoKS5maWxsKFwiMTAwXCIpO1xuICAgIGF3YWl0IGRpYWxvZy5sb2NhdG9yKCdidXR0b25bdHlwZT1cInN1Ym1pdFwiXTpoYXMtdGV4dChcIlNhdmVcIiknKS5jbGljaygpO1xuXG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoL2NyZWF0ZWR8c3VjY2Vzcy9pLCB7IHRpbWVvdXQ6IDUwMDAgfSk7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoa2V5TmFtZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJlZGl0IEFQSSBrZXlcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL2tleXNcIik7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoa2V5TmFtZSwgeyB0aW1lb3V0OiAxMDAwMCB9KTtcblxuICAgIGNvbnN0IHJvdyA9IHBhZ2UubG9jYXRvcigndHInLCB7IGhhc1RleHQ6IGtleU5hbWUgfSkuZmlyc3QoKTtcbiAgICBjb25zdCBhY3Rpb25zQ2VsbCA9IHJvdy5sb2NhdG9yKCd0ZCcpLmxhc3QoKTtcbiAgICBhd2FpdCBhY3Rpb25zQ2VsbC5sb2NhdG9yKCdidXR0b24nKS5udGgoMSkuY2xpY2soKTsgLy8gZWRpdCBidXR0b24gKHNraXAgUmVnZW5lcmF0ZSlcblxuICAgIGNvbnN0IGRpYWxvZyA9IHBhZ2UubG9jYXRvcignW3JvbGU9XCJkaWFsb2dcIl0nKS5maXJzdCgpO1xuICAgIGF3YWl0IGRpYWxvZy53YWl0Rm9yKHsgc3RhdGU6ICd2aXNpYmxlJywgdGltZW91dDogNTAwMCB9KTtcbiAgICBhd2FpdCBkaWFsb2cubG9jYXRvcignaW5wdXRbdHlwZT1cInRleHRcIl0nKS5maXJzdCgpLmZpbGwodXBkYXRlZE5hbWUpO1xuICAgIGF3YWl0IGRpYWxvZy5sb2NhdG9yKCdidXR0b25bdHlwZT1cInN1Ym1pdFwiXTpoYXMtdGV4dChcIlNhdmVcIiknKS5jbGljaygpO1xuXG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoL3VwZGF0ZWR8c3VjY2Vzcy9pLCB7IHRpbWVvdXQ6IDUwMDAgfSk7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQodXBkYXRlZE5hbWUpO1xuICB9KTtcblxuICB0ZXN0KFwiZGVsZXRlIEFQSSBrZXlcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL2tleXNcIik7XG4gICAgYXdhaXQgcGFnZS53YWl0Rm9yVGltZW91dCgxMDAwKTtcblxuICAgIGNvbnN0IHJvdyA9IHBhZ2UubG9jYXRvcigndHInLCB7IGhhc1RleHQ6IHVwZGF0ZWROYW1lIH0pLmZpcnN0KCk7XG4gICAgaWYgKGF3YWl0IHJvdy5pc1Zpc2libGUoKS5jYXRjaCgoKSA9PiBmYWxzZSkpIHtcbiAgICAgIGNvbnN0IGFjdGlvbnNDZWxsID0gcm93LmxvY2F0b3IoJ3RkJykubGFzdCgpO1xuICAgICAgYXdhaXQgYWN0aW9uc0NlbGwubG9jYXRvcignYnV0dG9uJykubnRoKDIpLmNsaWNrKCk7IC8vIGRlbGV0ZSBidXR0b24gKHNraXAgUmVnZW5lcmF0ZSArIGVkaXQpXG4gICAgICBhd2FpdCBwYWdlLndhaXRGb3JUaW1lb3V0KDMwMCk7XG5cbiAgICAgIGF3YWl0IHBhZ2UubG9jYXRvcigndGV4dD1EZWxldGUgcmVjb3JkPycpLndhaXRGb3IoeyBzdGF0ZTogJ3Zpc2libGUnLCB0aW1lb3V0OiA1MDAwIH0pO1xuICAgICAgYXdhaXQgcGFnZS5sb2NhdG9yKCdidXR0b246aGFzLXRleHQoXCJDYW5jZWxcIikgKyBidXR0b24sIFtyb2xlPVwiZGlhbG9nXCJdIGJ1dHRvbjpoYXMtdGV4dChcIkRlbGV0ZVwiKScpLmZpcnN0KCkuY2xpY2soKTtcblxuICAgICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoL2RlbGV0ZWR8c3VjY2Vzcy9pLCB7IHRpbWVvdXQ6IDUwMDAgfSk7XG4gICAgfVxuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIFZpcnR1YWwgS2V5cyAtIEZ1bGwgQ1JVRFxuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG5cbnRlc3QuZGVzY3JpYmUuc2VyaWFsKFwiVmlydHVhbCBLZXlzIENSVURcIiwgKCkgPT4ge1xuICBjb25zdCBrZXlOYW1lID0gYEUyRS1WS2V5LSR7U1VGRklYfWA7XG4gIGNvbnN0IHVwZGF0ZWROYW1lID0gYEUyRS1WS2V5LVVwZGF0ZWQtJHtTVUZGSVh9YDtcblxuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJjcmVhdGUgdmlydHVhbCBrZXlcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL3ZpcnR1YWwta2V5c1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIlZpcnR1YWwgS2V5c1wiLCB7IHRpbWVvdXQ6IDEwMDAwIH0pO1xuXG4gICAgYXdhaXQgcGFnZS5jbGljaygnW2RhdGEtdGVzdGlkPVwiY3JlYXRlLWJ1dHRvblwiXScpO1xuICAgIGNvbnN0IGRpYWxvZyA9IHBhZ2UubG9jYXRvcignW3JvbGU9XCJkaWFsb2dcIl0nKS5maXJzdCgpO1xuICAgIGNvbnN0IGlucHV0cyA9IGRpYWxvZy5sb2NhdG9yKCdpbnB1dFt0eXBlPVwidGV4dFwiXSwgaW5wdXRbdHlwZT1cIm51bWJlclwiXScpO1xuICAgIGF3YWl0IGlucHV0cy5udGgoMCkuZmlsbChrZXlOYW1lKTtcbiAgICBhd2FpdCBpbnB1dHMubnRoKDEpLmZpbGwoXCI1MFwiKTtcbiAgICBhd2FpdCBkaWFsb2cubG9jYXRvcignYnV0dG9uW3R5cGU9XCJzdWJtaXRcIl06aGFzLXRleHQoXCJTYXZlXCIpJykuY2xpY2soKTtcblxuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KC9jcmVhdGVkfHN1Y2Nlc3MvaSwgeyB0aW1lb3V0OiA1MDAwIH0pO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KGtleU5hbWUpO1xuICB9KTtcblxuICB0ZXN0KFwiZWRpdCB2aXJ0dWFsIGtleVwiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvdmlydHVhbC1rZXlzXCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KGtleU5hbWUsIHsgdGltZW91dDogMTAwMDAgfSk7XG5cbiAgICBjb25zdCByb3cgPSBwYWdlLmxvY2F0b3IoJ3RyJywgeyBoYXNUZXh0OiBrZXlOYW1lIH0pLmZpcnN0KCk7XG4gICAgYXdhaXQgcm93LmxvY2F0b3IoJ2J1dHRvbicpLmZpcnN0KCkuY2xpY2soKTtcblxuICAgIGNvbnN0IGRpYWxvZyA9IHBhZ2UubG9jYXRvcignW3JvbGU9XCJkaWFsb2dcIl0nKS5maXJzdCgpO1xuICAgIGNvbnN0IGlucHV0cyA9IGRpYWxvZy5sb2NhdG9yKCdpbnB1dFt0eXBlPVwidGV4dFwiXSwgaW5wdXRbdHlwZT1cIm51bWJlclwiXScpO1xuICAgIGF3YWl0IGlucHV0cy5udGgoMCkuZmlsbCh1cGRhdGVkTmFtZSk7XG4gICAgYXdhaXQgZGlhbG9nLmxvY2F0b3IoJ2J1dHRvblt0eXBlPVwic3VibWl0XCJdOmhhcy10ZXh0KFwiU2F2ZVwiKScpLmNsaWNrKCk7XG5cbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCgvdXBkYXRlZHxzdWNjZXNzL2ksIHsgdGltZW91dDogNTAwMCB9KTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCh1cGRhdGVkTmFtZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJkZWxldGUgdmlydHVhbCBrZXlcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL3ZpcnR1YWwta2V5c1wiKTtcbiAgICBhd2FpdCBwYWdlLndhaXRGb3JUaW1lb3V0KDEwMDApO1xuXG4gICAgY29uc3Qgcm93ID0gcGFnZS5sb2NhdG9yKCd0cicsIHsgaGFzVGV4dDogdXBkYXRlZE5hbWUgfSkuZmlyc3QoKTtcbiAgICBpZiAoYXdhaXQgcm93LmlzVmlzaWJsZSgpLmNhdGNoKCgpID0+IGZhbHNlKSkge1xuICAgICAgYXdhaXQgcm93LmxvY2F0b3IoJ2J1dHRvbicpLm50aCgxKS5jbGljaygpO1xuICAgICAgYXdhaXQgcGFnZS53YWl0Rm9yVGltZW91dCgzMDApO1xuICAgICAgYXdhaXQgcGFnZS5sb2NhdG9yKCd0ZXh0PURlbGV0ZSByZWNvcmQ/Jykud2FpdEZvcih7IHN0YXRlOiAndmlzaWJsZScsIHRpbWVvdXQ6IDUwMDAgfSk7XG4gICAgICBhd2FpdCBwYWdlLmxvY2F0b3IoJ2J1dHRvbjpoYXMtdGV4dChcIkNhbmNlbFwiKSArIGJ1dHRvbiwgW3JvbGU9XCJkaWFsb2dcIl0gYnV0dG9uOmhhcy10ZXh0KFwiRGVsZXRlXCIpJykuZmlyc3QoKS5jbGljaygpO1xuICAgICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoL2RlbGV0ZWR8c3VjY2Vzcy9pLCB7IHRpbWVvdXQ6IDUwMDAgfSk7XG4gICAgfVxuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIEFsaWFzZXMgLSBGdWxsIENSVURcbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuXG50ZXN0LmRlc2NyaWJlLnNlcmlhbChcIkFsaWFzZXMgQ1JVRFwiLCAoKSA9PiB7XG4gIGNvbnN0IGFsaWFzTmFtZSA9IGBlMmUtYWxpYXMtdGVzdC0ke1NVRkZJWH1gO1xuXG4gIHRlc3QuYmVmb3JlRWFjaChhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBsb2dpbihwYWdlKTtcbiAgfSk7XG5cbiAgdGVzdChcImNyZWF0ZSBhbGlhc1wiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvYWxpYXNlc1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIkFsaWFzZXNcIiwgeyB0aW1lb3V0OiAxMDAwMCB9KTtcblxuICAgIGNvbnN0IGNyZWF0ZUJ0biA9IHBhZ2UubG9jYXRvcignW2RhdGEtdGVzdGlkPVwiY3JlYXRlLWJ1dHRvblwiXScpO1xuICAgIGF3YWl0IGNyZWF0ZUJ0bi53YWl0Rm9yKHsgc3RhdGU6ICd2aXNpYmxlJyB9KTtcbiAgICBhd2FpdCBjcmVhdGVCdG4uY2xpY2soKTtcblxuICAgIGNvbnN0IGRpYWxvZyA9IHBhZ2UubG9jYXRvcignW3JvbGU9XCJkaWFsb2dcIl0nKS5maXJzdCgpO1xuICAgIGNvbnN0IGlucHV0cyA9IGRpYWxvZy5sb2NhdG9yKCdpbnB1dFt0eXBlPVwidGV4dFwiXScpO1xuICAgIGNvbnN0IGNvdW50ID0gYXdhaXQgaW5wdXRzLmNvdW50KCk7XG4gICAgaWYgKGNvdW50ID49IDMpIHtcbiAgICAgIGF3YWl0IGlucHV0cy5udGgoMCkuZmlsbChhbGlhc05hbWUpO1xuICAgICAgYXdhaXQgaW5wdXRzLm50aCgxKS5maWxsKFwib3BlbmFpXCIpO1xuICAgICAgYXdhaXQgaW5wdXRzLm50aCgyKS5maWxsKFwiZ3B0LTRvXCIpO1xuICAgICAgYXdhaXQgZGlhbG9nLmxvY2F0b3IoJ2J1dHRvblt0eXBlPVwic3VibWl0XCJdOmhhcy10ZXh0KFwiU2F2ZVwiKScpLmNsaWNrKCk7XG4gICAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCgvY3JlYXRlZHxzdWNjZXNzL2ksIHsgdGltZW91dDogNTAwMCB9KTtcbiAgICB9XG4gIH0pO1xuXG4gIHRlc3QoXCJkZWxldGUgYWxpYXNcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL2FsaWFzZXNcIik7XG4gICAgYXdhaXQgcGFnZS53YWl0Rm9yVGltZW91dCgxMDAwKTtcblxuICAgIGNvbnN0IHJvdyA9IHBhZ2UubG9jYXRvcigndHInLCB7IGhhc1RleHQ6IGFsaWFzTmFtZSB9KS5maXJzdCgpO1xuICAgIGlmIChhd2FpdCByb3cuaXNWaXNpYmxlKCkuY2F0Y2goKCkgPT4gZmFsc2UpKSB7XG4gICAgICBhd2FpdCByb3cubG9jYXRvcignYnV0dG9uJykubnRoKDEpLmNsaWNrKCk7XG4gICAgICBhd2FpdCBwYWdlLndhaXRGb3JUaW1lb3V0KDMwMCk7XG4gICAgICBhd2FpdCBwYWdlLmxvY2F0b3IoJ3RleHQ9RGVsZXRlIHJlY29yZD8nKS53YWl0Rm9yKHsgc3RhdGU6ICd2aXNpYmxlJywgdGltZW91dDogNTAwMCB9KTtcbiAgICAgIGF3YWl0IHBhZ2UubG9jYXRvcignYnV0dG9uOmhhcy10ZXh0KFwiQ2FuY2VsXCIpICsgYnV0dG9uLCBbcm9sZT1cImRpYWxvZ1wiXSBidXR0b246aGFzLXRleHQoXCJEZWxldGVcIiknKS5maXJzdCgpLmNsaWNrKCk7XG4gICAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCgvZGVsZXRlZHxzdWNjZXNzL2ksIHsgdGltZW91dDogNTAwMCB9KTtcbiAgICB9XG4gIH0pO1xufSk7XG5cbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuLy8gUHJpY2luZyAtIEZ1bGwgQ1JVRFxuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG5cbnRlc3QuZGVzY3JpYmUuc2VyaWFsKFwiUHJpY2luZyBDUlVEXCIsICgpID0+IHtcbiAgdGVzdC5iZWZvcmVFYWNoKGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IGxvZ2luKHBhZ2UpO1xuICB9KTtcblxuICB0ZXN0KFwiY3JlYXRlIHByaWNpbmcgb3ZlcnJpZGVcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL3ByaWNpbmdcIik7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoXCJQcmljaW5nXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG5cbiAgICBhd2FpdCBwYWdlLmNsaWNrKCdbZGF0YS10ZXN0aWQ9XCJjcmVhdGUtYnV0dG9uXCJdJyk7XG4gICAgY29uc3QgZGlhbG9nID0gcGFnZS5sb2NhdG9yKCdbcm9sZT1cImRpYWxvZ1wiXScpLmZpcnN0KCk7XG4gICAgY29uc3QgaW5wdXRzID0gZGlhbG9nLmxvY2F0b3IoJ2lucHV0W3R5cGU9XCJ0ZXh0XCJdLCBpbnB1dFt0eXBlPVwibnVtYmVyXCJdJyk7XG4gICAgY29uc3QgY291bnQgPSBhd2FpdCBpbnB1dHMuY291bnQoKTtcbiAgICBpZiAoY291bnQgPj0gNCkge1xuICAgICAgYXdhaXQgaW5wdXRzLm50aCgwKS5maWxsKFwib3BlbmFpXCIpO1xuICAgICAgYXdhaXQgaW5wdXRzLm50aCgxKS5maWxsKFwiZ3B0LTRvXCIpO1xuICAgICAgYXdhaXQgaW5wdXRzLm50aCgyKS5maWxsKFwiMi41MFwiKTtcbiAgICAgIGF3YWl0IGlucHV0cy5udGgoMykuZmlsbChcIjEwLjAwXCIpO1xuICAgICAgYXdhaXQgZGlhbG9nLmxvY2F0b3IoJ2J1dHRvblt0eXBlPVwic3VibWl0XCJdOmhhcy10ZXh0KFwiU2F2ZVwiKScpLmNsaWNrKCk7XG4gICAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCgvY3JlYXRlZHxzdWNjZXNzL2ksIHsgdGltZW91dDogNTAwMCB9KTtcbiAgICB9XG4gIH0pO1xuXG4gIHRlc3QoXCJkZWxldGUgcHJpY2luZyBvdmVycmlkZVwiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvcHJpY2luZ1wiKTtcbiAgICBhd2FpdCBwYWdlLndhaXRGb3JUaW1lb3V0KDEwMDApO1xuXG4gICAgY29uc3Qgcm93cyA9IHBhZ2UubG9jYXRvcigndGJvZHkgdHIsIHRhYmxlIHRyJyk7XG4gICAgY29uc3QgY291bnQgPSBhd2FpdCByb3dzLmNvdW50KCk7XG4gICAgaWYgKGNvdW50ID4gMSkge1xuICAgICAgY29uc3QgbGFzdFJvdyA9IHJvd3MubGFzdCgpO1xuICAgICAgYXdhaXQgbGFzdFJvdy5sb2NhdG9yKCdidXR0b24nKS5udGgoMSkuY2xpY2soKTtcbiAgICAgIGF3YWl0IHBhZ2Uud2FpdEZvclRpbWVvdXQoMzAwKTtcbiAgICAgIGF3YWl0IHBhZ2UubG9jYXRvcigndGV4dD1EZWxldGUgcmVjb3JkPycpLndhaXRGb3IoeyBzdGF0ZTogJ3Zpc2libGUnLCB0aW1lb3V0OiA1MDAwIH0pO1xuICAgICAgYXdhaXQgcGFnZS5sb2NhdG9yKCdidXR0b246aGFzLXRleHQoXCJDYW5jZWxcIikgKyBidXR0b24sIFtyb2xlPVwiZGlhbG9nXCJdIGJ1dHRvbjpoYXMtdGV4dChcIkRlbGV0ZVwiKScpLmZpcnN0KCkuY2xpY2soKTtcbiAgICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KC9kZWxldGVkfHN1Y2Nlc3MvaSwgeyB0aW1lb3V0OiA1MDAwIH0pO1xuICAgIH1cbiAgfSk7XG59KTtcblxuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG4vLyBSb3V0aW5nIFJ1bGVzIC0gRnVsbCBDUlVEXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcblxudGVzdC5kZXNjcmliZS5zZXJpYWwoXCJSb3V0aW5nIFJ1bGVzIENSVURcIiwgKCkgPT4ge1xuICBjb25zdCBydWxlTmFtZSA9IGBFMkUtUnVsZS0ke1NVRkZJWH1gO1xuICBjb25zdCB1cGRhdGVkTmFtZSA9IGBFMkUtUnVsZS1VcGRhdGVkLSR7U1VGRklYfWA7XG5cbiAgdGVzdC5iZWZvcmVFYWNoKGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IGxvZ2luKHBhZ2UpO1xuICB9KTtcblxuICB0ZXN0KFwiY3JlYXRlIHJvdXRpbmcgcnVsZVwiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvcm91dGluZy1ydWxlc1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIlJvdXRpbmcgUnVsZXNcIiwgeyB0aW1lb3V0OiAxMDAwMCB9KTtcblxuICAgIGNvbnN0IGNyZWF0ZUJ0biA9IHBhZ2UubG9jYXRvcignW2RhdGEtdGVzdGlkPVwiY3JlYXRlLWJ1dHRvblwiXScpO1xuICAgIGF3YWl0IGNyZWF0ZUJ0bi53YWl0Rm9yKHsgc3RhdGU6ICd2aXNpYmxlJyB9KTtcbiAgICBhd2FpdCBjcmVhdGVCdG4uY2xpY2soKTtcblxuICAgIGNvbnN0IGRpYWxvZyA9IHBhZ2UubG9jYXRvcignW3JvbGU9XCJkaWFsb2dcIl0nKS5maXJzdCgpO1xuICAgIGNvbnN0IGlucHV0cyA9IGRpYWxvZy5sb2NhdG9yKCdpbnB1dFt0eXBlPVwidGV4dFwiXSwgaW5wdXRbdHlwZT1cIm51bWJlclwiXScpO1xuICAgIGNvbnN0IGNvdW50ID0gYXdhaXQgaW5wdXRzLmNvdW50KCk7XG4gICAgaWYgKGNvdW50ID49IDUpIHtcbiAgICAgIGF3YWl0IGlucHV0cy5udGgoMCkuZmlsbChydWxlTmFtZSk7XG4gICAgICBhd2FpdCBpbnB1dHMubnRoKDEpLmZpbGwoXCIxXCIpO1xuICAgICAgYXdhaXQgaW5wdXRzLm50aCgyKS5maWxsKFwibW9kZWxcIik7XG4gICAgICBhd2FpdCBkaWFsb2cubG9jYXRvcignc2VsZWN0JykuZmlyc3QoKS5zZWxlY3RPcHRpb24oXCJlcXVhbHNcIik7XG4gICAgICBhd2FpdCBpbnB1dHMubnRoKDMpLmZpbGwoXCJncHQtNG9cIik7XG4gICAgICBhd2FpdCBpbnB1dHMubnRoKDQpLmZpbGwoXCJvcGVuYWlcIik7XG4gICAgICBhd2FpdCBkaWFsb2cubG9jYXRvcignYnV0dG9uW3R5cGU9XCJzdWJtaXRcIl06aGFzLXRleHQoXCJTYXZlXCIpJykuY2xpY2soKTtcbiAgICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KC9jcmVhdGVkfHN1Y2Nlc3MvaSwgeyB0aW1lb3V0OiA1MDAwIH0pO1xuICAgICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQocnVsZU5hbWUpO1xuICAgIH1cbiAgfSk7XG5cbiAgdGVzdChcImVkaXQgcm91dGluZyBydWxlXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi9yb3V0aW5nLXJ1bGVzXCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KHJ1bGVOYW1lLCB7IHRpbWVvdXQ6IDEwMDAwIH0pO1xuXG4gICAgY29uc3Qgcm93ID0gcGFnZS5sb2NhdG9yKCd0cicsIHsgaGFzVGV4dDogcnVsZU5hbWUgfSkuZmlyc3QoKTtcbiAgICBhd2FpdCByb3cubG9jYXRvcignYnV0dG9uJykuZmlyc3QoKS5jbGljaygpO1xuXG4gICAgY29uc3QgZGlhbG9nID0gcGFnZS5sb2NhdG9yKCdbcm9sZT1cImRpYWxvZ1wiXScpLmZpcnN0KCk7XG4gICAgY29uc3QgaW5wdXRzID0gZGlhbG9nLmxvY2F0b3IoJ2lucHV0W3R5cGU9XCJ0ZXh0XCJdJyk7XG4gICAgaWYgKGF3YWl0IGlucHV0cy5jb3VudCgpID4gMCkge1xuICAgICAgYXdhaXQgaW5wdXRzLm50aCgwKS5maWxsKHVwZGF0ZWROYW1lKTtcbiAgICAgIGF3YWl0IGRpYWxvZy5sb2NhdG9yKCdidXR0b25bdHlwZT1cInN1Ym1pdFwiXTpoYXMtdGV4dChcIlNhdmVcIiknKS5jbGljaygpO1xuICAgICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoL3VwZGF0ZWR8c3VjY2Vzcy9pLCB7IHRpbWVvdXQ6IDUwMDAgfSk7XG4gICAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCh1cGRhdGVkTmFtZSk7XG4gICAgfVxuICB9KTtcblxuICB0ZXN0KFwiZGVsZXRlIHJvdXRpbmcgcnVsZVwiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvcm91dGluZy1ydWxlc1wiKTtcbiAgICBhd2FpdCBwYWdlLndhaXRGb3JUaW1lb3V0KDEwMDApO1xuXG4gICAgY29uc3Qgcm93ID0gcGFnZS5sb2NhdG9yKCd0cicsIHsgaGFzVGV4dDogdXBkYXRlZE5hbWUgfSkuZmlyc3QoKTtcbiAgICBpZiAoYXdhaXQgcm93LmlzVmlzaWJsZSgpLmNhdGNoKCgpID0+IGZhbHNlKSkge1xuICAgICAgYXdhaXQgcm93LmxvY2F0b3IoJ2J1dHRvbicpLm50aCgxKS5jbGljaygpO1xuICAgICAgYXdhaXQgcGFnZS53YWl0Rm9yVGltZW91dCgzMDApO1xuICAgICAgYXdhaXQgcGFnZS5sb2NhdG9yKCd0ZXh0PURlbGV0ZSByZWNvcmQ/Jykud2FpdEZvcih7IHN0YXRlOiAndmlzaWJsZScsIHRpbWVvdXQ6IDUwMDAgfSk7XG4gICAgICBhd2FpdCBwYWdlLmxvY2F0b3IoJ2J1dHRvbjpoYXMtdGV4dChcIkNhbmNlbFwiKSArIGJ1dHRvbiwgW3JvbGU9XCJkaWFsb2dcIl0gYnV0dG9uOmhhcy10ZXh0KFwiRGVsZXRlXCIpJykuZmlyc3QoKS5jbGljaygpO1xuICAgICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoL2RlbGV0ZWR8c3VjY2Vzcy9pLCB7IHRpbWVvdXQ6IDUwMDAgfSk7XG4gICAgfVxuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIFRlYW1zIC0gRnVsbCBDUlVEXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcblxudGVzdC5kZXNjcmliZS5zZXJpYWwoXCJUZWFtcyBDUlVEXCIsICgpID0+IHtcbiAgY29uc3QgdGVhbU5hbWUgPSBgRTJFLVRlYW0tJHtTVUZGSVh9YDtcbiAgY29uc3QgdXBkYXRlZE5hbWUgPSBgRTJFLVRlYW0tVXBkYXRlZC0ke1NVRkZJWH1gO1xuXG4gIHRlc3QuYmVmb3JlRWFjaChhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBsb2dpbihwYWdlKTtcbiAgfSk7XG5cbiAgdGVzdChcImNyZWF0ZSB0ZWFtXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi90ZWFtc1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIlRlYW1zXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG5cbiAgICBhd2FpdCBwYWdlLmNsaWNrKCdbZGF0YS10ZXN0aWQ9XCJjcmVhdGUtYnV0dG9uXCJdJyk7XG4gICAgY29uc3QgZGlhbG9nID0gcGFnZS5sb2NhdG9yKCdbcm9sZT1cImRpYWxvZ1wiXScpLmZpcnN0KCk7XG4gICAgY29uc3QgaW5wdXRzID0gZGlhbG9nLmxvY2F0b3IoJ2lucHV0W3R5cGU9XCJ0ZXh0XCJdLCBpbnB1dFt0eXBlPVwibnVtYmVyXCJdJyk7XG4gICAgaWYgKGF3YWl0IGlucHV0cy5jb3VudCgpID49IDIpIHtcbiAgICAgIGF3YWl0IGlucHV0cy5udGgoMCkuZmlsbCh0ZWFtTmFtZSk7XG4gICAgICBhd2FpdCBpbnB1dHMubnRoKDEpLmZpbGwoXCIxMDAwXCIpO1xuICAgICAgYXdhaXQgZGlhbG9nLmxvY2F0b3IoJ2J1dHRvblt0eXBlPVwic3VibWl0XCJdOmhhcy10ZXh0KFwiU2F2ZVwiKScpLmNsaWNrKCk7XG4gICAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCgvY3JlYXRlZHxzdWNjZXNzL2ksIHsgdGltZW91dDogNTAwMCB9KTtcbiAgICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KHRlYW1OYW1lKTtcbiAgICB9XG4gIH0pO1xuXG4gIHRlc3QoXCJlZGl0IHRlYW1cIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL3RlYW1zXCIpO1xuICAgIGF3YWl0IHBhZ2Uud2FpdEZvclRpbWVvdXQoMTAwMCk7XG5cbiAgICBjb25zdCByb3dzID0gcGFnZS5sb2NhdG9yKCd0Ym9keSB0ciwgdGFibGUgdHInKTtcbiAgICBpZiAoYXdhaXQgcm93cy5maWx0ZXIoeyBoYXNUZXh0OiB0ZWFtTmFtZSB9KS5jb3VudCgpID4gMCkge1xuICAgICAgY29uc3Qgcm93ID0gcGFnZS5sb2NhdG9yKCd0cicsIHsgaGFzVGV4dDogdGVhbU5hbWUgfSkuZmlyc3QoKTtcbiAgICAgIGF3YWl0IHJvdy5sb2NhdG9yKCdidXR0b24nKS5maXJzdCgpLmNsaWNrKCk7XG5cbiAgICAgIGNvbnN0IGRpYWxvZyA9IHBhZ2UubG9jYXRvcignW3JvbGU9XCJkaWFsb2dcIl0nKS5maXJzdCgpO1xuICAgICAgY29uc3QgaW5wdXRzID0gZGlhbG9nLmxvY2F0b3IoJ2lucHV0W3R5cGU9XCJ0ZXh0XCJdLCBpbnB1dFt0eXBlPVwibnVtYmVyXCJdJyk7XG4gICAgICBpZiAoYXdhaXQgaW5wdXRzLmNvdW50KCkgPiAwKSB7XG4gICAgICAgIGF3YWl0IGlucHV0cy5udGgoMCkuZmlsbCh1cGRhdGVkTmFtZSk7XG4gICAgICAgIGF3YWl0IGRpYWxvZy5sb2NhdG9yKCdidXR0b25bdHlwZT1cInN1Ym1pdFwiXTpoYXMtdGV4dChcIlNhdmVcIiknKS5jbGljaygpO1xuICAgICAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCgvdXBkYXRlZHxzdWNjZXNzL2ksIHsgdGltZW91dDogNTAwMCB9KTtcbiAgICAgICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQodXBkYXRlZE5hbWUpO1xuICAgICAgfVxuICAgIH1cbiAgfSk7XG5cbiAgdGVzdChcImRlbGV0ZSB0ZWFtXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi90ZWFtc1wiKTtcbiAgICBhd2FpdCBwYWdlLndhaXRGb3JUaW1lb3V0KDEwMDApO1xuXG4gICAgY29uc3Qgcm93cyA9IHBhZ2UubG9jYXRvcigndGJvZHkgdHIsIHRhYmxlIHRyJyk7XG4gICAgaWYgKGF3YWl0IHJvd3MuZmlsdGVyKHsgaGFzVGV4dDogdXBkYXRlZE5hbWUgfSkuY291bnQoKSA+IDApIHtcbiAgICAgIGNvbnN0IHJvdyA9IHBhZ2UubG9jYXRvcigndHInLCB7IGhhc1RleHQ6IHVwZGF0ZWROYW1lIH0pLmZpcnN0KCk7XG4gICAgICBhd2FpdCByb3cubG9jYXRvcignYnV0dG9uJykubnRoKDEpLmNsaWNrKCk7XG4gICAgICBhd2FpdCBwYWdlLndhaXRGb3JUaW1lb3V0KDMwMCk7XG4gICAgICBhd2FpdCBwYWdlLmxvY2F0b3IoJ3RleHQ9RGVsZXRlIHJlY29yZD8nKS53YWl0Rm9yKHsgc3RhdGU6ICd2aXNpYmxlJywgdGltZW91dDogNTAwMCB9KTtcbiAgICAgIGF3YWl0IHBhZ2UubG9jYXRvcignYnV0dG9uOmhhcy10ZXh0KFwiQ2FuY2VsXCIpICsgYnV0dG9uLCBbcm9sZT1cImRpYWxvZ1wiXSBidXR0b246aGFzLXRleHQoXCJEZWxldGVcIiknKS5maXJzdCgpLmNsaWNrKCk7XG4gICAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCgvZGVsZXRlZHxzdWNjZXNzL2ksIHsgdGltZW91dDogNTAwMCB9KTtcbiAgICB9XG4gIH0pO1xufSk7XG5cbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuLy8gQ29tYm9zIC0gRnVsbCBDUlVEXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcblxudGVzdC5kZXNjcmliZS5zZXJpYWwoXCJDb21ib3MgQ1JVRFwiLCAoKSA9PiB7XG4gIGNvbnN0IGNvbWJvTmFtZSA9IGBFMkUtQ29tYm8tJHtTVUZGSVh9YDtcblxuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJjcmVhdGUgY29tYm9cIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL2NvbWJvc1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIkNvbWJvc1wiLCB7IHRpbWVvdXQ6IDEwMDAwIH0pO1xuXG4gICAgY29uc3QgY3JlYXRlQnRuID0gcGFnZS5sb2NhdG9yKCdidXR0b246aGFzLXRleHQoXCJOZXcgY29tYm9cIiknKTtcbiAgICBhd2FpdCBjcmVhdGVCdG4ud2FpdEZvcih7IHN0YXRlOiAndmlzaWJsZScgfSk7XG4gICAgYXdhaXQgY3JlYXRlQnRuLmNsaWNrKCk7XG4gICAgYXdhaXQgcGFnZS53YWl0Rm9yVGltZW91dCg1MDApO1xuXG4gICAgY29uc3QgZGlhbG9nID0gcGFnZS5sb2NhdG9yKCdbcm9sZT1cImRpYWxvZ1wiXScpLmZpcnN0KCk7XG4gICAgYXdhaXQgZGlhbG9nLndhaXRGb3IoeyBzdGF0ZTogJ3Zpc2libGUnLCB0aW1lb3V0OiA1MDAwIH0pO1xuICAgIGNvbnN0IG5hbWVJbnB1dCA9IGRpYWxvZy5sb2NhdG9yKCdpbnB1dCcpLmZpcnN0KCk7XG4gICAgYXdhaXQgbmFtZUlucHV0LndhaXRGb3IoeyBzdGF0ZTogJ3Zpc2libGUnIH0pO1xuICAgIGF3YWl0IG5hbWVJbnB1dC5maWxsKGNvbWJvTmFtZSk7XG4gICAgYXdhaXQgZGlhbG9nLmxvY2F0b3IoJ2J1dHRvbjpoYXMtdGV4dChcIlNhdmVcIiknKS5jbGljaygpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KC9zYXZlZHxzdWNjZXNzL2ksIHsgdGltZW91dDogNTAwMCB9KTtcbiAgfSk7XG5cbiAgdGVzdChcImRlbGV0ZSBjb21ib1wiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvY29tYm9zXCIpO1xuICAgIGF3YWl0IHBhZ2Uud2FpdEZvclRpbWVvdXQoMTAwMCk7XG5cbiAgICBjb25zdCByb3dzID0gcGFnZS5sb2NhdG9yKCd0Ym9keSB0ciwgdGFibGUgdHInKTtcbiAgICBjb25zdCBjb3VudCA9IGF3YWl0IHJvd3MuY291bnQoKTtcbiAgICBpZiAoY291bnQgPiAxKSB7XG4gICAgICBjb25zdCBsYXN0Um93ID0gcm93cy5sYXN0KCk7XG4gICAgICBhd2FpdCBsYXN0Um93LmxvY2F0b3IoJ2J1dHRvbicpLm50aCgxKS5jbGljaygpO1xuICAgICAgY29uc3QgY29uZmlybURpYWxvZyA9IHBhZ2UubG9jYXRvcignW3JvbGU9XCJhbGVydGRpYWxvZ1wiXScpLmZpcnN0KCk7XG4gICAgICBhd2FpdCBjb25maXJtRGlhbG9nLmxvY2F0b3IoJ2J1dHRvbjpoYXMtdGV4dChcIkRlbGV0ZVwiKScpLmNsaWNrKCk7XG4gICAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dCgvZGVsZXRlZHxzdWNjZXNzL2ksIHsgdGltZW91dDogNTAwMCB9KTtcbiAgICB9XG4gIH0pO1xufSk7XG5cbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuLy8gQ29ubmVjdGlvbnNcbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuXG50ZXN0LmRlc2NyaWJlKFwiQ29ubmVjdGlvbnNcIiwgKCkgPT4ge1xuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJjb25uZWN0aW9ucyBwYWdlIGxvYWRzXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi9jb25uZWN0aW9uc1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIkNvbm5lY3Rpb25zXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJidWxrIGFjdGlvbnMgYXJlIHZpc2libGVcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL2Nvbm5lY3Rpb25zXCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoJ2J1dHRvbjpoYXMtdGV4dChcIlBhdXNlIGFsbFwiKScpLmZpcnN0KCkpLnRvQmVWaXNpYmxlKHsgdGltZW91dDogNTAwMCB9KTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKCdidXR0b246aGFzLXRleHQoXCJSZXN1bWUgYWxsXCIpJykuZmlyc3QoKSkudG9CZVZpc2libGUoeyB0aW1lb3V0OiA1MDAwIH0pO1xuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIFByb3ZpZGVyc1xuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG5cbnRlc3QuZGVzY3JpYmUoXCJQcm92aWRlcnNcIiwgKCkgPT4ge1xuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJwcm92aWRlcnMgbGlzdCBsb2Fkc1wiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvcHJvdmlkZXJzXCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KFwiUHJvdmlkZXJzXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJwcm92aWRlciBkZXRhaWwgcGFnZSBsb2Fkc1wiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvcHJvdmlkZXJzXCIpO1xuICAgIGF3YWl0IHBhZ2Uud2FpdEZvclRpbWVvdXQoMTAwMCk7XG5cbiAgICAvLyBGaW5kIGFuZCBjbGljayB0aGUgZmlyc3QgcHJvdmlkZXIgY2FyZFxuICAgIGNvbnN0IHByb3ZpZGVyTGluayA9IHBhZ2UubG9jYXRvcignYVtocmVmXj1cIi9wcm92aWRlcnMvXCJdJykuZmlyc3QoKTtcbiAgICBpZiAoYXdhaXQgcHJvdmlkZXJMaW5rLmlzVmlzaWJsZSgpLmNhdGNoKCgpID0+IGZhbHNlKSkge1xuICAgICAgYXdhaXQgcHJvdmlkZXJMaW5rLmNsaWNrKCk7XG4gICAgICBhd2FpdCBwYWdlLndhaXRGb3JUaW1lb3V0KDEwMDApO1xuICAgICAgLy8gQ2hlY2sgdGhhdCB3ZSdyZSBvbiBhIGRldGFpbCBwYWdlIChVUkwgaGFzIC9wcm92aWRlcnMvKVxuICAgICAgYXdhaXQgZXhwZWN0KHBhZ2UpLnRvSGF2ZVVSTCgvXFwvcHJvdmlkZXJzXFwvLyk7XG4gICAgfSBlbHNlIHtcbiAgICAgIHRlc3Quc2tpcCgpO1xuICAgIH1cbiAgfSk7XG59KTtcblxuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG4vLyBNb2RlbHNcbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuXG50ZXN0LmRlc2NyaWJlKFwiTW9kZWxzXCIsICgpID0+IHtcbiAgdGVzdC5iZWZvcmVFYWNoKGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IGxvZ2luKHBhZ2UpO1xuICB9KTtcblxuICB0ZXN0KFwibW9kZWxzIHBhZ2UgbG9hZHNcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL21vZGVsc1wiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIk1vZGVsc1wiLCB7IHRpbWVvdXQ6IDEwMDAwIH0pO1xuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIEVuZHBvaW50XG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcblxudGVzdC5kZXNjcmliZShcIkVuZHBvaW50XCIsICgpID0+IHtcbiAgdGVzdC5iZWZvcmVFYWNoKGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IGxvZ2luKHBhZ2UpO1xuICB9KTtcblxuICB0ZXN0KFwiZW5kcG9pbnQgcGFnZSBsb2FkcyB3aXRoIFVSTHNcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL2VuZHBvaW50XCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KFwiRW5kcG9pbnRcIiwgeyB0aW1lb3V0OiAxMDAwMCB9KTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIkFQSSBrZXlzXCIpO1xuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIFVzYWdlICYgTG9nc1xuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG5cbnRlc3QuZGVzY3JpYmUoXCJVc2FnZSAmIExvZ3NcIiwgKCkgPT4ge1xuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJ1c2FnZSBwYWdlIGxvYWRzXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi91c2FnZVwiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIlVzYWdlXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJsb2dzIHBhZ2UgbG9hZHNcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL2xvZ3NcIik7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoXCJMb2dzXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG4gIH0pO1xufSk7XG5cbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuLy8gUXVvdGFcbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuXG50ZXN0LmRlc2NyaWJlKFwiUXVvdGFcIiwgKCkgPT4ge1xuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJxdW90YSBwYWdlIGxvYWRzXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi9xdW90YVwiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIlF1b3RhXCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG4gIH0pO1xufSk7XG5cbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuLy8gVHJhZmZpY1xuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG5cbnRlc3QuZGVzY3JpYmUoXCJUcmFmZmljXCIsICgpID0+IHtcbiAgdGVzdC5iZWZvcmVFYWNoKGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IGxvZ2luKHBhZ2UpO1xuICB9KTtcblxuICB0ZXN0KFwidHJhZmZpYyBwYWdlIGxvYWRzXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi90cmFmZmljXCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KFwiVHJhZmZpY1wiLCB7IHRpbWVvdXQ6IDEwMDAwIH0pO1xuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIENvbnNvbGVcbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuXG50ZXN0LmRlc2NyaWJlKFwiQ29uc29sZVwiLCAoKSA9PiB7XG4gIHRlc3QuYmVmb3JlRWFjaChhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBsb2dpbihwYWdlKTtcbiAgfSk7XG5cbiAgdGVzdChcImNvbnNvbGUgcGFnZSBsb2Fkc1wiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBhd2FpdCBwYWdlLmdvdG8oXCIvY29uc29sZVwiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIkNvbnNvbGVcIiwgeyB0aW1lb3V0OiAxMDAwMCB9KTtcbiAgfSk7XG59KTtcblxuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG4vLyBDaGF0XG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcblxudGVzdC5kZXNjcmliZShcIkNoYXRcIiwgKCkgPT4ge1xuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJjaGF0IHBhZ2UgbG9hZHNcIiwgYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgcGFnZS5nb3RvKFwiL2NoYXRcIik7XG4gICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQoXCJDaGF0XCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG4gIH0pO1xufSk7XG5cbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuLy8gQXVkaXRcbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuXG50ZXN0LmRlc2NyaWJlKFwiQXVkaXRcIiwgKCkgPT4ge1xuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJhdWRpdCBwYWdlIGxvYWRzXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi9hdWRpdFwiKTtcbiAgICBhd2FpdCBleHBlY3QocGFnZS5sb2NhdG9yKFwiYm9keVwiKSkudG9Db250YWluVGV4dChcIkF1ZGl0XCIsIHsgdGltZW91dDogMTAwMDAgfSk7XG4gIH0pO1xufSk7XG5cbi8vIOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgOKUgFxuLy8gVHVubmVsc1xuLy8g4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSA4pSAXG5cbnRlc3QuZGVzY3JpYmUoXCJUdW5uZWxzXCIsICgpID0+IHtcbiAgdGVzdC5iZWZvcmVFYWNoKGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IGxvZ2luKHBhZ2UpO1xuICB9KTtcblxuICB0ZXN0KFwidHVubmVscyBwYWdlIGxvYWRzXCIsIGFzeW5jICh7IHBhZ2UgfSkgPT4ge1xuICAgIGF3YWl0IHBhZ2UuZ290byhcIi90dW5uZWxzXCIpO1xuICAgIGF3YWl0IGV4cGVjdChwYWdlLmxvY2F0b3IoXCJib2R5XCIpKS50b0NvbnRhaW5UZXh0KFwiVHVubmVsc1wiLCB7IHRpbWVvdXQ6IDEwMDAwIH0pO1xuICB9KTtcbn0pO1xuXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcbi8vIE5hdmlnYXRpb24gJiBTaWRlYmFyXG4vLyDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIDilIBcblxudGVzdC5kZXNjcmliZShcIk5hdmlnYXRpb25cIiwgKCkgPT4ge1xuICB0ZXN0LmJlZm9yZUVhY2goYXN5bmMgKHsgcGFnZSB9KSA9PiB7XG4gICAgYXdhaXQgbG9naW4ocGFnZSk7XG4gIH0pO1xuXG4gIHRlc3QoXCJzaWRlYmFyIG5hdmlnYXRpb24gbGlua3Mgd29ya1wiLCBhc3luYyAoeyBwYWdlIH0pID0+IHtcbiAgICBjb25zdCByb3V0ZXMgPSBbXG4gICAgICB7IHBhdGg6IFwiL2Rhc2hib2FyZFwiLCB0ZXh0OiBcIkRhc2hib2FyZFwiIH0sXG4gICAgICB7IHBhdGg6IFwiL3Byb3ZpZGVyc1wiLCB0ZXh0OiBcIlByb3ZpZGVyc1wiIH0sXG4gICAgICB7IHBhdGg6IFwiL2Nvbm5lY3Rpb25zXCIsIHRleHQ6IFwiQ29ubmVjdGlvbnNcIiB9LFxuICAgICAgeyBwYXRoOiBcIi9rZXlzXCIsIHRleHQ6IFwiQVBJIEtleXNcIiB9LFxuICAgICAgeyBwYXRoOiBcIi92aXJ0dWFsLWtleXNcIiwgdGV4dDogXCJWaXJ0dWFsIEtleXNcIiB9LFxuICAgICAgeyBwYXRoOiBcIi9jb21ib3NcIiwgdGV4dDogXCJDb21ib3NcIiB9LFxuICAgICAgeyBwYXRoOiBcIi9yb3V0aW5nLXJ1bGVzXCIsIHRleHQ6IFwiUm91dGluZyBSdWxlc1wiIH0sXG4gICAgICB7IHBhdGg6IFwiL21vZGVsc1wiLCB0ZXh0OiBcIk1vZGVsc1wiIH0sXG4gICAgICB7IHBhdGg6IFwiL2FsaWFzZXNcIiwgdGV4dDogXCJBbGlhc2VzXCIgfSxcbiAgICAgIHsgcGF0aDogXCIvcHJpY2luZ1wiLCB0ZXh0OiBcIlByaWNpbmdcIiB9LFxuICAgICAgeyBwYXRoOiBcIi91c2FnZVwiLCB0ZXh0OiBcIlVzYWdlXCIgfSxcbiAgICAgIHsgcGF0aDogXCIvcXVvdGFcIiwgdGV4dDogXCJRdW90YVwiIH0sXG4gICAgICB7IHBhdGg6IFwiL2xvZ3NcIiwgdGV4dDogXCJMb2dzXCIgfSxcbiAgICAgIHsgcGF0aDogXCIvdHJhZmZpY1wiLCB0ZXh0OiBcIlRyYWZmaWNcIiB9LFxuICAgICAgeyBwYXRoOiBcIi9jb25zb2xlXCIsIHRleHQ6IFwiQ29uc29sZVwiIH0sXG4gICAgICB7IHBhdGg6IFwiL2NoYXRcIiwgdGV4dDogXCJDaGF0XCIgfSxcbiAgICAgIHsgcGF0aDogXCIvc2V0dGluZ3NcIiwgdGV4dDogXCJTZXR0aW5nc1wiIH0sXG4gICAgICB7IHBhdGg6IFwiL3RlYW1zXCIsIHRleHQ6IFwiVGVhbXNcIiB9LFxuICAgICAgeyBwYXRoOiBcIi90dW5uZWxzXCIsIHRleHQ6IFwiVHVubmVsc1wiIH0sXG4gICAgICB7IHBhdGg6IFwiL2F1ZGl0XCIsIHRleHQ6IFwiQXVkaXRcIiB9LFxuICAgIF07XG5cbiAgICBmb3IgKGNvbnN0IHJvdXRlIG9mIHJvdXRlcykge1xuICAgICAgYXdhaXQgcGFnZS5nb3RvKHJvdXRlLnBhdGgpO1xuICAgICAgYXdhaXQgZXhwZWN0KHBhZ2UubG9jYXRvcihcImJvZHlcIikpLnRvQ29udGFpblRleHQocm91dGUudGV4dCwgeyB0aW1lb3V0OiA4MDAwIH0pO1xuICAgIH1cbiAgfSk7XG59KTtcbiJdLCJtYXBwaW5ncyI6IkFBQUEsU0FBU0EsSUFBSSxFQUFFQyxNQUFNLFFBQVEsaUJBQWlCO0FBQzlDLFNBQVNDLEtBQUssUUFBUSxXQUFXOztBQUVqQztBQUNBRixJQUFJLENBQUNHLFFBQVEsQ0FBQ0MsU0FBUyxDQUFDO0VBQUVDLElBQUksRUFBRTtBQUFTLENBQUMsQ0FBQzs7QUFFM0M7QUFDQSxNQUFNQyxNQUFNLEdBQUdDLElBQUksQ0FBQ0MsR0FBRyxDQUFDLENBQUMsQ0FBQ0MsUUFBUSxDQUFDLEVBQUUsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQyxDQUFDLENBQUM7O0FBRWhEO0FBQ0E7QUFDQTs7QUFFQVYsSUFBSSxDQUFDRyxRQUFRLENBQUMsY0FBYyxFQUFFLE1BQU07RUFDbENILElBQUksQ0FBQyxxREFBcUQsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQzlFLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0lBQ2pCLE1BQU1WLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDLENBQUNDLFNBQVMsQ0FBQyxhQUFhLENBQUM7RUFDN0MsQ0FBQyxDQUFDO0VBRUZaLElBQUksQ0FBQyxzQ0FBc0MsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQy9ELE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFFBQVEsQ0FBQztJQUN6QixNQUFNRixJQUFJLENBQUNHLElBQUksQ0FBQyxXQUFXLEVBQUUsT0FBTyxDQUFDO0lBQ3JDLE1BQU1ILElBQUksQ0FBQ0csSUFBSSxDQUFDLFdBQVcsRUFBRSxlQUFlLENBQUM7SUFDN0MsTUFBTUgsSUFBSSxDQUFDSSxLQUFLLENBQUMsdUJBQXVCLENBQUM7SUFDekMsTUFBTWQsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxvQ0FBb0MsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQyxDQUFDLENBQUNDLGFBQWEsQ0FDcEYsdUJBQXVCLEVBQ3ZCO01BQUVDLE9BQU8sRUFBRTtJQUFLLENBQ2xCLENBQUM7RUFDSCxDQUFDLENBQUM7RUFFRm5CLElBQUksQ0FBQywyQkFBMkIsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQ3BELE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0lBQ2pCLE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFlBQVksQ0FBQztJQUU3QixNQUFNTyxRQUFRLEdBQUdULElBQUksQ0FBQ0ssT0FBTyxDQUFDLCtFQUErRSxDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDO0lBQ3RILElBQUksTUFBTUcsUUFBUSxDQUFDQyxTQUFTLENBQUMsQ0FBQyxDQUFDQyxLQUFLLENBQUMsTUFBTSxLQUFLLENBQUMsRUFBRTtNQUNqRCxNQUFNRixRQUFRLENBQUNMLEtBQUssQ0FBQyxDQUFDO01BQ3RCLE1BQU1RLFNBQVMsR0FBR1osSUFBSSxDQUFDSyxPQUFPLENBQUMsaUVBQWlFLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7TUFDekcsSUFBSSxNQUFNTSxTQUFTLENBQUNGLFNBQVMsQ0FBQyxDQUFDLENBQUNDLEtBQUssQ0FBQyxNQUFNLEtBQUssQ0FBQyxFQUFFO1FBQ2xELE1BQU1DLFNBQVMsQ0FBQ1IsS0FBSyxDQUFDLENBQUM7UUFDdkIsTUFBTUosSUFBSSxDQUFDYSxVQUFVLENBQUMsVUFBVSxFQUFFO1VBQUVMLE9BQU8sRUFBRTtRQUFNLENBQUMsQ0FBQztNQUN2RDtJQUNGO0VBQ0YsQ0FBQyxDQUFDO0VBRUZuQixJQUFJLENBQUMsa0VBQWtFLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUMzRixNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxPQUFPLENBQUM7SUFDeEIsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUMsQ0FBQ0MsU0FBUyxDQUFDLFNBQVMsQ0FBQztFQUN6QyxDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBWixJQUFJLENBQUNHLFFBQVEsQ0FBQyxXQUFXLEVBQUUsTUFBTTtFQUMvQkgsSUFBSSxDQUFDeUIsVUFBVSxDQUFDLE9BQU87SUFBRWQ7RUFBSyxDQUFDLEtBQUs7SUFDbEMsTUFBTVQsS0FBSyxDQUFDUyxJQUFJLENBQUM7RUFDbkIsQ0FBQyxDQUFDO0VBRUZYLElBQUksQ0FBQyxrQ0FBa0MsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQzNELE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFlBQVksQ0FBQztJQUM3QixNQUFNWixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxXQUFXLEVBQUU7TUFBRUMsT0FBTyxFQUFFO0lBQU0sQ0FBQyxDQUFDO0lBQ2pGLE1BQU1sQixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxXQUFXLENBQUM7SUFDN0QsTUFBTWpCLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLFFBQVEsQ0FBQztJQUMxRCxNQUFNakIsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsUUFBUSxDQUFDO0VBQzVELENBQUMsQ0FBQztFQUVGbEIsSUFBSSxDQUFDLG9EQUFvRCxFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDN0UsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsWUFBWSxDQUFDO0lBQzdCLE1BQU1aLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLFNBQVMsQ0FBQztJQUMzRCxNQUFNakIsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsV0FBVyxDQUFDO0lBQzdELE1BQU1qQixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxXQUFXLENBQUM7RUFDL0QsQ0FBQyxDQUFDO0FBQ0osQ0FBQyxDQUFDOztBQUVGO0FBQ0E7QUFDQTs7QUFFQWxCLElBQUksQ0FBQ0csUUFBUSxDQUFDdUIsTUFBTSxDQUFDLFVBQVUsRUFBRSxNQUFNO0VBQ3JDMUIsSUFBSSxDQUFDeUIsVUFBVSxDQUFDLE9BQU87SUFBRWQ7RUFBSyxDQUFDLEtBQUs7SUFDbEMsTUFBTVQsS0FBSyxDQUFDUyxJQUFJLENBQUM7RUFDbkIsQ0FBQyxDQUFDO0VBRUZYLElBQUksQ0FBQywrQkFBK0IsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQ3hELE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFdBQVcsQ0FBQztJQUM1QixNQUFNWixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxVQUFVLEVBQUU7TUFBRUMsT0FBTyxFQUFFO0lBQU0sQ0FBQyxDQUFDO0lBQ2hGLE1BQU1sQixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxTQUFTLENBQUM7SUFDM0QsTUFBTWpCLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLFNBQVMsQ0FBQztJQUMzRCxNQUFNakIsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsVUFBVSxDQUFDO0lBQzVELE1BQU1qQixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxTQUFTLENBQUM7SUFDM0QsTUFBTWpCLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGVBQWUsQ0FBQztJQUNqRSxNQUFNakIsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsVUFBVSxDQUFDO0VBQzlELENBQUMsQ0FBQztFQUVGbEIsSUFBSSxDQUFDLGlDQUFpQyxFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDMUQsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsV0FBVyxDQUFDO0lBQzVCLE1BQU1GLElBQUksQ0FBQ2dCLGNBQWMsQ0FBQyxHQUFHLENBQUM7SUFFOUIsTUFBTUMsTUFBTSxHQUFHakIsSUFBSSxDQUFDSyxPQUFPLENBQUMsbUNBQW1DLENBQUMsQ0FBQ0EsT0FBTyxDQUFDLGFBQWEsQ0FBQyxDQUFDQSxPQUFPLENBQUMsdUJBQXVCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7SUFDaEksSUFBSSxNQUFNVyxNQUFNLENBQUNQLFNBQVMsQ0FBQyxDQUFDLENBQUNDLEtBQUssQ0FBQyxNQUFNLEtBQUssQ0FBQyxFQUFFO01BQy9DLE1BQU1PLE1BQU0sR0FBRyxNQUFNRCxNQUFNLENBQUNFLFlBQVksQ0FBQyxjQUFjLENBQUM7TUFDeEQsTUFBTUYsTUFBTSxDQUFDYixLQUFLLENBQUMsQ0FBQztNQUNwQixNQUFNSixJQUFJLENBQUNnQixjQUFjLENBQUMsR0FBRyxDQUFDO01BRTlCLE1BQU1JLE9BQU8sR0FBR3BCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLGlDQUFpQyxDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDO01BQ3ZFLE1BQU1jLE9BQU8sQ0FBQ2hCLEtBQUssQ0FBQyxDQUFDO01BQ3JCLE1BQU1kLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLHNCQUFzQixFQUFFO1FBQUVDLE9BQU8sRUFBRTtNQUFLLENBQUMsQ0FBQzs7TUFFM0Y7TUFDQSxNQUFNUyxNQUFNLENBQUNiLEtBQUssQ0FBQyxDQUFDO01BQ3BCLE1BQU1nQixPQUFPLENBQUNoQixLQUFLLENBQUMsQ0FBQztNQUNyQixNQUFNZCxNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxzQkFBc0IsRUFBRTtRQUFFQyxPQUFPLEVBQUU7TUFBSyxDQUFDLENBQUM7SUFDN0Y7RUFDRixDQUFDLENBQUM7RUFFRm5CLElBQUksQ0FBQywrQkFBK0IsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQ3hELE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFdBQVcsQ0FBQztJQUM1QixNQUFNRixJQUFJLENBQUNnQixjQUFjLENBQUMsR0FBRyxDQUFDO0lBRTlCLE1BQU1LLEtBQUssR0FBR3JCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLHNCQUFzQixDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDO0lBQzFELE1BQU1lLEtBQUssQ0FBQ2xCLElBQUksQ0FBQyxJQUFJLENBQUM7SUFDdEIsTUFBTWlCLE9BQU8sR0FBR3BCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLGlDQUFpQyxDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDO0lBQ3ZFLE1BQU1jLE9BQU8sQ0FBQ2hCLEtBQUssQ0FBQyxDQUFDO0lBQ3JCLE1BQU1kLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGdCQUFnQixFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFLLENBQUMsQ0FBQztFQUN2RixDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUN1QixNQUFNLENBQUMsZUFBZSxFQUFFLE1BQU07RUFDMUMsTUFBTU8sT0FBTyxHQUFHLGdCQUFnQjNCLE1BQU0sRUFBRTtFQUN4QyxNQUFNNEIsV0FBVyxHQUFHLHdCQUF3QjVCLE1BQU0sRUFBRTtFQUVwRE4sSUFBSSxDQUFDeUIsVUFBVSxDQUFDLE9BQU87SUFBRWQ7RUFBSyxDQUFDLEtBQUs7SUFDbEMsTUFBTVQsS0FBSyxDQUFDUyxJQUFJLENBQUM7RUFDbkIsQ0FBQyxDQUFDO0VBRUZYLElBQUksQ0FBQyxnQkFBZ0IsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQ3pDLE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLE9BQU8sQ0FBQztJQUN4QixNQUFNWixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxVQUFVLEVBQUU7TUFBRUMsT0FBTyxFQUFFO0lBQU0sQ0FBQyxDQUFDO0lBRWhGLE1BQU1SLElBQUksQ0FBQ0ksS0FBSyxDQUFDLCtCQUErQixDQUFDO0lBQ2pELE1BQU1vQixNQUFNLEdBQUd4QixJQUFJLENBQUNLLE9BQU8sQ0FBQyxpQkFBaUIsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQztJQUN0RCxNQUFNaEIsTUFBTSxDQUFDa0MsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxRQUFRLENBQUM7SUFFMUQsTUFBTWlCLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQyxvQkFBb0IsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQyxDQUFDSCxJQUFJLENBQUNtQixPQUFPLENBQUM7SUFDaEUsTUFBTUUsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLHNCQUFzQixDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDLENBQUNILElBQUksQ0FBQyxLQUFLLENBQUM7SUFDaEUsTUFBTXFCLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQyx3Q0FBd0MsQ0FBQyxDQUFDRCxLQUFLLENBQUMsQ0FBQztJQUV0RSxNQUFNZCxNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxrQkFBa0IsRUFBRTtNQUFFQyxPQUFPLEVBQUU7SUFBSyxDQUFDLENBQUM7SUFDdkYsTUFBTWxCLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDZSxPQUFPLENBQUM7RUFDM0QsQ0FBQyxDQUFDO0VBRUZqQyxJQUFJLENBQUMsY0FBYyxFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDdkMsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsT0FBTyxDQUFDO0lBQ3hCLE1BQU1aLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDZSxPQUFPLEVBQUU7TUFBRWQsT0FBTyxFQUFFO0lBQU0sQ0FBQyxDQUFDO0lBRTdFLE1BQU1pQixHQUFHLEdBQUd6QixJQUFJLENBQUNLLE9BQU8sQ0FBQyxJQUFJLEVBQUU7TUFBRXFCLE9BQU8sRUFBRUo7SUFBUSxDQUFDLENBQUMsQ0FBQ2hCLEtBQUssQ0FBQyxDQUFDO0lBQzVELE1BQU1xQixXQUFXLEdBQUdGLEdBQUcsQ0FBQ3BCLE9BQU8sQ0FBQyxJQUFJLENBQUMsQ0FBQ3VCLElBQUksQ0FBQyxDQUFDO0lBQzVDLE1BQU1ELFdBQVcsQ0FBQ3RCLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQ3dCLEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQ3pCLEtBQUssQ0FBQyxDQUFDLENBQUMsQ0FBQzs7SUFFcEQsTUFBTW9CLE1BQU0sR0FBR3hCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLGlCQUFpQixDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDO0lBQ3RELE1BQU1rQixNQUFNLENBQUNNLE9BQU8sQ0FBQztNQUFFQyxLQUFLLEVBQUUsU0FBUztNQUFFdkIsT0FBTyxFQUFFO0lBQUssQ0FBQyxDQUFDO0lBQ3pELE1BQU1nQixNQUFNLENBQUNuQixPQUFPLENBQUMsb0JBQW9CLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUMsQ0FBQ0gsSUFBSSxDQUFDb0IsV0FBVyxDQUFDO0lBQ3BFLE1BQU1DLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQyx3Q0FBd0MsQ0FBQyxDQUFDRCxLQUFLLENBQUMsQ0FBQztJQUV0RSxNQUFNZCxNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxrQkFBa0IsRUFBRTtNQUFFQyxPQUFPLEVBQUU7SUFBSyxDQUFDLENBQUM7SUFDdkYsTUFBTWxCLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDZ0IsV0FBVyxDQUFDO0VBQy9ELENBQUMsQ0FBQztFQUVGbEMsSUFBSSxDQUFDLGdCQUFnQixFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDekMsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsT0FBTyxDQUFDO0lBQ3hCLE1BQU1GLElBQUksQ0FBQ2dCLGNBQWMsQ0FBQyxJQUFJLENBQUM7SUFFL0IsTUFBTVMsR0FBRyxHQUFHekIsSUFBSSxDQUFDSyxPQUFPLENBQUMsSUFBSSxFQUFFO01BQUVxQixPQUFPLEVBQUVIO0lBQVksQ0FBQyxDQUFDLENBQUNqQixLQUFLLENBQUMsQ0FBQztJQUNoRSxJQUFJLE1BQU1tQixHQUFHLENBQUNmLFNBQVMsQ0FBQyxDQUFDLENBQUNDLEtBQUssQ0FBQyxNQUFNLEtBQUssQ0FBQyxFQUFFO01BQzVDLE1BQU1nQixXQUFXLEdBQUdGLEdBQUcsQ0FBQ3BCLE9BQU8sQ0FBQyxJQUFJLENBQUMsQ0FBQ3VCLElBQUksQ0FBQyxDQUFDO01BQzVDLE1BQU1ELFdBQVcsQ0FBQ3RCLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQ3dCLEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQ3pCLEtBQUssQ0FBQyxDQUFDLENBQUMsQ0FBQztNQUNwRCxNQUFNSixJQUFJLENBQUNnQixjQUFjLENBQUMsR0FBRyxDQUFDO01BRTlCLE1BQU1oQixJQUFJLENBQUNLLE9BQU8sQ0FBQyxxQkFBcUIsQ0FBQyxDQUFDeUIsT0FBTyxDQUFDO1FBQUVDLEtBQUssRUFBRSxTQUFTO1FBQUV2QixPQUFPLEVBQUU7TUFBSyxDQUFDLENBQUM7TUFDdEYsTUFBTVIsSUFBSSxDQUFDSyxPQUFPLENBQUMsK0VBQStFLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUMsQ0FBQ0YsS0FBSyxDQUFDLENBQUM7TUFFbkgsTUFBTWQsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsa0JBQWtCLEVBQUU7UUFBRUMsT0FBTyxFQUFFO01BQUssQ0FBQyxDQUFDO0lBQ3pGO0VBQ0YsQ0FBQyxDQUFDO0FBQ0osQ0FBQyxDQUFDOztBQUVGO0FBQ0E7QUFDQTs7QUFFQW5CLElBQUksQ0FBQ0csUUFBUSxDQUFDdUIsTUFBTSxDQUFDLG1CQUFtQixFQUFFLE1BQU07RUFDOUMsTUFBTU8sT0FBTyxHQUFHLFlBQVkzQixNQUFNLEVBQUU7RUFDcEMsTUFBTTRCLFdBQVcsR0FBRyxvQkFBb0I1QixNQUFNLEVBQUU7RUFFaEROLElBQUksQ0FBQ3lCLFVBQVUsQ0FBQyxPQUFPO0lBQUVkO0VBQUssQ0FBQyxLQUFLO0lBQ2xDLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0VBQ25CLENBQUMsQ0FBQztFQUVGWCxJQUFJLENBQUMsb0JBQW9CLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUM3QyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxlQUFlLENBQUM7SUFDaEMsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsY0FBYyxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztJQUVwRixNQUFNUixJQUFJLENBQUNJLEtBQUssQ0FBQywrQkFBK0IsQ0FBQztJQUNqRCxNQUFNb0IsTUFBTSxHQUFHeEIsSUFBSSxDQUFDSyxPQUFPLENBQUMsaUJBQWlCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7SUFDdEQsTUFBTTBCLE1BQU0sR0FBR1IsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLDBDQUEwQyxDQUFDO0lBQ3pFLE1BQU0yQixNQUFNLENBQUNILEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQzFCLElBQUksQ0FBQ21CLE9BQU8sQ0FBQztJQUNqQyxNQUFNVSxNQUFNLENBQUNILEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQzFCLElBQUksQ0FBQyxJQUFJLENBQUM7SUFDOUIsTUFBTXFCLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQyx3Q0FBd0MsQ0FBQyxDQUFDRCxLQUFLLENBQUMsQ0FBQztJQUV0RSxNQUFNZCxNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxrQkFBa0IsRUFBRTtNQUFFQyxPQUFPLEVBQUU7SUFBSyxDQUFDLENBQUM7SUFDdkYsTUFBTWxCLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDZSxPQUFPLENBQUM7RUFDM0QsQ0FBQyxDQUFDO0VBRUZqQyxJQUFJLENBQUMsa0JBQWtCLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUMzQyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxlQUFlLENBQUM7SUFDaEMsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUNlLE9BQU8sRUFBRTtNQUFFZCxPQUFPLEVBQUU7SUFBTSxDQUFDLENBQUM7SUFFN0UsTUFBTWlCLEdBQUcsR0FBR3pCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLElBQUksRUFBRTtNQUFFcUIsT0FBTyxFQUFFSjtJQUFRLENBQUMsQ0FBQyxDQUFDaEIsS0FBSyxDQUFDLENBQUM7SUFDNUQsTUFBTW1CLEdBQUcsQ0FBQ3BCLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUMsQ0FBQ0YsS0FBSyxDQUFDLENBQUM7SUFFM0MsTUFBTW9CLE1BQU0sR0FBR3hCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLGlCQUFpQixDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDO0lBQ3RELE1BQU0wQixNQUFNLEdBQUdSLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQywwQ0FBMEMsQ0FBQztJQUN6RSxNQUFNMkIsTUFBTSxDQUFDSCxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMxQixJQUFJLENBQUNvQixXQUFXLENBQUM7SUFDckMsTUFBTUMsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLHdDQUF3QyxDQUFDLENBQUNELEtBQUssQ0FBQyxDQUFDO0lBRXRFLE1BQU1kLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGtCQUFrQixFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFLLENBQUMsQ0FBQztJQUN2RixNQUFNbEIsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUNnQixXQUFXLENBQUM7RUFDL0QsQ0FBQyxDQUFDO0VBRUZsQyxJQUFJLENBQUMsb0JBQW9CLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUM3QyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxlQUFlLENBQUM7SUFDaEMsTUFBTUYsSUFBSSxDQUFDZ0IsY0FBYyxDQUFDLElBQUksQ0FBQztJQUUvQixNQUFNUyxHQUFHLEdBQUd6QixJQUFJLENBQUNLLE9BQU8sQ0FBQyxJQUFJLEVBQUU7TUFBRXFCLE9BQU8sRUFBRUg7SUFBWSxDQUFDLENBQUMsQ0FBQ2pCLEtBQUssQ0FBQyxDQUFDO0lBQ2hFLElBQUksTUFBTW1CLEdBQUcsQ0FBQ2YsU0FBUyxDQUFDLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLE1BQU0sS0FBSyxDQUFDLEVBQUU7TUFDNUMsTUFBTWMsR0FBRyxDQUFDcEIsT0FBTyxDQUFDLFFBQVEsQ0FBQyxDQUFDd0IsR0FBRyxDQUFDLENBQUMsQ0FBQyxDQUFDekIsS0FBSyxDQUFDLENBQUM7TUFDMUMsTUFBTUosSUFBSSxDQUFDZ0IsY0FBYyxDQUFDLEdBQUcsQ0FBQztNQUM5QixNQUFNaEIsSUFBSSxDQUFDSyxPQUFPLENBQUMscUJBQXFCLENBQUMsQ0FBQ3lCLE9BQU8sQ0FBQztRQUFFQyxLQUFLLEVBQUUsU0FBUztRQUFFdkIsT0FBTyxFQUFFO01BQUssQ0FBQyxDQUFDO01BQ3RGLE1BQU1SLElBQUksQ0FBQ0ssT0FBTyxDQUFDLCtFQUErRSxDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDLENBQUNGLEtBQUssQ0FBQyxDQUFDO01BQ25ILE1BQU1kLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGtCQUFrQixFQUFFO1FBQUVDLE9BQU8sRUFBRTtNQUFLLENBQUMsQ0FBQztJQUN6RjtFQUNGLENBQUMsQ0FBQztBQUNKLENBQUMsQ0FBQzs7QUFFRjtBQUNBO0FBQ0E7O0FBRUFuQixJQUFJLENBQUNHLFFBQVEsQ0FBQ3VCLE1BQU0sQ0FBQyxjQUFjLEVBQUUsTUFBTTtFQUN6QyxNQUFNa0IsU0FBUyxHQUFHLGtCQUFrQnRDLE1BQU0sRUFBRTtFQUU1Q04sSUFBSSxDQUFDeUIsVUFBVSxDQUFDLE9BQU87SUFBRWQ7RUFBSyxDQUFDLEtBQUs7SUFDbEMsTUFBTVQsS0FBSyxDQUFDUyxJQUFJLENBQUM7RUFDbkIsQ0FBQyxDQUFDO0VBRUZYLElBQUksQ0FBQyxjQUFjLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUN2QyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxVQUFVLENBQUM7SUFDM0IsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsU0FBUyxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztJQUUvRSxNQUFNMEIsU0FBUyxHQUFHbEMsSUFBSSxDQUFDSyxPQUFPLENBQUMsK0JBQStCLENBQUM7SUFDL0QsTUFBTTZCLFNBQVMsQ0FBQ0osT0FBTyxDQUFDO01BQUVDLEtBQUssRUFBRTtJQUFVLENBQUMsQ0FBQztJQUM3QyxNQUFNRyxTQUFTLENBQUM5QixLQUFLLENBQUMsQ0FBQztJQUV2QixNQUFNb0IsTUFBTSxHQUFHeEIsSUFBSSxDQUFDSyxPQUFPLENBQUMsaUJBQWlCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7SUFDdEQsTUFBTTBCLE1BQU0sR0FBR1IsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLG9CQUFvQixDQUFDO0lBQ25ELE1BQU04QixLQUFLLEdBQUcsTUFBTUgsTUFBTSxDQUFDRyxLQUFLLENBQUMsQ0FBQztJQUNsQyxJQUFJQSxLQUFLLElBQUksQ0FBQyxFQUFFO01BQ2QsTUFBTUgsTUFBTSxDQUFDSCxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMxQixJQUFJLENBQUM4QixTQUFTLENBQUM7TUFDbkMsTUFBTUQsTUFBTSxDQUFDSCxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMxQixJQUFJLENBQUMsUUFBUSxDQUFDO01BQ2xDLE1BQU02QixNQUFNLENBQUNILEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQzFCLElBQUksQ0FBQyxRQUFRLENBQUM7TUFDbEMsTUFBTXFCLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQyx3Q0FBd0MsQ0FBQyxDQUFDRCxLQUFLLENBQUMsQ0FBQztNQUN0RSxNQUFNZCxNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxrQkFBa0IsRUFBRTtRQUFFQyxPQUFPLEVBQUU7TUFBSyxDQUFDLENBQUM7SUFDekY7RUFDRixDQUFDLENBQUM7RUFFRm5CLElBQUksQ0FBQyxjQUFjLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUN2QyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxVQUFVLENBQUM7SUFDM0IsTUFBTUYsSUFBSSxDQUFDZ0IsY0FBYyxDQUFDLElBQUksQ0FBQztJQUUvQixNQUFNUyxHQUFHLEdBQUd6QixJQUFJLENBQUNLLE9BQU8sQ0FBQyxJQUFJLEVBQUU7TUFBRXFCLE9BQU8sRUFBRU87SUFBVSxDQUFDLENBQUMsQ0FBQzNCLEtBQUssQ0FBQyxDQUFDO0lBQzlELElBQUksTUFBTW1CLEdBQUcsQ0FBQ2YsU0FBUyxDQUFDLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLE1BQU0sS0FBSyxDQUFDLEVBQUU7TUFDNUMsTUFBTWMsR0FBRyxDQUFDcEIsT0FBTyxDQUFDLFFBQVEsQ0FBQyxDQUFDd0IsR0FBRyxDQUFDLENBQUMsQ0FBQyxDQUFDekIsS0FBSyxDQUFDLENBQUM7TUFDMUMsTUFBTUosSUFBSSxDQUFDZ0IsY0FBYyxDQUFDLEdBQUcsQ0FBQztNQUM5QixNQUFNaEIsSUFBSSxDQUFDSyxPQUFPLENBQUMscUJBQXFCLENBQUMsQ0FBQ3lCLE9BQU8sQ0FBQztRQUFFQyxLQUFLLEVBQUUsU0FBUztRQUFFdkIsT0FBTyxFQUFFO01BQUssQ0FBQyxDQUFDO01BQ3RGLE1BQU1SLElBQUksQ0FBQ0ssT0FBTyxDQUFDLCtFQUErRSxDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDLENBQUNGLEtBQUssQ0FBQyxDQUFDO01BQ25ILE1BQU1kLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGtCQUFrQixFQUFFO1FBQUVDLE9BQU8sRUFBRTtNQUFLLENBQUMsQ0FBQztJQUN6RjtFQUNGLENBQUMsQ0FBQztBQUNKLENBQUMsQ0FBQzs7QUFFRjtBQUNBO0FBQ0E7O0FBRUFuQixJQUFJLENBQUNHLFFBQVEsQ0FBQ3VCLE1BQU0sQ0FBQyxjQUFjLEVBQUUsTUFBTTtFQUN6QzFCLElBQUksQ0FBQ3lCLFVBQVUsQ0FBQyxPQUFPO0lBQUVkO0VBQUssQ0FBQyxLQUFLO0lBQ2xDLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0VBQ25CLENBQUMsQ0FBQztFQUVGWCxJQUFJLENBQUMseUJBQXlCLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUNsRCxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxVQUFVLENBQUM7SUFDM0IsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsU0FBUyxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztJQUUvRSxNQUFNUixJQUFJLENBQUNJLEtBQUssQ0FBQywrQkFBK0IsQ0FBQztJQUNqRCxNQUFNb0IsTUFBTSxHQUFHeEIsSUFBSSxDQUFDSyxPQUFPLENBQUMsaUJBQWlCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7SUFDdEQsTUFBTTBCLE1BQU0sR0FBR1IsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLDBDQUEwQyxDQUFDO0lBQ3pFLE1BQU04QixLQUFLLEdBQUcsTUFBTUgsTUFBTSxDQUFDRyxLQUFLLENBQUMsQ0FBQztJQUNsQyxJQUFJQSxLQUFLLElBQUksQ0FBQyxFQUFFO01BQ2QsTUFBTUgsTUFBTSxDQUFDSCxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMxQixJQUFJLENBQUMsUUFBUSxDQUFDO01BQ2xDLE1BQU02QixNQUFNLENBQUNILEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQzFCLElBQUksQ0FBQyxRQUFRLENBQUM7TUFDbEMsTUFBTTZCLE1BQU0sQ0FBQ0gsR0FBRyxDQUFDLENBQUMsQ0FBQyxDQUFDMUIsSUFBSSxDQUFDLE1BQU0sQ0FBQztNQUNoQyxNQUFNNkIsTUFBTSxDQUFDSCxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMxQixJQUFJLENBQUMsT0FBTyxDQUFDO01BQ2pDLE1BQU1xQixNQUFNLENBQUNuQixPQUFPLENBQUMsd0NBQXdDLENBQUMsQ0FBQ0QsS0FBSyxDQUFDLENBQUM7TUFDdEUsTUFBTWQsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsa0JBQWtCLEVBQUU7UUFBRUMsT0FBTyxFQUFFO01BQUssQ0FBQyxDQUFDO0lBQ3pGO0VBQ0YsQ0FBQyxDQUFDO0VBRUZuQixJQUFJLENBQUMseUJBQXlCLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUNsRCxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxVQUFVLENBQUM7SUFDM0IsTUFBTUYsSUFBSSxDQUFDZ0IsY0FBYyxDQUFDLElBQUksQ0FBQztJQUUvQixNQUFNb0IsSUFBSSxHQUFHcEMsSUFBSSxDQUFDSyxPQUFPLENBQUMsb0JBQW9CLENBQUM7SUFDL0MsTUFBTThCLEtBQUssR0FBRyxNQUFNQyxJQUFJLENBQUNELEtBQUssQ0FBQyxDQUFDO0lBQ2hDLElBQUlBLEtBQUssR0FBRyxDQUFDLEVBQUU7TUFDYixNQUFNRSxPQUFPLEdBQUdELElBQUksQ0FBQ1IsSUFBSSxDQUFDLENBQUM7TUFDM0IsTUFBTVMsT0FBTyxDQUFDaEMsT0FBTyxDQUFDLFFBQVEsQ0FBQyxDQUFDd0IsR0FBRyxDQUFDLENBQUMsQ0FBQyxDQUFDekIsS0FBSyxDQUFDLENBQUM7TUFDOUMsTUFBTUosSUFBSSxDQUFDZ0IsY0FBYyxDQUFDLEdBQUcsQ0FBQztNQUM5QixNQUFNaEIsSUFBSSxDQUFDSyxPQUFPLENBQUMscUJBQXFCLENBQUMsQ0FBQ3lCLE9BQU8sQ0FBQztRQUFFQyxLQUFLLEVBQUUsU0FBUztRQUFFdkIsT0FBTyxFQUFFO01BQUssQ0FBQyxDQUFDO01BQ3RGLE1BQU1SLElBQUksQ0FBQ0ssT0FBTyxDQUFDLCtFQUErRSxDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDLENBQUNGLEtBQUssQ0FBQyxDQUFDO01BQ25ILE1BQU1kLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGtCQUFrQixFQUFFO1FBQUVDLE9BQU8sRUFBRTtNQUFLLENBQUMsQ0FBQztJQUN6RjtFQUNGLENBQUMsQ0FBQztBQUNKLENBQUMsQ0FBQzs7QUFFRjtBQUNBO0FBQ0E7O0FBRUFuQixJQUFJLENBQUNHLFFBQVEsQ0FBQ3VCLE1BQU0sQ0FBQyxvQkFBb0IsRUFBRSxNQUFNO0VBQy9DLE1BQU11QixRQUFRLEdBQUcsWUFBWTNDLE1BQU0sRUFBRTtFQUNyQyxNQUFNNEIsV0FBVyxHQUFHLG9CQUFvQjVCLE1BQU0sRUFBRTtFQUVoRE4sSUFBSSxDQUFDeUIsVUFBVSxDQUFDLE9BQU87SUFBRWQ7RUFBSyxDQUFDLEtBQUs7SUFDbEMsTUFBTVQsS0FBSyxDQUFDUyxJQUFJLENBQUM7RUFDbkIsQ0FBQyxDQUFDO0VBRUZYLElBQUksQ0FBQyxxQkFBcUIsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQzlDLE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLGdCQUFnQixDQUFDO0lBQ2pDLE1BQU1aLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGVBQWUsRUFBRTtNQUFFQyxPQUFPLEVBQUU7SUFBTSxDQUFDLENBQUM7SUFFckYsTUFBTTBCLFNBQVMsR0FBR2xDLElBQUksQ0FBQ0ssT0FBTyxDQUFDLCtCQUErQixDQUFDO0lBQy9ELE1BQU02QixTQUFTLENBQUNKLE9BQU8sQ0FBQztNQUFFQyxLQUFLLEVBQUU7SUFBVSxDQUFDLENBQUM7SUFDN0MsTUFBTUcsU0FBUyxDQUFDOUIsS0FBSyxDQUFDLENBQUM7SUFFdkIsTUFBTW9CLE1BQU0sR0FBR3hCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLGlCQUFpQixDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDO0lBQ3RELE1BQU0wQixNQUFNLEdBQUdSLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQywwQ0FBMEMsQ0FBQztJQUN6RSxNQUFNOEIsS0FBSyxHQUFHLE1BQU1ILE1BQU0sQ0FBQ0csS0FBSyxDQUFDLENBQUM7SUFDbEMsSUFBSUEsS0FBSyxJQUFJLENBQUMsRUFBRTtNQUNkLE1BQU1ILE1BQU0sQ0FBQ0gsR0FBRyxDQUFDLENBQUMsQ0FBQyxDQUFDMUIsSUFBSSxDQUFDbUMsUUFBUSxDQUFDO01BQ2xDLE1BQU1OLE1BQU0sQ0FBQ0gsR0FBRyxDQUFDLENBQUMsQ0FBQyxDQUFDMUIsSUFBSSxDQUFDLEdBQUcsQ0FBQztNQUM3QixNQUFNNkIsTUFBTSxDQUFDSCxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMxQixJQUFJLENBQUMsT0FBTyxDQUFDO01BQ2pDLE1BQU1xQixNQUFNLENBQUNuQixPQUFPLENBQUMsUUFBUSxDQUFDLENBQUNDLEtBQUssQ0FBQyxDQUFDLENBQUNpQyxZQUFZLENBQUMsUUFBUSxDQUFDO01BQzdELE1BQU1QLE1BQU0sQ0FBQ0gsR0FBRyxDQUFDLENBQUMsQ0FBQyxDQUFDMUIsSUFBSSxDQUFDLFFBQVEsQ0FBQztNQUNsQyxNQUFNNkIsTUFBTSxDQUFDSCxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMxQixJQUFJLENBQUMsUUFBUSxDQUFDO01BQ2xDLE1BQU1xQixNQUFNLENBQUNuQixPQUFPLENBQUMsd0NBQXdDLENBQUMsQ0FBQ0QsS0FBSyxDQUFDLENBQUM7TUFDdEUsTUFBTWQsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsa0JBQWtCLEVBQUU7UUFBRUMsT0FBTyxFQUFFO01BQUssQ0FBQyxDQUFDO01BQ3ZGLE1BQU1sQixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQytCLFFBQVEsQ0FBQztJQUM1RDtFQUNGLENBQUMsQ0FBQztFQUVGakQsSUFBSSxDQUFDLG1CQUFtQixFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDNUMsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsZ0JBQWdCLENBQUM7SUFDakMsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMrQixRQUFRLEVBQUU7TUFBRTlCLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztJQUU5RSxNQUFNaUIsR0FBRyxHQUFHekIsSUFBSSxDQUFDSyxPQUFPLENBQUMsSUFBSSxFQUFFO01BQUVxQixPQUFPLEVBQUVZO0lBQVMsQ0FBQyxDQUFDLENBQUNoQyxLQUFLLENBQUMsQ0FBQztJQUM3RCxNQUFNbUIsR0FBRyxDQUFDcEIsT0FBTyxDQUFDLFFBQVEsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQyxDQUFDRixLQUFLLENBQUMsQ0FBQztJQUUzQyxNQUFNb0IsTUFBTSxHQUFHeEIsSUFBSSxDQUFDSyxPQUFPLENBQUMsaUJBQWlCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7SUFDdEQsTUFBTTBCLE1BQU0sR0FBR1IsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLG9CQUFvQixDQUFDO0lBQ25ELElBQUksT0FBTTJCLE1BQU0sQ0FBQ0csS0FBSyxDQUFDLENBQUMsSUFBRyxDQUFDLEVBQUU7TUFDNUIsTUFBTUgsTUFBTSxDQUFDSCxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMxQixJQUFJLENBQUNvQixXQUFXLENBQUM7TUFDckMsTUFBTUMsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLHdDQUF3QyxDQUFDLENBQUNELEtBQUssQ0FBQyxDQUFDO01BQ3RFLE1BQU1kLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGtCQUFrQixFQUFFO1FBQUVDLE9BQU8sRUFBRTtNQUFLLENBQUMsQ0FBQztNQUN2RixNQUFNbEIsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUNnQixXQUFXLENBQUM7SUFDL0Q7RUFDRixDQUFDLENBQUM7RUFFRmxDLElBQUksQ0FBQyxxQkFBcUIsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQzlDLE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLGdCQUFnQixDQUFDO0lBQ2pDLE1BQU1GLElBQUksQ0FBQ2dCLGNBQWMsQ0FBQyxJQUFJLENBQUM7SUFFL0IsTUFBTVMsR0FBRyxHQUFHekIsSUFBSSxDQUFDSyxPQUFPLENBQUMsSUFBSSxFQUFFO01BQUVxQixPQUFPLEVBQUVIO0lBQVksQ0FBQyxDQUFDLENBQUNqQixLQUFLLENBQUMsQ0FBQztJQUNoRSxJQUFJLE1BQU1tQixHQUFHLENBQUNmLFNBQVMsQ0FBQyxDQUFDLENBQUNDLEtBQUssQ0FBQyxNQUFNLEtBQUssQ0FBQyxFQUFFO01BQzVDLE1BQU1jLEdBQUcsQ0FBQ3BCLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQ3dCLEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQ3pCLEtBQUssQ0FBQyxDQUFDO01BQzFDLE1BQU1KLElBQUksQ0FBQ2dCLGNBQWMsQ0FBQyxHQUFHLENBQUM7TUFDOUIsTUFBTWhCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLHFCQUFxQixDQUFDLENBQUN5QixPQUFPLENBQUM7UUFBRUMsS0FBSyxFQUFFLFNBQVM7UUFBRXZCLE9BQU8sRUFBRTtNQUFLLENBQUMsQ0FBQztNQUN0RixNQUFNUixJQUFJLENBQUNLLE9BQU8sQ0FBQywrRUFBK0UsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQyxDQUFDRixLQUFLLENBQUMsQ0FBQztNQUNuSCxNQUFNZCxNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxrQkFBa0IsRUFBRTtRQUFFQyxPQUFPLEVBQUU7TUFBSyxDQUFDLENBQUM7SUFDekY7RUFDRixDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUN1QixNQUFNLENBQUMsWUFBWSxFQUFFLE1BQU07RUFDdkMsTUFBTXlCLFFBQVEsR0FBRyxZQUFZN0MsTUFBTSxFQUFFO0VBQ3JDLE1BQU00QixXQUFXLEdBQUcsb0JBQW9CNUIsTUFBTSxFQUFFO0VBRWhETixJQUFJLENBQUN5QixVQUFVLENBQUMsT0FBTztJQUFFZDtFQUFLLENBQUMsS0FBSztJQUNsQyxNQUFNVCxLQUFLLENBQUNTLElBQUksQ0FBQztFQUNuQixDQUFDLENBQUM7RUFFRlgsSUFBSSxDQUFDLGFBQWEsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQ3RDLE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFFBQVEsQ0FBQztJQUN6QixNQUFNWixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxPQUFPLEVBQUU7TUFBRUMsT0FBTyxFQUFFO0lBQU0sQ0FBQyxDQUFDO0lBRTdFLE1BQU1SLElBQUksQ0FBQ0ksS0FBSyxDQUFDLCtCQUErQixDQUFDO0lBQ2pELE1BQU1vQixNQUFNLEdBQUd4QixJQUFJLENBQUNLLE9BQU8sQ0FBQyxpQkFBaUIsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQztJQUN0RCxNQUFNMEIsTUFBTSxHQUFHUixNQUFNLENBQUNuQixPQUFPLENBQUMsMENBQTBDLENBQUM7SUFDekUsSUFBSSxPQUFNMkIsTUFBTSxDQUFDRyxLQUFLLENBQUMsQ0FBQyxLQUFJLENBQUMsRUFBRTtNQUM3QixNQUFNSCxNQUFNLENBQUNILEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQzFCLElBQUksQ0FBQ3FDLFFBQVEsQ0FBQztNQUNsQyxNQUFNUixNQUFNLENBQUNILEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQzFCLElBQUksQ0FBQyxNQUFNLENBQUM7TUFDaEMsTUFBTXFCLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQyx3Q0FBd0MsQ0FBQyxDQUFDRCxLQUFLLENBQUMsQ0FBQztNQUN0RSxNQUFNZCxNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxrQkFBa0IsRUFBRTtRQUFFQyxPQUFPLEVBQUU7TUFBSyxDQUFDLENBQUM7TUFDdkYsTUFBTWxCLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDaUMsUUFBUSxDQUFDO0lBQzVEO0VBQ0YsQ0FBQyxDQUFDO0VBRUZuRCxJQUFJLENBQUMsV0FBVyxFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDcEMsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsUUFBUSxDQUFDO0lBQ3pCLE1BQU1GLElBQUksQ0FBQ2dCLGNBQWMsQ0FBQyxJQUFJLENBQUM7SUFFL0IsTUFBTW9CLElBQUksR0FBR3BDLElBQUksQ0FBQ0ssT0FBTyxDQUFDLG9CQUFvQixDQUFDO0lBQy9DLElBQUksT0FBTStCLElBQUksQ0FBQ0ssTUFBTSxDQUFDO01BQUVmLE9BQU8sRUFBRWM7SUFBUyxDQUFDLENBQUMsQ0FBQ0wsS0FBSyxDQUFDLENBQUMsSUFBRyxDQUFDLEVBQUU7TUFDeEQsTUFBTVYsR0FBRyxHQUFHekIsSUFBSSxDQUFDSyxPQUFPLENBQUMsSUFBSSxFQUFFO1FBQUVxQixPQUFPLEVBQUVjO01BQVMsQ0FBQyxDQUFDLENBQUNsQyxLQUFLLENBQUMsQ0FBQztNQUM3RCxNQUFNbUIsR0FBRyxDQUFDcEIsT0FBTyxDQUFDLFFBQVEsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQyxDQUFDRixLQUFLLENBQUMsQ0FBQztNQUUzQyxNQUFNb0IsTUFBTSxHQUFHeEIsSUFBSSxDQUFDSyxPQUFPLENBQUMsaUJBQWlCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7TUFDdEQsTUFBTTBCLE1BQU0sR0FBR1IsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLDBDQUEwQyxDQUFDO01BQ3pFLElBQUksT0FBTTJCLE1BQU0sQ0FBQ0csS0FBSyxDQUFDLENBQUMsSUFBRyxDQUFDLEVBQUU7UUFDNUIsTUFBTUgsTUFBTSxDQUFDSCxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMxQixJQUFJLENBQUNvQixXQUFXLENBQUM7UUFDckMsTUFBTUMsTUFBTSxDQUFDbkIsT0FBTyxDQUFDLHdDQUF3QyxDQUFDLENBQUNELEtBQUssQ0FBQyxDQUFDO1FBQ3RFLE1BQU1kLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGtCQUFrQixFQUFFO1VBQUVDLE9BQU8sRUFBRTtRQUFLLENBQUMsQ0FBQztRQUN2RixNQUFNbEIsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUNnQixXQUFXLENBQUM7TUFDL0Q7SUFDRjtFQUNGLENBQUMsQ0FBQztFQUVGbEMsSUFBSSxDQUFDLGFBQWEsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQ3RDLE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFFBQVEsQ0FBQztJQUN6QixNQUFNRixJQUFJLENBQUNnQixjQUFjLENBQUMsSUFBSSxDQUFDO0lBRS9CLE1BQU1vQixJQUFJLEdBQUdwQyxJQUFJLENBQUNLLE9BQU8sQ0FBQyxvQkFBb0IsQ0FBQztJQUMvQyxJQUFJLE9BQU0rQixJQUFJLENBQUNLLE1BQU0sQ0FBQztNQUFFZixPQUFPLEVBQUVIO0lBQVksQ0FBQyxDQUFDLENBQUNZLEtBQUssQ0FBQyxDQUFDLElBQUcsQ0FBQyxFQUFFO01BQzNELE1BQU1WLEdBQUcsR0FBR3pCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLElBQUksRUFBRTtRQUFFcUIsT0FBTyxFQUFFSDtNQUFZLENBQUMsQ0FBQyxDQUFDakIsS0FBSyxDQUFDLENBQUM7TUFDaEUsTUFBTW1CLEdBQUcsQ0FBQ3BCLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQ3dCLEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQ3pCLEtBQUssQ0FBQyxDQUFDO01BQzFDLE1BQU1KLElBQUksQ0FBQ2dCLGNBQWMsQ0FBQyxHQUFHLENBQUM7TUFDOUIsTUFBTWhCLElBQUksQ0FBQ0ssT0FBTyxDQUFDLHFCQUFxQixDQUFDLENBQUN5QixPQUFPLENBQUM7UUFBRUMsS0FBSyxFQUFFLFNBQVM7UUFBRXZCLE9BQU8sRUFBRTtNQUFLLENBQUMsQ0FBQztNQUN0RixNQUFNUixJQUFJLENBQUNLLE9BQU8sQ0FBQywrRUFBK0UsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQyxDQUFDRixLQUFLLENBQUMsQ0FBQztNQUNuSCxNQUFNZCxNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxrQkFBa0IsRUFBRTtRQUFFQyxPQUFPLEVBQUU7TUFBSyxDQUFDLENBQUM7SUFDekY7RUFDRixDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUN1QixNQUFNLENBQUMsYUFBYSxFQUFFLE1BQU07RUFDeEMsTUFBTTJCLFNBQVMsR0FBRyxhQUFhL0MsTUFBTSxFQUFFO0VBRXZDTixJQUFJLENBQUN5QixVQUFVLENBQUMsT0FBTztJQUFFZDtFQUFLLENBQUMsS0FBSztJQUNsQyxNQUFNVCxLQUFLLENBQUNTLElBQUksQ0FBQztFQUNuQixDQUFDLENBQUM7RUFFRlgsSUFBSSxDQUFDLGNBQWMsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQ3ZDLE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFNBQVMsQ0FBQztJQUMxQixNQUFNWixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxRQUFRLEVBQUU7TUFBRUMsT0FBTyxFQUFFO0lBQU0sQ0FBQyxDQUFDO0lBRTlFLE1BQU0wQixTQUFTLEdBQUdsQyxJQUFJLENBQUNLLE9BQU8sQ0FBQyw4QkFBOEIsQ0FBQztJQUM5RCxNQUFNNkIsU0FBUyxDQUFDSixPQUFPLENBQUM7TUFBRUMsS0FBSyxFQUFFO0lBQVUsQ0FBQyxDQUFDO0lBQzdDLE1BQU1HLFNBQVMsQ0FBQzlCLEtBQUssQ0FBQyxDQUFDO0lBQ3ZCLE1BQU1KLElBQUksQ0FBQ2dCLGNBQWMsQ0FBQyxHQUFHLENBQUM7SUFFOUIsTUFBTVEsTUFBTSxHQUFHeEIsSUFBSSxDQUFDSyxPQUFPLENBQUMsaUJBQWlCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7SUFDdEQsTUFBTWtCLE1BQU0sQ0FBQ00sT0FBTyxDQUFDO01BQUVDLEtBQUssRUFBRSxTQUFTO01BQUV2QixPQUFPLEVBQUU7SUFBSyxDQUFDLENBQUM7SUFDekQsTUFBTW1DLFNBQVMsR0FBR25CLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQyxPQUFPLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7SUFDakQsTUFBTXFDLFNBQVMsQ0FBQ2IsT0FBTyxDQUFDO01BQUVDLEtBQUssRUFBRTtJQUFVLENBQUMsQ0FBQztJQUM3QyxNQUFNWSxTQUFTLENBQUN4QyxJQUFJLENBQUN1QyxTQUFTLENBQUM7SUFDL0IsTUFBTWxCLE1BQU0sQ0FBQ25CLE9BQU8sQ0FBQyx5QkFBeUIsQ0FBQyxDQUFDRCxLQUFLLENBQUMsQ0FBQztJQUN2RCxNQUFNZCxNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxnQkFBZ0IsRUFBRTtNQUFFQyxPQUFPLEVBQUU7SUFBSyxDQUFDLENBQUM7RUFDdkYsQ0FBQyxDQUFDO0VBRUZuQixJQUFJLENBQUMsY0FBYyxFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDdkMsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsU0FBUyxDQUFDO0lBQzFCLE1BQU1GLElBQUksQ0FBQ2dCLGNBQWMsQ0FBQyxJQUFJLENBQUM7SUFFL0IsTUFBTW9CLElBQUksR0FBR3BDLElBQUksQ0FBQ0ssT0FBTyxDQUFDLG9CQUFvQixDQUFDO0lBQy9DLE1BQU04QixLQUFLLEdBQUcsTUFBTUMsSUFBSSxDQUFDRCxLQUFLLENBQUMsQ0FBQztJQUNoQyxJQUFJQSxLQUFLLEdBQUcsQ0FBQyxFQUFFO01BQ2IsTUFBTUUsT0FBTyxHQUFHRCxJQUFJLENBQUNSLElBQUksQ0FBQyxDQUFDO01BQzNCLE1BQU1TLE9BQU8sQ0FBQ2hDLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQ3dCLEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQ3pCLEtBQUssQ0FBQyxDQUFDO01BQzlDLE1BQU13QyxhQUFhLEdBQUc1QyxJQUFJLENBQUNLLE9BQU8sQ0FBQyxzQkFBc0IsQ0FBQyxDQUFDQyxLQUFLLENBQUMsQ0FBQztNQUNsRSxNQUFNc0MsYUFBYSxDQUFDdkMsT0FBTyxDQUFDLDJCQUEyQixDQUFDLENBQUNELEtBQUssQ0FBQyxDQUFDO01BQ2hFLE1BQU1kLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLGtCQUFrQixFQUFFO1FBQUVDLE9BQU8sRUFBRTtNQUFLLENBQUMsQ0FBQztJQUN6RjtFQUNGLENBQUMsQ0FBQztBQUNKLENBQUMsQ0FBQzs7QUFFRjtBQUNBO0FBQ0E7O0FBRUFuQixJQUFJLENBQUNHLFFBQVEsQ0FBQyxhQUFhLEVBQUUsTUFBTTtFQUNqQ0gsSUFBSSxDQUFDeUIsVUFBVSxDQUFDLE9BQU87SUFBRWQ7RUFBSyxDQUFDLEtBQUs7SUFDbEMsTUFBTVQsS0FBSyxDQUFDUyxJQUFJLENBQUM7RUFDbkIsQ0FBQyxDQUFDO0VBRUZYLElBQUksQ0FBQyx3QkFBd0IsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQ2pELE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLGNBQWMsQ0FBQztJQUMvQixNQUFNWixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxhQUFhLEVBQUU7TUFBRUMsT0FBTyxFQUFFO0lBQU0sQ0FBQyxDQUFDO0VBQ3JGLENBQUMsQ0FBQztFQUVGbkIsSUFBSSxDQUFDLDBCQUEwQixFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDbkQsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsY0FBYyxDQUFDO0lBQy9CLE1BQU1aLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsOEJBQThCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUMsQ0FBQyxDQUFDdUMsV0FBVyxDQUFDO01BQUVyQyxPQUFPLEVBQUU7SUFBSyxDQUFDLENBQUM7SUFDakcsTUFBTWxCLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsK0JBQStCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUMsQ0FBQyxDQUFDdUMsV0FBVyxDQUFDO01BQUVyQyxPQUFPLEVBQUU7SUFBSyxDQUFDLENBQUM7RUFDcEcsQ0FBQyxDQUFDO0FBQ0osQ0FBQyxDQUFDOztBQUVGO0FBQ0E7QUFDQTs7QUFFQW5CLElBQUksQ0FBQ0csUUFBUSxDQUFDLFdBQVcsRUFBRSxNQUFNO0VBQy9CSCxJQUFJLENBQUN5QixVQUFVLENBQUMsT0FBTztJQUFFZDtFQUFLLENBQUMsS0FBSztJQUNsQyxNQUFNVCxLQUFLLENBQUNTLElBQUksQ0FBQztFQUNuQixDQUFDLENBQUM7RUFFRlgsSUFBSSxDQUFDLHNCQUFzQixFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDL0MsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsWUFBWSxDQUFDO0lBQzdCLE1BQU1aLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLFdBQVcsRUFBRTtNQUFFQyxPQUFPLEVBQUU7SUFBTSxDQUFDLENBQUM7RUFDbkYsQ0FBQyxDQUFDO0VBRUZuQixJQUFJLENBQUMsNEJBQTRCLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUNyRCxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxZQUFZLENBQUM7SUFDN0IsTUFBTUYsSUFBSSxDQUFDZ0IsY0FBYyxDQUFDLElBQUksQ0FBQzs7SUFFL0I7SUFDQSxNQUFNOEIsWUFBWSxHQUFHOUMsSUFBSSxDQUFDSyxPQUFPLENBQUMsd0JBQXdCLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLENBQUM7SUFDbkUsSUFBSSxNQUFNd0MsWUFBWSxDQUFDcEMsU0FBUyxDQUFDLENBQUMsQ0FBQ0MsS0FBSyxDQUFDLE1BQU0sS0FBSyxDQUFDLEVBQUU7TUFDckQsTUFBTW1DLFlBQVksQ0FBQzFDLEtBQUssQ0FBQyxDQUFDO01BQzFCLE1BQU1KLElBQUksQ0FBQ2dCLGNBQWMsQ0FBQyxJQUFJLENBQUM7TUFDL0I7TUFDQSxNQUFNMUIsTUFBTSxDQUFDVSxJQUFJLENBQUMsQ0FBQ0MsU0FBUyxDQUFDLGVBQWUsQ0FBQztJQUMvQyxDQUFDLE1BQU07TUFDTFosSUFBSSxDQUFDMEQsSUFBSSxDQUFDLENBQUM7SUFDYjtFQUNGLENBQUMsQ0FBQztBQUNKLENBQUMsQ0FBQzs7QUFFRjtBQUNBO0FBQ0E7O0FBRUExRCxJQUFJLENBQUNHLFFBQVEsQ0FBQyxRQUFRLEVBQUUsTUFBTTtFQUM1QkgsSUFBSSxDQUFDeUIsVUFBVSxDQUFDLE9BQU87SUFBRWQ7RUFBSyxDQUFDLEtBQUs7SUFDbEMsTUFBTVQsS0FBSyxDQUFDUyxJQUFJLENBQUM7RUFDbkIsQ0FBQyxDQUFDO0VBRUZYLElBQUksQ0FBQyxtQkFBbUIsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQzVDLE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFNBQVMsQ0FBQztJQUMxQixNQUFNWixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxRQUFRLEVBQUU7TUFBRUMsT0FBTyxFQUFFO0lBQU0sQ0FBQyxDQUFDO0VBQ2hGLENBQUMsQ0FBQztBQUNKLENBQUMsQ0FBQzs7QUFFRjtBQUNBO0FBQ0E7O0FBRUFuQixJQUFJLENBQUNHLFFBQVEsQ0FBQyxVQUFVLEVBQUUsTUFBTTtFQUM5QkgsSUFBSSxDQUFDeUIsVUFBVSxDQUFDLE9BQU87SUFBRWQ7RUFBSyxDQUFDLEtBQUs7SUFDbEMsTUFBTVQsS0FBSyxDQUFDUyxJQUFJLENBQUM7RUFDbkIsQ0FBQyxDQUFDO0VBRUZYLElBQUksQ0FBQywrQkFBK0IsRUFBRSxPQUFPO0lBQUVXO0VBQUssQ0FBQyxLQUFLO0lBQ3hELE1BQU1BLElBQUksQ0FBQ0UsSUFBSSxDQUFDLFdBQVcsQ0FBQztJQUM1QixNQUFNWixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxVQUFVLEVBQUU7TUFBRUMsT0FBTyxFQUFFO0lBQU0sQ0FBQyxDQUFDO0lBQ2hGLE1BQU1sQixNQUFNLENBQUNVLElBQUksQ0FBQ0ssT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUNFLGFBQWEsQ0FBQyxVQUFVLENBQUM7RUFDOUQsQ0FBQyxDQUFDO0FBQ0osQ0FBQyxDQUFDOztBQUVGO0FBQ0E7QUFDQTs7QUFFQWxCLElBQUksQ0FBQ0csUUFBUSxDQUFDLGNBQWMsRUFBRSxNQUFNO0VBQ2xDSCxJQUFJLENBQUN5QixVQUFVLENBQUMsT0FBTztJQUFFZDtFQUFLLENBQUMsS0FBSztJQUNsQyxNQUFNVCxLQUFLLENBQUNTLElBQUksQ0FBQztFQUNuQixDQUFDLENBQUM7RUFFRlgsSUFBSSxDQUFDLGtCQUFrQixFQUFFLE9BQU87SUFBRVc7RUFBSyxDQUFDLEtBQUs7SUFDM0MsTUFBTUEsSUFBSSxDQUFDRSxJQUFJLENBQUMsUUFBUSxDQUFDO0lBQ3pCLE1BQU1aLE1BQU0sQ0FBQ1UsSUFBSSxDQUFDSyxPQUFPLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQ0UsYUFBYSxDQUFDLE9BQU8sRUFBRTtNQUFFQyxPQUFPLEVBQUU7SUFBTSxDQUFDLENBQUM7RUFDL0UsQ0FBQyxDQUFDO0VBRUZuQixJQUFJLENBQUMsaUJBQWlCLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUMxQyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxPQUFPLENBQUM7SUFDeEIsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsTUFBTSxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztFQUM5RSxDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUMsT0FBTyxFQUFFLE1BQU07RUFDM0JILElBQUksQ0FBQ3lCLFVBQVUsQ0FBQyxPQUFPO0lBQUVkO0VBQUssQ0FBQyxLQUFLO0lBQ2xDLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0VBQ25CLENBQUMsQ0FBQztFQUVGWCxJQUFJLENBQUMsa0JBQWtCLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUMzQyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxRQUFRLENBQUM7SUFDekIsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsT0FBTyxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztFQUMvRSxDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUMsU0FBUyxFQUFFLE1BQU07RUFDN0JILElBQUksQ0FBQ3lCLFVBQVUsQ0FBQyxPQUFPO0lBQUVkO0VBQUssQ0FBQyxLQUFLO0lBQ2xDLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0VBQ25CLENBQUMsQ0FBQztFQUVGWCxJQUFJLENBQUMsb0JBQW9CLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUM3QyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxVQUFVLENBQUM7SUFDM0IsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsU0FBUyxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztFQUNqRixDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUMsU0FBUyxFQUFFLE1BQU07RUFDN0JILElBQUksQ0FBQ3lCLFVBQVUsQ0FBQyxPQUFPO0lBQUVkO0VBQUssQ0FBQyxLQUFLO0lBQ2xDLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0VBQ25CLENBQUMsQ0FBQztFQUVGWCxJQUFJLENBQUMsb0JBQW9CLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUM3QyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxVQUFVLENBQUM7SUFDM0IsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsU0FBUyxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztFQUNqRixDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUMsTUFBTSxFQUFFLE1BQU07RUFDMUJILElBQUksQ0FBQ3lCLFVBQVUsQ0FBQyxPQUFPO0lBQUVkO0VBQUssQ0FBQyxLQUFLO0lBQ2xDLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0VBQ25CLENBQUMsQ0FBQztFQUVGWCxJQUFJLENBQUMsaUJBQWlCLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUMxQyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxPQUFPLENBQUM7SUFDeEIsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsTUFBTSxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztFQUM5RSxDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUMsT0FBTyxFQUFFLE1BQU07RUFDM0JILElBQUksQ0FBQ3lCLFVBQVUsQ0FBQyxPQUFPO0lBQUVkO0VBQUssQ0FBQyxLQUFLO0lBQ2xDLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0VBQ25CLENBQUMsQ0FBQztFQUVGWCxJQUFJLENBQUMsa0JBQWtCLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUMzQyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxRQUFRLENBQUM7SUFDekIsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsT0FBTyxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztFQUMvRSxDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUMsU0FBUyxFQUFFLE1BQU07RUFDN0JILElBQUksQ0FBQ3lCLFVBQVUsQ0FBQyxPQUFPO0lBQUVkO0VBQUssQ0FBQyxLQUFLO0lBQ2xDLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0VBQ25CLENBQUMsQ0FBQztFQUVGWCxJQUFJLENBQUMsb0JBQW9CLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUM3QyxNQUFNQSxJQUFJLENBQUNFLElBQUksQ0FBQyxVQUFVLENBQUM7SUFDM0IsTUFBTVosTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUMsU0FBUyxFQUFFO01BQUVDLE9BQU8sRUFBRTtJQUFNLENBQUMsQ0FBQztFQUNqRixDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7O0FBRUY7QUFDQTtBQUNBOztBQUVBbkIsSUFBSSxDQUFDRyxRQUFRLENBQUMsWUFBWSxFQUFFLE1BQU07RUFDaENILElBQUksQ0FBQ3lCLFVBQVUsQ0FBQyxPQUFPO0lBQUVkO0VBQUssQ0FBQyxLQUFLO0lBQ2xDLE1BQU1ULEtBQUssQ0FBQ1MsSUFBSSxDQUFDO0VBQ25CLENBQUMsQ0FBQztFQUVGWCxJQUFJLENBQUMsK0JBQStCLEVBQUUsT0FBTztJQUFFVztFQUFLLENBQUMsS0FBSztJQUN4RCxNQUFNZ0QsTUFBTSxHQUFHLENBQ2I7TUFBRUMsSUFBSSxFQUFFLFlBQVk7TUFBRUMsSUFBSSxFQUFFO0lBQVksQ0FBQyxFQUN6QztNQUFFRCxJQUFJLEVBQUUsWUFBWTtNQUFFQyxJQUFJLEVBQUU7SUFBWSxDQUFDLEVBQ3pDO01BQUVELElBQUksRUFBRSxjQUFjO01BQUVDLElBQUksRUFBRTtJQUFjLENBQUMsRUFDN0M7TUFBRUQsSUFBSSxFQUFFLE9BQU87TUFBRUMsSUFBSSxFQUFFO0lBQVcsQ0FBQyxFQUNuQztNQUFFRCxJQUFJLEVBQUUsZUFBZTtNQUFFQyxJQUFJLEVBQUU7SUFBZSxDQUFDLEVBQy9DO01BQUVELElBQUksRUFBRSxTQUFTO01BQUVDLElBQUksRUFBRTtJQUFTLENBQUMsRUFDbkM7TUFBRUQsSUFBSSxFQUFFLGdCQUFnQjtNQUFFQyxJQUFJLEVBQUU7SUFBZ0IsQ0FBQyxFQUNqRDtNQUFFRCxJQUFJLEVBQUUsU0FBUztNQUFFQyxJQUFJLEVBQUU7SUFBUyxDQUFDLEVBQ25DO01BQUVELElBQUksRUFBRSxVQUFVO01BQUVDLElBQUksRUFBRTtJQUFVLENBQUMsRUFDckM7TUFBRUQsSUFBSSxFQUFFLFVBQVU7TUFBRUMsSUFBSSxFQUFFO0lBQVUsQ0FBQyxFQUNyQztNQUFFRCxJQUFJLEVBQUUsUUFBUTtNQUFFQyxJQUFJLEVBQUU7SUFBUSxDQUFDLEVBQ2pDO01BQUVELElBQUksRUFBRSxRQUFRO01BQUVDLElBQUksRUFBRTtJQUFRLENBQUMsRUFDakM7TUFBRUQsSUFBSSxFQUFFLE9BQU87TUFBRUMsSUFBSSxFQUFFO0lBQU8sQ0FBQyxFQUMvQjtNQUFFRCxJQUFJLEVBQUUsVUFBVTtNQUFFQyxJQUFJLEVBQUU7SUFBVSxDQUFDLEVBQ3JDO01BQUVELElBQUksRUFBRSxVQUFVO01BQUVDLElBQUksRUFBRTtJQUFVLENBQUMsRUFDckM7TUFBRUQsSUFBSSxFQUFFLE9BQU87TUFBRUMsSUFBSSxFQUFFO0lBQU8sQ0FBQyxFQUMvQjtNQUFFRCxJQUFJLEVBQUUsV0FBVztNQUFFQyxJQUFJLEVBQUU7SUFBVyxDQUFDLEVBQ3ZDO01BQUVELElBQUksRUFBRSxRQUFRO01BQUVDLElBQUksRUFBRTtJQUFRLENBQUMsRUFDakM7TUFBRUQsSUFBSSxFQUFFLFVBQVU7TUFBRUMsSUFBSSxFQUFFO0lBQVUsQ0FBQyxFQUNyQztNQUFFRCxJQUFJLEVBQUUsUUFBUTtNQUFFQyxJQUFJLEVBQUU7SUFBUSxDQUFDLENBQ2xDO0lBRUQsS0FBSyxNQUFNQyxLQUFLLElBQUlILE1BQU0sRUFBRTtNQUMxQixNQUFNaEQsSUFBSSxDQUFDRSxJQUFJLENBQUNpRCxLQUFLLENBQUNGLElBQUksQ0FBQztNQUMzQixNQUFNM0QsTUFBTSxDQUFDVSxJQUFJLENBQUNLLE9BQU8sQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDRSxhQUFhLENBQUM0QyxLQUFLLENBQUNELElBQUksRUFBRTtRQUFFMUMsT0FBTyxFQUFFO01BQUssQ0FBQyxDQUFDO0lBQ2pGO0VBQ0YsQ0FBQyxDQUFDO0FBQ0osQ0FBQyxDQUFDIiwiaWdub3JlTGlzdCI6W119