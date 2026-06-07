import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: "list",
  use: {
    baseURL: "http://127.0.0.1:20128",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    command: "cd .. && ./g0router serve --data-dir ./e2e_data --port 20128",
    url: "http://127.0.0.1:20128/healthz",
    reuseExistingServer: !process.env.CI,
    timeout: 30000,
    env: {
      DATA_DIR: "./e2e_data",
      BIND_ADDRESS: "127.0.0.1",
      PORT: "20128",
    },
  },
});
