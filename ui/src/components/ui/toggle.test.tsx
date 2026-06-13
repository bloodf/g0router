import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Toggle } from "./toggle";

describe("Toggle", () => {
  it("renders a switch", () => {
    const html = renderToString(<Toggle checked={false} />);
    expect(html).toContain('role="switch"');
  });

  it("renders distinct classes for sm and md sizes", () => {
    const sm = renderToString(<Toggle size="sm" checked={false} />);
    const md = renderToString(<Toggle size="md" checked={false} />);
    expect(sm).not.toBe(md);
  });

  it("reflects checked state via aria-checked", () => {
    const on = renderToString(<Toggle checked />);
    expect(on).toContain('aria-checked="true"');
    const off = renderToString(<Toggle checked={false} />);
    expect(off).toContain('aria-checked="false"');
  });
});
