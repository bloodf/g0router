import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerModelsHandlers(page: Page, store: MockStore) {
  page.route("/api/models", async (route) => {
    if (route.request().method() === "GET") return json(route, Array.from(store.models.values()));
    return route.continue();
  });
  page.route("/api/models/disabled", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.disabledModels));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      store.disabledModels.add(body.model_id);
      const m = store.models.get(body.model_id);
      if (m) m.is_disabled = true;
      return json(route, {});
    }
    if (method === "DELETE") {
      const body = await route.request().postDataJSON();
      store.disabledModels.delete(body.model_id);
      const m = store.models.get(body.model_id);
      if (m) m.is_disabled = false;
      return json(route, {});
    }
    return route.continue();
  });
  page.route("/api/models/custom", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, store.customModels);
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const model = { id: store.nextId(), ...body, is_disabled: false, is_custom: true };
      store.customModels.push(model);
      store.models.set(model.id, model);
      return json(route, model);
    }
    return route.continue();
  });
  page.route(/\/api\/models\/custom\/[^/]+$/, async (route) => {
    if (route.request().method() === "DELETE") {
      const id = route.request().url().split("/").pop()!;
      store.customModels = store.customModels.filter((m) => m.id !== id);
      store.models.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
}
