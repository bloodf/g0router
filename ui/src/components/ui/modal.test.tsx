import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Modal } from "./modal";

describe("Modal", () => {
  it("renders overlay, panel, title, and 3 traffic-light dots when open", () => {
    const html = renderToString(
      <Modal open onClose={() => {}} title="My Dialog">
        <p>content</p>
      </Modal>
    );
    expect(html).toContain('data-testid="modal-overlay"');
    expect(html).toContain('role="dialog"');
    expect(html).toContain("My Dialog");
    expect(html).toContain('data-testid="modal-traffic-lights"');
    const trafficBlock = html.slice(html.indexOf('data-testid="modal-traffic-lights"'));
    const dots = (trafficBlock.match(/data-testid="traffic-dot"/g) ?? []).length;
    expect(dots).toBe(3);
    expect(html).toContain("content");
  });

  it("renders nothing when closed", () => {
    const html = renderToString(
      <Modal open={false} onClose={() => {}} title="Hidden">
        <p>content</p>
      </Modal>
    );
    expect(html).toBe("");
  });

  it("renders distinct class per size and sets aria-modal", () => {
    const sizes = ["sm", "md", "lg", "xl"] as const;
    const classes = sizes.map((size) =>
      renderToString(
        <Modal open onClose={() => {}} title="t" size={size}>
          x
        </Modal>
      )
    );
    expect(new Set(classes).size).toBe(sizes.length);
    expect(classes[0]).toContain('aria-modal="true"');
  });
});
