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
        require_api_key: true,
        rtk_enabled: true,
        caveman_enabled: false,
        caveman_level: "full",
        enable_request_logs: true,
        proxy_url: "http://127.0.0.1:8080",
        data_dir: "/tmp/g0router",
        log_retention_days: 30
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
