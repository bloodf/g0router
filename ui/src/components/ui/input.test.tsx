import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Input } from "./input";

describe("Input", () => {
  it("renders input and label text", () => {
    const html = renderToString(<Input label="Email" />);
    expect(html).toContain("Email");
    expect(html).toContain("<input");
  });

  it("renders error and hint text", () => {
    const errorHtml = renderToString(<Input label="Email" error="Required" />);
    expect(errorHtml).toContain("Required");
    const hintHtml = renderToString(<Input label="Email" hint="We never share it" />);
    expect(hintHtml).toContain("We never share it");
  });

  it("associates label htmlFor with input id and wires aria when error", () => {
    const html = renderToString(<Input label="Email" error="Required" />);
    const idMatch = html.match(/id="([^"]+)"/);
    expect(idMatch).not.toBeNull();
    const id = idMatch![1];
    expect(html).toContain(`for="${id}"`);
    expect(html).toContain('aria-invalid="true"');
    expect(html).toContain("aria-describedby");
  });
});
