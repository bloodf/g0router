import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getSettingsPath, type SettingsResponse } from "../api";
import { SettingsPage } from "./SettingsPage";

const settings: SettingsResponse = {
  RequireAPIKey: true,
  RTKEnabled: true,
  CavemanEnabled: false,
  CavemanLevel: "full",
  EnableRequestLogs: false,
  ProxyURL: "http://localhost:8081",
  DataDir: "/var/lib/g0router"
};

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

describe("SettingsPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows the loading state while settings are fetched", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<SettingsPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading settings");
  });

  it("renders the empty state when the settings API returns no body", async () => {
    const fetch = vi.fn(async () => new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetch);

    render(<SettingsPage />);

    expect(await screen.findByText("No runtime settings returned")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(getSettingsPath(), expect.objectContaining({ credentials: "same-origin" }));
  });

  it("loads settings into real controls and saves the full API contract", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getSettingsPath() && method === "GET") {
        return jsonResponse(settings);
      }
      if (path === getSettingsPath() && method === "PUT") {
        return jsonResponse({
          ...settings,
          RequireAPIKey: false,
          CavemanEnabled: true,
          CavemanLevel: "minimal",
          EnableRequestLogs: true,
          ProxyURL: "http://proxy.internal:9000"
        });
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<SettingsPage />);

    expect(await screen.findByLabelText("Require API key")).toBeChecked();
    expect(screen.getByLabelText("RTK enabled")).toBeChecked();
    expect(screen.getByLabelText("Caveman enabled")).not.toBeChecked();
    expect(screen.getByLabelText("Caveman level")).toHaveValue("full");
    expect(screen.getByLabelText("Enable request logs")).not.toBeChecked();
    expect(screen.getByLabelText("Proxy URL")).toHaveValue("http://localhost:8081");
    expect(screen.getByLabelText("Data directory")).toHaveValue("/var/lib/g0router");

    fireEvent.click(screen.getByLabelText("Require API key"));
    fireEvent.click(screen.getByLabelText("Caveman enabled"));
    fireEvent.change(screen.getByLabelText("Caveman level"), { target: { value: "minimal" } });
    fireEvent.click(screen.getByLabelText("Enable request logs"));
    fireEvent.change(screen.getByLabelText("Proxy URL"), { target: { value: "http://proxy.internal:9000" } });
    fireEvent.click(screen.getByRole("button", { name: "Save settings" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        getSettingsPath(),
        expect.objectContaining({
          body: JSON.stringify({
            RequireAPIKey: false,
            RTKEnabled: true,
            CavemanEnabled: true,
            CavemanLevel: "minimal",
            EnableRequestLogs: true,
            ProxyURL: "http://proxy.internal:9000",
            DataDir: "/var/lib/g0router"
          }),
          credentials: "same-origin",
          method: "PUT"
        })
      );
    });
    expect(await screen.findByText("Settings saved")).toBeInTheDocument();
  });

  it("renders recoverable errors and auth-expired errors", async () => {
    const fetch = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: "settings unavailable" }, { status: 500, statusText: "Server Error" }))
      .mockResolvedValueOnce(jsonResponse({ error: "control-plane auth required" }, { status: 403, statusText: "Forbidden" }));
    vi.stubGlobal("fetch", fetch);

    const { unmount } = render(<SettingsPage />);

    expect(await screen.findByText("Could not load settings")).toBeInTheDocument();
    expect(screen.getByText("settings unavailable")).toBeInTheDocument();

    unmount();
    render(<SettingsPage />);

    expect(await screen.findByText("Session expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });
});
