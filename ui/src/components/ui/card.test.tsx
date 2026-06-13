import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Card, CardHeader, CardTitle, CardContent } from "./card";

describe("Card", () => {
  it("renders children", () => {
    const html = renderToString(<Card>body</Card>);
    expect(html).toContain("body");
  });

  it("renders distinct class per padding variant", () => {
    const paddings = ["none", "sm", "md", "lg"] as const;
    const classes = paddings.map((padding) =>
      renderToString(<Card padding={padding}>x</Card>)
    );
    expect(new Set(classes).size).toBe(paddings.length);
  });

  it("composes header, title, and content", () => {
    const html = renderToString(
      <Card>
        <CardHeader>
          <CardTitle>Heading</CardTitle>
        </CardHeader>
        <CardContent>Inner</CardContent>
      </Card>
    );
    expect(html).toContain("Heading");
    expect(html).toContain("Inner");
  });
});
