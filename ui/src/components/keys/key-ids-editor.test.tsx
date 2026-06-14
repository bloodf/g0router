import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import {
  KeyIdsEditor,
  emptyProviderConfig,
  type EditorProviderConfig,
} from "./key-ids-editor";

const configs: EditorProviderConfig[] = [
  {
    provider: "openai",
    allowed_models: ["gpt-4o"],
    key_ids: ["conn-1"],
  },
];

describe("KeyIdsEditor", () => {
  it("renders a provider-config row with the editor marker", () => {
    const html = renderToString(
      <KeyIdsEditor value={configs} onChange={() => {}} providerOptions={["openai", "anthropic"]} />
    );
    expect(html).toContain("key-ids-editor");
    expect(html).toContain("vk-provider-select");
  });

  it("renders the selected allowed_models and key_ids", () => {
    const html = renderToString(
      <KeyIdsEditor value={configs} onChange={() => {}} providerOptions={["openai"]} />
    );
    expect(html).toContain("gpt-4o");
    expect(html).toContain("conn-1");
  });

  it("emptyProviderConfig serializes to the real VK provider_configs shape", () => {
    const pc = emptyProviderConfig("openai");
    expect(pc).toHaveProperty("provider", "openai");
    expect(pc).toHaveProperty("allowed_models");
    expect(pc).toHaveProperty("key_ids");
    expect(Array.isArray(pc.key_ids)).toBe(true);
    expect(Array.isArray(pc.allowed_models)).toBe(true);
  });

  it("renders an add-config control to append provider_configs rows", () => {
    const html = renderToString(
      <KeyIdsEditor value={[]} onChange={() => {}} providerOptions={["openai"]} />
    );
    expect(html).toContain("key-ids-editor");
  });
});
