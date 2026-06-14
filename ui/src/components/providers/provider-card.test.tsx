import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { ProviderCard } from "./provider-card";
import type { Provider } from "@/lib/types";

const provider: Provider = {
  id: "openai",
  name: "openai",
  display_name: "OpenAI",
  description: "GPT-4, GPT-3.5, DALL-E, Whisper",
  auth_types: ["api_key"],
  capabilities: ["chat", "embeddings"],
  connection_count: 2,
  status: "active",
};

describe("ProviderCard", () => {
  it("renders the display name and a status badge", () => {
    const html = renderToString(<ProviderCard provider={provider} />);
    expect(html).toContain("OpenAI");
    expect(html.toLowerCase()).toContain("active");
  });

  it("root element className contains the card-elev marker", () => {
    const html = renderToString(<ProviderCard provider={provider} />);
    expect(html).toMatch(/class="[^"]*card-elev/);
  });

  it("renders the connection_count", () => {
    const html = renderToString(<ProviderCard provider={provider} />);
    expect(html).toContain("2");
  });
});
