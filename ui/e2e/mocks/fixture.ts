import { test as base, expect } from "@playwright/test";
import { MockStore } from "./store";
import { setupMockApi } from "./handlers";

export type MockFixtures = {
  mockStore: MockStore;
};

// Worker-scoped store so serial tests can share entity data.
// Auth state is reset before each test; tests that need auth must call login().
export const test = base.extend<MockFixtures, { workerStore: MockStore }>({
  workerStore: [async ({}, use) => {
    const store = new MockStore();
    store.seedAll();
    await use(store);
  }, { scope: "worker" }],

  mockStore: async ({ workerStore }, use) => {
    // Reset auth to default (logged out) but keep all seeded entity data
    workerStore.auth = {
      require_login: true,
      has_users: true,
      authenticated: false,
      username: "admin",
      display_name: "Administrator",
      role: "admin",
    };
    await use(workerStore);
  },

  page: async ({ page, mockStore }, use) => {
    setupMockApi(page, mockStore);
    await use(page);
  },
});

export { expect };
