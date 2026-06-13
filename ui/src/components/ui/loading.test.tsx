import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Spinner, Loading } from "./loading";

describe("Loading", () => {
  it("Spinner renders and Loading renders its message", () => {
    expect(renderToString(<Spinner />)).toContain('role="status"');
    const html = renderToString(<Loading message="Fetching..." />);
    expect(html).toContain("Fetching...");
  });

  it("Spinner renders distinct classes per size", () => {
    const sm = renderToString(<Spinner size="sm" />);
    const lg = renderToString(<Spinner size="lg" />);
    expect(sm).not.toBe(lg);
  });

  it("Spinner exposes role=status for accessibility", () => {
    const html = renderToString(<Spinner />);
    expect(html).toContain('role="status"');
  });
});
