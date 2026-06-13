import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Badge } from "./badge";

describe("Badge", () => {
  it("renders children text", () => {
    const html = renderToString(<Badge>active</Badge>);
    expect(html).toContain("active");
  });

  it("renders distinct class per variant and size, and a dot when requested", () => {
    const variants = ["success", "error", "default", "neutral", "primary"] as const;
    const classes = variants.map((variant) =>
      renderToString(<Badge variant={variant}>x</Badge>)
    );
    expect(new Set(classes).size).toBe(variants.length);

    const sm = renderToString(<Badge size="sm">x</Badge>);
    const md = renderToString(<Badge size="md">x</Badge>);
    expect(sm).not.toBe(md);

    const withDot = renderToString(<Badge dot>x</Badge>);
    expect(withDot).toContain('data-testid="badge-dot"');
  });

  it("keeps text content visible (not icon-only)", () => {
    const html = renderToString(<Badge>online</Badge>);
    expect(html).toContain("online");
  });
});
