import { Page } from "@playwright/test";

export async function login(page: Page, username = "admin", password = "123456") {
  await page.goto("/login");
  await page.waitForSelector("#username", { state: "visible" });
  await page.fill("#username", username);
  await page.fill("#password", password);
  await page.click('button[type="submit"]');
  await page.waitForURL("**/dashboard", { timeout: 10000 });
}

export async function logout(page: Page) {
  // Try to find user menu or logout link
  const userMenu = page.locator('button:has-text("admin"), [data-testid="user-menu"], nav button').first();
  if (await userMenu.isVisible().catch(() => false)) {
    await userMenu.click();
    const logoutBtn = page.locator('text=Logout, text=Sair, button:has-text("logout")').first();
    if (await logoutBtn.isVisible().catch(() => false)) {
      await logoutBtn.click();
    }
  }
  await page.goto("/login");
}
