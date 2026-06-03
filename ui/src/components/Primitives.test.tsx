import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmptyState, ErrorState, LoadingState } from "./Primitives";

describe("async state primitives", () => {
  it("renders a loading state with a stable status role", () => {
    render(<LoadingState label="Loading providers" />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading providers");
  });

  it("renders an empty state with optional action", () => {
    render(<EmptyState title="No providers" description="Connect a provider account." action={<button>Add</button>} />);

    expect(screen.getByText("No providers")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add" })).toBeInTheDocument();
  });

  it("renders an error state with retry action", () => {
    const retry = vi.fn();

    render(<ErrorState title="Could not load settings" message="request failed" onRetry={retry} />);

    screen.getByRole("button", { name: "Retry" }).click();

    expect(screen.getByText("request failed")).toBeInTheDocument();
    expect(retry).toHaveBeenCalledTimes(1);
  });
});
