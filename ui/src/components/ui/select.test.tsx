import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Select } from "./select";

const options = [
  { value: "a", label: "Alpha" },
  { value: "b", label: "Beta" },
  { value: "c", label: "Gamma", disabled: true },
];

describe("Select", () => {
  it("renders all options from the array", () => {
    const html = renderToString(<Select label="Pick" options={options} />);
    expect(html).toContain("Alpha");
    expect(html).toContain("Beta");
    expect(html).toContain("Gamma");
  });

  it("marks disabled options as disabled", () => {
    const html = renderToString(<Select label="Pick" options={options} />);
    const gammaOption = html.match(/<option[^>]*value="c"[^>]*>/);
    expect(gammaOption).not.toBeNull();
    expect(gammaOption![0]).toContain("disabled");
  });

  it("associates label htmlFor with select id and wires aria when error", () => {
    const html = renderToString(
      <Select label="Pick" options={options} error="Required" />
    );
    const idMatch = html.match(/id="([^"]+)"/);
    expect(idMatch).not.toBeNull();
    const id = idMatch![1];
    expect(html).toContain(`for="${id}"`);
    expect(html).toContain('aria-invalid="true"');
    expect(html).toContain("aria-describedby");
    expect(html).toContain("Required");
  });
});
