import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

// Mock body mirrors the real Go pricing API (plan w6-g §1.4 / §1.2):
//   GET    /api/pricing               -> internal/admin/pricing.go:20 (nested
//          provider->model->{input,output,cached,reasoning,cache_creation})
//   PATCH  /api/pricing               -> pricing.go:30 ({provider:{model:{field:value}}})
//   DELETE /api/pricing?provider=&model= -> pricing.go:56 (empty provider resets all)
// The seed is PricingOverride[] (flat input_cost/output_cost, store.pricing Map);
// we project it into the real nested shape on GET and apply PATCH/DELETE against
// an overlay nested map. The legacy REST-collection model (POST / PUT-by-id) is
// dropped — the real Go has no such routes. NO index/seed edit; body only.
type Rates = { input: number; output: number; cached: number; reasoning: number; cache_creation: number };

export function registerPricingHandlers(page: Page, store: MockStore) {
  // Build the nested pricing map from the flat seed once per registration; PATCH
  // and DELETE mutate this overlay so the page sees its edits within a test.
  const nested: Record<string, Record<string, Rates>> = {};
  for (const p of store.pricing.values()) {
    (nested[p.provider] ??= {})[p.model] = {
      input: p.input_cost,
      output: p.output_cost,
      cached: 0,
      reasoning: 0,
      cache_creation: 0,
    };
  }

  page.route("/api/pricing", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, nested);
    if (method === "PATCH") {
      const body = (await route.request().postDataJSON()) as Record<string, Record<string, Partial<Rates>>>;
      for (const provider of Object.keys(body)) {
        for (const model of Object.keys(body[provider])) {
          const cur = ((nested[provider] ??= {})[model] ??= { input: 0, output: 0, cached: 0, reasoning: 0, cache_creation: 0 });
          Object.assign(cur, body[provider][model]);
        }
      }
      return json(route, nested);
    }
    if (method === "DELETE") {
      const url = new URL(route.request().url());
      const provider = url.searchParams.get("provider") || "";
      const model = url.searchParams.get("model") || "";
      if (!provider) {
        for (const k of Object.keys(nested)) delete nested[k];
      } else if (model && nested[provider]) {
        delete nested[provider][model];
      } else {
        delete nested[provider];
      }
      return json(route, nested);
    }
    return route.continue();
  });

  // Defensive: any stray REST-by-id call (no longer used by the page) 404s so a
  // mismatch surfaces loudly rather than silently continuing to the network.
  page.route(/\/api\/pricing\/[^/]+$/, async (route) => {
    return error(route, "Not found", 404);
  });
}
