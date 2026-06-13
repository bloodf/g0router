import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { ConfirmModal } from "./confirm-modal";

describe("ConfirmModal", () => {
  it("renders title, message, and both buttons", () => {
    const html = renderToString(
      <ConfirmModal
        open
        title="Delete item?"
        message="This cannot be undone."
        confirmLabel="Delete"
        cancelLabel="Keep"
        onConfirm={() => {}}
        onCancel={() => {}}
      />
    );
    expect(html).toContain("Delete item?");
    expect(html).toContain("This cannot be undone.");
    expect(html).toContain("Delete");
    expect(html).toContain("Keep");
  });

  it("danger variant gives the confirm button danger styling; primary gives primary", () => {
    const danger = renderToString(
      <ConfirmModal
        open
        variant="danger"
        title="t"
        message="m"
        confirmLabel="Go"
        cancelLabel="No"
        onConfirm={() => {}}
        onCancel={() => {}}
      />
    );
    const primary = renderToString(
      <ConfirmModal
        open
        variant="primary"
        title="t"
        message="m"
        confirmLabel="Go"
        cancelLabel="No"
        onConfirm={() => {}}
        onCancel={() => {}}
      />
    );
    expect(danger).toContain("bg-destructive");
    expect(primary).toContain("bg-primary");
    expect(danger).not.toBe(primary);
  });

  it("confirm and cancel buttons carry accessible names from props", () => {
    const html = renderToString(
      <ConfirmModal
        open
        title="t"
        message="m"
        confirmLabel="Proceed"
        cancelLabel="Abort"
        onConfirm={() => {}}
        onCancel={() => {}}
      />
    );
    expect(html).toContain("Proceed");
    expect(html).toContain("Abort");
  });
});
