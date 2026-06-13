import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Tooltip, TooltipProvider } from "./tooltip";

describe("Tooltip", () => {
  it("renders its trigger child in the closed state", () => {
    const html = renderToString(
      <TooltipProvider>
        <Tooltip content="Hello">
          <button>trigger</button>
        </Tooltip>
      </TooltipProvider>
    );
    expect(html).toContain("trigger");
  });

  it("accepts side and color props without error", () => {
    const html = renderToString(
      <TooltipProvider>
        <Tooltip content="Hi" side="right" color="primary">
          <button>t</button>
        </Tooltip>
      </TooltipProvider>
    );
    expect(html).toContain("t");
  });

  it("renders inside the provider without throwing", () => {
    expect(() =>
      renderToString(
        <TooltipProvider>
          <Tooltip content="Hi" color="dark">
            <span>x</span>
          </Tooltip>
        </TooltipProvider>
      )
    ).not.toThrow();
  });
});
