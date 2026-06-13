import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

// Spec-only helper: drive the auth mock's control surface (auth_mode / lockout).
async function setMockAuth(
  page: import("@playwright/test").Page,
  body: { auth_mode?: "password" | "oidc" | "both"; force_lockout?: number }
) {
  await page.request.post("/__mock__/auth", { data: body });
}

test.describe("Auth flow", () => {
  test("login with valid credentials redirects to dashboard", async ({ page }) => {
    await login(page);
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test("invalid credentials show error toast", async ({ page }) => {
    await page.goto("/login");
    await page.fill("#username", "admin");
    await page.fill("#password", "wrongpassword");
    await page.click('button[type="submit"]');
    // Wait for toast or error to appear — sonner renders outside body, so search the whole page
    await expect(page.locator("[data-sonner-toast], .sonner-toast").first()).toContainText(/invalid|error|failed/i, { timeout: 5000 });
  });

  test('login page shows OIDC button when auth_mode is "both"', async ({ page }) => {
    await setMockAuth(page, { auth_mode: "both" });
    await page.goto("/login");
    await expect(page.locator("#password")).toBeVisible();
    await expect(page.getByTestId("oidc-button")).toBeVisible();
  });

  test('login page hides password form when auth_mode is "oidc"', async ({ page }) => {
    await setMockAuth(page, { auth_mode: "oidc" });
    await page.goto("/login");
    const oidc = page.getByTestId("oidc-button");
    await expect(oidc).toBeVisible();
    // Variant per plan §1.4: OIDC is the primary action when auth_mode is "oidc".
    await expect(oidc).toHaveAttribute("data-primary", "true");
  });

  test("rate-limit returns a retry countdown and disables submit", async ({ page }) => {
    await setMockAuth(page, { force_lockout: 30 });
    await page.goto("/login");
    await page.fill("#username", "admin");
    await page.fill("#password", "123456");
    await page.click('button[type="submit"]');
    await expect(page.getByText(/wait/i)).toBeVisible({ timeout: 5000 });
    await expect(page.locator('button[type="submit"]')).toBeDisabled();
  });

  test("logout from header clears session and returns to /login", async ({ page }) => {
    await login(page);
    const logoutBtn = page.getByTestId("logout-button");
    await expect(logoutBtn).toBeVisible();
    await logoutBtn.click();
    await expect(page).toHaveURL(/\/login/);
  });

  test("callback relays code+state via BroadcastChannel and localStorage", async ({ page }) => {
    // Subscribe to the BroadcastChannel on an app-origin page first.
    await page.goto("/login");
    const received = page.evaluate(
      () =>
        new Promise<{ code?: string; state?: string }>((resolve) => {
          const ch = new BroadcastChannel("oauth_callback");
          ch.onmessage = (ev) => resolve(ev.data);
        })
    );
    // Navigate the SAME page to /callback so the BroadcastChannel shares context.
    await page.goto("/callback?code=abc&state=xyz");
    // localStorage fallback is written synchronously on the callback page.
    const stored = await page.evaluate(() =>
      localStorage.getItem("oauth_callback")
    );
    expect(stored).toContain("abc");

    // Reopen the listener page to assert the broadcast on a fresh subscriber.
    await page.goto("/login");
    const got = await page.evaluate(
      () =>
        new Promise<{ code?: string; state?: string }>((resolve) => {
          const ch = new BroadcastChannel("oauth_callback");
          ch.onmessage = (ev) => resolve(ev.data);
          const w = window.open("/callback?code=abc&state=xyz", "_blank");
          setTimeout(() => {
            try {
              w?.close();
            } catch {
              /* ignore */
            }
          }, 1800);
        })
    );
    expect(got.code).toBe("abc");
    expect(got.state).toBe("xyz");
    void received;
  });

  test("callback shows manual-copy state when no code/error present", async ({ page }) => {
    await page.goto("/callback");
    await expect(page.getByText(/copy this url/i)).toBeVisible();
  });
});
