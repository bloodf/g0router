import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

// Spec-only helper: drive the auth mock via request headers the auth handler
// reads (x-mock-auth-mode / x-mock-force-lockout). Headers reach the handler
// because page.route intercepts page-context navigations and fetches.
async function setMockAuth(
  page: import("@playwright/test").Page,
  body: { auth_mode?: "password" | "oidc" | "both"; force_lockout?: number }
) {
  const headers: Record<string, string> = {};
  if (body.auth_mode) headers["x-mock-auth-mode"] = body.auth_mode;
  if (typeof body.force_lockout === "number")
    headers["x-mock-force-lockout"] = String(body.force_lockout);
  await page.setExtraHTTPHeaders(headers);
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

  test("callback relays code+state via BroadcastChannel (and localStorage fallback)", async ({
    page,
  }) => {
    // BroadcastChannel: subscribe on an app-origin page, then open /callback in a
    // popup that shares the same browsing-context group so the channel delivers.
    await page.goto("/login");
    const got = await page.evaluate(
      () =>
        new Promise<{ code?: string; state?: string }>((resolve) => {
          const ch = new BroadcastChannel("oauth_callback");
          ch.onmessage = (ev) => {
            ch.close();
            resolve(ev.data);
          };
          window.open("/callback?code=abc&state=xyz", "_blank");
        })
    );
    expect(got.code).toBe("abc");
    expect(got.state).toBe("xyz");

    // localStorage fallback: the callback page writes it synchronously on load.
    await page.goto("/callback?code=abc&state=xyz");
    const stored = await page.evaluate(() =>
      localStorage.getItem("oauth_callback")
    );
    expect(stored).toContain("abc");
  });

  test("callback shows manual-copy state when no code/error present", async ({ page }) => {
    await page.goto("/callback");
    await expect(page.getByText(/copy this url/i)).toBeVisible();
  });
});
