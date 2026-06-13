import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Button } from "./button";

describe("Button", () => {
  it("renders children", () => {
    const html = renderToString(<Button>Click me</Button>);
    expect(html).toContain("Click me");
  });

  it("renders distinct class per variant", () => {
    const variants = ["primary", "secondary", "ghost", "outline", "danger"] as const;
    const classes = variants.map((variant) =>
      renderToString(<Button variant={variant}>x</Button>)
    );
    const unique = new Set(classes);
    expect(unique.size).toBe(variants.length);
  });

  it("loading renders spinner, disables, and sets aria-busy", () => {
    const html = renderToString(<Button loading>Save</Button>);
    expect(html).toContain('role="status"');
    expect(html).toContain("disabled");
    expect(html).toContain('aria-busy="true"');
  });
});
