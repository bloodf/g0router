import { render, screen, within } from "@testing-library/react";
import App from "./App";
import {
  getApiKeysPath,
  getCombosPath,
  getConnectionsPath,
  getMcpServersPath,
  getQuotaPath,
  getSettingsPath,
  getUsagePath
} from "./api";

describe("App", () => {
  it("renders all control-plane navigation destinations", () => {
    render(<App />);

    expect(screen.getByRole("heading", { name: "g0router" })).toBeInTheDocument();
    const primaryNav = screen.getByRole("navigation", { name: "Primary" });
    expect(primaryNav).toBeInTheDocument();

    for (const label of ["Dashboard", "Endpoint", "Providers", "Usage", "Quota", "Combos", "MCP", "Settings"]) {
      expect(within(primaryNav).getByRole("button", { name: label })).toBeInTheDocument();
    }
  });

  it("renders the requested dashboard, provider, usage, quota, and settings sections", () => {
    render(<App />);

    expect(screen.getByRole("heading", { name: "Gateway overview" })).toBeInTheDocument();
    expect(screen.getByText("Endpoint controls")).toBeInTheDocument();
    expect(screen.getByText("Provider connections")).toBeInTheDocument();
    expect(screen.getByText("Usage analytics")).toBeInTheDocument();
    expect(screen.getByText("Quota monitor")).toBeInTheDocument();
    expect(screen.getByText("Combo routing")).toBeInTheDocument();
    expect(screen.getByText("MCP gateway")).toBeInTheDocument();
    expect(screen.getByText("Runtime settings")).toBeInTheDocument();
  });
});

describe("api helpers", () => {
  it("exposes typed management API paths", () => {
    expect(getConnectionsPath()).toBe("/api/connections");
    expect(getApiKeysPath()).toBe("/api/keys");
    expect(getUsagePath()).toBe("/api/usage");
    expect(getQuotaPath("openai")).toBe("/api/usage/quota/openai");
    expect(getCombosPath()).toBe("/api/combos");
    expect(getMcpServersPath()).toBe("/api/mcp/instances");
    expect(getSettingsPath()).toBe("/api/settings");
  });
});
