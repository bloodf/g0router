import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

// Mirrors the REAL Go /api/virtual-keys CRUD (internal/admin/virtualkeys.go, plan
// §1.4 / §1.6 / §8 ESC-2). Response bodies use the Go envelope shapes:
//   GET  /api/virtual-keys -> {data:{virtual_keys:[virtualKeyDTO]}}
//   POST /api/virtual-keys -> {data:{virtual_key:virtualKeyDTO}}
//   GET/PUT /api/virtual-keys/{id} -> {data:{virtual_key:virtualKeyDTO}}
//   DELETE /api/virtual-keys/{id} -> {data:{message}}
// Create/update REQUIRE provider_configs with a non-empty key_ids per config
// (virtualkeys.go:55-66) — the KeyIDs editor (plan §1.6) writes provider_configs.
interface ProviderConfigLike {
  provider?: string;
  allowed_models?: string[];
  key_ids?: string[];
}

function validateProviderConfigs(body: { provider_configs?: ProviderConfigLike[] }): string | null {
  const configs = body.provider_configs;
  if (!configs || configs.length === 0) return "provider_configs is required";
  for (let i = 0; i < configs.length; i++) {
    const pc = configs[i];
    if (!pc.provider) return `provider_configs[${i}].provider is required`;
    if (!pc.allowed_models || pc.allowed_models.length === 0) return `provider_configs[${i}].allowed_models is required`;
    if (!pc.key_ids || pc.key_ids.length === 0) return `provider_configs[${i}].key_ids is required`;
  }
  return null;
}

export function registerVirtualKeysHandlers(page: Page, store: MockStore) {
  page.route("/api/virtual-keys", async (route) => {
    const method = route.request().method();
    if (method === "GET") {
      return json(route, { virtual_keys: Array.from(store.virtualKeys.values()) });
    }
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      if (!body.name) return error(route, "name is required", 400);
      const invalid = validateProviderConfigs(body);
      if (invalid) return error(route, invalid, 400);
      const id = store.nextId();
      const now = Math.floor(Date.now() / 1000);
      const vk = {
        id,
        key: `vk-${Math.random().toString(36).slice(2, 12)}`,
        name: body.name,
        provider_configs: body.provider_configs,
        budget: body.budget,
        rate_limit_rpm: body.rate_limit_rpm,
        is_active: body.is_active ?? true,
        created_at: now,
        updated_at: now,
      };
      store.virtualKeys.set(id, vk as never);
      return json(route, { virtual_key: vk });
    }
    return route.continue();
  });
  page.route(/\/api\/virtual-keys\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const vk = store.virtualKeys.get(id);
      return vk ? json(route, { virtual_key: vk }) : error(route, "virtual key not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.virtualKeys.get(id);
      if (!existing) return error(route, "virtual key not found", 404);
      if (!body.name) return error(route, "name is required", 400);
      const invalid = validateProviderConfigs(body);
      if (invalid) return error(route, invalid, 400);
      const updated = {
        ...existing,
        name: body.name,
        provider_configs: body.provider_configs,
        budget: body.budget,
        rate_limit_rpm: body.rate_limit_rpm,
        is_active: body.is_active ?? (existing as { is_active?: boolean }).is_active ?? true,
        updated_at: Math.floor(Date.now() / 1000),
      };
      store.virtualKeys.set(id, updated as never);
      return json(route, { virtual_key: updated });
    }
    if (method === "DELETE") {
      store.virtualKeys.delete(id);
      return json(route, { message: "Virtual key deleted successfully" });
    }
    return route.continue();
  });
}
