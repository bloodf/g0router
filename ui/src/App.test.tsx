import { render, screen } from "@testing-library/react";
import App from "./App";

describe("App", () => {
  it("renders the control-plane dashboard shell", () => {
    render(<App />);

    expect(screen.getByRole("heading", { name: "g0router" })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Primary" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByText("Gateway status")).toBeInTheDocument();
    expect(screen.getByText("Provider health")).toBeInTheDocument();
    expect(screen.getByText("Request flow")).toBeInTheDocument();
  });
});
