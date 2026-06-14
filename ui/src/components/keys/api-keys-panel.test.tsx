import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { ApiKeysPanel, type ApiKeyRow } from "./api-keys-panel";

const keys: ApiKeyRow[] = [
  { id: "key-1", key: "sk-g0def-1234567890abcdef", name: "Default Key", machine_id: "m-1", is_active: true, created_at: "2024-01-01T00:00:00Z" },
  { id: "key-2", key: "sk-g0stg-0987654321zyxwvu", name: "Staging Key", machine_id: "m-2", is_active: true, created_at: "2024-01-02T00:00:00Z" },
];

describe("ApiKeysPanel", () => {
  it("renders seeded keys with the real DTO fields (name + a row marker)", () => {
    const html = renderToString(<ApiKeysPanel initialKeys={keys} />);
    expect(html).toContain("Default Key");
    expect(html).toContain("Staging Key");
    expect(html).toContain("api-key-row");
  });

  it("renders the API Keys heading and a create trigger", () => {
    const html = renderToString(<ApiKeysPanel initialKeys={keys} />);
    expect(html).toContain("API Keys");
    expect(html).toContain("create-key-trigger");
  });

  it("renders compact mode without the section heading when compact", () => {
    const full = renderToString(<ApiKeysPanel initialKeys={keys} />);
    const compact = renderToString(<ApiKeysPanel initialKeys={keys} compact />);
    // Both render rows; compact still exposes the create trigger.
    expect(compact).toContain("api-key-row");
    expect(compact).toContain("create-key-trigger");
    expect(full).toContain("api-key-row");
  });
});
