import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getSettingsPath } from "../api";
import { SettingsSecurityPage } from "./SettingsSecurityPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

describe("SettingsSecurityPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads security-sensitive settings from the real settings endpoint", async () => {
    const fetch = vi.fn(async () =>
      jsonResponse({
        RequireAPIKey: true,
        RTKEnabled: true,
        CavemanEnabled: false,
        CavemanLevel: "full",
        EnableRequestLogs: true,
        ProxyURL: "http://127.0.0.1:8080",
        DataDir: "/tmp/g0router"
      })
    );
    vi.stubGlobal("fetch", fetch);

    render(<SettingsSecurityPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "Settings and security" })).toBeInTheDocument();
    expect(screen.getByLabelText("Require API key")).toBeChecked();
    expect(screen.getByLabelText("Enable request logs")).toBeChecked();
    expect(fetch).toHaveBeenCalledWith(getSettingsPath(), expect.objectContaining({ credentials: "same-origin" }));
  });
});
