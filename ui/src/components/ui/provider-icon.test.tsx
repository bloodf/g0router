import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import {
  ProviderIcon,
  providerInitials,
  providerColor,
} from "./provider-icon";

describe("ProviderIcon", () => {
  it("renders an img with the /providers/<slug>.png src", () => {
    const html = renderToString(<ProviderIcon slug="openai" name="OpenAI" />);
    expect(html).toContain("<img");
    expect(html).toContain('src="/providers/openai.png"');
  });

  it("providerInitials returns the first two letters uppercased", () => {
    expect(providerInitials("openai")).toBe("OP");
    expect(providerInitials("Anthropic")).toBe("AN");
  });

  it("providerColor is deterministic and differs across names", () => {
    expect(providerColor("openai")).toBe(providerColor("openai"));
    expect(providerColor("openai")).not.toBe(providerColor("anthropic"));
  });

  it("uses the provider name as the img alt text", () => {
    const html = renderToString(<ProviderIcon slug="anthropic" name="Anthropic" />);
    expect(html).toContain('alt="Anthropic"');
  });
});
