import type { PricingOverride } from "../../src/lib/types";

export function seedPricing(): PricingOverride[] {
  return [
    { id: "price-1", provider: "openai", model: "gpt-4o", input_cost: 2.0, output_cost: 8.0 },
    { id: "price-2", provider: "anthropic", model: "claude-sonnet-4", input_cost: 2.5, output_cost: 12.0 },
  ];
}
