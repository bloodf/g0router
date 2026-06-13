import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Skeleton, CardSkeleton } from "./skeleton";

describe("Skeleton", () => {
  it("renders a skeleton element", () => {
    const html = renderToString(<Skeleton />);
    expect(html).toContain('aria-hidden="true"');
  });

  it("CardSkeleton composes multiple Skeletons", () => {
    const html = renderToString(<CardSkeleton />);
    const count = (html.match(/aria-hidden="true"/g) ?? []).length;
    expect(count).toBeGreaterThanOrEqual(2);
  });

  it("passes className through", () => {
    const html = renderToString(<Skeleton className="custom-skel" />);
    expect(html).toContain("custom-skel");
  });
});
