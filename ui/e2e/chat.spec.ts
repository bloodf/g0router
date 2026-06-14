import { test, expect } from "./mocks/fixture";
import { login } from "./helpers";

test.describe("Chat", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("chat page loads", async ({ page }) => {
    await page.goto("/chat");
    await expect(page.locator("body")).toContainText("Chat", { timeout: 10000 });
  });

  test("send message and receive mock response", async ({ page }) => {
    await page.goto("/chat");
    const input = page.locator('input[aria-label="Message"]').first();
    await input.fill("Hello mock assistant");
    await input.press("Enter");
    await expect(page.locator("body")).toContainText("mock assistant", { timeout: 15000 });
  });

  test("provider/model selector renders", async ({ page }) => {
    await page.goto("/chat");
    await expect(
      page.locator("[data-testid='chat-model-select']")
    ).toBeVisible({ timeout: 10000 });
  });

  test("assistant turn is appended into the message list", async ({ page }) => {
    await page.goto("/chat");
    const input = page.locator('input[aria-label="Message"]').first();
    await input.fill("ping");
    await input.press("Enter");
    const assistant = page.locator("[data-testid='chat-message-assistant']").last();
    await expect(assistant).toBeVisible({ timeout: 15000 });
    await expect(assistant).toContainText("mock assistant");
  });
});
