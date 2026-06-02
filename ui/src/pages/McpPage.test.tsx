import { render, screen, within } from "@testing-library/react";
import { McpPage } from "./McpPage";

describe("McpPage", () => {
  it("shows per-instance MCP management fields without secrets", () => {
    render(<McpPage />);

    expect(screen.getByRole("heading", { name: "MCP gateway" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add instance" })).toBeInTheDocument();

    const atlassianA = screen.getByRole("row", { name: /atlassian-a/i });
    expect(within(atlassianA).getByText("http")).toBeInTheDocument();
    expect(within(atlassianA).getByText("account-a")).toBeInTheDocument();
    expect(within(atlassianA).getByText("2")).toBeInTheDocument();
    expect(within(atlassianA).getByText("healthy")).toBeInTheDocument();

    const atlassianB = screen.getByRole("row", { name: /atlassian-b/i });
    expect(within(atlassianB).getByText("account-b")).toBeInTheDocument();
    expect(screen.getByText("Complete auth")).toBeInTheDocument();
    expect(screen.queryByText(/token|secret/i)).not.toBeInTheDocument();
  });
});
