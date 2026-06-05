import { render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getAuditPath, type AuditListResponse } from "../api";
import { AuditPage } from "./AuditPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

function makeEntry(overrides: Partial<{ id: number; timestamp: string; actor_api_key_id: string; action: string; target: string; details: string }> = {}) {
  return {
    id: 1,
    timestamp: "2026-06-05T10:00:00Z",
    actor_api_key_id: "key-abc",
    action: "create",
    target: "connections/conn-1",
    details: "created connection openai/main",
    ...overrides
  };
}

const listResponse: AuditListResponse = {
  object: "list",
  data: [
    makeEntry({ id: 2, timestamp: "2026-06-05T11:00:00Z", action: "delete", target: "keys/key-1", details: "deleted api key dev-key", actor_api_key_id: "key-xyz" }),
    makeEntry({ id: 1, timestamp: "2026-06-05T10:00:00Z", action: "create", target: "connections/conn-1", details: "created connection openai/main", actor_api_key_id: "key-abc" })
  ],
  limit: 50,
  offset: 0,
  total: 2
};

describe("AuditPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows loading state while entries are fetched", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<AuditPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading audit log");
  });

  it("renders the empty state when audit log has no rows", async () => {
    const fetch = vi.fn(async () =>
      jsonResponse({ object: "list", data: [], limit: 50, offset: 0, total: 0 })
    );
    vi.stubGlobal("fetch", fetch);

    render(<AuditPage />);

    expect(await screen.findByText("No audit log entries")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(getAuditPath(), expect.objectContaining({ credentials: "same-origin" }));
  });

  it("renders audit entries newest-first with all fields", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(listResponse)));

    render(<AuditPage />);

    const table = await screen.findByRole("table", { name: "Audit log" });
    const rows = within(table).getAllByRole("row");
    // header + 2 data rows
    expect(rows).toHaveLength(3);

    const firstDataRow = rows[1];
    expect(within(firstDataRow).getByText("delete")).toBeInTheDocument();
    expect(within(firstDataRow).getByText("keys/key-1")).toBeInTheDocument();
    expect(within(firstDataRow).getByText("deleted api key dev-key")).toBeInTheDocument();

    const secondDataRow = rows[2];
    expect(within(secondDataRow).getByText("create")).toBeInTheDocument();
    expect(within(secondDataRow).getByText("connections/conn-1")).toBeInTheDocument();
  });

  it("calls the audit endpoint with correct credentials", async () => {
    const fetch = vi.fn(async () => jsonResponse(listResponse));
    vi.stubGlobal("fetch", fetch);

    render(<AuditPage />);

    await screen.findByRole("table", { name: "Audit log" });
    expect(fetch).toHaveBeenCalledWith(
      getAuditPath(),
      expect.objectContaining({ credentials: "same-origin" })
    );
  });

  it("renders error state on fetch failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ error: "audit unavailable" }, { status: 500, statusText: "Server Error" }))
    );

    render(<AuditPage />);

    expect(await screen.findByText("Could not load audit log")).toBeInTheDocument();
    expect(screen.getByText("audit unavailable")).toBeInTheDocument();
  });

  it("renders auth-expired state on 401", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ error: "control-plane auth required" }, { status: 401, statusText: "Unauthorized" }))
    );

    render(<AuditPage />);

    expect(await screen.findByText("Session expired")).toBeInTheDocument();
  });
});
