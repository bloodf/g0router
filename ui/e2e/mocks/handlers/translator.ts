import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

// Self-contained translator mock (w6-i §1.6/§1.9). No new store field, no seed
// edit — the sample payloads live inline here. Serves the minimal contract the
// textarea-variant translator page needs:
//   GET  /api/translator/load?file=<name>  -> a seeded source payload
//   POST /api/translator/translate         -> a transformed payload
// There is NO real /api/translator/* Go backend (w6-i §1.2); this is the binding
// capability contract for the translator surface until the serial Go follow-up.

const SAMPLE_CLIENT_REQUEST = {
  model: "gpt-4o",
  provider: "openai",
  messages: [{ role: "user", content: "Translate this request" }],
  stream: false,
};

export function registerTranslatorHandlers(page: Page, _store: MockStore) {
  page.route(/\/api\/translator\/load(\?.*)?$/, async (route) => {
    if (route.request().method() === "GET") {
      return json(route, {
        file: "sample",
        payload: JSON.stringify(SAMPLE_CLIENT_REQUEST, null, 2),
      });
    }
    return route.continue();
  });

  page.route("/api/translator/translate", async (route) => {
    if (route.request().method() === "POST") {
      const body = await route.request().postDataJSON();
      // Echo the incoming payload back as the "OpenAI intermediate" form with a
      // deterministic marker the spec can assert on.
      const transformed = {
        translated: true,
        from: body.from ?? "client",
        to: body.to ?? "openai",
        payload: body.payload ?? "",
      };
      return json(route, {
        payload: JSON.stringify(transformed, null, 2),
      });
    }
    return route.continue();
  });
}
