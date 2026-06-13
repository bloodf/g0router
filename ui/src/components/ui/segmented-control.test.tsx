import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { SegmentedControl } from "./segmented-control";

const options = [
  { value: "day", label: "Day" },
  { value: "week", label: "Week" },
  { value: "month", label: "Month" },
];

describe("SegmentedControl", () => {
  it("renders all option labels", () => {
    const html = renderToString(
      <SegmentedControl options={options} value="day" onChange={() => {}} />
    );
    expect(html).toContain("Day");
    expect(html).toContain("Week");
    expect(html).toContain("Month");
  });

  it("styles the selected option distinctly from the others", () => {
    const html = renderToString(
      <SegmentedControl options={options} value="week" onChange={() => {}} />
    );
    const selected = html.match(/aria-selected="true"/g) ?? [];
    const unselected = html.match(/aria-selected="false"/g) ?? [];
    expect(selected.length).toBe(1);
    expect(unselected.length).toBe(2);
  });

  it("exposes tablist and tab roles with aria-selected", () => {
    const html = renderToString(
      <SegmentedControl options={options} value="day" onChange={() => {}} />
    );
    expect(html).toContain('role="tablist"');
    expect(html).toContain('role="tab"');
    expect(html).toContain('aria-selected="true"');
  });
});
